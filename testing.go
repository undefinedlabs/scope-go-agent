package scopeagent

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/undefinedlabs/go-agent/errors"
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

	return &Test{
		ctx:  ctx,
		span: span,
		t:    t,
	}
}

func (test *Test) End() {
	if r := recover(); r != nil {
		test.span.SetTag("test.status", "ERROR")
		errors.LogError(test.span, r, 1)
		test.span.Finish()
		_ = GlobalAgent.Flush()
		panic(r)
	}
	test.span.Finish()
	if test.t.Failed() {
		test.span.SetTag("test.status", "FAIL")
	} else if test.t.Skipped() {
		test.span.SetTag("test.status", "SKIP")
	} else {
		test.span.SetTag("test.status", "PASS")
	}
}

func (test *Test) Context() context.Context {
	return test.ctx
}
