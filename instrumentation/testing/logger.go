package testing

import (
	"fmt"
	"go.undefinedlabs.com/scopeagent/reflection"
	"sync"
	"testing"
	_ "unsafe"

	"github.com/opentracing/opentracing-go/log"
	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/tags"
)

var (
	patchLock          sync.Mutex

	errorPatch  *mpatch.Patch
	errorfPatch *mpatch.Patch
	fatalPatch  *mpatch.Patch
	fatalfPatch *mpatch.Patch
	logPatch    *mpatch.Patch
	logfPatch   *mpatch.Patch
	skipPatch   *mpatch.Patch
	skipfPatch  *mpatch.Patch
)

//go:linkname lError testing.(*common).Error
func lError(t *testing.T, args ...interface{})

//go:linkname lErrorf testing.(*common).Errorf
func lErrorf(t *testing.T, format string, args ...interface{})

//go:linkname lFatal testing.(*common).Fatal
func lFatal(t *testing.T, args ...interface{})

//go:linkname lFatalf testing.(*common).Fatalf
func lFatalf(t *testing.T, format string, args ...interface{})

//go:linkname lLog testing.(*common).Log
func lLog(t *testing.T, args ...interface{})

//go:linkname lLogf testing.(*common).Logf
func lLogf(t *testing.T, format string, args ...interface{})

//go:linkname lSkip testing.(*common).Skip
func lSkip(t *testing.T, args ...interface{})

//go:linkname lSkipf testing.(*common).Skipf
func lSkipf(t *testing.T, format string, args ...interface{})

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

	if errorPatch != nil {
		logOnError(errorPatch.Unpatch())
	}
	if errorfPatch != nil {
		logOnError(errorfPatch.Unpatch())
	}
	if fatalPatch != nil {
		logOnError(fatalPatch.Unpatch())
	}
	if fatalfPatch != nil {
		logOnError(fatalfPatch.Unpatch())
	}
	if logPatch != nil {
		logOnError(logPatch.Unpatch())
	}
	if logfPatch != nil {
		logOnError(logfPatch.Unpatch())
	}
	if skipPatch != nil {
		logOnError(skipPatch.Unpatch())
	}
	if skipfPatch != nil {
		logOnError(skipfPatch.Unpatch())
	}
}

func patchError() {
	patchWithArgs(&errorPatch, lError, func(test *Test, args ...interface{}) {
		test.t.Helper()
		test.span.LogFields(
			log.String(tags.EventType, tags.LogEvent),
			log.String(tags.EventMessage, fmt.Sprint(args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
			log.String("log.internal_level", "Error"),
			log.String("log.logger", "testing"),
		)
	}, func(t *testing.T, args ...interface{}) {
		t.Helper()
		t.Error(args)
	})
}

func patchErrorf() {
	patchWithFormatAndArgs(&errorfPatch, lErrorf, func(test *Test, format string, args ...interface{}) {
		test.t.Helper()
		test.span.LogFields(
			log.String(tags.EventType, tags.LogEvent),
			log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
			log.String("log.internal_level", "Error"),
			log.String("log.logger", "testing"),
		)
	}, func(t *testing.T, format string, args ...interface{}) {
		t.Helper()
		t.Errorf(format, args)
	})
}

func patchFatal() {
	patchWithArgs(&fatalPatch, lFatal, func(test *Test, args ...interface{}) {
		test.t.Helper()
		test.span.LogFields(
			log.String(tags.EventType, tags.EventTestFailure),
			log.String(tags.EventMessage, fmt.Sprint(args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String("log.internal_level", "Fatal"),
			log.String("log.logger", "testing"),
		)
	}, func(t *testing.T, args ...interface{}) {
		t.Helper()
		t.Fatal(args)
	})
}

func patchFatalf() {
	patchWithFormatAndArgs(&fatalfPatch, lFatalf, func(test *Test, format string, args ...interface{}) {
		test.t.Helper()
		test.span.LogFields(
			log.String(tags.EventType, tags.EventTestFailure),
			log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String("log.internal_level", "Fatal"),
			log.String("log.logger", "testing"),
		)
	}, func(t *testing.T, format string, args ...interface{}) {
		t.Helper()
		t.Fatalf(format, args)
	})
}

func patchLog() {
	patchWithArgs(&logPatch, lLog, func(test *Test, args ...interface{}) {
		test.t.Helper()
		test.span.LogFields(
			log.String(tags.EventType, tags.LogEvent),
			log.String(tags.EventMessage, fmt.Sprint(args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String(tags.LogEventLevel, tags.LogLevel_INFO),
			log.String("log.internal_level", "Log"),
			log.String("log.logger", "testing"),
		)
	}, func(t *testing.T, args ...interface{}) {
		t.Helper()
		t.Log(args)
	})
}

func patchLogf() {
	patchWithFormatAndArgs(&logfPatch, lLogf, func(test *Test, format string, args ...interface{}) {
		test.t.Helper()
		test.span.LogFields(
			log.String(tags.EventType, tags.LogEvent),
			log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String(tags.LogEventLevel, tags.LogLevel_INFO),
			log.String("log.internal_level", "Log"),
			log.String("log.logger", "testing"),
		)
	}, func(t *testing.T, format string, args ...interface{}) {
		t.Helper()
		t.Logf(format, args)
	})
}

func patchSkip() {
	patchWithArgs(&skipPatch, lSkip, func(test *Test, args ...interface{}) {
		test.t.Helper()
		test.span.LogFields(
			log.String(tags.EventType, tags.EventTestSkip),
			log.String(tags.EventMessage, fmt.Sprint(args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String("log.internal_level", "Skip"),
			log.String("log.logger", "testing"),
		)
	}, func(t *testing.T, args ...interface{}) {
		t.Helper()
		t.Skip(args)
	})
}

func patchSkipf() {
	patchWithFormatAndArgs(&skipfPatch, lSkipf, func(test *Test, format string, args ...interface{}) {
		test.t.Helper()
		test.span.LogFields(
			log.String(tags.EventType, tags.EventTestSkip),
			log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String("log.internal_level", "Skip"),
			log.String("log.logger", "testing"),
		)
	}, func(t *testing.T, format string, args ...interface{}) {
		t.Helper()
		t.Skipf(format, args)
	})
}

func patchWithArgs(patchValue **mpatch.Patch, method interface{},
	spanFunc func(test *Test, args ...interface{}),
	oFunc func(t *testing.T, args ...interface{})) {

	lPatch, err := mpatch.PatchMethod(method, func(t *testing.T, args ...interface{}) {
		if t == nil {
			instrumentation.Logger().Println("testing.T is nil")
			return
		}
		t.Helper()
		test := GetTest(t)
		if test == nil {
			instrumentation.Logger().Printf("test struct for %v doesn't exist\n", t.Name())
			return
		}
		if test.span != nil {
			spanFunc(test, args...)
		}
		mu := reflection.GetTestMutex(test.t)
		if mu != nil {
			mu.Lock()
			(*patchValue).Unpatch()
			mu.Unlock()
			oFunc(t, args...)
			mu.Lock()
			(*patchValue).Patch()
			mu.Unlock()
		} else {
			(*patchValue).Unpatch()
			oFunc(t, args...)
			(*patchValue).Patch()
		}
	})
	logOnError(err)
	*patchValue = lPatch
}

func patchWithFormatAndArgs(patchValue **mpatch.Patch, method interface{},
	spanFunc func(test *Test, format string, args ...interface{}),
	oFunc func(t *testing.T, format string, args ...interface{})) {

	lPatch, err := mpatch.PatchMethod(method, func(t *testing.T, format string, args ...interface{}) {
		if t == nil {
			instrumentation.Logger().Println("testing.T is nil")
			return
		}
		t.Helper()
		test := GetTest(t)
		if test == nil {
			instrumentation.Logger().Printf("test struct for %v doesn't exist\n", t.Name())
			return
		}
		if test.span != nil {
			spanFunc(test, format, args...)
		}
		mu := reflection.GetTestMutex(test.t)
		if mu != nil {
			mu.Lock()
			(*patchValue).Unpatch()
			mu.Unlock()
			oFunc(t, format, args...)
			mu.Lock()
			(*patchValue).Patch()
			mu.Unlock()
		} else {
			(*patchValue).Unpatch()
			oFunc(t, format, args...)
			(*patchValue).Patch()
		}
	})
	logOnError(err)
	*patchValue = lPatch
}

func logOnError(err error) {
	if err != nil {
		instrumentation.Logger().Println(err)
	}
}
