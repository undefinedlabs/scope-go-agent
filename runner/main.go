package runner

import (
	"errors"
	"reflect"
	"runtime"
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

		repository    string
		branch        string
		commit        string
		serviceName   string
		configuration *testRunnerSession
	}
	testDescriptor struct {
		test    testing.InternalTest
		fqn     string
		ran     int
		failed  bool
		skipped bool
	}
	benchmarkDescriptor struct {
		benchmark testing.InternalBenchmark
		fqn       string
		ran       int
		failed    bool
		skipped   bool
	}
)

var runner *testRunner
var cfgLoader sessionLoader

func Run(m *testing.M, repository string, branch string, commit string, serviceName string) int {
	runner = getRunner(m, repository, branch, commit, serviceName)
	return runner.Run()
}

func getRunner(m *testing.M, repository string, branch string, commit string, serviceName string) *testRunner {
	cfgLoader = &dummySessionLoader{}
	runner := &testRunner{
		m:           m,
		repository:  repository,
		branch:      branch,
		commit:      commit,
		serviceName: serviceName,
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
			r.tests = append(r.tests, testDescriptor{
				test:   test,
				fqn:    r.getFqnOfTest(test.F),
				ran:    0,
				failed: false,
			})
		}
	}
	if bPointer, err := r.getFieldPointer("benchmarks"); err == nil {
		r.intBenchmarks = (*[]testing.InternalBenchmark)(bPointer)
		for _, benchmark := range *r.intBenchmarks {
			r.benchmarks = append(r.benchmarks, benchmarkDescriptor{
				benchmark: benchmark,
				fqn:       r.getFqnOfBenchmark(benchmark.F),
				ran:       0,
				failed:    false,
			})
		}
	}
	r.configuration = cfgLoader.LoadSessionConfiguration(r.repository, r.branch, r.commit, r.serviceName)
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

func (r *testRunner) getFqnOfTest(tFunc func(*testing.T)) string {
	funcVal := runtime.FuncForPC(reflect.ValueOf(tFunc).Pointer())
	return funcVal.Name()
}

func (r *testRunner) getFqnOfBenchmark(bFunc func(*testing.B)) string {
	funcVal := runtime.FuncForPC(reflect.ValueOf(bFunc).Pointer())
	return funcVal.Name()
}
