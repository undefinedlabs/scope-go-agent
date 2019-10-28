package scopeagent // import "go.undefinedlabs.com/scopeagent"

import (
	"go.undefinedlabs.com/scopeagent/agent"
	scopetesting "go.undefinedlabs.com/scopeagent/instrumentation/testing"
	"runtime"
	"testing"
)

var defaultAgent *agent.Agent

// Helper function to run a `testing.M` object and gracefully stopping the agent afterwards
func Run(m *testing.M, opts ...agent.Option) int {
	opts = append(opts, agent.WithTestingModeEnabled())
	newAgent, err := agent.NewAgent(opts...)
	if err != nil {
		return m.Run()
	}

	defer newAgent.Stop()
	defaultAgent = newAgent
	return m.Run()
}

// Instruments the given test, returning a `Test` object that can be used to extend the test trace
func StartTest(t *testing.T, opts ...scopetesting.Option) *scopetesting.Test {
	opts = append(opts, scopetesting.WithOnPanicHandler(func(test *scopetesting.Test) {
		if defaultAgent != nil {
			_ = defaultAgent.Flush()
			defaultAgent.PrintReport()
		}
	}))
	pc, _, _, _ := runtime.Caller(1)
	return scopetesting.StartTestFromCaller(t, pc, opts...)
}

// Instruments the given benchmark
func StartBenchmark(b *testing.B, benchFunc func(b *testing.B)) {
	pc, _, _, _ := runtime.Caller(1)
	scopetesting.StartBenchmark(b, pc, benchFunc)
}