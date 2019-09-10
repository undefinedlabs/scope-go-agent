package scopeagent

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"os/exec"
	"strings"
)

type envCarrier struct {
	Env	*[]string
}
func (carrier *envCarrier) Set(key, val string) {
	var newCarrier []string
	keyUpper := strings.ToUpper(key)
	ctxKey := "CTX_" + keyUpper
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
				err := handler(kv[0][4:], kv[1])
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

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
func (test *Test) extract(env []string) (opentracing.SpanContext, error) {
	var carrier opentracing.TextMapReader
	carrier = &envCarrier{Env:&env}
	return GlobalAgent.Tracer.Extract(opentracing.TextMap, carrier)
}