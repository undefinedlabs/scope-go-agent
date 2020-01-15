package runner

import (
	"errors"
	"io/ioutil"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"unsafe"
)

type (
	testRunner struct {
		m                *testing.M
		failRetriesCount int
		exitOnError      bool
		logger           *log.Logger
		failed           bool
		failedlock       *sync.Mutex
	}
	testDescriptor struct {
		runner  *testRunner
		test    testing.InternalTest
		ran     int
		failed  bool
		flaky   bool
		error   bool
		skipped bool
	}
)

var runner *testRunner
var runnerRegexName = regexp.MustCompile(`(?m)([\w -:_]*)\/\[runner.[\w:]*]\/\[group](\/[\w -:_]*)?`)

// Gets the test name
func GetOriginalTestName(name string) string {
	match := runnerRegexName.FindStringSubmatch(name)
	if match == nil || len(match) == 0 {
		return name
	}
	return match[1] + match[2]
}

// Runs a test suite
func Run(m *testing.M, exitOnError bool, failRetriesCount int, logger *log.Logger) int {
	if logger == nil {
		logger = log.New(ioutil.Discard, "", 0)
	}
	runner := &testRunner{
		m:                m,
		exitOnError:      exitOnError,
		failRetriesCount: failRetriesCount,
		logger:           logger,
		failed:           false,
		failedlock:       &sync.Mutex{},
	}
	runner.init()
	return runner.m.Run()
}

// Initialize test runner, replace the internal test with an indirection
func (r *testRunner) init() {
	if tPointer, err := getFieldPointerOfM(r.m, "tests"); err == nil {
		tests := make([]testing.InternalTest, 0)
		internalTests := (*[]testing.InternalTest)(tPointer)
		for _, test := range *internalTests {
			td := &testDescriptor{
				runner: r,
				test:   test,
				ran:    0,
				failed: false,
			}
			tests = append(tests, testing.InternalTest{
				Name: test.Name,
				F:    td.run,
			})
		}
		// Replace internal tests
		*internalTests = tests
	}
}

// Internal test runner, each test calls this method in order to handle retries and process exiting
func (td *testDescriptor) run(t *testing.T) {
	run := 1
	maxRetries := td.runner.failRetriesCount
	exitOnError := td.runner.exitOnError
	var rc interface{}

	for {
		var innerTest *testing.T
		title := "Run"
		if run > 1 {
			title = "Retry:" + strconv.Itoa(run-1)
		}
		title = "[runner." + title + "]"
		t.Run(title, func(it *testing.T) {
			// We need to run another subtest in order to support t.Parallel()
			// https://stackoverflow.com/a/53950628
			it.Run("[group]", func(gt *testing.T) {
				defer func() {
					rc = recover()
					if rc != nil {
						gt.FailNow()
					}
				}()
				innerTest = gt
				td.test.F(gt)
			})
		})
		if rc != nil {
			if exitOnError {
				panic(rc)
			}
			td.runner.logger.Println("PANIC RECOVER:", rc)
			td.error = true
		}
		td.skipped = innerTest.Skipped()
		if td.skipped {
			t.SkipNow()
			break
		}
		td.ran++

		if innerTest.Failed() {
			// Current run failure
			td.failed = true
		} else if td.failed {
			// Current run ok but previous run with fail -> Flaky
			td.failed = false
			td.flaky = true
			td.runner.logger.Println("FLAKY TEST DETECTED:", t.Name())
			break
		} else {
			// Current run ok and previous run (if any) not marked as failed
			break
		}

		if run > maxRetries {
			break
		}
		run++
	}

	// Set the global failed flag
	td.runner.failedlock.Lock()
	td.runner.failed = td.runner.failed || td.failed || td.error
	setTestFailureFlag(getTestParent(t), td.runner.failed)
	td.runner.failedlock.Unlock()

	if td.error && exitOnError {
		// If after all recovers and retries the test finish with error and we have the exitOnError flag,
		// we panic with the latest recovered data
		panic(rc)
	}
	if !td.error && !td.failed {
		// If test pass or flaky
		setTestFailureFlag(t, false)
	}
}

// Sets the test failure flag
func setTestFailureFlag(t *testing.T, value bool) {
	if ptr, err := getFieldPointerOfT(t, "failed"); err == nil {
		*(*bool)(ptr) = value
	}
}

// Gets the parent from a test
func getTestParent(t *testing.T) *testing.T {
	if parentPtr, err := getFieldPointerOfT(t, "parent"); err == nil {
		parentTPointer := (**testing.T)(parentPtr)
		if parentTPointer != nil && *parentTPointer != nil {
			return *parentTPointer
		}
	}
	return nil
}

// Gets a pointer of a private or public field in a testing.M struct
func getFieldPointerOfM(m *testing.M, fieldName string) (unsafe.Pointer, error) {
	val := reflect.Indirect(reflect.ValueOf(m))
	member := val.FieldByName(fieldName)
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return ptrToY, nil
	}
	return nil, errors.New("field can't be retrieved")
}

// Gets a pointer of a private or public field in a testing.T struct
func getFieldPointerOfT(t *testing.T, fieldName string) (unsafe.Pointer, error) {
	val := reflect.Indirect(reflect.ValueOf(t))
	member := val.FieldByName(fieldName)
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return ptrToY, nil
	}
	return nil, errors.New("field can't be retrieved")
}
