package testing

import (
	"fmt"
	"sync"
	"testing"
	_ "unsafe"

	"github.com/opentracing/opentracing-go/log"
	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/tags"
)

var (
	patchLock sync.Mutex

	errorPatch  *mpatch.Patch
	errorfPatch *mpatch.Patch
	fatalPatch  *mpatch.Patch
	fatalfPatch *mpatch.Patch
	logPatch    *mpatch.Patch
	logfPatch   *mpatch.Patch
	skipPatch   *mpatch.Patch
	skipfPatch  *mpatch.Patch
)

//go:linkname llog testing.(*common).log
func llog(t *testing.T, s string)

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
		s := fmt.Sprintln(args...)
		if test.span != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.LogEvent),
				log.String(tags.EventMessage, s),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
				log.String("log.internal_level", "Error"),
				log.String("log.logger", "testing"),
			)
		}
		llog(test.t, s)
		test.t.Fail()
	})
}

func patchErrorf() {
	patchWithFormatAndArgs(&errorfPatch, lErrorf, func(test *Test, format string, args ...interface{}) {
		test.t.Helper()
		s := fmt.Sprintf(format, args...)
		if test.span != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.LogEvent),
				log.String(tags.EventMessage, s),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
				log.String("log.internal_level", "Error"),
				log.String("log.logger", "testing"),
			)
		}
		llog(test.t, s)
		test.t.Fail()
	})
}

func patchFatal() {
	patchWithArgs(&fatalPatch, lFatal, func(test *Test, args ...interface{}) {
		test.t.Helper()
		s := fmt.Sprintln(args...)
		if test.span != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.EventTestFailure),
				log.String(tags.EventMessage, s),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String("log.internal_level", "Fatal"),
				log.String("log.logger", "testing"),
			)
		}
		llog(test.t, s)
		test.t.FailNow()
	})
}

func patchFatalf() {
	patchWithFormatAndArgs(&fatalfPatch, lFatalf, func(test *Test, format string, args ...interface{}) {
		test.t.Helper()
		s := fmt.Sprintf(format, args...)
		if test.span != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.EventTestFailure),
				log.String(tags.EventMessage, s),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String("log.internal_level", "Fatal"),
				log.String("log.logger", "testing"),
			)
		}
		llog(test.t, s)
		test.t.FailNow()
	})
}

func patchLog() {
	patchWithArgs(&logPatch, lLog, func(test *Test, args ...interface{}) {
		test.t.Helper()
		s := fmt.Sprintln(args...)
		if test.span != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.LogEvent),
				log.String(tags.EventMessage, s),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String(tags.LogEventLevel, tags.LogLevel_INFO),
				log.String("log.internal_level", "Log"),
				log.String("log.logger", "testing"),
			)
		}
		llog(test.t, s)
	})
}

func patchLogf() {
	patchWithFormatAndArgs(&logfPatch, lLogf, func(test *Test, format string, args ...interface{}) {
		test.t.Helper()
		s := fmt.Sprintf(format, args...)
		if test.span != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.LogEvent),
				log.String(tags.EventMessage, s),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String(tags.LogEventLevel, tags.LogLevel_INFO),
				log.String("log.internal_level", "Log"),
				log.String("log.logger", "testing"),
			)
		}
		llog(test.t, s)
	})
}

func patchSkip() {
	patchWithArgs(&skipPatch, lSkip, func(test *Test, args ...interface{}) {
		test.t.Helper()
		s := fmt.Sprintln(args...)
		if test.span != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.EventTestSkip),
				log.String(tags.EventMessage, s),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String("log.internal_level", "Skip"),
				log.String("log.logger", "testing"),
			)
		}
		llog(test.t, s)
		test.t.SkipNow()
	})
}

func patchSkipf() {
	patchWithFormatAndArgs(&skipfPatch, lSkipf, func(test *Test, format string, args ...interface{}) {
		test.t.Helper()
		s := fmt.Sprintf(format, args...)
		if test.span != nil {
			test.span.LogFields(
				log.String(tags.EventType, tags.EventTestSkip),
				log.String(tags.EventMessage, s),
				log.String(tags.EventSource, getSourceFileAndNumber()),
				log.String("log.internal_level", "Skip"),
				log.String("log.logger", "testing"),
			)
		}
		llog(test.t, s)
		test.t.SkipNow()
	})
}

func patchWithArgs(patchValue **mpatch.Patch, method interface{}, methodBody func(test *Test, args ...interface{})) {
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
		methodBody(test, args...)
	})
	logOnError(err)
	*patchValue = lPatch
}

func patchWithFormatAndArgs(patchValue **mpatch.Patch, method interface{}, methodBody func(test *Test, format string, args ...interface{})) {
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
		methodBody(test, format, args...)
	})
	logOnError(err)
	*patchValue = lPatch
}

func logOnError(err error) {
	if err != nil {
		instrumentation.Logger().Println(err)
	}
}
