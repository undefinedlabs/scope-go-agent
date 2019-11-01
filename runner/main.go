package runner

import (
	"errors"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"unsafe"
)

type (
	testRunner struct {
		m             *testing.M
		intTests      *[]testing.InternalTest
		intBenchmarks *[]testing.InternalBenchmark

		tests      []testDescriptor
		benchmarks []benchmarkDescriptor
	}
	testDescriptor struct {
		test        testing.InternalTest
		packageName string
		ran         bool
		failed      bool
	}
	benchmarkDescriptor struct {
		benchmark   testing.InternalBenchmark
		packageName string
		ran         bool
		failed      bool
	}
)

var runner *testRunner

func Run(m *testing.M) int {
	runner = getRunner(m)
	return runner.Run()
}

func getRunner(m *testing.M) *testRunner {
	runner := &testRunner{
		m: m,
	}
	runner.init()
	return runner
}

func (r *testRunner) Run() int {
	return r.m.Run()
}

func (r *testRunner) init() {
	if tPointer, err := r.getFieldPointer("tests"); err == nil {
		r.intTests = (*[]testing.InternalTest)(tPointer)
		for _, test := range *r.intTests {
			funcVal := runtime.FuncForPC(reflect.ValueOf(test.F).Pointer())
			funcFullName := funcVal.Name()
			funcNameIndex := strings.LastIndex(funcFullName, test.Name)
			if funcNameIndex < 1 {
				funcNameIndex = len(funcFullName)
			}
			r.tests = append(r.tests, testDescriptor{
				test:        test,
				packageName: funcFullName[:funcNameIndex-1],
				ran:         false,
				failed:      false,
			})
		}
	}
	if bPointer, err := r.getFieldPointer("benchmarks"); err == nil {
		r.intBenchmarks = (*[]testing.InternalBenchmark)(bPointer)
		for _, benchmark := range *r.intBenchmarks {
			funcVal := runtime.FuncForPC(reflect.ValueOf(benchmark.F).Pointer())
			funcFullName := funcVal.Name()
			funcNameIndex := strings.LastIndex(funcFullName, benchmark.Name)
			if funcNameIndex < 1 {
				funcNameIndex = len(funcFullName)
			}
			r.benchmarks = append(r.benchmarks, benchmarkDescriptor{
				benchmark:   benchmark,
				packageName: funcFullName[:funcNameIndex-1],
				ran:         false,
				failed:      false,
			})
		}
	}


}

func (r *testRunner) getFieldPointer(fieldName string) (unsafe.Pointer, error) {
	val := reflect.Indirect(reflect.ValueOf(r.m))
	member := val.FieldByName(fieldName)
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return ptrToY, nil
	}
	return nil, errors.New("field can't be retrieved")
}

type ()
