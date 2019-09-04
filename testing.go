package scopeagent

import (
	"bufio"
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/undefinedlabs/go-agent/errors"
	log2 "log"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
)

type Test struct {
	ctx    context.Context
	span   opentracing.Span
	t      *testing.T
	stdOut *StdIO
	stdErr *StdIO
}
type StdIO struct {
	oldIO     *os.File
	readPipe  *os.File
	writePipe *os.File
	sync      *sync.WaitGroup
}

func InstrumentTest(t *testing.T, f func(ctx context.Context, t *testing.T)) {
	test := StartTest(t)
	defer test.End()
	f(test.Context(), t)
}

func StartTest(t *testing.T) *Test {
	pc, _, _, _ := runtime.Caller(1)
	parts := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	pl := len(parts)
	packageName := ""
	funcName := parts[pl-1]

	if parts[pl-2][0] == '(' {
		funcName = parts[pl-2] + "." + funcName
		packageName = strings.Join(parts[0:pl-2], ".")
	} else {
		packageName = strings.Join(parts[0:pl-1], ".")
	}

	span, ctx := opentracing.StartSpanFromContext(context.Background(), t.Name(), opentracing.Tags{
		"span.kind":  "test",
		"test.name":  funcName,
		"test.suite": packageName,
	})
	span.SetBaggageItem("trace.kind", "test")

	// Replaces stdout and stderr
	stdOut := newStdIO(&os.Stdout)
	stdErr := newStdIO(&os.Stderr)
	log2.SetOutput(stdOut.writePipe)

	test := &Test{
		ctx:    ctx,
		span:   span,
		t:      t,
		stdOut: stdOut,
		stdErr: stdErr,
	}

	// Starts stdIO pipe handlers
	if test.stdOut != nil {
		go stdIOHandler(test, test.stdOut, false)
	}
	if test.stdErr != nil {
		go stdIOHandler(test, test.stdErr, true)
	}

	return test
}

func (test *Test) End() {
	if r := recover(); r != nil {
		test.span.SetTag("test.status", "ERROR")
		test.stdOut.restore(&os.Stdout)
		test.stdErr.restore(&os.Stderr)
		test.span.SetTag("error", true)
		errors.LogError(test.span, r, 1)
		test.span.Finish()
		_ = GlobalAgent.Flush()
		panic(r)
	}
	if test.t.Failed() {
		test.span.SetTag("test.status", "FAIL")
		test.span.SetTag("error", true)
		test.span.LogFields(
			log.String(EventType, EventTestFailure),
			log.String(EventMessage, "Test has failed"),
		)
	} else if test.t.Skipped() {
		test.span.SetTag("test.status", "SKIP")
	} else {
		test.span.SetTag("test.status", "PASS")
	}

	test.stdOut.restore(&os.Stdout)
	test.stdErr.restore(&os.Stderr)
	log2.SetOutput(os.Stderr)
	test.span.Finish()
}

func (test *Test) Context() context.Context {
	return test.ctx
}

// Handles the StdIO pipe for stdout and stderr
func stdIOHandler(test *Test, stdio *StdIO, isError bool) {
	stdio.sync.Add(1)
	defer stdio.sync.Done()
	reader := bufio.NewReader(stdio.readPipe)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if isError {
			test.span.LogFields(
				log.String(EventType, LogEvent),
				log.String(EventMessage, line),
				log.String(LogEventLevel, LogLevel_ERROR),
			)
		} else {
			test.span.LogFields(
				log.String(EventType, LogEvent),
				log.String(EventMessage, line),
				log.String(LogEventLevel, LogLevel_VERBOSE),
			)
		}
		_, _ = stdio.oldIO.WriteString(line)
	}
}

// Creates and replaces a file instance with a pipe
func newStdIO(file **os.File) *StdIO {
	rPipe, wPipe, err := os.Pipe()
	if err == nil {
		stdIO := &StdIO{
			oldIO:     *file,
			readPipe:  rPipe,
			writePipe: wPipe,
			sync:      new(sync.WaitGroup),
		}
		*file = wPipe
		return stdIO
	}
	return nil
}

// Restores the old file instance
func (stdIO *StdIO) restore(file **os.File) {
	_ = (*file).Sync()
	_ = stdIO.readPipe.Sync()
	_ = stdIO.writePipe.Close()
	_ = stdIO.readPipe.Close()
	stdIO.sync.Wait()
	*file = stdIO.oldIO
}
