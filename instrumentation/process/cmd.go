package process

import (
	"context"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	"os/exec"
	"path/filepath"
)

// Injects the span context to the command environment variables
func InjectToCmd(ctx context.Context, command *exec.Cmd) *exec.Cmd {
	if command.Env == nil {
		command.Env = []string{}
	}
	err := InjectFromContext(ctx, &command.Env)
	if err != nil {
		fmt.Println(err)
	}
	return command
}

// Injects a new span context to the command environment variables
func InjectToCmdWithSpan(ctx context.Context, command *exec.Cmd) (opentracing.Span, context.Context) {
	innerSpan, innerCtx := opentracing.StartSpanFromContextWithTracer(ctx, instrumentation.Tracer(), "Exec: "+filepath.Base(command.Args[0]))
	innerSpan.SetTag("Args", command.Args)
	innerSpan.SetTag("Path", command.Path)
	innerSpan.SetTag("Dir", command.Dir)
	InjectToCmd(innerCtx, command)
	return innerSpan, innerCtx
}
