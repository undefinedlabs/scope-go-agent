package testing

import (
	"fmt"
	"github.com/opentracing/opentracing-go/log"
	"go.undefinedlabs.com/scopeagent/tags"
	"runtime"
)

// ***************************
// TB interface implementation
func (test *Test) private() {}
func (test *Test) Error(args ...interface{}) {
	var source string
	if _, file, line, ok := runtime.Caller(1); ok == true {
		source = fmt.Sprintf("%s:%d", file, line)
	}
	test.span.LogFields(
		log.String(tags.EventType, tags.LogEvent),
		log.String(tags.EventMessage, fmt.Sprint(args...)),
		log.String(tags.EventSource, source),
		log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
		log.String("log.internal_level", "Error"),
		log.String("log.logger", "ScopeAgent"),
	)
	test.innerLoggerUsed = true
	test.t.Error(args...)
}
func (test *Test) Errorf(format string, args ...interface{}) {
	var source string
	if _, file, line, ok := runtime.Caller(1); ok == true {
		source = fmt.Sprintf("%s:%d", file, line)
	}
	test.span.LogFields(
		log.String(tags.EventType, tags.LogEvent),
		log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
		log.String(tags.EventSource, source),
		log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
		log.String("log.internal_level", "Error"),
		log.String("log.logger", "ScopeAgent"),
	)
	test.innerLoggerUsed = true
	test.t.Errorf(format, args...)
}
func (test *Test) Fail() {
	test.t.Fail()
}
func (test *Test) FailNow() {
	test.t.FailNow()
}
func (test *Test) Failed() bool {
	return test.t.Failed()
}
func (test *Test) Fatal(args ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok == true {
		test.failReasonSource = fmt.Sprintf("%s:%d", file, line)
	}
	test.failReason = fmt.Sprint(args...)
	test.innerLoggerUsed = true
	test.t.Fatal(args...)
}
func (test *Test) Fatalf(format string, args ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok == true {
		test.failReasonSource = fmt.Sprintf("%s:%d", file, line)
	}
	test.failReason = fmt.Sprintf(format, args...)
	test.innerLoggerUsed = true
	test.t.Fatalf(format, args...)
}
func (test *Test) Log(args ...interface{}) {
	var source string
	if _, file, line, ok := runtime.Caller(1); ok == true {
		source = fmt.Sprintf("%s:%d", file, line)
	}
	test.span.LogFields(
		log.String(tags.EventType, tags.LogEvent),
		log.String(tags.EventMessage, fmt.Sprint(args...)),
		log.String(tags.EventSource, source),
		log.String(tags.LogEventLevel, tags.LogLevel_INFO),
		log.String("log.internal_level", "Log"),
		log.String("log.logger", "ScopeAgent"),
	)
	test.innerLoggerUsed = true
	test.t.Log(args...)
}
func (test *Test) Logf(format string, args ...interface{}) {
	var source string
	if _, file, line, ok := runtime.Caller(1); ok == true {
		source = fmt.Sprintf("%s:%d", file, line)
	}
	test.span.LogFields(
		log.String(tags.EventType, tags.LogEvent),
		log.String(tags.EventMessage, fmt.Sprintf(format, args...)),
		log.String(tags.EventSource, source),
		log.String(tags.LogEventLevel, tags.LogLevel_INFO),
		log.String("log.internal_level", "Log"),
		log.String("log.logger", "ScopeAgent"),
	)
	test.innerLoggerUsed = true
	test.t.Logf(format, args...)
}
func (test *Test) Name() string {
	return test.t.Name()
}
func (test *Test) Skip(args ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok == true {
		test.skipReasonSource = fmt.Sprintf("%s:%d", file, line)
	}
	test.skipReason = fmt.Sprint(args...)
	test.innerLoggerUsed = true
	test.t.Skip(args...)
}
func (test *Test) SkipNow() {
	test.t.SkipNow()
}
func (test *Test) Skipf(format string, args ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok == true {
		test.skipReasonSource = fmt.Sprintf("%s:%d", file, line)
	}
	test.skipReason = fmt.Sprintf(format, args...)
	test.innerLoggerUsed = true
	test.t.Skipf(format, args...)
}
func (test *Test) Skipped() bool {
	return test.t.Skipped()
}
func (test *Test) Helper() {
	test.t.Helper()
}
