package gocheck

import (
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	_ "unsafe"

	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent/instrumentation"
	scopetesting "go.undefinedlabs.com/scopeagent/instrumentation/testing"
	"go.undefinedlabs.com/scopeagent/instrumentation/testing/config"
	"go.undefinedlabs.com/scopeagent/reflection"
	"go.undefinedlabs.com/scopeagent/runner"
	"go.undefinedlabs.com/scopeagent/tags"

	goerrors "github.com/go-errors/errors"
	"github.com/opentracing/opentracing-go/log"
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
		fn      func(*chk.C)
		err     *goerrors.Error
		options *runner.Options
		test    *Test
		writer  io.Writer
	}

	testLogWriter struct {
		test *Test
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

var (
	testMap      = map[*chk.C]*testData{}
	testMapMutex = sync.RWMutex{}
)

//go:linkname nSRunner gopkg.in/check%2ev1.newSuiteRunner
func nSRunner(suite interface{}, runConf *chk.RunConf) *suiteRunner

//go:linkname lTestingT gopkg.in/check%2ev1.TestingT
func lTestingT(testingT *testing.T)

//go:linkname writeLog gopkg.in/check%2ev1.(*C).writeLog
func writeLog(c *chk.C, buf []byte)

func Init() {
	var nSRunnerPatch *mpatch.Patch
	var err error
	nSRunnerPatch, err = mpatch.PatchMethod(nSRunner, func(suite interface{}, runConf *chk.RunConf) *suiteRunner {
		nSRunnerPatch.Unpatch()
		defer nSRunnerPatch.Patch()
		runnerOptions := runner.GetRunnerOptions()

		tWriter := &testLogWriter{}

		r := nSRunner(suite, runConf)
		for idx := range r.tests {
			item := r.tests[idx]
			tData := &testData{options: runnerOptions, writer: tWriter}

			instTest := func(c *chk.C) {
				if isTestCached(c) {
					writeCachedResult(item)
					return
				}
				tData.c = c
				testMapMutex.Lock()
				testMap[c] = tData
				setLogWriter(c, &tData.writer)
				testMapMutex.Unlock()
				defer func() {
					testMapMutex.Lock()
					delete(testMap, c)
					testMapMutex.Unlock()
				}()

				test := startTest(item, c)
				tData.test = test
				tData.writer.(*testLogWriter).test = test
				defer test.end(c)
				item.Call([]reflect.Value{reflect.ValueOf(c)})
			}
			tData.fn = instTest

			if runnerOptions != nil {
				instTest = getRunnerTestFunc(tData)
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

func setLogWriter(c *chk.C, w *io.Writer) {
	if ptr, err := reflection.GetFieldPointerOf(c, "logw"); err == nil {
		cWriter := *(*io.Writer)(ptr)
		if cWriter == nil {
			*(*io.Writer)(ptr) = *w
		} else if cWriter != *w {
			*w = io.MultiWriter(cWriter, *w)
			*(*io.Writer)(ptr) = *w
		}
	}
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
func getRunnerTestFunc(tData *testData) func(*chk.C) {
	runnerExecution := func(td *testData) {
		defer func() {
			if rc := recover(); rc != nil {
				// using go-errors to preserve stacktrace
				td.err = goerrors.Wrap(rc, 2)
				td.c.Fail()
			}
		}()
		td.fn(td.c)
	}
	return func(c *chk.C) {
		if isTestCached(c) {
			tData.fn(c)
			return
		}
		tData.c = c
		run := 1
		for {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			go func() {
				defer wg.Done()
				runnerExecution(tData)
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

// Write data to a test event
func (w *testLogWriter) Write(p []byte) (n int, err error) {
	if w.test == nil || w.test.span == nil {
		return 0, nil
	}

	pcs := make([]uintptr, 64)
	count := runtime.Callers(2, pcs)
	pcs = pcs[:count]
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		name := frame.Function

		// If the frame is not in the gopkg.in/check we skip it
		if !strings.Contains(name, "gopkg.in/check") {
			if !more {
				break
			}
			continue
		}

		// we only log if in the stackframe we see the log or lof method
		if strings.HasSuffix(name, "log") || strings.HasSuffix(name, "logf") {
			frame, more = frames.Next()
			if !more {
				break
			}
			frame, more = frames.Next()
			helperName := frame.Function
			if strings.HasSuffix(helperName, "logCaller") {
				break
			}
			eventType := tags.LogEvent
			eventLevel := tags.LogLevel_INFO
			_, file, line, _ := getCallerInsideSourceRoot(frame, more, frames)
			source := fmt.Sprintf("%s:%d", file, line)
			if strings.HasSuffix(helperName, "Fatal") {
				eventType = tags.EventTestFailure
				eventLevel = tags.LogLevel_ERROR

			} else if strings.HasSuffix(helperName, "Error") || strings.HasSuffix(helperName, "Errorf") {
				eventLevel = tags.LogLevel_ERROR
			}
			fields := []log.Field{
				log.String(tags.EventType, eventType),
				log.String(tags.EventMessage, string(p)),
				log.String(tags.LogEventLevel, eventLevel),
				log.String("log.logger", "gopkg.in/check.v1"),
			}
			if file != "" {
				fields = append(fields, log.String(tags.EventSource, source))
			}
			w.test.span.LogFields(fields...)
			return len(p), nil
		}
		if !more {
			break
		}
	}
	return 0, nil
}

//go:noinline
func getCallerInsideSourceRoot(frame runtime.Frame, more bool, frames *runtime.Frames) (pc uintptr, file string, line int, ok bool) {
	isWindows := runtime.GOOS == "windows"
	sourceRoot := instrumentation.GetSourceRoot()
	if isWindows {
		sourceRoot = strings.ToLower(sourceRoot)
	}
	for {
		file := filepath.Clean(frame.File)
		dir := filepath.Dir(file)
		if isWindows {
			dir = strings.ToLower(dir)
		}
		if strings.Index(dir, sourceRoot) != -1 {
			return frame.PC, file, frame.Line, true
		}
		if !more {
			break
		}
		frame, more = frames.Next()
	}
	return
}
