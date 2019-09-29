package scopeagent // import "go.undefinedlabs.com/scopeagent"

import (
	"go.undefinedlabs.com/scopeagent/agent"
	"go.undefinedlabs.com/scopeagent/instrumentation"
)

var (
	GlobalAgent *agent.Agent
)

func init() {
	defaultAgent, err := agent.NewAgent()
	if err == nil {
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
