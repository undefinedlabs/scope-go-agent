package errors

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"go.undefinedlabs.com/scopeagent/tracer"
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

var MarkSpanAsError = errors.New("")

// Write exception event in span using the recover data from panic
func LogError(span opentracing.Span, recoverData interface{}, skipFrames int) {
	var exceptionFields = getExceptionLogFields(recoverData, skipFrames+1)
	span.LogFields(exceptionFields...)
	span.SetTag("error", true)
}

func LogErrorInRawSpan(rawSpan *tracer.RawSpan, err **errors.Error) {
	if rawSpan.Tags == nil {
		rawSpan.Tags = opentracing.Tags{}
	}
	if *err == MarkSpanAsError {
		rawSpan.Tags["error"] = true
	} else {
		var exceptionFields = getExceptionLogFields(*err, 1)
		if rawSpan.Logs == nil {
			rawSpan.Logs = []opentracing.LogRecord{}
		}
		rawSpan.Logs = append(rawSpan.Logs, opentracing.LogRecord{
			Timestamp: time.Now(),
			Fields:    exceptionFields,
		})
		rawSpan.Tags["error"] = true
		*err = MarkSpanAsError
	}
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

// Get the current error with the fixed stacktrace
func GetCurrentError(recoverData interface{}) *errors.Error {
	return errors.Wrap(recoverData, 1)
}

func getExceptionLogFields(recoverData interface{}, skipFrames int) []log.Field {
	if recoverData != nil {
		err := errors.Wrap(recoverData, 2+skipFrames)
		errMessage := err.Error()
		errStack := err.StackFrames() //filterStackFrames(err.StackFrames())
		exceptionData := getExceptionFrameData(errMessage, errStack)
		source := ""

		if errStack != nil && len(errStack) > 0 {
			for _, currentFrame := range errStack {
				if currentFrame.Package != "runtime" && currentFrame.File != "" {
					source = fmt.Sprintf("%s:%d", currentFrame.File, currentFrame.LineNumber)
					break
				}
			}
		}

		fields := make([]log.Field, 5)
		fields[0] = log.String(EventType, "error")
		fields[1] = log.String(EventSource, source)
		fields[2] = log.String(EventMessage, errMessage)
		fields[3] = log.String(EventStack, getStringStack(err, errStack))
		fields[4] = log.Object(EventException, exceptionData)
		return fields
	}
	return nil
}

func getStringStack(err *errors.Error, errStack []errors.StackFrame) string {
	var frames []string
	for _, frame := range errStack {
		frames = append(frames, frame.String())
	}
	return fmt.Sprintf("[%s]: %s\n\n%s", err.TypeName(), err.Error(), strings.Join(frames, ""))
}

// Filter stack frames from the go-agent
func filterStackFrames(errStack []errors.StackFrame) []errors.StackFrame {
	var stack []errors.StackFrame
	for _, frame := range errStack {
		if strings.Contains(frame.Package, "undefinedlabs/go-agent") {
			continue
		}
		stack = append(stack, frame)
	}
	return stack
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
	exStack := map[string]interface{}{
		"frames": exFrames,
	}
	return map[string]interface{}{
		"message":    errMessage,
		"stacktrace": exStack,
	}
}
