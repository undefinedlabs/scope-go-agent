package scopeagent

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/opentracing/opentracing-go"
	oLog "github.com/opentracing/opentracing-go/log"
	"github.com/undefinedlabs/go-agent/contexts"
	"github.com/undefinedlabs/go-agent/errors"
	"github.com/undefinedlabs/go-agent/monpatch"
)

var (
	patcher sync.Once
)

const currentTestKey = "currentTest"

type Test struct {
	ctx    context.Context
	span   opentracing.Span
	t      *testing.T
	stdOut *StdIO
	stdErr *StdIO
}
type StdIO struct {
	oldIO     *os.File
	readPipe  *os.File
	writePipe *os.File
	sync      *sync.WaitGroup
}

func InstrumentTest(t *testing.T, f func(ctx context.Context, t *testing.T)) {
	test := StartTest(t)
	defer test.End()
	f(test.Context(), t)
}

func StartTest(t *testing.T) *Test {
	patchLogger()
	pc, _, _, _ := runtime.Caller(1)
	parts := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	pl := len(parts)
	packageName := ""
	funcName := parts[pl-1]

	if parts[pl-2][0] == '(' {
		funcName = parts[pl-2] + "." + funcName
		packageName = strings.Join(parts[0:pl-2], ".")
	} else {
		packageName = strings.Join(parts[0:pl-1], ".")
	}

	span, ctx := opentracing.StartSpanFromContext(context.Background(), t.Name(), opentracing.Tags{
		"span.kind":  "test",
		"test.name":  funcName,
		"test.suite": packageName,
	})
	span.SetBaggageItem("trace.kind", "test")

	// Replaces stdout and stderr
	stdOut := newStdIO(&os.Stdout)
	stdErr := newStdIO(&os.Stderr)
	log.SetOutput(stdOut.writePipe)

	test := &Test{
		ctx:    ctx,
		span:   span,
		t:      t,
		stdOut: stdOut,
		stdErr: stdErr,
	}
	contexts.SetGoRoutineData(currentTestKey, test)

	// Starts stdIO pipe handlers
	if test.stdOut != nil {
		go stdIOHandler(test, test.stdOut, false)
	}
	if test.stdErr != nil {
		go stdIOHandler(test, test.stdErr, true)
	}

	return test
}

func (test *Test) End() {
	if r := recover(); r != nil {
		test.span.SetTag("test.status", "ERROR")
		test.stdOut.restore(&os.Stdout)
		test.stdErr.restore(&os.Stderr)
		test.span.SetTag("error", true)
		errors.LogError(test.span, r, 1)
		test.span.Finish()
		_ = GlobalAgent.Flush()
		panic(r)
	}
	if test.t.Failed() {
		test.span.SetTag("test.status", "FAIL")
		test.span.SetTag("error", true)
		test.span.LogFields(
			oLog.String(EventType, EventTestFailure),
			oLog.String(EventMessage, "Test has failed"),
		)
	} else if test.t.Skipped() {
		test.span.SetTag("test.status", "SKIP")
	} else {
		test.span.SetTag("test.status", "PASS")
	}
	test.stdOut.restore(&os.Stdout)
	test.stdErr.restore(&os.Stderr)
	log.SetOutput(os.Stderr)
	test.span.Finish()
	contexts.SetGoRoutineData(currentTestKey, nil)
}

func (test *Test) Context() context.Context {
	return test.ctx
}

func patchLogger() {

	patcher.Do(func() {

		commonType := reflect.ValueOf(testing.T{}).FieldByName("common").Type()
		commonTypeReference := reflect.New(commonType).Type()

		var traceFatalGuard *monpatch.PatchGuard
		traceFatalGuard = monpatch.PatchInstanceMethod(commonTypeReference, "Fatal",
			func(t *testing.T, args ...interface{}) {
				traceFatalGuard.Unpatch()
				defer traceFatalGuard.Restore()

				currentTest := contexts.GetGoRoutineData(currentTestKey)
				if currentTest != nil {
					test := currentTest.(*Test)
					var source string
					if _, file, line, ok := runtime.Caller(1); ok == true {
						source = fmt.Sprintf("%s:%d", file, line)
					}
					test.span.LogFields(
						oLog.String("event", "log"),
						oLog.String("message", fmt.Sprint(args)),
						oLog.String(EventSource, source),
						oLog.String("log.level", "ERROR"),
						oLog.String("log.logger", "testing.T"),
					)
					test.span.SetTag("test.status", "FAIL")
					test.span.SetTag("error", true)
				}

				t.Fatal(args)
			})

		var traceFatalfGuard *monpatch.PatchGuard
		traceFatalfGuard = monpatch.PatchInstanceMethod(commonTypeReference, "Fatalf",
			func(t *testing.T, format string, args ...interface{}) {
				traceFatalfGuard.Unpatch()
				defer traceFatalfGuard.Restore()

				currentTest := contexts.GetGoRoutineData(currentTestKey)
				if currentTest != nil {
					test := currentTest.(*Test)
					var source string
					if _, file, line, ok := runtime.Caller(1); ok == true {
						source = fmt.Sprintf("%s:%d", file, line)
					}
					test.span.LogFields(
						oLog.String("event", "log"),
						oLog.String("message", fmt.Sprintf(format, args)),
						oLog.String(EventSource, source),
						oLog.String("log.level", "ERROR"),
						oLog.String("log.logger", "testing.T"),
					)
					test.span.SetTag("test.status", "FAIL")
					test.span.SetTag("error", true)
				}

				t.Fatalf(format, args)
			})

		var traceErrorGuard *monpatch.PatchGuard
		traceErrorGuard = monpatch.PatchInstanceMethod(commonTypeReference, "Error",
			func(t *testing.T, args ...interface{}) {
				traceErrorGuard.Unpatch()
				defer traceErrorGuard.Restore()

				currentTest := contexts.GetGoRoutineData(currentTestKey)
				if currentTest != nil {
					test := currentTest.(*Test)
					var source string
					if _, file, line, ok := runtime.Caller(1); ok == true {
						source = fmt.Sprintf("%s:%d", file, line)
					}
					test.span.LogFields(
						oLog.String("event", "log"),
						oLog.String("message", fmt.Sprint(args)),
						oLog.String(EventSource, source),
						oLog.String("log.level", "ERROR"),
						oLog.String("log.logger", "testing.T"),
					)
				}

				t.Error(args)
			})

		var traceErrorfGuard *monpatch.PatchGuard
		traceErrorfGuard = monpatch.PatchInstanceMethod(commonTypeReference, "Errorf",
			func(t *testing.T, format string, args ...interface{}) {
				traceErrorfGuard.Unpatch()
				defer traceErrorfGuard.Restore()

				currentTest := contexts.GetGoRoutineData(currentTestKey)
				if currentTest != nil {
					test := currentTest.(*Test)
					var source string
					if _, file, line, ok := runtime.Caller(1); ok == true {
						source = fmt.Sprintf("%s:%d", file, line)
					}
					test.span.LogFields(
						oLog.String("event", "log"),
						oLog.String("message", fmt.Sprintf(format, args)),
						oLog.String(EventSource, source),
						oLog.String("log.level", "ERROR"),
						oLog.String("log.logger", "testing.T"),
					)
				}

				t.Errorf(format, args)
			})

		var traceLogGuard *monpatch.PatchGuard
		traceLogGuard = monpatch.PatchInstanceMethod(commonTypeReference, "Log",
			func(t *testing.T, args ...interface{}) {
				traceLogGuard.Unpatch()
				defer traceLogGuard.Restore()

				currentTest := contexts.GetGoRoutineData(currentTestKey)
				if currentTest != nil {
					test := currentTest.(*Test)
					var source string
					if _, file, line, ok := runtime.Caller(1); ok == true {
						source = fmt.Sprintf("%s:%d", file, line)
					}
					test.span.LogFields(
						oLog.String("event", "log"),
						oLog.String("message", fmt.Sprint(args)),
						oLog.String(EventSource, source),
						oLog.String("log.level", "INFO"),
						oLog.String("log.logger", "testing.T"),
					)
				}

				t.Log(args)
			})

		var traceLogfGuard *monpatch.PatchGuard
		traceLogfGuard = monpatch.PatchInstanceMethod(commonTypeReference, "Logf",
			func(t *testing.T, format string, args ...interface{}) {
				traceLogfGuard.Unpatch()
				defer traceLogfGuard.Restore()

				currentTest := contexts.GetGoRoutineData(currentTestKey)
				if currentTest != nil {
					test := currentTest.(*Test)
					var source string
					if _, file, line, ok := runtime.Caller(1); ok == true {
						source = fmt.Sprintf("%s:%d", file, line)
					}
					test.span.LogFields(
						oLog.String("event", "log"),
						oLog.String("message", fmt.Sprintf(format, args)),
						oLog.String(EventSource, source),
						oLog.String("log.level", "INFO"),
						oLog.String("log.logger", "testing.T"),
					)
				}

				t.Logf(format, args)
			})

		var logOutputGuard *monpatch.PatchGuard
		logOutputGuard = monpatch.PatchInstanceMethod(reflect.TypeOf(new(log.Logger)), "Output", func(l *log.Logger, calldepth int, s string) error {
			logOutputGuard.Unpatch()
			defer logOutputGuard.Restore()

			funcPc, _, _, _ := runtime.Caller(1)
			funcName := runtime.FuncForPC(funcPc).Name()

			currentTest := contexts.GetGoRoutineData(currentTestKey)
			if currentTest != nil {
				test := currentTest.(*Test)
				var source string
				if _, file, line, ok := runtime.Caller(2); ok == true {
					source = fmt.Sprintf("%s:%d", file, line)
				}
				if isFatal := strings.Contains(funcName, "Fatal"); isFatal || strings.Contains(funcName, "Panic") {
					test.span.LogFields(
						oLog.String("event", "log"),
						oLog.String("message", s),
						oLog.String(EventSource, source),
						oLog.String("log.level", "ERROR"),
						oLog.String("log.logger", "log.Logger"),
					)
					if isFatal {
						test.span.SetTag("test.status", "FAIL")
						test.span.SetTag("error", true)
						test.span.Finish()
						_ = GlobalAgent.Flush()
					}
				} else {
					test.span.LogFields(
						oLog.String("event", "log"),
						oLog.String("message", s),
						oLog.String(EventSource, source),
						oLog.String("log.level", "VERBOSE"),
						oLog.String("log.logger", "log.Logger"),
					)
				}
			}

			return l.Output(calldepth, s)
		})
	})

}

// Handles the StdIO pipe for stdout and stderr
func stdIOHandler(test *Test, stdio *StdIO, isError bool) {
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
				oLog.String(EventType, LogEvent),
				oLog.String(EventMessage, line),
				oLog.String(LogEventLevel, LogLevel_ERROR),
			)
		} else {
			test.span.LogFields(
				oLog.String(EventType, LogEvent),
				oLog.String(EventMessage, line),
				oLog.String(LogEventLevel, LogLevel_VERBOSE),
			)
		}
		_, _ = stdio.oldIO.WriteString(line)
	}
}

// Creates and replaces a file instance with a pipe
func newStdIO(file **os.File) *StdIO {
	rPipe, wPipe, err := os.Pipe()
	if err == nil {
		stdIO := &StdIO{
			oldIO:     *file,
			readPipe:  rPipe,
			writePipe: wPipe,
			sync:      new(sync.WaitGroup),
		}
		*file = wPipe
		return stdIO
	}
	return nil
}

// Restores the old file instance
func (stdIO *StdIO) restore(file **os.File) {
	_ = (*file).Sync()
	_ = stdIO.readPipe.Sync()
	_ = stdIO.writePipe.Close()
	_ = stdIO.readPipe.Close()
	stdIO.sync.Wait()
	*file = stdIO.oldIO
}
