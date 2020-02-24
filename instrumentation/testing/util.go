package testing

import (
	"fmt"
	"path"
	"runtime"
	"strings"

	"go.undefinedlabs.com/scopeagent/ast"
	"go.undefinedlabs.com/scopeagent/instrumentation"
)

func getFuncName(fullName string) string {
	testNameSlash := strings.IndexByte(fullName, '/')
	funcName := fullName
	if testNameSlash >= 0 {
		funcName = fullName[:testNameSlash]
	}
	return funcName
}

func getPackageName(pc uintptr, fullName string) string {
	funcName := getFuncName(fullName)
	funcFullName := runtime.FuncForPC(pc).Name()
	funcNameIndex := strings.LastIndex(funcFullName, funcName)
	if funcNameIndex < 1 {
		funcNameIndex = len(funcFullName)
	}
	packageName := funcFullName[:funcNameIndex-1]
	sourceRoot := instrumentation.GetSourceRoot()
	if len(packageName) > 0 && packageName[0] == '_' && strings.Index(packageName, sourceRoot) != -1 {
		packageName = strings.Replace(packageName, path.Dir(sourceRoot)+"/", "", -1)[1:]
	}
	return packageName
}

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
