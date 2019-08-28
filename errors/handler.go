package errors

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)
const (
	EventType		= "event"
	EventSource		= "source"
	EventMessage	= "message"
	EventStack		= "stack"
	EventException	= "exception"
)

func LogError(span opentracing.Span, recoverData interface{}, skipFrames int) {
	var exceptionFields = getExceptionLogFields(recoverData, skipFrames + 1)
	span.LogFields(exceptionFields...)
}

func getExceptionLogFields(recoverData interface{}, skipFrames int) []log.Field {
	if recoverData != nil {
		err := errors.Wrap(recoverData, 2 + skipFrames)
		errMessage := err.Error()
		errStack := err.StackFrames()
		exceptionData := getExceptionFrameData(errMessage, errStack)
		source := ""

		exFrames := exceptionData["stacktrace"].(map[string]interface{})["frames"].([]map[string]interface{})
		if exFrames != nil && len(exFrames) > 0 {
			lastFrame := exFrames[0]
			source = fmt.Sprintf("%s:%d", lastFrame["file"], lastFrame["line"])
		}

		fields := make([]log.Field, 5)
		fields[0] = log.String(EventType, "error")
		fields[1] = log.String(EventSource, source)
		fields[2] = log.String(EventMessage, errMessage)
		fields[3] = log.String(EventStack, err.ErrorStack())
		fields[4] = log.Object(EventException, exceptionData)
		return fields
	}
	return nil
}

func getExceptionFrameData(errMessage string, errStack []errors.StackFrame) map[string]interface{} {
	var exFrames []map[string]interface{}
	for _, frame := range errStack {
		exFrames = append(exFrames, map[string]interface{}{
			"name":   frame.Name,
			"module": frame.Package,
			"file":   frame.File,
			"line":   frame.LineNumber,
		})
	}
	exStack := map[string]interface{} {
		"frames" : exFrames,
	}
	return map[string]interface{} {
		"message": errMessage,
		"stacktrace": exStack,
	}
}