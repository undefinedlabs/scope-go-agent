package logrus

import (
	"fmt"
	"path/filepath"

	"github.com/opentracing/opentracing-go"
	otLog "github.com/opentracing/opentracing-go/log"

	log "github.com/sirupsen/logrus"

	"go.undefinedlabs.com/scopeagent/tags"
	"go.undefinedlabs.com/scopeagent/tracer"
)

type (
	ScopeHook struct {
		LogLevels []log.Level
	}
)

var scopeHook = &ScopeHook{}

// Adds an scope hook in the logger if is not already added.
func AddScopeHook(logger *log.Logger) {
	// We check first if the logger already contains a ScopeHook instance
	for _, hooks := range logger.Hooks {
		for _, hook := range hooks {
			if _, ok := hook.(*ScopeHook); ok {
				return
			}
		}
	}
	logger.AddHook(scopeHook)
}

// Fire will be called when some logging function is called with current hook
// It will format log entry to string and write it to appropriate writer
func (hook *ScopeHook) Fire(entry *log.Entry) error {
	if entry.Context == nil {
		return nil
	}

	// If context is found, we try to find the a span from the context and write the logs
	if span := opentracing.SpanFromContext(entry.Context); span != nil {

		logLevel := tags.LogLevel_VERBOSE
		if entry.Level == log.PanicLevel || entry.Level == log.FatalLevel || entry.Level == log.ErrorLevel {
			logLevel = tags.LogLevel_ERROR
		} else if entry.Level == log.WarnLevel {
			logLevel = tags.LogLevel_WARNING
		} else if entry.Level == log.InfoLevel {
			logLevel = tags.LogLevel_INFO
		} else if entry.Level == log.DebugLevel {
			logLevel = tags.LogLevel_DEBUG
		} else if entry.Level == log.TraceLevel {
			logLevel = tags.LogLevel_VERBOSE
		}

		fields := []otLog.Field{
			otLog.String(tags.EventType, tags.LogEvent),
			otLog.String(tags.LogEventLevel, logLevel),
			otLog.String("log.logger", "logrus"),
			otLog.String("log.level", entry.Level.String()),
			otLog.String(tags.EventMessage, entry.Message),
		}

		if entry.Caller != nil && entry.Caller.File != "" && entry.Caller.Line != 0 {
			fields = append(fields, otLog.String(tags.EventSource, fmt.Sprintf("%s:%d", filepath.Clean(entry.Caller.File), entry.Caller.Line)))
		}

		if entry.Data != nil {
			for k, v := range entry.Data {
				fields = append(fields, otLog.Object(k, v))
			}
		}

		if ownSpan, ok := span.(tracer.Span); ok {
			ownSpan.LogFieldsWithTimestamp(entry.Time, fields...)
		} else {
			span.LogFields(fields...)
		}
	}
	return nil
}

// Levels define on which log levels this hook would trigger
func (hook *ScopeHook) Levels() []log.Level {
	if hook.LogLevels == nil {
		hook.LogLevels = []log.Level{
			log.PanicLevel,
			log.FatalLevel,
			log.ErrorLevel,
			log.WarnLevel,
			log.InfoLevel,
			log.DebugLevel,
			log.TraceLevel,
		}
	}
	return hook.LogLevels
}
