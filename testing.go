package scopeagent

import (
	"context"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/undefinedlabs/go-agent/contexts"
	"github.com/undefinedlabs/go-agent/errors"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"bou.ke/monkey"
)

var (
	patcher		sync.Once
)

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
	//**
	//patcher.Do(func() {
		var fatalGuard *monkey.PatchGuard
		monkey.PatchInstanceMethod(reflect.TypeOf(t), "Fatal", func (tInst *testing.T, args ...interface{}) {
			fatalGuard.Unpatch()
			defer fatalGuard.Restore()

			line := fmt.Sprintln(args...)
			fmt.Println("INTERCEPTED: " + line)

			tInst.Fatal(args)
		})
	//})
	//**
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
	contexts.SetGoRoutineData("currentSpan", span)

	return &Test{
		ctx:  ctx,
		span: span,
		t:    t,
	}
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
	contexts.SetGoRoutineData("currentSpan", nil)
}

func (test *Test) Context() context.Context {
	return test.ctx
}
