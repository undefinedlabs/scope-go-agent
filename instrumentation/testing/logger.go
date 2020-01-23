package testing

import (
	"reflect"
	"sync"
	"testing"
	"unsafe"

	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent/instrumentation"
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
	var t testing.T
	typeOfT := reflect.TypeOf(t)
	if cm, ok := typeOfT.FieldByName("common"); ok {
		commonPtr = reflect.PtrTo(cm.Type)
	}
}

func PatchTestingLogger() {
	patchPointersMutex.Lock()
	defer patchPointersMutex.Unlock()
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
	patchPointersMutex.Lock()
	defer patchPointersMutex.Unlock()
	for _, patch := range patches {
		logOnError(patch.Unpatch())
	}
	patches = map[string]*mpatch.Patch{}
	patchPointers = map[uintptr]bool{}
}

func patchError() {
	fn := func(test *Test, argsValues []reflect.Value) {
		args := getArgs(argsValues[0])
		test.Error(args...)
	}
	patch("Error", fn)
	patchPointers[reflect.ValueOf(fn).Pointer()] = true
}

func patchErrorf() {
	fn := func(test *Test, argsValues []reflect.Value) {
		format := argsValues[0].String()
		args := getArgs(argsValues[1])
		test.Errorf(format, args...)
	}
	patch("Errorf", fn)
	patchPointers[reflect.ValueOf(fn).Pointer()] = true
}

func patchFatal() {
	fn := func(test *Test, argsValues []reflect.Value) {
		args := getArgs(argsValues[0])
		test.Fatal(args...)
	}
	patch("Fatal", fn)
	patchPointers[reflect.ValueOf(fn).Pointer()] = true
}

func patchFatalf() {
	fn := func(test *Test, argsValues []reflect.Value) {
		format := argsValues[0].String()
		args := getArgs(argsValues[1])
		test.Fatalf(format, args...)
	}
	patch("Fatalf", fn)
	patchPointers[reflect.ValueOf(fn).Pointer()] = true
}

func patchLog() {
	fn := func(test *Test, argsValues []reflect.Value) {
		args := getArgs(argsValues[0])
		test.Log(args...)
	}
	patch("Log", fn)
	patchPointers[reflect.ValueOf(fn).Pointer()] = true
}

func patchLogf() {
	fn := func(test *Test, argsValues []reflect.Value) {
		format := argsValues[0].String()
		args := getArgs(argsValues[1])
		test.Logf(format, args...)
	}
	patch("Logf", fn)
	patchPointers[reflect.ValueOf(fn).Pointer()] = true
}

func patchSkip() {
	fn := func(test *Test, argsValues []reflect.Value) {
		args := getArgs(argsValues[0])
		test.Skip(args...)
	}
	patch("Skip", fn)
	patchPointers[reflect.ValueOf(fn).Pointer()] = true
}

func patchSkipf() {
	fn := func(test *Test, argsValues []reflect.Value) {
		format := argsValues[0].String()
		args := getArgs(argsValues[1])
		test.Skipf(format, args...)
	}
	patch("Skipf", fn)
	patchPointers[reflect.ValueOf(fn).Pointer()] = true
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
