package scopeagent // import "go.undefinedlabs.com/scopeagent"

import (
	"github.com/opentracing/opentracing-go"
	"go.undefinedlabs.com/scopeagent/agent"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/instrumentation/nethttp"
	scopetesting "go.undefinedlabs.com/scopeagent/instrumentation/testing"
	"runtime"
	"testing"
)

var globalAgent *agent.Agent

// Tries to autoinstall the Scope agent if we can autodetect the API key, otherwise does nothing
// If an agent could be installed, it also applies the autoinstrumentation by default
func init() {
	defaultAgent, err := agent.NewAgent()
	if err != nil {
		return
	}

	globalAgent = defaultAgent
	instrumentation.SetTracer(globalAgent.Tracer())

	if agent.GetBoolEnv("SCOPE_SET_GLOBAL_TRACER", false) {
		opentracing.SetGlobalTracer(globalAgent.Tracer())
	}

	if agent.GetBoolEnv("SCOPE_AUTO_INSTRUMENT", true) {
		if err := PatchAll(); err != nil {
			panic(err)
		}
	}
}

// Returns the autoinstalled agent instance, if any
func GlobalAgent() *agent.Agent {
	return globalAgent
}

// Helper function to run a `testing.M` object and gracefully stopping the agent afterwards
func Run(m *testing.M) int {
	if globalAgent != nil {
		globalAgent.SetTestingMode(true)
		defer globalAgent.Stop()
	}
	return m.Run()
}

// Gracefully stops the Scope agent, flushing any buffers before returning
func Stop() {
	if globalAgent != nil {
		globalAgent.Stop()
	}
}

// Instruments the given test, returning a `Test` object that can be used to extend the test trace
func StartTest(t *testing.T, opts ...scopetesting.Option) *scopetesting.Test {
	opts = append(opts, scopetesting.WithOnPanicHandler(func(test *scopetesting.Test) {
		if globalAgent != nil {
			_ = globalAgent.Flush()
			globalAgent.PrintReport()
		}
	}))
	pc, _, _, _ := runtime.Caller(1)
	return scopetesting.StartTestFromCaller(t, pc, opts...)
}

// Apply all the available autoinstrumentation
func PatchAll() error {
	if err := nethttp.PatchHttpDefaultClient(); err != nil {
		return err
	}
	return nil
}
