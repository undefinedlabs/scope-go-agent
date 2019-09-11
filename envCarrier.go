package scopeagent

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"os"
	"os/exec"
	"strings"
)

type envCarrier struct {
	Env	*[]string
}
func (carrier *envCarrier) Set(key, val string) {
	var newCarrier []string
	keyUpper := strings.ToUpper(key)
	ctxKey := escape("CTX_" + keyUpper)
	if carrier.Env != nil {
		for _, item := range *carrier.Env {
			if strings.Index(item, ctxKey) < 0 {
				newCarrier = append(newCarrier, item)
			}
		}
	}
	newCarrier = append(newCarrier, fmt.Sprintf("%s=%s", ctxKey, val))
	carrier.Env = &newCarrier
}
func (carrier *envCarrier) ForeachKey(handler func(key, val string) error) error {
	if carrier.Env != nil {
		for _, item := range *carrier.Env {
			if strings.Index(item, "CTX_") >= 0 {
				kv := strings.Split(item, "=")
				err := handler(unescape(kv[0][4:]), kv[1])
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// We need to sanitize the env vars due:
// Environment variable names used by the utilities in the Shell and Utilities volume of IEEE Std 1003.1-2001
// consist solely of uppercase letters, digits, and the '_' (underscore)
func escape(value string) string {
	return value
}
func unescape(value string) string {
	return value
}


// Injects the test context to the command environment variables
func (test *Test) Inject(command *exec.Cmd) *exec.Cmd {
	var carrier opentracing.TextMapWriter
	carrier = &envCarrier{}
	err := GlobalAgent.Tracer.Inject(test.span.Context(), opentracing.TextMap, carrier)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	command.Env = append(command.Env, *carrier.(*envCarrier).Env...)
	return command
}

// Extract the context from an environment variables array
func extract(env []string) (opentracing.SpanContext, error) {
	var carrier opentracing.TextMapReader
	carrier = &envCarrier{Env:&env}
	return GlobalAgent.Tracer.Extract(opentracing.TextMap, carrier)
}

// Gets the current span context from the environment variables
func getContextFromEnvironment() (opentracing.SpanContext, error) {
	return extract(os.Environ())
}