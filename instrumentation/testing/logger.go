package testing

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"unsafe"

	"github.com/opentracing/opentracing-go/log"
	"github.com/undefinedlabs/monkey"

	"go.undefinedlabs.com/scopeagent/tags"
)

var (
	// *testing.common type
	commonPtr reflect.Type

	// patch guards
	guards []*monkey.PatchGuard
)

func init() {
	var t testing.T
	typeOfT := reflect.TypeOf(t)
	if cm, ok := typeOfT.FieldByName("common"); ok {
		commonPtr = reflect.PtrTo(cm.Type)
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
	for _, guard := range guards {
		guard.Unpatch()
	}
	guards = nil
}

func patchError() {
	patch("Error", func(t *testing.T, argsValues []reflect.Value) {
		args := getArgs(argsValues[0])

		test := GetTest(t)
		if test != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.LogEvent),
				log.String(tags.EventMessage, fmt.Sprint(args...)),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
				log.String("log.internal_level", "Error"),
				log.String("log.logger", "testing"),
			)
		}

		t.Error(args...)
	})
}
func patchErrorf() {
	patch("Errorf", func(t *testing.T, argsValues []reflect.Value) {
		format := argsValues[0].String()
		args := getArgs(argsValues[1])

		test := GetTest(t)
		if test != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.LogEvent),
				log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
				log.String("log.internal_level", "Error"),
				log.String("log.logger", "testing"),
			)
		}

		t.Errorf(format, args...)
	})
}
func patchFatal() {
	patch("Fatal", func(t *testing.T, argsValues []reflect.Value) {
		args := getArgs(argsValues[0])

		test := GetTest(t)
		if test != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.EventTestFailure),
				log.String(tags.EventMessage, fmt.Sprint(args...)),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String("log.internal_level", "Fatal"),
				log.String("log.logger", "testing"),
			)
		}

		t.Fatal(args...)
	})
}
func patchFatalf() {
	patch("Fatalf", func(t *testing.T, argsValues []reflect.Value) {
		format := argsValues[0].String()
		args := getArgs(argsValues[1])

		test := GetTest(t)
		if test != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.EventTestFailure),
				log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String("log.internal_level", "Fatal"),
				log.String("log.logger", "testing"),
			)
		}

		t.Fatalf(format, args...)
	})
}
func patchLog() {
	patch("Log", func(t *testing.T, argsValues []reflect.Value) {
		args := getArgs(argsValues[0])

		test := GetTest(t)
		if test != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.LogEvent),
				log.String(tags.EventMessage, fmt.Sprint(args...)),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String(tags.LogEventLevel, tags.LogLevel_INFO),
				log.String("log.internal_level", "Log"),
				log.String("log.logger", "testing"),
			)
		}

		t.Log(args...)
	})
}
func patchLogf() {
	patch("Logf", func(t *testing.T, argsValues []reflect.Value) {
		format := argsValues[0].String()
		args := getArgs(argsValues[1])

		test := GetTest(t)
		if test != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.LogEvent),
				log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String(tags.LogEventLevel, tags.LogLevel_INFO),
				log.String("log.internal_level", "Log"),
				log.String("log.logger", "testing"),
			)
		}

		t.Logf(format, args...)
	})
}
func patchSkip() {
	patch("Skip", func(t *testing.T, argsValues []reflect.Value) {
		args := getArgs(argsValues[0])

		test := GetTest(t)
		if test != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.EventTestSkip),
				log.String(tags.EventMessage, fmt.Sprint(args...)),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String("log.internal_level", "Skip"),
				log.String("log.logger", "testing"),
			)
		}

		t.Skip(args...)
	})
}
func patchSkipf() {
	patch("Skipf", func(t *testing.T, argsValues []reflect.Value) {
		format := argsValues[0].String()
		args := getArgs(argsValues[1])

		test := GetTest(t)
		if test != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.EventTestSkip),
				log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String("log.internal_level", "Skip"),
				log.String("log.logger", "testing"),
			)
		}

		t.Skipf(format, args...)
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

func getSourceFileAndNumber() string {
	var source string
	if _, file, line, ok := runtime.Caller(5); ok == true {
		source = fmt.Sprintf("%s:%d", file, line)
	}
	return source
}

func patch(methodName string, methodBody func(t *testing.T, argsValues []reflect.Value)) {
	if method, ok := commonPtr.MethodByName(methodName); ok {
		var guard *monkey.PatchGuard
		newFunc := reflect.MakeFunc(method.Type, func(in []reflect.Value) []reflect.Value {
			guard.Unpatch()
			defer guard.Restore()
			t := (*testing.T)(unsafe.Pointer(in[0].Pointer()))
			in = in[1:]
			methodBody(t, in)
			return nil
		})
		guard = monkey.PatchInstanceMethod(commonPtr, methodName, newFunc)
		guards = append(guards, guard)
	}
}
