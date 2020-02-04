package autoinstrument

import (
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent/agent"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/instrumentation/logging"
	"go.undefinedlabs.com/scopeagent/instrumentation/nethttp"
	scopetesting "go.undefinedlabs.com/scopeagent/instrumentation/testing"
)

var (
	once         sync.Once
	defaultAgent *agent.Agent
)

func init() {
	once.Do(func() {
		if envDMPatch, set := os.LookupEnv("SCOPE_DISABLE_MONKEY_PATCHING"); !set || envDMPatch == "" {
			// We monkey patch the `testing.M.Run()` func to patch and unpatch the testing logger methods
			var m *testing.M
			mType := reflect.TypeOf(m)
			if mRunMethod, ok := mType.MethodByName("Run"); ok {
				var runPatch *mpatch.Patch
				var err error
				runPatch, err = mpatch.PatchMethodByReflect(mRunMethod, func(m *testing.M) int {
					logOnError(runPatch.Unpatch())
					defer func() {
						logOnError(runPatch.Patch())
					}()
					scopetesting.PatchTestingLogger()
					defer scopetesting.UnpatchTestingLogger()
					nethttp.PatchHttpDefaultClient()

					newAgent, err := agent.NewAgent(agent.WithSetGlobalTracer(), agent.WithTestingModeEnabled())
					if err != nil {
						return m.Run()
					}

					logging.PatchStandardLogger()

					scopetesting.Init(m)
					scopetesting.SetDefaultPanicHandler(func(test *scopetesting.Test) {
						if defaultAgent != nil {
							_ = defaultAgent.Flush()
							defaultAgent.PrintReport()
						}
					})

					defer newAgent.Stop()
					defaultAgent = newAgent
					return newAgent.Run(m)
				})
				logOnError(err)
			}
		}
	})
}

func logOnError(err error) {
	if err != nil {
		instrumentation.Logger().Println(err)
	}
}