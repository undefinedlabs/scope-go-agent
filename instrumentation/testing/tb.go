package testing

import (
	"fmt"
	"github.com/opentracing/opentracing-go/log"
	"path/filepath"

	"go.undefinedlabs.com/scopeagent/errors"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/tags"
)

// ***************************
// TB interface implementation
func (test *Test) private() {}

func (test *Test) Error(args ...interface{}) {
	test.t.Helper()
	if test.span != nil {
		test.span.LogFields(
			log.String(tags.EventType, tags.LogEvent),
			log.String(tags.EventMessage, fmt.Sprint(args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
			log.String("log.internal_level", "Error"),
			log.String("log.logger", "testing"),
		)
	}
	test.t.Error(args...)
}

func (test *Test) Errorf(format string, args ...interface{}) {
	test.t.Helper()
	if test.span != nil {
		test.span.LogFields(
			log.String(tags.EventType, tags.LogEvent),
			log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
			log.String("log.internal_level", "Error"),
			log.String("log.logger", "testing"),
		)
	}
	test.t.Errorf(format, args...)
}

func (test *Test) Fail() {
	test.t.Helper()
	test.t.Fail()
}

func (test *Test) FailNow() {
	test.t.Helper()
	test.t.FailNow()
}

func (test *Test) Failed() bool {
	test.t.Helper()
	return test.t.Failed()
}

func (test *Test) Fatal(args ...interface{}) {
	test.t.Helper()
	if test.span != nil {
		test.span.LogFields(
			log.String(tags.EventType, tags.EventTestFailure),
			log.String(tags.EventMessage, fmt.Sprint(args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String("log.internal_level", "Fatal"),
			log.String("log.logger", "testing"),
		)
	}
	test.t.Fatal(args...)
}

func (test *Test) Fatalf(format string, args ...interface{}) {
	test.t.Helper()
	if test.span != nil {
		test.span.LogFields(
			log.String(tags.EventType, tags.EventTestFailure),
			log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String("log.internal_level", "Fatal"),
			log.String("log.logger", "testing"),
		)
	}
	test.t.Fatalf(format, args...)
}

func (test *Test) Log(args ...interface{}) {
	test.t.Helper()
	if test.span != nil {
		test.span.LogFields(
			log.String(tags.EventType, tags.LogEvent),
			log.String(tags.EventMessage, fmt.Sprint(args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String(tags.LogEventLevel, tags.LogLevel_INFO),
			log.String("log.internal_level", "Log"),
			log.String("log.logger", "testing"),
		)
	}
	test.t.Log(args...)
}

func (test *Test) Logf(format string, args ...interface{}) {
	test.t.Helper()
	if test.span != nil {
		test.span.LogFields(
			log.String(tags.EventType, tags.LogEvent),
			log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String(tags.LogEventLevel, tags.LogLevel_INFO),
			log.String("log.internal_level", "Log"),
			log.String("log.logger", "testing"),
		)
	}
	test.t.Logf(format, args...)
}

func (test *Test) Name() string {
	return test.t.Name()
}

func (test *Test) Skip(args ...interface{}) {
	test.t.Helper()
	if test.span != nil {
		test.span.LogFields(
			log.String(tags.EventType, tags.EventTestSkip),
			log.String(tags.EventMessage, fmt.Sprint(args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String("log.internal_level", "Skip"),
			log.String("log.logger", "testing"),
		)
	}
	test.t.Skip(args...)
}

func (test *Test) SkipNow() {
	test.t.Helper()
	test.t.SkipNow()
}

func (test *Test) Skipf(format string, args ...interface{}) {
	test.t.Helper()
	if test.span != nil {
		test.span.LogFields(
			log.String(tags.EventType, tags.EventTestSkip),
			log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
			log.String(tags.EventSource, getSourceFileAndNumber()),
			log.String("log.internal_level", "Skip"),
			log.String("log.logger", "testing"),
		)
	}
	test.t.Skipf(format, args...)
}

func (test *Test) Skipped() bool {
	return test.t.Skipped()
}

// Deprecated: use `testing.T.Helper` instead
func (test *Test) Helper() {
	test.t.Helper()
}

// Log panic data with stacktrace
func (test *Test) LogPanic(recoverData interface{}, skipFrames int) {
	errors.LogPanic(test.ctx, recoverData, skipFrames+1)
}

func getSourceFileAndNumber() string {
	var source string
	if _, file, line, ok := instrumentation.GetCallerInsideSourceRoot(2); ok == true {
		file = filepath.Clean(file)
		source = fmt.Sprintf("%s:%d", file, line)
	}
	return source
}
