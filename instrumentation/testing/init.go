package testing

import (
	"os"
	"reflect"
	"testing"

	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent/reflection"
)

// Initialize the testing instrumentation
func Init(m *testing.M) {
	if tPointer, err := reflection.GetFieldPointerOfM(m, "tests"); err == nil {
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
	if bPointer, err := reflection.GetFieldPointerOfM(m, "benchmarks"); err == nil {
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

	if envDMPatch, set := os.LookupEnv("SCOPE_DISABLE_MONKEY_PATCHING"); !set || envDMPatch == "" {
		// We monkey patch the `testing.M.Run()` func to patch and unpatch the testing logger methods
		mType := reflect.ValueOf(m).Type()
		if mRunMethod, ok := mType.MethodByName("Run"); ok {
			var runPatch *mpatch.Patch
			var err error
			runPatch, err = mpatch.PatchMethodByReflect(mRunMethod, func(m *testing.M) int {
				logOnError(runPatch.Unpatch())
				defer func() {
					logOnError(runPatch.Patch())
				}()
				PatchTestingLogger()
				defer UnpatchTestingLogger()
				return m.Run()
			})
			logOnError(err)
		}
	}
}
