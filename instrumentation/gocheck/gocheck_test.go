package gocheck

import (
	"testing"

	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
	. "gopkg.in/check.v1"
)

var (
	failCount  = 0
	fatalCount = 0
	panicCount = 0
)

// Hook up gocheck into the "go test" runner.
func TestM(t *testing.T) {
	TestingT(t)
}

type MySuite struct{}

var _ = Suite(&MySuite{})

/*
func (s *MySuite) TestHelloWorld(c *C) {
	// panic("")
	//c.Assert(42, chk.Equals, "42")
	//c.Assert(io.ErrClosedPipe, chk.ErrorMatches, "io: .*on closed pipe")
	//c.Check(42, chk.Equals, 42)
}
*/

func (s *MySuite) TestPass(c *C) {
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

/*
func (s *MySuite) TestExpected(c *C) {
	c.ExpectFailure("expected failure")
}
*/
