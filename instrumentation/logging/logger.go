package logging

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"go.undefinedlabs.com/scopeagent/tags"
)

const (
	logRegexTemplate = `(?m)^%s(?:(?P<date>\d{4}\/\d{1,2}\/\d{1,2}) )?(?:(?P<time>\d{1,2}:\d{1,2}:\d{1,2}(?:.\d{1,6})?) )?(?:(?:(?P<file>[\w\-. \/\\:]+):(?P<line>\d+)): )?(.*)\n?$`
	timeLayout       = "2006/01/02T15:04:05.000000"
)

type (
	otWriter struct {
		logRecordsMutex sync.RWMutex
		logRecords      []opentracing.LogRecord
		regex           *regexp.Regexp
		logger          *stdlog.Logger
	}
	logItem struct {
		time       time.Time
		file       string
		lineNumber string
		message    string
	}
	loggerPatchInfo struct {
		current  io.Writer
		previous io.Writer
	}
)

var (
	patchedLoggersMutex sync.Mutex
	patchedLoggers      = map[io.Writer]loggerPatchInfo{}
	stdLoggerWriter     io.Writer
	loggerContextMutex  sync.RWMutex
	loggerContext       = map[*stdlog.Logger]context.Context{}
)

// Patch the standard logger
func PatchStandardLogger() {
	stdLoggerWriter := getStdLoggerWriter()
	otWriter := newInstrumentedWriter(nil, stdlog.Prefix(), stdlog.Flags())
	stdlog.SetOutput(io.MultiWriter(stdLoggerWriter, otWriter))
	recorders = append(recorders, otWriter)
}

// Unpatch the standard logger
func UnpatchStandardLogger() {
	stdlog.SetOutput(stdLoggerWriter)
}

// Patch a logger
func PatchLogger(logger *stdlog.Logger) {
	patchedLoggersMutex.Lock()
	defer patchedLoggersMutex.Unlock()
	currentWriter := getLoggerWriter(logger)
	if patchInfo, ok := patchedLoggers[currentWriter]; ok {
		currentWriter = patchInfo.previous
	}
	otWriter := newInstrumentedWriter(logger, logger.Prefix(), logger.Flags())
	mWriter := io.MultiWriter(currentWriter, otWriter)
	logger.SetOutput(mWriter)
	recorders = append(recorders, otWriter)
	patchedLoggers[mWriter] = loggerPatchInfo{
		current:  otWriter,
		previous: currentWriter,
	}
}

// Unpatch a logger
func UnpatchLogger(logger *stdlog.Logger) {
	patchedLoggersMutex.Lock()
	defer patchedLoggersMutex.Unlock()
	currentWriter := getLoggerWriter(logger)
	if logInfo, ok := patchedLoggers[currentWriter]; ok {
		logger.SetOutput(logInfo.previous)
		delete(patchedLoggers, currentWriter)
	}
}

// Create a new logger with a context
func WithContext(logger *stdlog.Logger, ctx context.Context) *stdlog.Logger {
	rLogger := stdlog.New(getLoggerWriter(logger), logger.Prefix(), logger.Flags())
	loggerContextMutex.Lock()
	defer loggerContextMutex.Unlock()
	loggerContext[rLogger] = ctx
	PatchLogger(rLogger)
	return rLogger
}

// Create a new instrumented writer for loggers
func newInstrumentedWriter(logger *stdlog.Logger, prefix string, flag int) *otWriter {
	return &otWriter{
		regex:  regexp.MustCompile(fmt.Sprintf(logRegexTemplate, prefix)),
		logger: logger,
	}
}

// Write data to the channel and the base writer
func (w *otWriter) Write(p []byte) (n int, err error) {
	w.process(p)
	return len(p), nil
}

// Start recording opentracing.LogRecord from logger
func (w *otWriter) Reset() {
	w.logRecordsMutex.Lock()
	defer w.logRecordsMutex.Unlock()
	w.logRecords = nil
}

// Stop recording opentracing.LogRecord and return all recorded items
func (w *otWriter) GetRecords() []opentracing.LogRecord {
	w.logRecordsMutex.RLock()
	defer w.Reset()
	defer w.logRecordsMutex.RUnlock()
	return w.logRecords
}

// Process bytes and create new log items struct to store
func (w *otWriter) process(p []byte) {
	if len(p) == 0 {
		// Nothing to process
		return
	}
	logBuffer := string(p)
	matches := w.regex.FindAllStringSubmatch(logBuffer, -1)
	if matches == nil || len(matches) == 0 {
		return
	}
	var item *logItem
	for _, match := range matches {
		// In case a new log line we store the previous one and create a new log item
		if match[1] != "" || match[2] != "" || match[3] != "" || match[4] != "" {
			if item != nil {
				w.storeLogRecord(item)
			}
			now := time.Now()
			if match[1] != "" && match[2] != "" {
				pTime, err := time.Parse(timeLayout, fmt.Sprintf("%sT%s", match[1], match[2]))
				if err == nil {
					now = pTime
				}
			}
			item = &logItem{
				time:       now,
				file:       match[3],
				lineNumber: match[4],
			}
		}
		if item != nil {
			if item.message == "" {
				item.message = match[5]
			} else {
				// Multiline log item support
				item.message = item.message + "\n" + match[5]
			}
		}
	}
	if item != nil {
		w.storeLogRecord(item)
	}
}

// Stores a new log record from the logItem
func (w *otWriter) storeLogRecord(item *logItem) {
	fields := []log.Field{
		log.String(tags.EventType, tags.LogEvent),
		log.String(tags.LogEventLevel, tags.LogLevel_VERBOSE),
		log.String("log.logger", "log.std"),
		log.String(tags.EventMessage, item.message),
	}
	if item.file != "" && item.lineNumber != "" {
		item.file = filepath.Clean(item.file)
		fields = append(fields, log.String(tags.EventSource, fmt.Sprintf("%s:%s", item.file, item.lineNumber)))
	}
	span := getContextSpanFromLogger(w.logger)
	if span != nil {
		span.LogFields(fields...)
		return
	}
	w.logRecordsMutex.Lock()
	defer w.logRecordsMutex.Unlock()
	w.logRecords = append(w.logRecords, opentracing.LogRecord{
		Timestamp: item.time,
		Fields:    fields,
	})
}

func getContextSpanFromLogger(logger *stdlog.Logger) opentracing.Span {
	loggerContextMutex.RLock()
	defer loggerContextMutex.RUnlock()
	if ctx, ok := loggerContext[logger]; ok {
		return opentracing.SpanFromContext(ctx)
	}
	return nil
}
