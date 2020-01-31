package runner

import (
	"fmt"
	"testing"
)

var (
	okCount    = 0
	failCount  = 0
	errorCount = 0
	flakyCount = 0
)

func TestMain(m *testing.M) {
	Run(m, false, 4, nil)
	fmt.Println(okCount, failCount, errorCount, flakyCount)
	if okCount != 1 {
		panic("TestOk ran an unexpected number of times")
	}
	if failCount != 5 {
		panic("TestFail ran an unexpected number of times")
	}
	if errorCount != 5 {
		panic("TestError ran an unexpected number of times")
	}
	if flakyCount != 3 {
		panic("TestFlaky ran an unexpected number of times")
	}
}

func TestOk(t *testing.T) {
	okCount++
	t.Log("Ok")
}

func TestFail(t *testing.T) {
	failCount++
	t.Fatal("Fail")
}

func TestError(t *testing.T) {
	errorCount++
	panic("this is a panic")
}

func TestFlaky(t *testing.T) {
	flakyCount++
	if flakyCount != 3 {
		t.Fatal("this is flaky")
	}
}
