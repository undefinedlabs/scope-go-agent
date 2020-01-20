package testing

import (
	"fmt"
	"runtime"
)

// ***************************
// TB interface implementation
func (test *Test) private() {}

// Deprecated: use `testing.T.Error` instead
func (test *Test) Error(args ...interface{}) {
	test.t.Error(args...)
}

// Deprecated: use `testing.T.Error` instead
func (test *Test) Errorf(format string, args ...interface{}) {
	test.t.Errorf(format, args...)
}

// Deprecated: use `testing.T.Fail` instead
func (test *Test) Fail() {
	test.t.Fail()
}

// Deprecated: use `testing.T.FailNow` instead
func (test *Test) FailNow() {
	test.t.FailNow()
}

// Deprecated: use `testing.T.Failed` instead
func (test *Test) Failed() bool {
	return test.t.Failed()
}

// Deprecated: use `testing.T.Fatal` instead
func (test *Test) Fatal(args ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok == true {
		test.failReasonSource = fmt.Sprintf("%s:%d", file, line)
	}
	test.failReason = fmt.Sprint(args...)
	test.t.Fatal(args...)
}

// Deprecated: use `testing.T.Fatalf` instead
func (test *Test) Fatalf(format string, args ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok == true {
		test.failReasonSource = fmt.Sprintf("%s:%d", file, line)
	}
	test.failReason = fmt.Sprintf(format, args...)
	test.t.Fatalf(format, args...)
}

// Deprecated: use `testing.T.Log` instead
func (test *Test) Log(args ...interface{}) {
	test.t.Log(args...)
}

// Deprecated: use `testing.T.Logf` instead
func (test *Test) Logf(format string, args ...interface{}) {
	test.t.Logf(format, args...)
}

// Deprecated: use `testing.T.Name` instead
func (test *Test) Name() string {
	return test.t.Name()
}

// Deprecated: use `testing.T.Skip` instead
func (test *Test) Skip(args ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok == true {
		test.skipReasonSource = fmt.Sprintf("%s:%d", file, line)
	}
	test.skipReason = fmt.Sprint(args...)
	test.t.Skip(args...)
}

// Deprecated: use `testing.T.SkipNow` instead
func (test *Test) SkipNow() {
	test.t.SkipNow()
}

// Deprecated: use `testing.T.Skipf` instead
func (test *Test) Skipf(format string, args ...interface{}) {
	if _, file, line, ok := runtime.Caller(1); ok == true {
		test.skipReasonSource = fmt.Sprintf("%s:%d", file, line)
	}
	test.skipReason = fmt.Sprintf(format, args...)
	test.t.Skipf(format, args...)
}

// Deprecated: use `testing.T.Skipped` instead
func (test *Test) Skipped() bool {
	return test.t.Skipped()
}

// Deprecated: use `testing.T.Helper` instead
func (test *Test) Helper() {
	test.t.Helper()
}
