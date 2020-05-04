package gocheck

import (
	"fmt"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"
	_ "unsafe"

	chk "gopkg.in/check.v1"
)

type methodType struct {
	reflect.Value
	Info reflect.Method
}

type resultTracker struct {
	result          chk.Result
	_lastWasProblem bool
	_waiting        int
	_missed         int
	_expectChan     chan *chk.C
	_doneChan       chan *chk.C
	_stopChan       chan bool
}
type tempDir struct {
	sync.Mutex
	path    string
	counter int
}
type outputWriter struct {
	m                    sync.Mutex
	writer               io.Writer
	wroteCallProblemLast bool
	Stream               bool
	Verbose              bool
}
type suiteRunner struct {
	suite                     interface{}
	setUpSuite, tearDownSuite *methodType
	setUpTest, tearDownTest   *methodType
	tests                     []*methodType
	tracker                   *resultTracker
	tempDir                   *tempDir
	keepDir                   bool
	output                    *outputWriter
	reportedProblemLast       bool
	benchTime                 time.Duration
	benchMem                  bool
}

//go:linkname nSRunner gopkg.in/check%2ev1.newSuiteRunner
func nSRunner(suite interface{}, runConf *chk.RunConf) *suiteRunner
//go:linkname nSRunnerRun gopkg.in/check%2ev1.(*suiteRunner).run
func nSRunnerRun(runner *suiteRunner) *chk.Result

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {

	sr := nSRunner(&MySuite{}, &chk.RunConf{})
	r := nSRunnerRun(sr)

	_ = r

	st := &struct {
	}{}

	fType := reflect.FuncOf([]reflect.Type{reflect.TypeOf(st), reflect.TypeOf(&chk.C{})}, nil, false)
	fVal := reflect.MakeFunc(fType, func(args []reflect.Value) (results []reflect.Value) {
		cArg := (*chk.C)(unsafe.Pointer(args[1].Pointer()))
		fmt.Println(cArg)
		return nil
	})

	fValIface := fVal.Interface().(func(*struct{}, *chk.C))

	_ = fValIface

	//reflect.StructOf()

	//chk.newSuiteRunner

	ms := &MySuite{}

	//s := nSRunner(ms, &chk.RunConf{})
	//_ = s

	mt, _ := reflect.TypeOf(ms).MethodByName("TestHelloWorld")
	fmt.Println(mt)
	thwFunc := reflect.MakeFunc(mt.Type, func(args []reflect.Value) (results []reflect.Value) {
		cArg := (*chk.C)(unsafe.Pointer(args[0].Pointer()))

		fmt.Println(cArg)

		return nil
	})

	_ = thwFunc

	tmpStruct := struct{}{}
	/*
		thwFuncIface := thwFunc.Interface().(func(*struct{}, *chk.C))
		thwFuncIface(&tmpStruct, &chk.C{})
		thwFuncIface(&tmpStruct, &chk.C{})
		thwFuncIface(&tmpStruct, &chk.C{})
	*/

	fmt.Println(fType.NumMethod())
	fValIface(&tmpStruct, &chk.C{})
	fValIface(&tmpStruct, &chk.C{})

	chk.TestingT(t)
}

type MySuite struct{}

var _ = chk.Suite(&MySuite{})

func (s *MySuite) SetUpTest(c *chk.C) {
	fmt.Println("Start Test", c.TestName())
}

func (s *MySuite) TearDownTest(c *chk.C) {
	fmt.Println("End Test", c.TestName())
}

func (s *MySuite) TestHelloWorld(c *chk.C) {
	c.Assert(42, chk.Equals, "42")
	c.Assert(io.ErrClosedPipe, chk.ErrorMatches, "io: .*on closed pipe")
	c.Check(42, chk.Equals, 42)
}
