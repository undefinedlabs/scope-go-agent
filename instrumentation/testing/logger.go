package testing

import (
	"reflect"
	"runtime"
	"testing"
	"unsafe"

	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent/instrumentation"
)

var (
	commonPtr       reflect.Type         // *testing.common type
	patches         []*mpatch.Patch      // patches
	skippedPointers = map[uintptr]bool{} // pointers to skip
)

func init() {
	// We get the *testing.common type to use in the patch method
	var t testing.T
	typeOfT := reflect.TypeOf(t)
	if cm, ok := typeOfT.FieldByName("common"); ok {
		commonPtr = reflect.PtrTo(cm.Type)
	}

	// We extract all methods pointer of Test, to avoid logging twice
	var test *Test
	typeOfTest := reflect.TypeOf(test)
	for i := 0; i < typeOfTest.NumMethod(); i++ {
		method := typeOfTest.Method(i)
		skippedPointers[method.Func.Pointer()] = true
	}
}

func PatchTestingLogger() {

	patchError()
	patchErrorf()
	patchFatal()
	patchFatalf()
	patchLog()
	patchLogf()
	patchSkip()
	patchSkipf()
}

func UnpatchTestingLogger() {
	for _, patch := range patches {
		logOnError(patch.Unpatch())
	}
	patches = nil
}

func patchError() {
	patch("Error", func(test *Test, argsValues []reflect.Value) {
		args := getArgs(argsValues[0])
		test.Error(args...)
	})
}
func patchErrorf() {
	patch("Errorf", func(test *Test, argsValues []reflect.Value) {
		format := argsValues[0].String()
		args := getArgs(argsValues[1])
		test.Errorf(format, args...)
	})
}
func patchFatal() {
	patch("Fatal", func(test *Test, argsValues []reflect.Value) {
		args := getArgs(argsValues[0])
		test.Fatal(args...)
	})
}
func patchFatalf() {
	patch("Fatalf", func(test *Test, argsValues []reflect.Value) {
		format := argsValues[0].String()
		args := getArgs(argsValues[1])
		test.Fatalf(format, args...)
	})
}
func patchLog() {
	patch("Log", func(test *Test, argsValues []reflect.Value) {
		args := getArgs(argsValues[0])
		test.Log(args...)
	})
}
func patchLogf() {
	patch("Logf", func(test *Test, argsValues []reflect.Value) {
		format := argsValues[0].String()
		args := getArgs(argsValues[1])
		test.Logf(format, args...)
	})
}
func patchSkip() {
	patch("Skip", func(test *Test, argsValues []reflect.Value) {
		args := getArgs(argsValues[0])
		test.Skip(args...)
	})
}
func patchSkipf() {
	patch("Skipf", func(test *Test, argsValues []reflect.Value) {
		format := argsValues[0].String()
		args := getArgs(argsValues[1])
		test.Skipf(format, args...)
	})
}

func getArgs(in reflect.Value) []interface{} {
	var args []interface{}
	if in.Kind() == reflect.Slice {
		for i := 0; i < in.Len(); i++ {
			args = append(args, in.Index(i).Interface())
		}
	}
	return args
}

func patch(methodName string, methodBody func(test *Test, argsValues []reflect.Value)) {
	if method, ok := commonPtr.MethodByName(methodName); ok {
		var patch *mpatch.Patch
		var err error
		patch, err = mpatch.PatchMethodByReflect(method,
			reflect.MakeFunc(method.Type, func(in []reflect.Value) []reflect.Value {
				logOnError(patch.Unpatch())
				defer func() {
					logOnError(patch.Patch())
				}()
				// We check if the caller is not a method of Test struct, to avoid duplicate logs
				if pc, _, _, ok := runtime.Caller(3); ok {
					fnc := runtime.FuncForPC(pc)
					if _, ok := skippedPointers[fnc.Entry()]; ok {
						return nil
					}
				}
				t := (*testing.T)(unsafe.Pointer(in[0].Pointer()))
				if t != nil {
					in = in[1:]
					test := GetTest(t)
					if test != nil {
						methodBody(test, in)
					} else {
						instrumentation.Logger().Printf("test struct for %v doesn't exist\n", t.Name())
					}
				} else {
					instrumentation.Logger().Println("testing.T is nil")
				}
				return nil
			}),
		)
		logOnError(err)
		patches = append(patches, patch)
	}
}

func logOnError(err error) {
	if err != nil {
		instrumentation.Logger().Println(err)
	}
}
