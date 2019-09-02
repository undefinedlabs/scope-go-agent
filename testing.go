package scopeagent

import (
	"bou.ke/monkey"
	"context"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/opentracing/opentracing-go"
	oLog "github.com/opentracing/opentracing-go/log"
	"github.com/undefinedlabs/go-agent/contexts"
	"github.com/undefinedlabs/go-agent/errors"
)

var (
	patcher sync.Once
)

const currentTestKey  = "currentTest"

type Test struct {
	ctx  context.Context
	span opentracing.Span
	t    *testing.T
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

	test := &Test{
		ctx:  ctx,
		span: span,
		t:    t,
	}
	contexts.SetGoRoutineData(currentTestKey, test)

	return test
}

func (test *Test) End() {
	if r := recover(); r != nil {
		test.span.SetTag("test.status", "ERROR")
		test.span.SetTag("error", true)
		errors.LogError(test.span, r, 1)
		test.span.Finish()
		_ = GlobalAgent.Flush()
		panic(r)
	}
	if test.t.Failed() {
		test.span.SetTag("test.status", "FAIL")
		test.span.SetTag("error", true)
	} else if test.t.Skipped() {
		test.span.SetTag("test.status", "SKIP")
	} else {
		test.span.SetTag("test.status", "PASS")
	}
	test.span.Finish()
	contexts.SetGoRoutineData(currentTestKey, nil)
}

func (test *Test) Context() context.Context {
	return test.ctx
}


func patchLogger() {

	patcher.Do(func() {
		var logOutputGuard *monkey.PatchGuard
		logOutputGuard = monkey.PatchInstanceMethod(reflect.TypeOf(new(log.Logger)), "Output", func(l *log.Logger, calldepth int, s string) error {
			logOutputGuard.Unpatch()
			defer logOutputGuard.Restore()

			funcPc, _, _, _ := runtime.Caller(1)
			funcName := runtime.FuncForPC(funcPc).Name()

			currentTest := contexts.GetGoRoutineData(currentTestKey)
			if currentTest != nil {
				test := currentTest.(*Test)

				if isFatal := strings.Contains(funcName, "Fatal"); isFatal || strings.Contains(funcName, "Panic") {
					test.span.LogFields(
						oLog.String("event", "log"),
						oLog.String("message", s),
						oLog.String("log.level", "ERROR"),
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
						oLog.String("log.level", "VERBOSE"),
					)
				}
			}

			return l.Output(calldepth, s)
		})
	})

}