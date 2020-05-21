package gocheck

import (
	"fmt"
	goerrors "github.com/go-errors/errors"
	"go.undefinedlabs.com/scopeagent/instrumentation/testing/config"
	"go.undefinedlabs.com/scopeagent/reflection"
	"go.undefinedlabs.com/scopeagent/runner"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"
	_ "unsafe"

	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent/instrumentation"
	scopetesting "go.undefinedlabs.com/scopeagent/instrumentation/testing"

	chk "gopkg.in/check.v1"
)

type (
	methodType struct {
		reflect.Value
		Info reflect.Method
	}

	resultTracker struct {
		result          chk.Result
		_lastWasProblem bool
		_waiting        int
		_missed         int
		_expectChan     chan *chk.C
		_doneChan       chan *chk.C
		_stopChan       chan bool
	}

	tempDir struct {
		sync.Mutex
		path    string
		counter int
	}

	outputWriter struct {
		m                    sync.Mutex
		writer               io.Writer
		wroteCallProblemLast bool
		Stream               bool
		Verbose              bool
	}

	suiteRunner struct {
		suite                     interface{}
		setUpSuite, tearDownSuite *methodType
		setUpTest, tearDownTest   *methodType
		tests                     []*methodType
		tracker                   *resultTracker
		tempDir                   *tempDir
		keepDir                   bool
		output                    *outputWriter
		reportedProblemLast       bool
		benchTime                 time.Duration
		benchMem                  bool
	}

	testStatus uint32

	testData struct {
		c       *chk.C
		test    func(*chk.C)
		err     *goerrors.Error
		options *runner.Options
	}
)

const (
	testSucceeded testStatus = iota
	testFailed
	testSkipped
	testPanicked
	testFixturePanicked
	testMissed
)

//go:linkname nSRunner gopkg.in/check%2ev1.newSuiteRunner
func nSRunner(suite interface{}, runConf *chk.RunConf) *suiteRunner

//go:linkname lTestingT gopkg.in/check%2ev1.TestingT
func lTestingT(testingT *testing.T)

func init() {
	var nSRunnerPatch *mpatch.Patch
	var err error
	nSRunnerPatch, err = mpatch.PatchMethod(nSRunner, func(suite interface{}, runConf *chk.RunConf) *suiteRunner {
		nSRunnerPatch.Unpatch()
		defer nSRunnerPatch.Patch()
		runnerOptions := runner.GetRunnerOptions()

		r := nSRunner(suite, runConf)
		for idx := range r.tests {
			item := r.tests[idx]
			instTest := func(c *chk.C) {
				if isTestCached(c) {
					writeCachedResult(item)
					return
				}
				test := startTest(item, c)
				defer test.end(c)
				item.Call([]reflect.Value{reflect.ValueOf(c)})
			}

			if runnerOptions != nil {
				instTest = getRunnerTestFunc(instTest, runnerOptions)
			}

			r.tests[idx] = &methodType{reflect.ValueOf(instTest), item.Info}
		}
		return r
	})
	logOnError(err)

	var lTestingTPatch *mpatch.Patch
	lTestingTPatch, err = mpatch.PatchMethod(lTestingT, func(testingT *testing.T) {
		lTestingTPatch.Unpatch()
		defer lTestingTPatch.Patch()

		// We tell the runner to ignore retries on this testing.T
		runner.IgnoreRetries(testingT)

		// We get the instrumented test struct and clean it, that removes the results of that test to be sent to scope
		*scopetesting.GetTest(testingT) = scopetesting.Test{}

		// We call the original go-check TestingT func
		lTestingT(testingT)
	})
	logOnError(err)
}

func logOnError(err error) {
	if err != nil {
		instrumentation.Logger().Println(err)
	}
}

// gets test status
func getTestStatus(c *chk.C) testStatus {
	var status uint32
	if ptr, err := reflection.GetFieldPointerOf(c, "_status"); err == nil {
		status = *(*uint32)(ptr)
	}
	return testStatus(status)
}

// sets test status
func setTestStatus(c *chk.C, status testStatus) {
	sValue := uint32(status)
	if ptr, err := reflection.GetFieldPointerOf(c, "_status"); err == nil {
		*(*uint32)(ptr) = sValue
	}
}

// gets the test reason
func getTestReason(c *chk.C) string {
	if ptr, err := reflection.GetFieldPointerOf(c, "reason"); err == nil {
		return *(*string)(ptr)
	}
	return ""
}

// gets if the test must fail
func getTestMustFail(c *chk.C) bool {
	if ptr, err := reflection.GetFieldPointerOf(c, "mustFail"); err == nil {
		return *(*bool)(ptr)
	}
	return false
}

// gets if the test should retry
func shouldRetry(c *chk.C) bool {
	switch status := getTestStatus(c); status {
	case testFailed, testPanicked, testFixturePanicked:
		if getTestMustFail(c) {
			return false
		}
		return true
	}
	if getTestMustFail(c) {
		return true
	}
	return false
}

// gets the test func with the test runner algorithm
func getRunnerTestFunc(tFunc func(*chk.C), options *runner.Options) func(*chk.C) {
	tData := testData{
		test:    tFunc,
		options: options,
	}
	runnerExecution := func(td *testData) {
		defer func() {
			if rc := recover(); rc != nil {
				// using go-errors to preserve stacktrace
				td.err = goerrors.Wrap(rc, 2)
				td.c.Fail()
			}
		}()
		td.test(td.c)
	}
	return func(c *chk.C) {
		if isTestCached(c) {
			tData.test(c)
			return
		}
		tData.c = c
		run := 1
		for {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			go func() {
				defer wg.Done()
				runnerExecution(&tData)
			}()
			wg.Wait()
			if !shouldRetry(tData.c) {
				break
			}
			if run > tData.options.FailRetries {
				break
			}
			if tData.err != nil {
				if !tData.options.PanicAsFail {
					tData.options.OnPanic(nil, tData.err)
					panic(tData.err.ErrorStack())
				}
				tData.options.Logger.Printf("test '%s' - panic recover: %v",
					tData.c.TestName(), tData.err)
			}
			setTestStatus(tData.c, testSucceeded)
			fmt.Printf("FAIL: Retrying '%s' [%d/%d]\n", c.TestName(), run, tData.options.FailRetries)
			run++
		}
	}
}

// gets if the test is cached
func isTestCached(c *chk.C) bool {
	fqn := c.TestName()
	cachedMap := config.GetCachedTestsMap()
	if _, ok := cachedMap[fqn]; ok {
		instrumentation.Logger().Printf("Test '%v' is cached.", fqn)
		fmt.Print("[SCOPE CACHED] ")
		return true
	}
	instrumentation.Logger().Printf("Test '%v' is not cached.", fqn)
	return false
}
