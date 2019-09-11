package scopeagent

import (
	"context"
	"errors"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"os"
	"os/exec"
	"strings"
)

const (
	environmentKeyPrefix = "CTX_"
)

var (
	escapeMap          map[string]string
	processSpanContext *opentracing.SpanContext
)

func init() {
	escapeMap = map[string]string{
		".": "__dt__",
		"-": "__dh__",
	}
	if envCtx, err := ExtractFromEnvVars(os.Environ()); err == nil {
		processSpanContext = &envCtx
	}
}

// Environment Carrier
type envCarrier struct {
	Env *[]string
}

func (carrier *envCarrier) Set(key, val string) {
	var newCarrier []string
	keyUpper := strings.ToUpper(key)
	ctxKey := escape(environmentKeyPrefix + keyUpper)
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
			if strings.Index(item, environmentKeyPrefix) >= 0 {
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
	for key, val := range escapeMap {
		value = strings.ReplaceAll(value, key, val)
	}
	return value
}
func unescape(value string) string {
	for key, val := range escapeMap {
		value = strings.ReplaceAll(value, val, key)
	}
	return value
}

// Injects the test context to the command environment variables
func (test *Test) Inject(command *exec.Cmd) *exec.Cmd {
	command.Env = append(command.Env, GetSpanContextEnvVars(test.span.Context())...)
	appendScopeEnvVars(command)
	return command
}

// Injects the span context to the command environment variables
func Inject(ctx context.Context, command *exec.Cmd) *exec.Cmd {
	command.Env = append(command.Env, GetContextEnvVars(ctx)...)
	appendScopeEnvVars(command)
	return command
}

// Injects the span context to an environment variables array
func GetSpanContextEnvVars(ctx opentracing.SpanContext) []string {
	var carrier opentracing.TextMapWriter
	carrier = &envCarrier{}
	err := GlobalAgent.Tracer.Inject(ctx, opentracing.TextMap, carrier)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	return *carrier.(*envCarrier).Env
}

// Gets the span context env vars
func GetContextEnvVars(ctx context.Context) []string {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		return GetSpanContextEnvVars(span.Context())
	}
	return []string{}
}

// Extract the context from an environment variables array
func ExtractFromEnvVars(env []string) (opentracing.SpanContext, error) {
	var carrier opentracing.TextMapReader
	carrier = &envCarrier{Env: &env}
	return GlobalAgent.Tracer.Extract(opentracing.TextMap, carrier)
}

// Gets the current span context from the environment variables
func ProcessSpanContext() (opentracing.SpanContext, error) {
	if processSpanContext == nil {
		return nil, errors.New("process span context not found")
	}
	return *processSpanContext, nil
}

// Append all SCOPE_* environment variables to a child command
func appendScopeEnvVars(command *exec.Cmd) {
	var othersVars []string
	envVars := os.Environ()
	for _, item := range envVars {
		if strings.Index(item, "SCOPE_") == 0 {
			kv := strings.Split(item, "=")
			for _, cmdEnvItem := range command.Env {
				if strings.Index(cmdEnvItem, kv[0]) == 0 {
					othersVars = append(othersVars, cmdEnvItem)
				}
			}
		}
	}
	if len(othersVars) > 0 {
		command.Env = append(command.Env, othersVars...)
	}
}