package instrumentation

import (
	"fmt"
	"runtime"
	"testing"

	"go.undefinedlabs.com/scopeagent/ast"
)

func TestSplitPackageAndName(t *testing.T) {
	cases := map[string][]string{
		"pkg.TestBase":                 {"pkg", "TestBase"},
		"pkg.TestBase.func1":           {"pkg", "TestBase.func1"},
		"github.com/org/proj.TestBase": {"github.com/org/proj", "TestBase"},
	}

	for key, expected := range cases {
		t.Run(key, func(t *testing.T) {
			pkg, fname := splitPackageAndName(key)
			if pkg != expected[0] {
				t.Fatalf("value '%s', expected: '%s'", pkg, expected[0])
			}
			if fname != expected[1] {
				t.Fatalf("value '%s', expected: '%s'", fname, expected[1])
			}
		})
	}
}

func TestGetTestCodeBoundaries(t *testing.T) {
	var pc uintptr
	func() { pc, _, _, _ = runtime.Caller(1) }()
	testName := t.Name()

	pkg, fname, bound := GetPackageAndNameAndBoundaries(pc)

	if pkg != "go.undefinedlabs.com/scopeagent/instrumentation" {
		t.Fatalf("value '%s' not expected", pkg)
	}
	if fname != testName {
		t.Fatalf("value '%s' not expected", fname)
	}
	boundaryExpected, _ := ast.GetFuncSourceForName(pc, testName)
	calcExpected := fmt.Sprintf("%s:%d:%d", boundaryExpected.File, boundaryExpected.Start.Line, boundaryExpected.End.Line)
	if bound != calcExpected {
		t.Fatalf("value '%s' not expected", bound)
	}
}
