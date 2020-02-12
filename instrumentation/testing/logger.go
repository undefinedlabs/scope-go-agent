package testing

import (
	"reflect"
	"sync"
	"testing"
	"unsafe"

	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/reflection"
)

var (
	commonPtr          reflect.Type // *testing.common type
	patchLock          sync.Mutex
	patchesMutex       sync.RWMutex
	patches            = map[string]*mpatch.Patch{} // patches
	patchPointersMutex sync.RWMutex
	patchPointers      = map[uintptr]bool{} // pointers of patch funcs
)

func init() {
	// We get the *testing.common type to use in the patch method
	if cPtr, err := reflection.GetTypePointer(testing.T{}, "common"); err == nil {
		commonPtr = cPtr
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
	patchLock.Lock()
	defer patchLock.Unlock()
	patchPointersMutex.Lock()
	defer patchPointersMutex.Unlock()
	for _, patch := range patches {
		logOnError(patch.Unpatch())
	}
	patches = map[string]*mpatch.Patch{}
	patchPointers = map[uintptr]bool{}
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
	patchesMutex.Lock()
	defer patchesMutex.Unlock()
	patchPointersMutex.Lock()
	defer patchPointersMutex.Unlock()

	var method reflect.Method
	var ok bool
	if method, ok = commonPtr.MethodByName(methodName); !ok {
		return
	}

	var methodPatch *mpatch.Patch
	var err error
	methodPatch, err = mpatch.PatchMethodWithMakeFunc(method, func(in []reflect.Value) []reflect.Value {
		t := (*testing.T)(unsafe.Pointer(in[0].Pointer()))
		if t == nil {
			instrumentation.Logger().Println("testing.T is nil")
			return nil
		}
		test := GetTest(t)
		if test == nil {
			instrumentation.Logger().Printf("test struct for %v doesn't exist\n", t.Name())
			return nil
		}
		methodBody(test, in[1:])
		return nil
	})
	logOnError(err)
	if err == nil {
		patches[methodName] = methodPatch
		patchPointers[reflect.ValueOf(methodBody).Pointer()] = true
	}
}

func logOnError(err error) {
	if err != nil {
		instrumentation.Logger().Println(err)
	}
}

func isAPatchPointer(ptr uintptr) bool {
	patchPointersMutex.RLock()
	defer patchPointersMutex.RUnlock()
	if _, ok := patchPointers[ptr]; ok {
		return true
	}
	return false
}

func getMethodPatch(methodName string) *mpatch.Patch {
	patchesMutex.RLock()
	defer patchesMutex.RUnlock()
	return patches[methodName]
}
