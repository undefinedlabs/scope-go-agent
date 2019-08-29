package scopeagent

import (
	"bufio"
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
)

type Test struct {
	ctx  	context.Context
	span 	opentracing.Span
	t    	*testing.T
	stdOut	*StdIO
	stdErr	*StdIO
}
type StdIO struct {
	oldIO		*os.File
	readPipe	*os.File
	writePipe	*os.File
	sync		*sync.WaitGroup
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

	stdOut := newStdIO(&os.Stdout)
	stdErr := newStdIO(&os.Stderr)

	test := &Test{
		ctx:  ctx,
		span: span,
		t:    t,
		stdOut: stdOut,
		stdErr: stdErr,
	}

	if test.stdOut != nil {
		go stdIOHandler(test, test.stdOut, false)
	}
	if test.stdErr != nil {
		go stdIOHandler(test, test.stdErr, true)
	}

	return test
}

func (test *Test) End() {
	defer test.span.Finish()

	if r := recover(); r != nil {
		test.span.SetTag("test.status", "ERROR")
	} else if test.t.Failed() {
		test.span.SetTag("test.status", "FAIL")
	} else if test.t.Skipped() {
		test.span.SetTag("test.status", "SKIP")
	} else {
		test.span.SetTag("test.status", "PASS")
	}

	test.stdOut.restore(&os.Stdout)
	test.stdErr.restore(&os.Stderr)
	test.span.Finish()
}

func (test *Test) Context() context.Context {
	return test.ctx
}


func stdIOHandler (test *Test, stdio *StdIO, isError bool) {
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
			_, _ = stdio.oldIO.WriteString("** Adding 1 Error LOG on " + test.t.Name() + "\n")
		} else {
			test.span.LogFields(
				log.String(EventType, LogEvent),
				log.String(EventMessage, line),
				log.String(LogEventLevel, LogLevel_VERBOSE),
			)
			_, _ = stdio.oldIO.WriteString("** Adding 1 Verbose LOG on " + test.t.Name() + "\n")
		}
		_, _ = stdio.oldIO.WriteString(line)
	}
}

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

func (stdIO *StdIO) restore(file **os.File) {
	_ = stdIO.writePipe.Close()
	_ = stdIO.readPipe.Close()
	stdIO.sync.Wait()
	*file = stdIO.oldIO
}