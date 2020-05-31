package testing

import (
	"runtime"
	"strings"
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
	llogPatch *mpatch.Patch
)

//go:linkname llog testing.(*common).log
func llog(t *testing.T, s string)

//go:linkname llogdepth testing.(*common).logDepth
func llogdepth(t *testing.T, s string, depth int)

func PatchTestingLogger() {
	patchLock.Lock()
	defer patchLock.Unlock()
	var err error
	llogPatch, err = mpatch.PatchMethod(llog, func(t *testing.T, s string) {
		pc, _, _, ok := runtime.Caller(1)
		if ok {
			name := runtime.FuncForPC(pc).Name()
			test := GetTest(t)
			if test != nil && test.span != nil {
				if strings.HasSuffix(name, ".Error") || strings.HasSuffix(name, ".Errorf") {
					test.span.LogFields(
						log.String(tags.EventType, tags.LogEvent),
						log.String(tags.EventMessage, s),
						log.String(tags.EventSource, getSourceFileAndNumber()),
						log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
						log.String("log.internal_level", "Error"),
						log.String("log.logger", "testing"),
					)
				} else if strings.HasSuffix(name, ".Fatal") || strings.HasSuffix(name, ".Fatalf") {
					test.span.LogFields(
						log.String(tags.EventType, tags.EventTestFailure),
						log.String(tags.EventMessage, s),
						log.String(tags.EventSource, getSourceFileAndNumber()),
						log.String("log.internal_level", "Fatal"),
						log.String("log.logger", "testing"),
					)
				} else if strings.HasSuffix(name, ".Log") || strings.HasSuffix(name, ".Logf") {
					test.span.LogFields(
						log.String(tags.EventType, tags.LogEvent),
						log.String(tags.EventMessage, s),
						log.String(tags.EventSource, getSourceFileAndNumber()),
						log.String(tags.LogEventLevel, tags.LogLevel_INFO),
						log.String("log.internal_level", "Log"),
						log.String("log.logger", "testing"),
					)
				} else if strings.HasSuffix(name, ".Skip") || strings.HasSuffix(name, ".Skipf") {
					test.span.LogFields(
						log.String(tags.EventType, tags.EventTestSkip),
						log.String(tags.EventMessage, s),
						log.String(tags.EventSource, getSourceFileAndNumber()),
						log.String("log.internal_level", "Skip"),
						log.String("log.logger", "testing"),
					)
				}
			}
		}
		llogdepth(t, s, 3)
	})
	logOnError(err)
}

func UnpatchTestingLogger() {
	patchLock.Lock()
	defer patchLock.Unlock()
	if llogPatch != nil {
		logOnError(llogPatch.Unpatch())
	}
}

func logOnError(err error) {
	if err != nil {
		instrumentation.Logger().Println(err)
	}
}
