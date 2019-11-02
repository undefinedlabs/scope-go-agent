package runner

import (
	"errors"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"unsafe"
)

type (
	testRunner struct {
		m             *testing.M
		intTests      *[]testing.InternalTest
		intBenchmarks *[]testing.InternalBenchmark

		tests      *map[string]*testDescriptor
		benchmarks *map[string]*benchmarkDescriptor

		repository    string
		branch        string
		commit        string
		serviceName   string
		configuration *testRunnerSession

		exitCode int
	}
	testDescriptor struct {
		test           testing.InternalTest
		fqn            string
		ran            int
		failed         bool
		skipped        bool
		retryOnFailure bool
		added          bool
	}
	benchmarkDescriptor struct {
		benchmark      testing.InternalBenchmark
		fqn            string
		ran            int
		failed         bool
		skipped        bool
		retryOnFailure bool
		added          bool
	}
)

var runner *testRunner
var cfgLoader sessionLoader

func Run(m *testing.M, repository string, branch string, commit string, serviceName string) int {
	cfgLoader = &dummySessionLoader{} // Need to be replaced with the actual configuration loader
	runner := &testRunner{
		m:           m,
		repository:  repository,
		branch:      branch,
		commit:      commit,
		serviceName: serviceName,
	}
	runner.init()
	return runner.Run()
}

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
					F: r.testProcessor,
				})
				desc.added = true
			}
			desc.retryOnFailure = iTest.RetryOnFailure
		}
		if desc, ok := (*r.benchmarks)[iTest.Fqn]; ok {
			if iTest.Skip {
				desc.skipped = true
			} else {
				benchmarks = append(benchmarks, desc.benchmark)
				desc.added = true
			}
			desc.retryOnFailure = iTest.RetryOnFailure
		}
	}
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
	*r.intTests = tests
	*r.intBenchmarks = benchmarks
	r.exitCode = r.m.Run()

	return r.exitCode
}

func (r *testRunner) testProcessor(t *testing.T) {
	t.Helper()
	if item, ok := (*r.tests)[t.Name()]; ok {
		run := 1
		for {
			var innerTest *testing.T
			t.Run("Run:" + strconv.Itoa(run), func(it *testing.T) {
				it.Helper()
				innerTest = it
				item.test.F(it)
			})
			r.getTestResultsInfo(innerTest)
			//fmt.Println(innerTest)
			run++
			if run > 4 {
				break
			}
		}

	} else {
		t.FailNow()
	}
}

func (r *testRunner) init() {
	if tPointer, err := r.getFieldPointer("tests"); err == nil {
		r.intTests = (*[]testing.InternalTest)(tPointer)
		r.tests = &map[string]*testDescriptor{}
		for _, test := range *r.intTests {
			fqn := r.getFqnOfTest(test.F)
			(*r.tests)[fqn] = &testDescriptor{
				test:           test,
				fqn:            fqn,
				ran:            0,
				failed:         false,
				retryOnFailure: true,
				added:          false,
			}
		}
	}
	if bPointer, err := r.getFieldPointer("benchmarks"); err == nil {
		r.intBenchmarks = (*[]testing.InternalBenchmark)(bPointer)
		r.benchmarks = &map[string]*benchmarkDescriptor{}

		for _, benchmark := range *r.intBenchmarks {
			fqn := r.getFqnOfBenchmark(benchmark.F)
			(*r.benchmarks)[fqn] = &benchmarkDescriptor{
				benchmark:      benchmark,
				fqn:            fqn,
				ran:            0,
				failed:         false,
				retryOnFailure: true,
				added:          false,
			}
		}
	}
	r.configuration = cfgLoader.LoadSessionConfiguration(r.repository, r.branch, r.commit, r.serviceName)
}

func (r *testRunner) getFieldPointer(fieldName string) (unsafe.Pointer, error) {
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

func (r *testRunner) getTestResultsInfo(t *testing.T) {

}