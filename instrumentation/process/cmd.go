package process

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/opentracing/opentracing-go"

	"go.undefinedlabs.com/scopeagent/instrumentation"
)

// Injects the span context to the command environment variables
func InjectToCmd(ctx context.Context, command *exec.Cmd) *exec.Cmd {
	if command.Env == nil {
		command.Env = []string{}
	}
	err := InjectFromContext(ctx, &command.Env)
	if err != nil {
		instrumentation.Logger().Println(err)
	}
	return command
}

// Injects a new span context to the command environment variables
func InjectToCmdWithSpan(ctx context.Context, command *exec.Cmd) (opentracing.Span, context.Context) {

	var operationNameBuilder = new(strings.Builder)
	operationNameBuilder.WriteString("Exec: ")
	operationNameBuilder.WriteString(filepath.Base(command.Args[0]))
	operationNameBuilder.WriteByte(' ')
	for _, item := range command.Args[1:] {
		if strings.ContainsAny(item, " ") {
			operationNameBuilder.WriteByte('"')
			operationNameBuilder.WriteString(item)
			operationNameBuilder.WriteByte('"')
		} else {
			operationNameBuilder.WriteString(item)
		}
		operationNameBuilder.WriteByte(' ')
	}

	innerSpan, innerCtx := opentracing.StartSpanFromContextWithTracer(ctx, instrumentation.Tracer(), operationNameBuilder.String())
	innerSpan.SetTag("Args", command.Args)
	innerSpan.SetTag("Path", command.Path)
	innerSpan.SetTag("Dir", command.Dir)
	InjectToCmd(innerCtx, command)
	return innerSpan, innerCtx
}
