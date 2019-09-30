package testing

import (
	"bufio"
	"fmt"
	"github.com/opentracing/opentracing-go/log"
	"go.undefinedlabs.com/scopeagent/tags"
	stdlog "log"
	"os"
	"strings"
	"sync"
)

type stdIO struct {
	oldIO     *os.File
	readPipe  *os.File
	writePipe *os.File
	sync      *sync.WaitGroup
}

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
	reader := bufio.NewReader(stdio.readPipe)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		nLine := line
		flags := stdlog.Flags()
		sliceCount := 0
		if flags&(stdlog.Ldate|stdlog.Ltime|stdlog.Lmicroseconds) != 0 {
			if flags&stdlog.Ldate != 0 {
				sliceCount = sliceCount + 11
			}
			if flags&(stdlog.Ltime|stdlog.Lmicroseconds) != 0 {
				sliceCount = sliceCount + 9
				if flags&stdlog.Lmicroseconds != 0 {
					sliceCount = sliceCount + 7
				}
			}
			nLine = nLine[sliceCount:]
		}
		test.span.LogFields(
			log.String(tags.EventType, tags.LogEvent),
			log.String(tags.EventMessage, nLine),
			log.String(tags.LogEventLevel, tags.LogLevel_VERBOSE),
			log.String("log.logger", "std.Logger"),
		)

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
		fmt.Println(err)
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
