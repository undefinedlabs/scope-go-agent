package testing

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"go.undefinedlabs.com/scopeagent/reflection"
)

func TestLogBufferRegex(t *testing.T) {
	test := StartTest(t)
	defer test.End()

	expectedLogLines := []string{
		"Hello World",
		"Hello        World     With         Spaces",
		"Hello\n World\nMulti\n        Line",
	}

	for _, item := range expectedLogLines {
		t.Log(item)
	}

	logBuffer := extractTestOutput(t)
	logs := string(*logBuffer)
	for idx, matches := range findMatchesLogRegex(logs) {
		if expectedLogLines[idx] != matches[3] {
			t.FailNow()
		}
	}
}

func TestExtractSubTestLogBuffer(t *testing.T) {
	t.Run("SubTest", TestLogBufferRegex)
}

func BenchmarkTestInit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tests := append(make([]testing.InternalTest, 0),
			testing.InternalTest{Name: "Test01", F: func(t *testing.T) {}},
			testing.InternalTest{Name: "Test02", F: func(t *testing.T) {}},
			testing.InternalTest{Name: "Test03", F: func(t *testing.T) {}},
			testing.InternalTest{Name: "Test04", F: func(t *testing.T) {}},
			testing.InternalTest{Name: "Test05", F: func(t *testing.T) {}},
		)
		benchmarks := append(make([]testing.InternalBenchmark, 0),
			testing.InternalBenchmark{Name: "Test01", F: func(b *testing.B) {}},
			testing.InternalBenchmark{Name: "Test02", F: func(b *testing.B) {}},
			testing.InternalBenchmark{Name: "Test03", F: func(b *testing.B) {}},
			testing.InternalBenchmark{Name: "Test04", F: func(b *testing.B) {}},
			testing.InternalBenchmark{Name: "Test05", F: func(b *testing.B) {}},
		)
		Init(testing.MainStart(nil, tests, benchmarks, nil))
	}
}

func BenchmarkLoggerPatcher(b *testing.B) {
	for i := 0; i < b.N; i++ {
		PatchTestingLogger()
		UnpatchTestingLogger()
	}
}

func TestLoggerPatcher(t *testing.T) {
	tm := time.Now()
	PatchTestingLogger()
	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(x int) {
			defer wg.Done()
			t.Log(fmt.Sprintf("Hello world %d", x))
		}(i)
	}
	wg.Wait()
	UnpatchTestingLogger()
	if time.Since(tm) > 2*time.Second {
		t.Fatal("Test is too slow")
	}
}

func TestIsParallelByReflection(t *testing.T) {
	t.Parallel()
	tm := time.Now()
	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = reflection.GetIsParallel(t)
		}()
	}
	wg.Wait()
	if time.Since(tm) > time.Second {
		t.Fatal("Test is too slow")
	}
}
