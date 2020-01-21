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

	"go.undefinedlabs.com/scopeagent/ast"
	"go.undefinedlabs.com/scopeagent/instrumentation"
)

type (
	Benchmark struct {
		b *testing.B
	}
)

var (
	instrumentedBenchmarkMutex sync.RWMutex
	instrumentedBenchmark      = map[*testing.B]*Benchmark{}
	benchNameRegex             = regexp.MustCompile(`([\w-_:!@#\$%&()=]*)(\/\*\&\/)?`)
)

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
	result := b.Run("*&", func(b1 *testing.B) {
		addInstrumentedBenchmark(b1, &Benchmark{b: b1})
		benchFunc(b1)
		bChild = b1
	})
	if bChild == nil {
		return
	}
	if getBenchmarkHasSub(bChild) > 0 {
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
		var nameSegments []string
		for _, match := range benchNameRegex.FindAllStringSubmatch(fullTestName, -1) {
			if match[1] != "" {
				nameSegments = append(nameSegments, match[1])
			}
		}
		fullTestName = strings.Join(nameSegments, "/")
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

func getBenchmarkHasSub(b *testing.B) int32 {
	val := reflect.Indirect(reflect.ValueOf(b))
	member := val.FieldByName("hasSub")
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return *(*int32)(ptrToY)
	}
	return 0
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
