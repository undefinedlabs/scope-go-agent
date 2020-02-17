package runner

import (
	"fmt"
	"testing"
)

var (
	okCount     = 0
	failCount   = 0
	errorCount  = 0
	flakyCount  = 0
	failSubTest = 0
)

func TestMain(m *testing.M) {
	Run(m, Options{
		FailRetries: 4,
		PanicAsFail: true,
		Logger:      nil,
		OnPanic: func(t *testing.T, err error) {
			fmt.Printf("the test '%s' has paniked with error: %s", t.Name(), err)
		},
	})
	fmt.Println(okCount, failCount, errorCount, flakyCount, failSubTest)
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
	if failCount != 5 {
		panic("TestFailSubTest ran an unexpected number of times")
	}
}

func TestOk(t *testing.T) {
	if GetOriginalTestName(t.Name()) != "TestOk" {
		t.Fatal("test name is invalid.")
	}
	okCount++
	t.Log("Ok")
}

func TestFail(t *testing.T) {
	if GetOriginalTestName(t.Name()) != "TestFail" {
		t.Fatal("test name is invalid.")
	}
	failCount++
	t.Fatal("Fail")
}

func TestError(t *testing.T) {
	if GetOriginalTestName(t.Name()) != "TestError" {
		t.Fatal("test name is invalid.")
	}
	errorCount++
	panic("this is a panic")
}

func TestFlaky(t *testing.T) {
	if GetOriginalTestName(t.Name()) != "TestFlaky" {
		t.Fatal("test name is invalid.")
	}
	flakyCount++
	if flakyCount != 3 {
		t.Fatal("this is flaky")
	}
}

func TestFailSubTest(t *testing.T) {
	t.Run("SubTest", func(t *testing.T) {
		if GetOriginalTestName(t.Name()) != "TestFailSubTest/SubTest" {
			t.Fatal("test name is invalid.")
		}

		t.Run("SubSub", func(t *testing.T) {
			if GetOriginalTestName(t.Name()) != "TestFailSubTest/SubTest/SubSub" {
				t.Fatal("test name is invalid.")
			}
		})

		failSubTest++
		t.Fatal("Subtest fail")
	})
}
