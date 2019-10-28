package testing

import (
	"context"
	stdErrors "errors"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"go.undefinedlabs.com/scopeagent/ast"
	"go.undefinedlabs.com/scopeagent/errors"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/tags"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"
)

type (
	Test struct {
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
		onPanicHandler   func(*Test)
	}

	Option func(*Test)
)

// Options for starting a new test
func WithContext(ctx context.Context) Option {
	return func(test *Test) {
		test.ctx = ctx
	}
}

func WithOnPanicHandler(f func(*Test)) Option {
	return func(test *Test) {
		test.onPanicHandler = f
	}
}

// Starts a new test
func StartTest(t *testing.T, opts ...Option) *Test {
	pc, _, _, _ := runtime.Caller(1)
	return StartTestFromCaller(t, pc, opts...)
}

// Starts a new test with and uses the caller pc info for Name and Suite
func StartTestFromCaller(t *testing.T, pc uintptr, opts ...Option) *Test {
	test := &Test{t: t}

	for _, opt := range opts {
		opt(test)
	}

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

	sourceBounds, _ := ast.GetFuncSourceForName(pc, funcName)
	var testCode string
	if sourceBounds != nil {
		testCode = fmt.Sprintf("%s:%d:%d", sourceBounds.File, sourceBounds.Start.Line, sourceBounds.End.Line)
	}

	var startOptions []opentracing.StartSpanOption
	startOptions = append(startOptions, opentracing.Tags{
		"span.kind":      "test",
		"test.name":      fullTestName,
		"test.suite":     packageName,
		"test.code":      testCode,
		"test.framework": "testing",
		"test.language":  "go",
	})

	if test.ctx == nil {
		test.ctx = context.Background()
	}

	span, ctx := opentracing.StartSpanFromContextWithTracer(test.ctx, instrumentation.Tracer(), t.Name(), startOptions...)
	span.SetBaggageItem("trace.kind", "test")
	test.span = span
	test.ctx = ctx

	test.startCapturingLogs()

	return test
}

// Ends the current test
func (test *Test) End() {
	if r := recover(); r != nil {
		test.stopCapturingLogs()
		test.span.SetTag("test.status", "ERROR")
		test.span.SetTag("error", true)
		errors.LogError(test.span, r, 1)
		test.span.Finish()
		if test.onPanicHandler != nil {
			test.onPanicHandler(test)
		}
		panic(r)
	}
	if test.t.Failed() {
		test.span.SetTag("test.status", "FAIL")
		test.span.SetTag("error", true)
		if test.failReason != "" {
			test.span.LogFields(
				log.String(tags.EventType, tags.EventTestFailure),
				log.String(tags.EventMessage, test.failReason),
				log.String(tags.EventSource, test.failReasonSource),
			)
		} else {
			test.span.LogFields(
				log.String(tags.EventType, tags.EventTestFailure),
				log.String(tags.EventMessage, "Test has failed"),
			)
		}
	} else if test.t.Skipped() {
		test.span.SetTag("test.status", "SKIP")
		if test.skipReason != "" {
			test.span.LogFields(
				log.String(tags.EventType, tags.EventTestSkip),
				log.String(tags.EventMessage, test.skipReason),
				log.String(tags.EventSource, test.skipReasonSource),
			)
		} else {
			test.span.LogFields(
				log.String(tags.EventType, tags.EventTestSkip),
				log.String(tags.EventMessage, "Test has skipped"),
			)
		}
	} else {
		test.span.SetTag("test.status", "PASS")
	}

	test.stopCapturingLogs()
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
		childTest := StartTestFromCaller(childT, pc)
		defer childTest.End()
		f(childT)
	})
}

//Extract benchmark result from the private result field in testing.B
func extractBenchmarkResult(b *testing.B) (*testing.BenchmarkResult, error) {
	val := reflect.Indirect(reflect.ValueOf(b))
	member := val.FieldByName("result")
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return (*testing.BenchmarkResult)(ptrToY), nil
	}
	return nil, stdErrors.New("result can't be retrieved")
}

// Starts a new benchmark using a pc s a caller
func StartBenchmark(b *testing.B, pc uintptr, benchFunc func(b *testing.B)) {
	var bChild *testing.B
	b.ResetTimer()
	b.ReportAllocs()
	startTime := time.Now()
	result := b.Run("Scope", func(b1 *testing.B) {
		benchFunc(b1)
		bChild = b1
	})
	endTime := time.Now()
	results, err := extractBenchmarkResult(bChild)
	if err != nil {
		panic(err)
	}

	fullTestName := b.Name()
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

	sourceBounds, _ := ast.GetFuncSourceForName(pc, funcName)
	var testCode string
	if sourceBounds != nil {
		testCode = fmt.Sprintf("%s:%d:%d", sourceBounds.File, sourceBounds.Start.Line, sourceBounds.End.Line)
	}

	var startOptions []opentracing.StartSpanOption
	startOptions = append(startOptions, opentracing.Tags{
		"span.kind":      "test",
		"test.name":      fullTestName,
		"test.suite":     packageName,
		"test.code":      testCode,
		"test.framework": "benchmark",
		"test.language":  "go",
	}, opentracing.StartTime(startTime))

	span, _ := opentracing.StartSpanFromContextWithTracer(context.Background(), instrumentation.Tracer(), b.Name(), startOptions...)
	span.SetBaggageItem("trace.kind", "test")
	span.SetTag("test.type", "benchmark")
	avg := float32(results.T.Nanoseconds()) / float32(results.N)
	span.SetTag("benchmark.numbers", results.N)
	span.SetTag("benchmark.duration.avg", avg)
	span.SetTag("benchmark.duration.total", results.T.Nanoseconds())
	span.SetTag("benchmark.memory.avgAllocations", results.AllocsPerOp())
	span.SetTag("benchmark.memory.avgBytesAllocations", results.AllocedBytesPerOp())
	if result {
		span.SetTag("test.status", "PASS")
	} else {
		span.SetTag("test.status", "FAIL")
	}
	span.FinishWithOptions(opentracing.FinishOptions{
		FinishTime:  endTime,
	})
}
