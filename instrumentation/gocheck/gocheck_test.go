package gocheck

import (
	"fmt"
	"io"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

type MySuite struct{}

var _ = Suite(&MySuite{})


func (s *MySuite) SetUpTest(c *C) {
	fmt.Println("Start Test", c.TestName())
}

func (s *MySuite) TearDownTest(c *C) {
	fmt.Println("End Test", c.TestName())
}

func (s *MySuite) TestHelloWorld(c *C) {
	c.Assert(42, Equals, "42")
	c.Assert(io.ErrClosedPipe, ErrorMatches, "io: .*on closed pipe")
	c.Check(42, Equals, 42)
}