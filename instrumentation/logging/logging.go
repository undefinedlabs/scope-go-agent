package logging

import (
	"bufio"
	"fmt"
	stdlog "log"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/tags"
)

const (
	LOG_REGEX_TEMPLATE = `^%s(?:(?P<date>\d{4}\/\d{1,2}\/\d{1,2}) )?(?:(?P<time>\d{1,2}:\d{1,2}:\d{1,2}(?:.\d{1,6})?) )?(?:(?:(?P<file>[\w\-. /\\:]+):(?P<line>\d+)): )?(.*)\n?$`
)

type stdIO struct {
	oldIO     *os.File
	readPipe  *os.File
	writePipe *os.File
	sync      *sync.WaitGroup
}

var (
	loggerStdIO *stdIO
	stdOut      *stdIO
	stdErr      *stdIO
	currentSpan opentracing.Span
)

// Initialize logging instrumentation
func Init() {
	// Replaces stdout and stderr
	lgStdIO := newStdIO(&os.Stderr, false)
	if lgStdIO != nil && lgStdIO.writePipe != nil {
		stdlog.SetOutput(lgStdIO.writePipe)
	}
	loggerStdIO = lgStdIO
	stdOut = newStdIO(&os.Stdout, true)
	stdErr = newStdIO(&os.Stderr, true)

	// Starts stdIO pipe handlers
	if stdOut != nil {
		go stdIOHandler(stdOut, false)
	}
	if stdErr != nil {
		go stdIOHandler(stdErr, true)
	}
	if loggerStdIO != nil {
		go loggerStdIOHandler(loggerStdIO)
	}
}

// Finalize logging instrumentation
func Finalize() {
	stdOut.restore(&os.Stdout, true)
	stdErr.restore(&os.Stderr, true)
	loggerStdIO.restore(&os.Stderr, false)
	stdlog.SetOutput(os.Stderr)
}

// Sets the current span for logger
func SetCurrentSpan(span opentracing.Span) {
	currentSpan = span
}

// Creates and replaces a file instance with a pipe
func newStdIO(file **os.File, replace bool) *stdIO {
	rPipe, wPipe, err := os.Pipe()
	if err == nil {
		stdIO := &stdIO{
			readPipe:  rPipe,
			writePipe: wPipe,
			sync:      new(sync.WaitGroup),
		}
		if file != nil {
			stdIO.oldIO = *file
			if replace {
				*file = wPipe
			}
		}
		return stdIO
	} else {
		instrumentation.Logger().Println(err)
	}
	return nil
}

// Restores the old file instance
func (stdIO *stdIO) restore(file **os.File, replace bool) {
	if file != nil {
		_ = (*file).Sync()
	}
	if stdIO == nil {
		return
	}
	if stdIO.readPipe != nil {
		_ = stdIO.readPipe.Sync()
	}
	if stdIO.writePipe != nil {
		_ = stdIO.writePipe.Sync()
		_ = stdIO.writePipe.Close()
	}
	if stdIO.readPipe != nil {
		_ = stdIO.readPipe.Close()
	}
	stdIO.sync.Wait()
	if replace && file != nil {
		*file = stdIO.oldIO
	}
}

// Handles the StdIO pipe for stdout and stderr
func stdIOHandler(stdio *stdIO, isError bool) {
	stdio.sync.Add(1)
	defer stdio.sync.Done()
	reader := bufio.NewReader(stdio.readPipe)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if currentSpan != nil && len(strings.TrimSpace(line)) > 0 {
			if isError {
				currentSpan.LogFields(
					log.String(tags.EventType, tags.LogEvent),
					log.String(tags.EventMessage, line),
					log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
				)
			} else {
				currentSpan.LogFields(
					log.String(tags.EventType, tags.LogEvent),
					log.String(tags.EventMessage, line),
					log.String(tags.LogEventLevel, tags.LogLevel_VERBOSE),
				)
			}
		}
		_, _ = stdio.oldIO.WriteString(line)
	}
}

// Handles the StdIO for a logger
func loggerStdIOHandler(stdio *stdIO) {
	stdio.sync.Add(1)
	defer stdio.sync.Done()
	commonFields := []log.Field{
		log.String(tags.EventType, tags.LogEvent),
		log.String(tags.LogEventLevel, tags.LogLevel_VERBOSE),
		log.String("log.logger", "std.Logger"),
	}
	reader := bufio.NewReader(stdio.readPipe)
	re := regexp.MustCompile(fmt.Sprintf(LOG_REGEX_TEMPLATE, stdlog.Prefix()))
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		if currentSpan != nil {
			matches := re.FindStringSubmatch(line)
			file := matches[3]
			lineNumber := matches[4]
			message := matches[5]
			fields := append(commonFields, log.String(tags.EventMessage, message))
			if file != "" && line != "" {
				fields = append(fields, log.String(tags.EventSource, fmt.Sprintf("%s:%s", file, lineNumber)))
			}
			currentSpan.LogFields(fields...)
		}

		if stdio.oldIO != nil {
			_, _ = stdio.oldIO.WriteString(line)
		}
	}
}
