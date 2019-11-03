package scopeagent

import (
	"math"
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

func TestFlaky(t *testing.T) {
	test := StartTest(t)
	defer test.End()
	value := rand.Intn(8)
	t.Log("Value", value)
	if value <= 5 {
		t.FailNow()
	}
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

func TestError(t *testing.T) {
	test := StartTest(t)
	defer test.End()
	a := 0
	b := 5 / a
	_ = b
	t.FailNow()
}

func Benchmark01(b *testing.B) {
	StartBenchmark(b, func(b *testing.B) {
		a := 1
		for i := 0; i < b.N; i++ {
			a = a*i*a + b.N
			math.Log(float64(a))
		}
	})
}

func Benchmark02(b *testing.B) {
	StartBenchmark(b, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			math.Log(float64(b.N))
		}
	})
}
