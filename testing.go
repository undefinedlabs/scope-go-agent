package scopeagent

import (
	"bufio"
	"context"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"go.undefinedlabs.com/scopeagent/ast"
	"go.undefinedlabs.com/scopeagent/errors"
	log2 "log"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
)

type Test struct {
	testing.TB
	ctx              context.Context
	span             opentracing.Span
	t                *testing.T
	stdOut           *stdIO
	stdErr           *stdIO
	loggerStdIO      *stdIO
	failReason       string
	failReasonSource string
	skipReason       string
	skipReasonSource string
}
type stdIO struct {
	oldIO     *os.File
	readPipe  *os.File
	writePipe *os.File
	sync      *sync.WaitGroup
}

// Instrument a test
func InstrumentTest(t *testing.T, f func(ctx context.Context, t *testing.T)) {
	test := StartTest(t)
	defer test.End()
	f(test.Context(), t)
}

// Starts a new test
func StartTest(t *testing.T) *Test {
	pc, _, _, _ := runtime.Caller(1)
	return startTestFromCaller(t, pc)
}

// Starts a new test with and uses the caller pc info for Name and Suite
func startTestFromCaller(t *testing.T, pc uintptr) *Test {
	fullTestName := t.Name()
	testNameSlash := strings.IndexByte(fullTestName, '/')
	if testNameSlash < 0 {
		testNameSlash = len(fullTestName)
	}
	funcName := fullTestName[:testNameSlash]

	funcFullName := runtime.FuncForPC(pc).Name()
	funcNameIndex := strings.LastIndex(funcFullName, funcName)
	if funcNameIndex < 1 {
		funcNameIndex = len(funcFullName)
	}
	packageName := funcFullName[:funcNameIndex-1]

	sourceBounds := ast.GetFuncSourceForName(pc, funcName)
	var testCode string
	if sourceBounds != nil {
		testCode = fmt.Sprintf("%s:%d:%d", sourceBounds.File, sourceBounds.Start.Line, sourceBounds.End.Line)
	}

	span, ctx := opentracing.StartSpanFromContext(context.Background(), t.Name(), opentracing.Tags{
		"span.kind":  "test",
		"test.name":  fullTestName,
		"test.suite": packageName,
		"test.code":  testCode,
	})
	span.SetBaggageItem("trace.kind", "test")

	// Replaces stdout and stderr
	loggerStdIO := newStdIO(&os.Stderr, false)
	stdOut := newStdIO(&os.Stdout, true)
	stdErr := newStdIO(&os.Stderr, true)
	log2.SetOutput(loggerStdIO.writePipe)

	test := &Test{
		ctx:         ctx,
		span:        span,
		t:           t,
		stdOut:      stdOut,
		stdErr:      stdErr,
		loggerStdIO: loggerStdIO,
	}

	// Starts stdIO pipe handlers
	if test.stdOut != nil {
		go stdIOHandler(test, test.stdOut, false)
	}
	if test.stdErr != nil {
		go stdIOHandler(test, test.stdErr, true)
	}
	if test.loggerStdIO != nil {
		go loggerStdIOHandler(test, test.loggerStdIO)
	}

	return test
}

// Ends the current test
func (test *Test) End() {
	if r := recover(); r != nil {
		test.span.SetTag("test.status", "ERROR")
		test.stdOut.restore(&os.Stdout, true)
		test.stdErr.restore(&os.Stderr, true)
		test.loggerStdIO.restore(&os.Stderr, false)
		log2.SetOutput(os.Stderr)
		test.span.SetTag("error", true)
		errors.LogError(test.span, r, 1)
		test.span.Finish()
		_ = GlobalAgent.Flush()
		GlobalAgent.printReport()
		panic(r)
	}
	if test.t.Failed() {
		test.span.SetTag("test.status", "FAIL")
		test.span.SetTag("error", true)
		if test.failReason != "" {
			test.span.LogFields(
				log.String(EventType, EventTestFailure),
				log.String(EventMessage, test.failReason),
				log.String(EventSource, test.failReasonSource),
			)
		} else {
			test.span.LogFields(
				log.String(EventType, EventTestFailure),
				log.String(EventMessage, "Test has failed"),
			)
		}
	} else if test.t.Skipped() {
		test.span.SetTag("test.status", "SKIP")
		if test.skipReason != "" {
			test.span.LogFields(
				log.String(EventType, EventTestSkip),
				log.String(EventMessage, test.skipReason),
				log.String(EventSource, test.skipReasonSource),
			)
		} else {
			test.span.LogFields(
				log.String(EventType, EventTestSkip),
				log.String(EventMessage, "Test has skipped"),
			)
		}
	} else {
		test.span.SetTag("test.status", "PASS")
	}

	test.stdOut.restore(&os.Stdout, true)
	test.stdErr.restore(&os.Stderr, true)
	test.loggerStdIO.restore(&os.Stderr, false)
	log2.SetOutput(os.Stderr)
	test.span.Finish()
}

// Gets the test context
func (test *Test) Context() context.Context {
	return test.ctx
}

// Runs a sub test
func (test *Test) Run(name string, f func(t *testing.T)) {
	pc, _, _, _ := runtime.Caller(1)
	test.t.Run(name, func(childT *testing.T) {
		childTest := startTestFromCaller(childT, pc)
		defer childTest.End()
		f(childT)
	})
}

// Handles the StdIO pipe for stdout and stderr
func stdIOHandler(test *Test, stdio *stdIO, isError bool) {
	stdio.sync.Add(1)
	defer stdio.sync.Done()
	reader := bufio.NewReader(stdio.readPipe)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if isError {
			test.span.LogFields(
				log.String(EventType, LogEvent),
				log.String(EventMessage, line),
				log.String(LogEventLevel, LogLevel_ERROR),
			)
		} else {
			test.span.LogFields(
				log.String(EventType, LogEvent),
				log.String(EventMessage, line),
				log.String(LogEventLevel, LogLevel_VERBOSE),
			)
		}
		_, _ = stdio.oldIO.WriteString(line)
	}
}

// Handles the StdIO for a logger
func loggerStdIOHandler(test *Test, stdio *stdIO) {
	stdio.sync.Add(1)
	defer stdio.sync.Done()
	reader := bufio.NewReader(stdio.readPipe)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		nLine := line
		flags := log2.Flags()
		sliceCount := 0
		if flags&(log2.Ldate|log2.Ltime|log2.Lmicroseconds) != 0 {
			if flags&log2.Ldate != 0 {
				sliceCount = sliceCount + 11
			}
			if flags&(log2.Ltime|log2.Lmicroseconds) != 0 {
				sliceCount = sliceCount + 9
				if flags&log2.Lmicroseconds != 0 {
					sliceCount = sliceCount + 7
				}
			}
			nLine = nLine[sliceCount:]
		}
		test.span.LogFields(
			log.String(EventType, LogEvent),
			log.String(EventMessage, nLine),
			log.String(LogEventLevel, LogLevel_VERBOSE),
			log.String("log.logger", "std.Logger"),
		)

		if stdio.oldIO != nil {
			_, _ = stdio.oldIO.WriteString(line)
		}
	}
}

// Creates and replaces a file instance with a pipe
func newStdIO(file **os.File, replace bool) *stdIO {
	rPipe, wPipe, err := os.Pipe()
	if err == nil {
		stdIO := &stdIO{
			readPipe:  rPipe,
			writePipe: wPipe,
			sync:      new(sync.WaitGroup),
		}
		if file != nil {
			stdIO.oldIO = *file
			if replace {
				*file = wPipe
			}
		}
		return stdIO
	}
	return nil
}

// Restores the old file instance
func (stdIO *stdIO) restore(file **os.File, replace bool) {
	if file != nil {
		_ = (*file).Sync()
	}
	_ = stdIO.readPipe.Sync()
	_ = stdIO.writePipe.Close()
	_ = stdIO.readPipe.Close()
	stdIO.sync.Wait()
	if replace && file != nil {
		*file = stdIO.oldIO
	}
}

// ***************************
// TB interface implementation
func (test *Test) private() {}
func (test *Test) Error(args ...interface{}) {
	var source string
	if _, file, line, ok := runtime.Caller(1); ok == true {
		source = fmt.Sprintf("%s:%d", file, line)
	}
	test.span.LogFields(
		log.String(EventType, LogEvent),
		log.String(EventMessage, fmt.Sprint(args...)),
		log.String(EventSource, source),
		log.String(LogEventLevel, LogLevel_ERROR),
		log.String("log.internal_level", "Error"),
		log.String("log.logger", "ScopeAgent"),
	)
	test.t.Error(args...)
}
func (test *Test) Errorf(format string, args ...interface{}) {
	var source string
	if _, file, line, ok := runtime.Caller(1); ok == true {
		source = fmt.Sprintf("%s:%d", file, line)
	}
	test.span.LogFields(
		log.String(EventType, LogEvent),
		log.String(EventMessage, fmt.Sprintf(format, args...)),
		log.String(EventSource, source),
		log.String(LogEventLevel, LogLevel_ERROR),
		log.String("log.internal_level", "Error"),
		log.String("log.logger", "ScopeAgent"),
	)
	test.t.Errorf(format, args...)
}
func (test *Test) Fail() {
	test.t.Fail()
}
func (test *Test) FailNow() {
	test.t.FailNow()
}
func (test *Test) Failed() bool {
	return test.t.Failed()
}
func (test *Test) Fatal(args ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok == true {
		test.failReasonSource = fmt.Sprintf("%s:%d", file, line)
	}
	test.failReason = fmt.Sprint(args...)
	test.t.Fatal(args...)
}
func (test *Test) Fatalf(format string, args ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok == true {
		test.failReasonSource = fmt.Sprintf("%s:%d", file, line)
	}
	test.failReason = fmt.Sprintf(format, args...)
	test.t.Fatalf(format, args...)
}
func (test *Test) Log(args ...interface{}) {
	var source string
	if _, file, line, ok := runtime.Caller(1); ok == true {
		source = fmt.Sprintf("%s:%d", file, line)
	}
	test.span.LogFields(
		log.String(EventType, LogEvent),
		log.String(EventMessage, fmt.Sprint(args...)),
		log.String(EventSource, source),
		log.String(LogEventLevel, LogLevel_INFO),
		log.String("log.internal_level", "Log"),
		log.String("log.logger", "ScopeAgent"),
	)
	test.t.Log(args...)
}
func (test *Test) Logf(format string, args ...interface{}) {
	var source string
	if _, file, line, ok := runtime.Caller(1); ok == true {
		source = fmt.Sprintf("%s:%d", file, line)
	}
	test.span.LogFields(
		log.String(EventType, LogEvent),
		log.String(EventMessage, fmt.Sprintf(format, args...)),
		log.String(EventSource, source),
		log.String(LogEventLevel, LogLevel_INFO),
		log.String("log.internal_level", "Log"),
		log.String("log.logger", "ScopeAgent"),
	)
	test.t.Logf(format, args...)
}
func (test *Test) Name() string {
	return test.t.Name()
}
func (test *Test) Skip(args ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok == true {
		test.skipReasonSource = fmt.Sprintf("%s:%d", file, line)
	}
	test.skipReason = fmt.Sprint(args...)
	test.t.Skip(args...)
}
func (test *Test) SkipNow() {
	test.t.SkipNow()
}
func (test *Test) Skipf(format string, args ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok == true {
		test.skipReasonSource = fmt.Sprintf("%s:%d", file, line)
	}
	test.skipReason = fmt.Sprintf(format, args...)
	test.t.Skipf(format, args...)
}
func (test *Test) Skipped() bool {
	return test.t.Skipped()
}
func (test *Test) Helper() {
	test.t.Helper()
}
