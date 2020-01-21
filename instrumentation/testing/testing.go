package testing

import (
	"context"
	stdErrors "errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"go.undefinedlabs.com/scopeagent/ast"
	"go.undefinedlabs.com/scopeagent/errors"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/instrumentation/logging"
	"go.undefinedlabs.com/scopeagent/tags"
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
	}

	Option func(*Test)

	Benchmark struct {
		b *testing.B
	}
)

var (
	testMapMutex               sync.RWMutex
	testMap                    = map[*testing.T]*Test{}
	autoInstrumentedTestsMutex sync.RWMutex
	autoInstrumentedTests      = map[*testing.T]bool{}

	instrumentedBenchmarkMutex sync.RWMutex
	instrumentedBenchmark      = map[*testing.B]*Benchmark{}

	defaultPanicHandler = func(test *Test) {}

	TESTING_LOG_REGEX = regexp.MustCompile(`(?m)^ {4}(?P<file>[\w\/\.]+):(?P<line>\d+): (?P<message>(.*\n {8}.*)*.*)`)
)

// Initialize the testing instrumentation
func Init(m *testing.M) {
	if tPointer, err := getFieldPointerOfM(m, "tests"); err == nil {
		intTests := (*[]testing.InternalTest)(tPointer)
		tests := make([]testing.InternalTest, 0)
		for _, test := range *intTests {
			funcValue := test.F
			funcPointer := reflect.ValueOf(funcValue).Pointer()
			tests = append(tests, testing.InternalTest{
				Name: test.Name,
				F: func(t *testing.T) { // Creating a new test function as an indirection of the original test
					addAutoInstrumentedTest(t)
					tStruct := StartTestFromCaller(t, funcPointer)
					defer tStruct.end()
					funcValue(t)
				},
			})
		}
		// Replace internal tests with new test indirection
		*intTests = tests
	}
	if bPointer, err := getFieldPointerOfM(m, "benchmarks"); err == nil {
		intBenchmarks := (*[]testing.InternalBenchmark)(bPointer)
		var benchmarks []testing.InternalBenchmark
		for _, benchmark := range *intBenchmarks {
			funcValue := benchmark.F
			funcPointer := reflect.ValueOf(funcValue).Pointer()
			benchmarks = append(benchmarks, testing.InternalBenchmark{
				Name: benchmark.Name,
				F: func(b *testing.B) { // Indirection of the original benchmark
					startBenchmark(b, funcPointer, funcValue)
				},
			})
		}
		*intBenchmarks = benchmarks
	}
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
	// Get or create a new Test struct
	// If we get an old struct we replace the current span and context with a new one.
	// Useful if we want to overwrite the Start call with options
	test, exist := getOrCreateTest(t)
	if exist {
		// If there is already one we want to replace it, so we clear the context
		test.ctx = context.Background()
	}

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

	logging.Reset()

	return test
}

// Ends the current test
func (test *Test) End() {
	autoInstrumentedTestsMutex.RLock()
	defer autoInstrumentedTestsMutex.RUnlock()
	// First we detect if the current test is auto-instrumented, if not we call the end method (needed in sub tests)
	if _, ok := autoInstrumentedTests[test.t]; !ok {
		test.end()
	}
}

// Gets the test context
func (test *Test) Context() context.Context {
	return test.ctx
}

// Runs an auto instrumented sub test
func (test *Test) Run(name string, f func(t *testing.T)) {
	pc, _, _, _ := runtime.Caller(1)
	test.t.Run(name, func(childT *testing.T) {
		addAutoInstrumentedTest(childT)
		childTest := StartTestFromCaller(childT, pc)
		defer childTest.end()
		f(childT)
	})
}

// Ends the current test (this method is called from the auto-instrumentation)
func (test *Test) end() {
	finishTime := time.Now()

	// Remove the Test struct from the hash map, so a call to Start while we end this instance will create a new struct
	removeTest(test.t)
	// Stop and get records generated by loggers
	logRecords := logging.GetRecords()
	// Extract logging buffer from testing.T
	test.extractTestLoggerOutput()

	finishOptions := opentracing.FinishOptions{
		FinishTime: finishTime,
		LogRecords: logRecords,
	}

	if r := recover(); r != nil {
		test.span.SetTag("test.status", tags.TestStatus_FAIL)
		test.span.SetTag("error", true)
		if r != errors.MarkSpanAsError {
			errors.LogError(test.span, r, 1)
		}
		test.span.FinishWithOptions(finishOptions)
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

	test.span.FinishWithOptions(finishOptions)
}

func (test *Test) extractTestLoggerOutput() {
	output := extractTestOutput(test.t)
	if output == nil {
		return
	}
	outStr := string(*output)
	for _, matches := range findMatchesLogRegex(outStr) {
		test.span.LogFields([]log.Field{
			log.String(tags.EventType, tags.LogEvent),
			log.String(tags.LogEventLevel, tags.LogLevel_VERBOSE),
			log.String("log.logger", "test.Logger"),
			log.String(tags.EventMessage, matches[3]),
			log.String(tags.EventSource, fmt.Sprintf("%s:%s", matches[1], matches[2])),
		}...)
	}
}

func findMatchesLogRegex(output string) [][]string {
	allMatches := TESTING_LOG_REGEX.FindAllStringSubmatch(output, -1)
	for _, matches := range allMatches {
		matches[3] = strings.ReplaceAll(matches[3], "\n        ", "\n")
	}
	return allMatches
}

func extractTestOutput(t *testing.T) *[]byte {
	val := reflect.Indirect(reflect.ValueOf(t))
	member := val.FieldByName("output")
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return (*[]byte)(ptrToY)
	}
	return nil
}

// Gets or create a test struct
func getOrCreateTest(t *testing.T) (test *Test, exists bool) {
	testMapMutex.Lock()
	defer testMapMutex.Unlock()
	if testPtr, ok := testMap[t]; ok {
		test = testPtr
		exists = true
	} else {
		test = &Test{t: t, onPanicHandler: defaultPanicHandler}
		testMap[t] = test
		exists = false
	}
	return
}

// Removes a test struct from the map
func removeTest(t *testing.T) {
	testMapMutex.Lock()
	defer testMapMutex.Unlock()
	delete(testMap, t)
}

// Gets the Test struct from testing.T
func GetTest(t *testing.T) *Test {
	testMapMutex.RLock()
	defer testMapMutex.RUnlock()
	if test, ok := testMap[t]; ok {
		return test
	}
	return nil
}

// Adds an auto instrumented test to the map
func addAutoInstrumentedTest(t *testing.T) {
	autoInstrumentedTestsMutex.Lock()
	defer autoInstrumentedTestsMutex.Unlock()
	autoInstrumentedTests[t] = true
}

// Starts a new benchmark using a pc as caller
func StartBenchmark(b *testing.B, pc uintptr, benchFunc func(b *testing.B)) {
	if !isBenchmarkInstrumented(b) {
		// If the current benchmark is not instrumented, we instrument it.
		startBenchmark(b, pc, benchFunc)
	} else {
		// If the benchmark is already instrumented, we passthrough to the benchFunc
		benchFunc(b)
	}
}

// Runs an auto instrumented sub benchmark
func (bench *Benchmark) Run(name string, f func(b *testing.B)) bool {
	pc, _, _, _ := runtime.Caller(1)
	return bench.b.Run(name, func(innerB *testing.B) {
		startBenchmark(innerB, pc, f)
	})
}

// Adds an instrumented benchmark to the map
func addInstrumentedBenchmark(b *testing.B, value *Benchmark) {
	instrumentedBenchmarkMutex.Lock()
	defer instrumentedBenchmarkMutex.Unlock()
	instrumentedBenchmark[b] = value
}

// Gets if the benchmark is instrumented
func isBenchmarkInstrumented(b *testing.B) bool {
	instrumentedBenchmarkMutex.RLock()
	defer instrumentedBenchmarkMutex.RUnlock()
	_, ok := instrumentedBenchmark[b]
	return ok
}

// Gets the Benchmark struct from *testing.Benchmark
func GetBenchmark(b *testing.B) *Benchmark {
	instrumentedBenchmarkMutex.RLock()
	defer instrumentedBenchmarkMutex.RUnlock()
	if bench, ok := instrumentedBenchmark[b]; ok {
		return bench
	}
	return nil
}

func startBenchmark(b *testing.B, pc uintptr, benchFunc func(b *testing.B)) {
	var bChild *testing.B
	b.ReportAllocs()
	b.ResetTimer()
	startTime := time.Now()
	result := b.Run("*", func(b1 *testing.B) {
		addInstrumentedBenchmark(b1, &Benchmark{b: b1})
		benchFunc(b1)
		bChild = b1
	})
	if bChild == nil {
		return
	}
	results, err := extractBenchmarkResult(bChild)
	if err != nil {
		instrumentation.Logger().Printf("Error while extracting the benchmark result object: %v\n", err)
		return
	}

	// Extracting the benchmark func name (by removing any possible sub-benchmark suffix `{bench_func}/{sub_benchmark}`)
	// to search the func source code bounds and to calculate the package name.
	fullTestName := b.Name()

	// We detect if the parent benchmark is instrumented, and if so we remove the "*" SubBenchmark from the previous instrumentation
	parentBenchmark := getParentBenchmark(b)
	if parentBenchmark != nil && isBenchmarkInstrumented(parentBenchmark) {
		parentName := parentBenchmark.Name()
		if strings.Index(fullTestName, parentName) == 0 && len(parentName) > 2 {
			fullTestName = parentName[:len(parentName)-2] + fullTestName[len(parentName):]
		}
	}

	testNameSlash := strings.IndexByte(fullTestName, '/')
	funcName := fullTestName
	if testNameSlash >= 0 {
		funcName = fullTestName[:testNameSlash]
	}
	packageName := getBenchmarkSuiteName(b)

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

	span, _ := opentracing.StartSpanFromContextWithTracer(context.Background(), instrumentation.Tracer(), fullTestName, startOptions...)
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

func getParentBenchmark(b *testing.B) *testing.B {
	val := reflect.Indirect(reflect.ValueOf(b))
	member := val.FieldByName("parent")
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return *(**testing.B)(ptrToY)
	}
	return nil
}

func getBenchmarkSuiteName(b *testing.B) string {
	val := reflect.Indirect(reflect.ValueOf(b))
	member := val.FieldByName("importPath")
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return *(*string)(ptrToY)
	}
	return ""
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
