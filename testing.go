package scopeagent

import (
	"bufio"
	"context"
	"fmt"
	log2 "log"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/undefinedlabs/go-agent/ast"
	"github.com/undefinedlabs/go-agent/errors"
	"github.com/undefinedlabs/go-agent/tracer"
)

type Test struct {
	ctx         context.Context
	span        opentracing.Span
	t           *testing.T
	stdOut      *StdIO
	stdErr      *StdIO
	loggerStdIO *StdIO
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
	pcName := runtime.FuncForPC(pc).Name()
	parts := strings.Split(pcName, ".")
	pl := len(parts)
	packageName := ""
	funcName := parts[pl-1]

	if parts[pl-2][0] == '(' {
		funcName = parts[pl-2] + "." + funcName
		packageName = strings.Join(parts[0:pl-2], ".")
	} else {
		packageName = strings.Join(parts[0:pl-1], ".")
	}

	if checkIfNewTestProcessNeeded(t, funcName) {
		return nil
	}

	sourceBounds := ast.GetFuncSource(pc)
	var testCode string
	if sourceBounds != nil {
		testCode = fmt.Sprintf("%s:%d:%d", sourceBounds.File, sourceBounds.Start.Line, sourceBounds.End.Line)
	}

	span, ctx := opentracing.StartSpanFromContext(context.Background(), t.Name(), opentracing.Tags{
		"span.kind":  "test",
		"test.name":  funcName,
		"test.suite": packageName,
		"test.code":  testCode,
	})

	// Check if we have to read values from out of process context
	if agentId, traceId, spanId, ok := getOutOfProcessContext(); ok {
		scopeSpan := span.(tracer.ScopeSpan)
		//fmt.Printf("Using AgentId: %s, TraceId: %x, SpanId. %x\n", agentId, traceId, spanId)
		GlobalAgent.metadata[AgentID] = agentId
		scopeSpan.SetTraceAndSpanId(traceId, spanId)
	}

	span.SetBaggageItem("trace.kind", "test")

	// Replaces stdout and stderr
	loggerStdIO := newStdIO(&os.Stderr, false)
	stdOut := newStdIO(&os.Stdout, true)
	stdErr := newStdIO(&os.Stderr, true)
	log2.SetOutput(loggerStdIO.writePipe)

	test := &Test{
		ctx:         ctx,
		span:        span,
		t:           t,
		stdOut:      stdOut,
		stdErr:      stdErr,
		loggerStdIO: loggerStdIO,
	}

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

	return test
}

func (test *Test) End() {
	defer checkIfFlushNeeded()
	if r := recover(); r != nil {
		test.span.SetTag("test.status", "ERROR")
		test.stdOut.restore(&os.Stdout, true)
		test.stdErr.restore(&os.Stderr, true)
		test.loggerStdIO.restore(&os.Stderr, false)
		log2.SetOutput(os.Stderr)
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

	test.stdOut.restore(&os.Stdout, true)
	test.stdErr.restore(&os.Stderr, true)
	test.loggerStdIO.restore(&os.Stderr, false)
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

// Handles the StdIO for a logger
func loggerStdIOHandler(test *Test, stdio *StdIO) {
	stdio.sync.Add(1)
	defer stdio.sync.Done()
	reader := bufio.NewReader(stdio.readPipe)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		nLine := line
		flags := log2.Flags()
		sliceCount := 0
		if flags&(log2.Ldate|log2.Ltime|log2.Lmicroseconds) != 0 {
			if flags&log2.Ldate != 0 {
				sliceCount = sliceCount + 11
			}
			if flags&(log2.Ltime|log2.Lmicroseconds) != 0 {
				sliceCount = sliceCount + 9
				if flags&log2.Lmicroseconds != 0 {
					sliceCount = sliceCount + 7
				}
			}
			nLine = nLine[sliceCount:]
		}
		test.span.LogFields(
			log.String(EventType, LogEvent),
			log.String(EventMessage, nLine),
			log.String(LogEventLevel, LogLevel_VERBOSE),
			log.String("log.logger", "std.Logger"),
		)

		if stdio.oldIO != nil {
			_, _ = stdio.oldIO.WriteString(line)
		}
	}
}

// Creates and replaces a file instance with a pipe
func newStdIO(file **os.File, replace bool) *StdIO {
	rPipe, wPipe, err := os.Pipe()
	if err == nil {
		stdIO := &StdIO{
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
	}
	return nil
}

// Restores the old file instance
func (stdIO *StdIO) restore(file **os.File, replace bool) {
	if file != nil {
		_ = (*file).Sync()
	}
	_ = stdIO.readPipe.Sync()
	_ = stdIO.writePipe.Close()
	_ = stdIO.readPipe.Close()
	stdIO.sync.Wait()
	if replace && file != nil {
		*file = stdIO.oldIO
	}
}
