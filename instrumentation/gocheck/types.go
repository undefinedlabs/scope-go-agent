package gocheck

import (
	"fmt"
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

///go:linkname nSRunnerRun gopkg.in/check%2ev1.(*suiteRunner).run
//func nSRunnerRun(runner *suiteRunner) *chk.Result

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
				test := startTest(item, c)
				defer test.end(c)
				item.Call([]reflect.Value{reflect.ValueOf(c)})
			}

			if runnerOptions != nil {
				instTest = getRunnerTest(instTest)
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

func getTestStatus(c *chk.C) testStatus {
	var status uint32
	if ptr, err := reflection.GetFieldPointerOf(c, "_status"); err == nil {
		status = *(*uint32)(ptr)
	}
	return testStatus(status)
}

func shouldRetry(c *chk.C) bool {
	switch status := getTestStatus(c); status {
	case testFailed:
	case testPanicked:
	case testFixturePanicked:
		return true
	}
	return false
}

func getRunnerTest(tFunc func(*chk.C)) func(*chk.C) {
	instTest := tFunc

	runnerExecution := func(c *chk.C) {
		fmt.Println(c.TestName(), "Start Runner")
		defer func() {
			fmt.Println(c.TestName(), "End Runner")
		}()
		instTest(c)
	}
	return func(c *chk.C) {
		runnerExecution(c)
	}
}
