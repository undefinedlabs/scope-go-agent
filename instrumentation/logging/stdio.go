package logging

import (
	"bufio"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/tags"
)

type instrumentedIO struct {
	orig            **os.File
	base            *os.File
	rPipe           *os.File
	wPipe           *os.File
	hSync           sync.WaitGroup
	logRecordsMutex sync.RWMutex
	logRecords      []opentracing.LogRecord
	isError         bool
}

var (
	patchedStdOut *instrumentedIO
	patchedStdErr *instrumentedIO
)

// Patch standard output
func PatchStdOut() {
	patchedStdOut = patchIO(&os.Stdout, false)
	recorders = append(recorders, patchedStdOut)
}

// Unpatch standard output
func UnpatchStdOut() {
	if patchedStdOut != nil {
		patchedStdOut.Restore()
		patchedStdOut = nil
	}
}

// Patch standard error
func PatchStdErr() {
	patchedStdErr = patchIO(&os.Stderr, true)
	recorders = append(recorders, patchedStdErr)
}

// Unpatch standard error
func UnpatchStdErr() {
	if patchedStdErr != nil {
		patchedStdErr.Restore()
		patchedStdErr = nil
	}
}

// Patch IO File
func patchIO(base **os.File, isError bool) *instrumentedIO {
	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		instrumentation.Logger().Println(err)
		return nil
	}
	instIO := &instrumentedIO{
		orig:    base,
		base:    *base,
		rPipe:   rPipe,
		wPipe:   wPipe,
		isError: isError,
	}
	*base = wPipe
	instIO.hSync.Add(1)
	go instIO.ioHandler()
	return instIO
}

// Start recording opentracing.LogRecord from logger
func (i *instrumentedIO) Reset() {
	i.logRecordsMutex.Lock()
	defer i.logRecordsMutex.Unlock()
	i.logRecords = nil
}

// Stop recording opentracing.LogRecord and return all recorded items
func (i *instrumentedIO) GetRecords() []opentracing.LogRecord {
	i.logRecordsMutex.Lock()
	defer i.logRecordsMutex.Unlock()
	defer func() {
		i.logRecords = nil
	}()
	_ = i.wPipe.Sync()
	_ = i.rPipe.Sync()
	return i.logRecords
}

// Close handler
func (i *instrumentedIO) Restore() {
	i.wPipe.Sync()
	i.rPipe.Sync()
	i.wPipe.Close()
	i.rPipe.Close()
	i.hSync.Wait()

	if i.orig != nil {
		*i.orig = i.base
	}
}

// Handles the StdIO pipe for stdout and stderr
func (i *instrumentedIO) ioHandler() {
	defer i.hSync.Done()
	reader := bufio.NewReader(i.rPipe)
	fields := []log.Field{
		log.String(tags.EventType, tags.LogEvent),
		log.String("log.logger", "std"),
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			// Error or EOF
			break
		}
		i.logRecordsMutex.RLock()
		if len(strings.TrimSpace(line)) > 0 {
			now := time.Now()
			if i.isError {
				fields = append(fields,
					log.String(tags.EventMessage, line),
					log.String(tags.LogEventLevel, tags.LogLevel_ERROR))
			} else {
				fields = append(fields,
					log.String(tags.EventMessage, line),
					log.String(tags.LogEventLevel, tags.LogLevel_VERBOSE))
			}
			i.logRecords = append(i.logRecords, opentracing.LogRecord{
				Timestamp: now,
				Fields:    fields,
			})
		}
		i.logRecordsMutex.RUnlock()
		_, _ = (*i.base).WriteString(line)
	}
}
