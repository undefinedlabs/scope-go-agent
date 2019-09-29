package scopeagent // import "go.undefinedlabs.com/scopeagent"

import (
	"github.com/opentracing/opentracing-go"
	"go.undefinedlabs.com/scopeagent/agent"
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
		opentracing.SetGlobalTracer(GlobalAgent.Tracer)
	}

	if agent.GetBoolEnv("SCOPE_AUTO_INSTRUMENT", true) {
		if err := agent.PatchAll(); err != nil {
			panic(err)
		}
	}
}
