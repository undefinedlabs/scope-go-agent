package runner

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"testing"
	"time"
	"unsafe"
)

type (
	testRunner struct {
		m             *testing.M
		intTests      *[]testing.InternalTest
		intBenchmarks *[]testing.InternalBenchmark

		tests      *map[string]*testDescriptor
		benchmarks *map[string]*benchmarkDescriptor

		sessionLoader SessionLoader

		repository    string
		branch        string
		commit        string
		serviceName   string
		configuration *testRunnerSession

		exitCode int
	}
	testDescriptor struct {
		test                       testing.InternalTest
		fqn                        string
		ran                        int
		failed                     bool
		flaky                      bool
		error                      bool
		skipped                    bool
		retryOnFailure             bool
		includeStatusInTestResults bool
		added                      bool
		rules                      *runnerRules
	}
	benchmarkDescriptor struct {
		benchmark testing.InternalBenchmark
		fqn       string
		skipped   bool
		added     bool
	}
	internalTestResult struct {
		ran        bool      // Test or benchmark (or one of its subtests) was executed.
		failed     bool      // Test or benchmark has failed.
		skipped    bool      // Test of benchmark has been skipped.
		done       bool      // Test is finished and all subtests have completed.
		finished   bool      // Test function has completed.
		raceErrors int       // number of races detected during test
		name       string    // Name of test or benchmark.
		start      time.Time // Time test or benchmark started
		duration   time.Duration
	}
)

var runner *testRunner
var runnerRegexName *regexp.Regexp

// Runs a test suite
func Run(m *testing.M, repository string, branch string, commit string, serviceName string) int {
	runner := &testRunner{
		m:             m,
		repository:    repository,
		branch:        branch,
		commit:        commit,
		serviceName:   serviceName,
		sessionLoader: &fileSessionLoader{}, // Need to be replaced with the actual configuration loader
	}
	runner.init()
	return runner.Run()
}

// Gets the test name
func GetTestName(name string) string {
	if runnerRegexName == nil {
		runnerRegexName = regexp.MustCompile(`(?m)([\w -:_]*)\/\[runner.[\w:]*]`)
	}
	match := runnerRegexName.FindStringSubmatch(name)
	if match == nil || len(match) == 0 {
		return name
	}
	return match[1]
}

// Runs the test suite
func (r *testRunner) Run() int {
	if r.configuration == nil || r.configuration.Tests == nil {
		return r.m.Run()
	}

	tests := make([]testing.InternalTest, 0)
	benchmarks := make([]testing.InternalBenchmark, 0)

	// Tests and Benchmarks selection and order
	for _, iTest := range r.configuration.Tests {
		if desc, ok := (*r.tests)[iTest.Fqn]; ok {
			if iTest.Skip {
				desc.skipped = true
			} else {
				tests = append(tests, testing.InternalTest{
					Name: desc.fqn,
					F:    r.testProcessor,
				})
				desc.added = true
				desc.rules = iTest.Rules
			}
			desc.retryOnFailure = iTest.RetryOnFailure
			desc.includeStatusInTestResults = iTest.IncludeStatusInTestResults
		}
		if desc, ok := (*r.benchmarks)[iTest.Fqn]; ok {
			if iTest.Skip {
				desc.skipped = true
			} else {
				benchmarks = append(benchmarks, desc.benchmark)
			}
		}
	}

	//Insert missing tests and benchmarks at the end of the list
	for _, value := range *r.tests {
		if value.added || value.skipped {
			continue
		}
		value.added = true
		tests = append(tests, testing.InternalTest{
			Name: value.fqn,
			F:    r.testProcessor,
		})
	}
	for _, value := range *r.benchmarks {
		if value.added || value.skipped {
			continue
		}
		value.added = true
		benchmarks = append(benchmarks, value.benchmark)
	}

	// Replace internal tests and benchmark and Run
	*r.intTests = tests
	*r.intBenchmarks = benchmarks
	r.m.Run()

	return r.exitCode
}

// Internal test processor, each test calls the processor in order to handle retries, process exiting and exitcodes
func (r *testRunner) testProcessor(t *testing.T) {
	t.Helper()
	name := t.Name()
	if item, ok := (*r.tests)[name]; ok {

		//Sets the original test name
		if pointer, err := getFieldPointerOfT(t, "name"); err == nil {
			*(*string)(pointer) = item.test.Name
		}

		run := 1
		rules := r.configuration.Rules
		if item.rules != nil {
			rules = *item.rules
		}
		var rc interface{}

		for {
			var innerTest *testing.T
			title := "Run"
			if run > 1 {
				title = "Retry:" + strconv.Itoa(run-1)
			}
			title = "[runner." + title + "]"
			t.Run(title, func(it *testing.T) {
				defer func() {
					rc = recover()
					if rc != nil {
						it.FailNow()
					}
				}()
				it.Helper()
				innerTest = it
				item.test.F(it)
			})
			innerTestInfo := r.getTestResultsInfo(innerTest)
			if rc != nil {
				if rules.ExitOnError && (!item.retryOnFailure || rules.ErrorRetries == 0) {
					panic(rc)
				}
				fmt.Println("PANIC RECOVER:", rc)
				item.error = true
			}
			item.skipped = innerTestInfo.skipped
			item.ran++
			if item.skipped {
				break
			}
			maxLoop := rules.PassRetries
			// Current run failure
			if innerTestInfo.failed {
				item.failed = true
				if !item.retryOnFailure {
					break
				}
				maxLoop = rules.FailRetries
			}
			// Current run ok but previous with fail -> Flaky
			if !innerTestInfo.failed && item.failed {
				item.failed = false
				item.flaky = true
				maxLoop = rules.FailRetries
			}
			// Already flaky
			if item.flaky {
				maxLoop = rules.FailRetries
			}
			// Error - Panic
			if item.error {
				maxLoop = rules.ErrorRetries
			}

			if run > maxLoop {
				break
			}
			run++
		}
		if item.flaky {
			fmt.Println("*** FLAKY", item.fqn)
		}
		if item.failed && rules.ExitOnFail {
			panic(fmt.Sprintf("Test '%s' has been failed.", item.fqn))
		}
		if item.error && rules.ExitOnError {
			panic(rc)
		}
		if item.includeStatusInTestResults && (item.error || item.failed || item.flaky) {
			r.exitCode = 1
		}
	} else {
		t.FailNow()
	}
}

func (r *testRunner) init() {
	if tPointer, err := r.getFieldPointerOfM("tests"); err == nil {
		r.intTests = (*[]testing.InternalTest)(tPointer)
		r.tests = &map[string]*testDescriptor{}
		for _, test := range *r.intTests {
			fqn := r.getFqnOfTest(test.F)
			(*r.tests)[fqn] = &testDescriptor{
				test:                       test,
				fqn:                        fqn,
				ran:                        0,
				failed:                     false,
				retryOnFailure:             true,
				added:                      false,
				includeStatusInTestResults: true,
			}
		}
	}
	if bPointer, err := r.getFieldPointerOfM("benchmarks"); err == nil {
		r.intBenchmarks = (*[]testing.InternalBenchmark)(bPointer)
		r.benchmarks = &map[string]*benchmarkDescriptor{}

		for _, benchmark := range *r.intBenchmarks {
			fqn := r.getFqnOfBenchmark(benchmark.F)
			(*r.benchmarks)[fqn] = &benchmarkDescriptor{
				benchmark: benchmark,
				fqn:       fqn,
				added:     false,
			}
		}
	}
	r.configuration = r.sessionLoader.LoadSessionConfiguration(r.repository, r.branch, r.commit, r.serviceName)
}

func (r *testRunner) getFieldPointerOfM(fieldName string) (unsafe.Pointer, error) {
	val := reflect.Indirect(reflect.ValueOf(r.m))
	member := val.FieldByName(fieldName)
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return ptrToY, nil
	}
	return nil, errors.New("field can't be retrieved")
}

func (r *testRunner) getFqnOfTest(tFunc func(*testing.T)) string {
	funcVal := runtime.FuncForPC(reflect.ValueOf(tFunc).Pointer())
	return funcVal.Name()
}

func (r *testRunner) getFqnOfBenchmark(bFunc func(*testing.B)) string {
	funcVal := runtime.FuncForPC(reflect.ValueOf(bFunc).Pointer())
	return funcVal.Name()
}

func getFieldPointerOfT(t *testing.T, fieldName string) (unsafe.Pointer, error) {
	val := reflect.Indirect(reflect.ValueOf(t))
	member := val.FieldByName(fieldName)
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return ptrToY, nil
	}
	return nil, errors.New("field can't be retrieved")
}

func (r *testRunner) getTestResultsInfo(t *testing.T) *internalTestResult {
	iTestResults := &internalTestResult{}
	if ptr, err := getFieldPointerOfT(t, "ran"); err == nil {
		iTestResults.ran = *(*bool)(ptr)
	}
	if ptr, err := getFieldPointerOfT(t, "failed"); err == nil {
		iTestResults.failed = *(*bool)(ptr)
	}
	if ptr, err := getFieldPointerOfT(t, "skipped"); err == nil {
		iTestResults.skipped = *(*bool)(ptr)
	}
	if ptr, err := getFieldPointerOfT(t, "done"); err == nil {
		iTestResults.done = *(*bool)(ptr)
	}
	if ptr, err := getFieldPointerOfT(t, "finished"); err == nil {
		iTestResults.finished = *(*bool)(ptr)
	}
	if ptr, err := getFieldPointerOfT(t, "raceErrors"); err == nil {
		iTestResults.raceErrors = *(*int)(ptr)
	}
	if ptr, err := getFieldPointerOfT(t, "name"); err == nil {
		iTestResults.name = *(*string)(ptr)
	}
	if ptr, err := getFieldPointerOfT(t, "start"); err == nil {
		iTestResults.start = *(*time.Time)(ptr)
	}
	if ptr, err := getFieldPointerOfT(t, "duration"); err == nil {
		iTestResults.duration = *(*time.Duration)(ptr)
	}
	return iTestResults
}
