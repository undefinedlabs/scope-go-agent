package logging

import (
	"fmt"
	"io"
	stdlog "log"
	"regexp"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"go.undefinedlabs.com/scopeagent/tags"
)

const (
	LOG_REGEX_TEMPLATE = `(?m)^%s(?:(?P<date>\d{4}\/\d{1,2}\/\d{1,2}) )?(?:(?P<time>\d{1,2}:\d{1,2}:\d{1,2}(?:.\d{1,6})?) )?(?:(?:(?P<file>[\w\-. \/\\:]+):(?P<line>\d+)): )?(.*)\n?$`
)

type (
	OTWriter struct {
		base            io.Writer
		logFlags        int
		logRecords      []opentracing.LogRecord
		logRecordsMutex sync.RWMutex
		regex           *regexp.Regexp
		timeLayout      string
	}
	logItem struct {
		time       time.Time
		file       string
		lineNumber string
		message    string
	}
)

var (
	otWriters       []*OTWriter
	oldLoggerWriter io.Writer
)

// Patch the standard logger
func PatchStandardLogger() {
	oldLoggerWriter = getStdLoggerWriter()
	loggerWriter := newInstrumentedWriter(oldLoggerWriter, stdlog.Prefix(), stdlog.Flags())
	stdlog.SetOutput(loggerWriter)
	otWriters = append(otWriters, loggerWriter)
	loggerWriter.StartRecord()
}

// Patch a logger
func PatchLogger(logger *stdlog.Logger) {
	nWriter := newInstrumentedWriter(logger.Writer(), logger.Prefix(), logger.Flags())
	logger.SetOutput(nWriter)
	otWriters = append(otWriters, nWriter)
	nWriter.StartRecord()
}

//
// We are doing like this because there is no way to call span.LogFields with a custom timestamp on each event.
// The only way is to create an opentracing.LogRecord array and call later:
//  span.FinishWithOptions(opentracing.FinishOptions{
//		LogRecords: logRecords,
//	}
//

// Start record in all registered writers (used by the StartTest in order to generate new records for the span)
func StartRecord() {
	for _, writer := range otWriters {
		writer.StartRecord()
	}
}

// Stop record all registered writers (used by End in order to retrieve the records from the log and insert them in the span)
func StopRecord() []opentracing.LogRecord {
	var records []opentracing.LogRecord
	for _, writer := range otWriters {
		records = append(records, writer.StopRecord()...)
	}
	return records
}

// Create a new instrumented writer for loggers
func newInstrumentedWriter(base io.Writer, prefix string, flag int) *OTWriter {
	if baseWriter, ok := base.(*OTWriter); ok {
		// Avoid decorating twice
		return baseWriter
	}
	writer := &OTWriter{
		base:     base,
		logFlags: flag,
		regex:    regexp.MustCompile(fmt.Sprintf(LOG_REGEX_TEMPLATE, prefix)),
	}
	if flag == stdlog.LstdFlags|stdlog.Lmicroseconds {
		writer.timeLayout = "2006/01/02T15:04:05.000000"
	}
	return writer
}

// Write data to the channel and the base writer
func (w *OTWriter) Write(p []byte) (n int, err error) {
	w.logRecordsMutex.RLock()
	defer w.logRecordsMutex.RUnlock()
	if w.logRecords != nil {
		w.process(p)
	}
	return w.base.Write(p)
}

// Start recording opentracing.LogRecord from logger
func (w *OTWriter) StartRecord() {
	w.logRecordsMutex.Lock()
	defer w.logRecordsMutex.Unlock()
	w.logRecords = make([]opentracing.LogRecord, 0)
}

// Stop recording opentracing.LogRecord and return all recorded items
func (w *OTWriter) StopRecord() []opentracing.LogRecord {
	w.logRecordsMutex.Lock()
	defer w.logRecordsMutex.Unlock()
	defer func() {
		w.logRecords = nil
	}()
	return w.logRecords
}

// Process bytes and create new log items struct to store
func (w *OTWriter) process(p []byte) {
	if w.logRecords == nil || len(p) == 0 {
		// Nothing to process
		return
	}
	logBuffer := string(p)
	matches := w.regex.FindAllStringSubmatch(logBuffer, -1)
	if matches == nil || len(matches) == 0 {
		// If there are no matches we return without cleaning the buffer
		return
	}
	var item *logItem
	for _, match := range matches {
		// In case a new log line we store the previous one and create a new log item
		if match[1] != "" || match[2] != "" || match[3] != "" || match[4] != "" {
			w.storeLogRecord(item)
			now := time.Now()
			if w.timeLayout != "" {
				pTime, err := time.Parse(w.timeLayout, fmt.Sprintf("%sT%s", match[1], match[2]))
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
		if item.message == "" {
			item.message = match[5]
		} else {
			// Multiline log item support
			item.message = item.message + "\n" + match[5]
		}
	}
	w.storeLogRecord(item)
}

// Stores a new log record from the logItem
func (w *OTWriter) storeLogRecord(item *logItem) {
	if item == nil {
		return
	}
	fields := []log.Field{
		log.String(tags.EventType, tags.LogEvent),
		log.String(tags.LogEventLevel, tags.LogLevel_VERBOSE),
		log.String("log.logger", "std.Logger"),
		log.String(tags.EventMessage, item.message),
	}
	if item.file != "" && item.lineNumber != "" {
		fields = append(fields, log.String(tags.EventSource, fmt.Sprintf("%s:%s", item.file, item.lineNumber)))
	}
	w.logRecords = append(w.logRecords, opentracing.LogRecord{
		Timestamp: item.time,
		Fields:    fields,
	})
}
