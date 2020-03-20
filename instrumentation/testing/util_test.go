package testing

import (
	_ "fmt"
	_ "runtime"
	_ "testing"

	_ "go.undefinedlabs.com/scopeagent/ast"
)

/*
func TestGetFuncName(t *testing.T) {
	cases := map[string]string{
		"TestBase":                           "TestBase",
		"TestBase/Sub01":                     "TestBase",
		"TestBase/Sub 02":                    "TestBase",
		"TestBase/Sub/Sub02":                 "TestBase",
		"TestBase/Sub/Sub02/Sub03":           "TestBase",
		"TestBase/Sub/Sub02/Sub03/S u b 0 4": "TestBase",
	}

	for key, expected := range cases {
		t.Run(key, func(t *testing.T) {
			value := getFuncName(key)
			if value != expected {
				t.Fatalf("value '%s', expected: '%s'", value, expected)
			}
		})
	}
}

func TestGetPackageName(t *testing.T) {
	var pc uintptr
	func() { pc, _, _, _ = runtime.Caller(1) }()
	testName := t.Name()
	packName := getPackageName(pc, testName)

	subTestName := ""
	t.Run("sub-test", func(t *testing.T) {
		subTestName = t.Name()
	})
	packName02 := getPackageName(pc, subTestName)

	if testName != "TestGetPackageName" {
		t.Fatalf("value '%s' not expected", testName)
	}
	if subTestName != "TestGetPackageName/sub-test" {
		t.Fatalf("value '%s' not expected", testName)
	}
	if packName != "go.undefinedlabs.com/scopeagent/instrumentation/testing" {
		t.Fatalf("value '%s' not expected", packName)
	}
	if packName != packName02 {
		t.Fatalf("value '%s' not expected", packName02)
	}
}

func TestGetTestCodeBoundaries(t *testing.T) {
	var pc uintptr
	func() { pc, _, _, _ = runtime.Caller(1) }()
	testName := t.Name()

	actualBoundary := getTestCodeBoundaries(pc, testName)
	boundaryExpected, _ := ast.GetFuncSourceForName(pc, testName)
	calcExpected := fmt.Sprintf("%s:%d:%d", boundaryExpected.File, boundaryExpected.Start.Line, boundaryExpected.End.Line)
	if actualBoundary != calcExpected {
		t.Fatalf("value '%s' not expected", actualBoundary)
	}
}
*/
