package scopeagent // import "go.undefinedlabs.com/scopeagent"

import (
	"go.undefinedlabs.com/scopeagent/agent"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	scopetesting "go.undefinedlabs.com/scopeagent/instrumentation/testing"
	"runtime"
	"testing"
)

var (
	GlobalAgent *agent.Agent
)

func init() {
	defaultAgent, err := agent.NewAgent()
	if err != nil {
		return
	}

	GlobalAgent = defaultAgent

	if agent.GetBoolEnv("SCOPE_SET_GLOBAL_TRACER", true) {
		GlobalAgent.SetAsGlobalTracer()
	}

	if agent.GetBoolEnv("SCOPE_AUTO_INSTRUMENT", true) {
		if err := instrumentation.PatchAll(); err != nil {
			panic(err)
		}
	}
}

func Run(m *testing.M) int {
	if GlobalAgent != nil {
		GlobalAgent.SetTestingMode(true)
		defer GlobalAgent.Stop()
	}
	result := m.Run()
	return result
}

func StartTest(t *testing.T, opts ...scopetesting.Option) *scopetesting.Test {
	opts = append(opts, scopetesting.WithOnPanicHandler(func(test *scopetesting.Test) {
		if GlobalAgent != nil {
			_ = GlobalAgent.Flush()
			GlobalAgent.PrintReport()
		}
	}))
	pc, _, _, _ := runtime.Caller(1)
	return scopetesting.StartTestFromCaller(t, pc, opts...)
}
