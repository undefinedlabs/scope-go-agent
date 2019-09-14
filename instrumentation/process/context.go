package process

import (
	"context"
	"errors"
	"github.com/opentracing/opentracing-go"
	"go.undefinedlabs.com/scopeagent/tracer"
	"os"
	"sync"
)

var (
	processSpanContext *opentracing.SpanContext
	once sync.Once
)

// Injects a context to the environment variables array
func InjectFromContext(ctx context.Context, env *[]string) error {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		return Inject(span.Context(), env)
	}
	return errors.New("there are no spans in the context")
}

// Injects the span context to the environment variables array
func Inject(sm opentracing.SpanContext, env *[]string) error {
	return opentracing.GlobalTracer().Inject(sm, tracer.EnvironmentVariableFormat, env)
}

// Extracts the span context from an environment variables array
func Extract(env *[]string) (opentracing.SpanContext, error) {
	return opentracing.GlobalTracer().Extract(tracer.EnvironmentVariableFormat, env)
}

// Gets the current span context from the environment variables
func ProcessSpanContext() (opentracing.SpanContext, error) {
	once.Do(func() {
		env := os.Environ()
		if envCtx, err := Extract(&env); err == nil {
			processSpanContext = &envCtx
		}
	})
	if processSpanContext == nil {
		return nil, errors.New("process span context not found")
	}
	return *processSpanContext, nil
}