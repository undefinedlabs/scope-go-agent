package errors

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

const (
	EventType      = "event"
	EventSource    = "source"
	EventMessage   = "message"
	EventStack     = "stack"
	EventException = "exception"
)

type StackFrames struct {
	File       string
	LineNumber int
	Name       string
	Package    string
}

// Write exception event in span using the recover data from panic
func LogError(span opentracing.Span, recoverData interface{}, skipFrames int) {
	var exceptionFields = getExceptionLogFields(recoverData, skipFrames+1)
	span.LogFields(exceptionFields...)
}

// Gets the current stack frames array
func GetCurrentStackFrames(skip int) []StackFrames {
	skip = skip + 1
	err := errors.New(nil)
	errStack := err.StackFrames()
	nLength := len(errStack) - skip
	if nLength < 0 {
		return nil
	}
	stackFrames := make([]StackFrames, nLength)
	for idx, frame := range errStack {
		if idx >= skip {
			stackFrames[idx-skip] = StackFrames{
				File:       frame.File,
				LineNumber: frame.LineNumber,
				Name:       frame.Name,
				Package:    frame.Package,
			}
		}
	}
	return stackFrames
}

func getExceptionLogFields(recoverData interface{}, skipFrames int) []log.Field {
	if recoverData != nil {
		err := errors.Wrap(recoverData, 2+skipFrames)
		errMessage := err.Error()
		errStack := err.StackFrames()
		exceptionData := getExceptionFrameData(errMessage, errStack)
		source := ""

		exFrames := exceptionData["stacktrace"].(map[string]interface{})["frames"].([]map[string]interface{})
		if exFrames != nil && len(exFrames) > 0 {
			for i, _ := range exFrames {
				currentFrame := exFrames[i]
				if currentFrame["file"] != nil {
					source = fmt.Sprintf("%s:%d", currentFrame["file"], currentFrame["line"])
					break
				}
			}
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
		if frame.Package == "runtime" {
			exFrames = append(exFrames, map[string]interface{}{
				"name":   frame.Name,
				"module": frame.Package,
			})
		} else {
			exFrames = append(exFrames, map[string]interface{}{
				"name":   frame.Name,
				"module": frame.Package,
				"file":   frame.File,
				"line":   frame.LineNumber,
			})
		}
	}
	exStack := map[string]interface{}{
		"frames": exFrames,
	}
	return map[string]interface{}{
		"message":    errMessage,
		"stacktrace": exStack,
	}
}
