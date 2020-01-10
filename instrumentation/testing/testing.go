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
	"go.undefinedlabs.com/scopeagent/instrumentation/logging"
	"go.undefinedlabs.com/scopeagent/tags"
	"math"
	"reflect"
	"runtime"
	"strings"
	"sync"
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
		failReason       string
		failReasonSource string
		skipReason       string
		skipReasonSource string
		onPanicHandler   func(*Test)
		hasEnded         bool
	}

	Option func(*Test)
)

var (
	mutex   sync.Mutex
	testMap map[*testing.T]*Test

	intTests *[]testing.InternalTest

	defaultPanicHandler func(*Test)
)

// Initialize vars
func init() {
	mutex = sync.Mutex{}
	testMap = map[*testing.T]*Test{}
	defaultPanicHandler = func(t *Test) {}
}

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
	mutex.Lock()
	if testPtr, ok := testMap[t]; ok {
		testPtr.hasEnded = true
		testPtr.stopCapturingLogs()
		delete(testMap, testPtr.t)
	}

	test := &Test{t: t, onPanicHandler: defaultPanicHandler}

	testMap[t] = test
	mutex.Unlock()

	for _, opt := range opts {
		opt(test)
	}

	// Extracting the benchmark func name (by removing any possible sub-benchmark suffix `{bench_func}/{sub_benchmark}`)
	// to search the func source code bounds and to calculate the package name.
	fullTestName := t.Name()
	testNameSlash := strings.IndexByte(fullTestName, '/')
	funcName := fullTestName
	if testNameSlash >= 0 {
		funcName = fullTestName[:testNameSlash]
	}

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
	logging.SetCurrentSpan(span)

	return test
}

// Ends the current test
func (test *Test) End() {
	mutex.Lock()
	if test.hasEnded {
		mutex.Unlock()
		return
	}
	test.hasEnded = true
	mutex.Unlock()
	if r := recover(); r != nil {
		test.span.SetTag("test.status", tags.TestStatus_FAIL)
		test.span.SetTag("error", true)
		errors.LogError(test.span, r, 1)
		logging.SetCurrentSpan(nil)
		test.span.Finish()
		if test.onPanicHandler != nil {
			test.onPanicHandler(test)
		}
		panic(r)
	}
	if test.t.Failed() {
		test.span.SetTag("test.status", tags.TestStatus_FAIL)
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
		test.span.SetTag("test.status", tags.TestStatus_SKIP)
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
		test.span.SetTag("test.status", tags.TestStatus_PASS)
	}

	logging.SetCurrentSpan(nil)
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

// Starts a new benchmark using a pc as caller
func StartBenchmark(b *testing.B, pc uintptr, benchFunc func(b *testing.B)) {
	var bChild *testing.B
	b.ReportAllocs()
	b.ResetTimer()
	startTime := time.Now()
	result := b.Run("*", func(b1 *testing.B) {
		benchFunc(b1)
		bChild = b1
	})
	results, err := extractBenchmarkResult(bChild)
	if err != nil {
		instrumentation.Logger().Printf("Error while extracting the benchmark result object: %v\n", err)
		return
	}

	// Extracting the benchmark func name (by removing any possible sub-benchmark suffix `{bench_func}/{sub_benchmark}`)
	// to search the func source code bounds and to calculate the package name.
	fullTestName := b.Name()
	testNameSlash := strings.IndexByte(fullTestName, '/')
	funcName := fullTestName
	if testNameSlash >= 0 {
		funcName = fullTestName[:testNameSlash]
	}

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
		"test.type":      "benchmark",
	}, opentracing.StartTime(startTime))

	span, _ := opentracing.StartSpanFromContextWithTracer(context.Background(), instrumentation.Tracer(), b.Name(), startOptions...)
	span.SetBaggageItem("trace.kind", "test")
	avg := math.Round((float64(results.T.Nanoseconds())/float64(results.N))*100) / 100
	span.SetTag("benchmark.runs", results.N)
	span.SetTag("benchmark.duration.mean", avg)
	span.SetTag("benchmark.memory.mean_allocations", results.AllocsPerOp())
	span.SetTag("benchmark.memory.mean_bytes_allocations", results.AllocedBytesPerOp())
	if result {
		span.SetTag("test.status", "PASS")
	} else {
		span.SetTag("test.status", "FAIL")
	}
	span.FinishWithOptions(opentracing.FinishOptions{
		FinishTime: startTime.Add(results.T),
	})
}

// Initialize the testing instrumentation
func Init(m *testing.M) {
	tests := make([]testing.InternalTest, 0)
	if tPointer, err := getFieldPointerOfM(m, "tests"); err == nil {
		intTests = (*[]testing.InternalTest)(tPointer)
		for _, test := range *intTests {
			funcValue := test.F
			funcPointer := reflect.ValueOf(funcValue).Pointer()
			tests = append(tests, testing.InternalTest{
				Name: test.Name,
				F: func(t *testing.T) { // Creating a new test function as an indirection of the original test
					tStruct := StartTestFromCaller(t, funcPointer)
					defer tStruct.End()
					funcValue(t)
				},
			})
		}
		// Replace internal tests with new test indirection
		*intTests = tests
	}
}

// Gets the Test struct from testing.T
func GetTest(t *testing.T) *Test {
	mutex.Lock()
	defer mutex.Unlock()
	if test, ok := testMap[t]; ok {
		return test
	}
	return nil
}

// Sets the default panic handler
func SetDefaultPanicHandler(handler func(*Test)) {
	if handler != nil {
		defaultPanicHandler = handler
	}
}

// Gets a private field from the testing.M struct using reflection
func getFieldPointerOfM(m *testing.M, fieldName string) (unsafe.Pointer, error) {
	val := reflect.Indirect(reflect.ValueOf(m))
	member := val.FieldByName(fieldName)
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return ptrToY, nil
	}
	return nil, stdErrors.New("field can't be retrieved")
}
