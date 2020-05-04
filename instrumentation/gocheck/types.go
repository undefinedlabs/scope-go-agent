package gocheck

import (
	"io"
	"reflect"
	"sync"
	"time"
	_ "unsafe"

	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent/instrumentation"

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
)

//go:linkname nSRunner gopkg.in/check%2ev1.newSuiteRunner
func nSRunner(suite interface{}, runConf *chk.RunConf) *suiteRunner

///go:linkname nSRunnerRun gopkg.in/check%2ev1.(*suiteRunner).run
//func nSRunnerRun(runner *suiteRunner) *chk.Result

func init() {
	var patch *mpatch.Patch
	var err error
	nsr := nSRunner
	patch, err = mpatch.PatchMethod(nsr, func(suite interface{}, runConf *chk.RunConf) *suiteRunner {
		patch.Unpatch()
		defer patch.Patch()

		r := nsr(suite, runConf)
		for idx := range r.tests {
			item := r.tests[idx]
			nFunc := func(c *chk.C) {
				startTest(item, c)
				defer endTest(item, c)
				item.Call([]reflect.Value{reflect.ValueOf(c)})
			}
			r.tests[idx] = &methodType{reflect.ValueOf(nFunc), item.Info}
		}
		return r
	})
	logOnError(err)
}

func logOnError(err error) {
	if err != nil {
		instrumentation.Logger().Println(err)
	}
}
