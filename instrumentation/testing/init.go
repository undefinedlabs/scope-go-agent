package testing

import (
	"reflect"
	"sync"
	"testing"

	"go.undefinedlabs.com/scopeagent/reflection"
)

// Initialize the testing instrumentation
func Init(m *testing.M) {
	if tPointer, err := reflection.GetFieldPointerOf(m, "tests"); err == nil {
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
	if bPointer, err := reflection.GetFieldPointerOf(m, "benchmarks"); err == nil {
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

func getTestMutex(t *testing.T) *sync.RWMutex {
	if ptr, err := reflection.GetFieldPointerOf(t, "mu"); err == nil {
		return (*sync.RWMutex)(ptr)
	}
	return nil
}

func getIsParallel(t *testing.T) bool {
	mu := getTestMutex(t)
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	if pointer, err := reflection.GetFieldPointerOf(t, "isParallel"); err == nil {
		return *(*bool)(pointer)
	}
	return false
}
