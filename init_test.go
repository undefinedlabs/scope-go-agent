package scopeagent

import (
	"math/rand"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(Run(m))
}

func TestSkipped(t *testing.T) {
	test := StartTest(t)
	defer test.End()
}

func TestFirstTest(t *testing.T) {
	test := StartTest(t)
	defer test.End()
}

func TestFail(t *testing.T) {
	test := StartTest(t)
	defer test.End()
	t.FailNow()
}

func TestFlaky(t *testing.T) {
	test := StartTest(t)
	defer test.End()
	value := rand.Intn(8)
	t.Log("Value", value)
	if value <= 5 {
		t.FailNow()
	}
}

func TestError(t *testing.T) {
	test := StartTest(t)
	defer test.End()
	a := 0
	b := 5 / a
	_ = b
	t.FailNow()
}
