package gocheck

import (
	"testing"

	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
	. "gopkg.in/check.v1"
)

var (
	failCount     = 0
	fatalCount    = 0
	panicCount    = 0
	expectedCount = 0
	errorCount    = 0
)

// Hook up gocheck into the "go test" runner.
func TestM(t *testing.T) {
	TestingT(t)
}

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestPass(c *C) {
	c.Log("Hello", "World")
	c.Logf("Hello: %v", "World 2")
}
func (s *MySuite) TestSkip(c *C) {
	c.Skip("My skip reason")
}
func (s *MySuite) TestFail(c *C) {
	if failCount < 2 {
		failCount++
		c.Fail()
	}
}
func (s *MySuite) TestFatal(c *C) {
	if fatalCount < 2 {
		fatalCount++
		c.Fatal("fatal error")
	}
}
func (s *MySuite) TestPanic(c *C) {
	if panicCount < 2 {
		panicCount++
		panic("Custom panic")
	}
}
func (s *MySuite) TestExpected(c *C) {
	c.ExpectFailure("expected failure")
	expectedCount++
	if expectedCount > 2 {
		c.Fail()
	}
}
func (s *MySuite) TestError(c *C) {
	if errorCount < 2 {
		errorCount++
		c.Error("This is an error")
	}
}
