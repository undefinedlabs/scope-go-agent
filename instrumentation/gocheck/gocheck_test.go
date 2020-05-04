package gocheck

import (
	"testing"

	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func TestM(t *testing.T) {
	TestingT(t)
}

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestHelloWorld(c *C) {
	// panic("")
	//c.Assert(42, chk.Equals, "42")
	//c.Assert(io.ErrClosedPipe, chk.ErrorMatches, "io: .*on closed pipe")
	//c.Check(42, chk.Equals, 42)
}

func (s MySuite) TestOther(c *C) {

}
