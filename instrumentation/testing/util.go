package testing

import (
	"fmt"
	"path"
	"runtime"
	"strings"

	"go.undefinedlabs.com/scopeagent/ast"
	"go.undefinedlabs.com/scopeagent/instrumentation"
)

// Gets the Test/Benchmark parent func name without sub-benchmark and sub-test segments
func getFuncName(fullName string) string {
	testNameSlash := strings.IndexByte(fullName, '/')
	funcName := fullName
	if testNameSlash >= 0 {
		funcName = fullName[:testNameSlash]
	}
	return funcName
}

// Gets the Package name
func getPackageName(pc uintptr, fullName string) string {
	// Parent test/benchmark name
	funcName := getFuncName(fullName)
	// Full func name (format ex: {packageName}.{test/benchmark name}.{inner function of sub benchmark/test}
	funcFullName := runtime.FuncForPC(pc).Name()

	// We select the packageName as substring from start to the index of the test/benchmark name minus 1
	funcNameIndex := strings.LastIndex(funcFullName, funcName)
	if funcNameIndex < 1 {
		funcNameIndex = len(funcFullName)
	}
	packageName := funcFullName[:funcNameIndex-1]

	// If the package has the format: _/{path...}
	// We convert the path from absolute to relative to the source root
	sourceRoot := instrumentation.GetSourceRoot()
	if len(packageName) > 0 && packageName[0] == '_' && strings.Index(packageName, sourceRoot) != -1 {
		packageName = strings.Replace(packageName, path.Dir(sourceRoot)+"/", "", -1)[1:]
	}

	return packageName
}

// Gets the source code boundaries of a test or benchmark in the format: {file}:{startLine}:{endLine}
func getTestCodeBoundaries(pc uintptr, fullName string) string {
	funcName := getFuncName(fullName)
	sourceBounds, err := ast.GetFuncSourceForName(pc, funcName)
	if err != nil {
		instrumentation.Logger().Printf("error calculating the source boundaries for '%s [%s]': %v", funcName, fullName, err)
	}
	if sourceBounds != nil {
		return fmt.Sprintf("%s:%d:%d", sourceBounds.File, sourceBounds.Start.Line, sourceBounds.End.Line)
	}
	return ""
}
