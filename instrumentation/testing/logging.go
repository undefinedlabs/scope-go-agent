package testing

import (
	"bufio"
	"fmt"
	stdlog "log"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/opentracing/opentracing-go/log"

	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/tags"
)

type stdIO struct {
	oldIO     *os.File
	readPipe  *os.File
	writePipe *os.File
	sync      *sync.WaitGroup
}

var LOG_REGEX_TEMPLATE = `^%s(?:(?P<date>\d{4}\/\d{1,2}\/\d{1,2}) )?(?:(?P<time>\d{1,2}:\d{1,2}:\d{1,2}(?:.\d{1,6})?) )?(?:(?:(?P<file>[\w\-. \/\\:]+):(?P<line>\d+)): )?(.*)\n?$`

func (test *Test) startCapturingLogs() {
	// Replaces stdout and stderr
	loggerStdIO := newStdIO(&os.Stderr, false)
	if loggerStdIO != nil && loggerStdIO.writePipe != nil {
		stdlog.SetOutput(loggerStdIO.writePipe)
	}
	test.loggerStdIO = loggerStdIO
	test.stdOut = newStdIO(&os.Stdout, true)
	test.stdErr = newStdIO(&os.Stderr, true)

	// Starts stdIO pipe handlers
	if test.stdOut != nil {
		go stdIOHandler(test, test.stdOut, false)
	}
	if test.stdErr != nil {
		go stdIOHandler(test, test.stdErr, true)
	}
	if test.loggerStdIO != nil {
		go loggerStdIOHandler(test, test.loggerStdIO)
	}
}

func (test *Test) stopCapturingLogs() {
	test.stdOut.restore(&os.Stdout, true)
	test.stdErr.restore(&os.Stderr, true)
	test.loggerStdIO.restore(&os.Stderr, false)
	stdlog.SetOutput(os.Stderr)
}

// Handles the StdIO pipe for stdout and stderr
func stdIOHandler(test *Test, stdio *stdIO, isError bool) {
	stdio.sync.Add(1)
	defer stdio.sync.Done()
	reader := bufio.NewReader(stdio.readPipe)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if len(strings.TrimSpace(line)) > 0 {
			if isError {
				test.span.LogFields(
					log.String(tags.EventType, tags.LogEvent),
					log.String(tags.EventMessage, line),
					log.String(tags.LogEventLevel, tags.LogLevel_ERROR),
				)
			} else {
				test.span.LogFields(
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
func loggerStdIOHandler(test *Test, stdio *stdIO) {
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
		matches := re.FindStringSubmatch(line)
		file := matches[3]
		lineNumber := matches[4]
		message := matches[5]
		fields := append(commonFields, log.String(tags.EventMessage, message))
		if file != "" && line != "" {
			fields = append(fields, log.String(tags.EventSource, fmt.Sprintf("%s:%s", file, lineNumber)))
		}
		test.span.LogFields(fields...)

		if stdio.oldIO != nil {
			_, _ = stdio.oldIO.WriteString(line)
		}
	}
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
