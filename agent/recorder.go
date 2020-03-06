package agent

import (
	"bytes"
	"compress/gzip"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vmihailenco/msgpack"

	"go.undefinedlabs.com/scopeagent/tracer"
)

const retryBackoff = 1 * time.Second
const numOfRetries = 3

type (
	ScopeSpanRecorder interface {
		RecordSpan(span tracer.RawSpan)
		Stop()
		Stats() RecorderStats
	}
	RecorderStats interface {
		Write()
		HasTests() bool
		HasTestsNotSent() bool
		HasTestRejected() bool
		HasTestSent() bool
	}
	recorderStats struct {
		logger    *log.Logger
		statsOnce sync.Once

		totalSpans        int64
		sendSpansCalls    int64
		sendSpansOk       int64
		sendSpansKo       int64
		sendSpansRetries  int64
		spansSent         int64
		spansNotSent      int64
		spansRejected     int64
		totalTestSpans    int64
		testSpansSent     int64
		testSpansNotSent  int64
		testSpansRejected int64
	}

	PayloadSpan  map[string]interface{}
	PayloadEvent map[string]interface{}
)

func NewScopeSpanRecorder(agent *Agent) ScopeSpanRecorder {
	if val, ok := os.LookupEnv("SCOPE_CLI_UNIX_SOCKET"); ok && val != ""{
		return newWrapperSpanRecorder(agent, true)
	}
	if val, ok := os.LookupEnv("SCOPE_CLI_TCP"); ok && val != "" {
		return newWrapperSpanRecorder(agent, false)
	}
	return newInProcSpanRecorder(agent)
}

// Encodes `payload` using msgpack and compress it with gzip
func encodePayload(payload map[string]interface{}) (*bytes.Buffer, error) {
	binaryPayload, err := msgpack.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err = zw.Write(binaryPayload)
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	return &buf, nil
}

func isTestSpan(span tracer.RawSpan) bool {
	return span.Tags["span.kind"] == "test"
}

func (s *recorderStats) Write() {
	s.statsOnce.Do(func() {
		s.logger.Printf("** Recorder statistics **\n")
		s.logger.Printf("  Total spans: %d\n", atomic.LoadInt64(&s.totalSpans))
		s.logger.Printf("     Spans sent: %d\n", atomic.LoadInt64(&s.spansSent))
		s.logger.Printf("     Spans not sent: %d\n", atomic.LoadInt64(&s.spansNotSent))
		s.logger.Printf("     Spans rejected: %d\n", atomic.LoadInt64(&s.spansRejected))
		s.logger.Printf("  Total test spans: %d\n", atomic.LoadInt64(&s.totalTestSpans))
		s.logger.Printf("     Test spans sent: %d\n", atomic.LoadInt64(&s.testSpansSent))
		s.logger.Printf("     Test spans not sent: %d\n", atomic.LoadInt64(&s.testSpansNotSent))
		s.logger.Printf("     Test spans rejected: %d\n", atomic.LoadInt64(&s.testSpansRejected))
		s.logger.Printf("  SendSpans calls: %d\n", atomic.LoadInt64(&s.sendSpansCalls))
		s.logger.Printf("     SendSpans OK: %d\n", atomic.LoadInt64(&s.sendSpansOk))
		s.logger.Printf("     SendSpans KO: %d\n", atomic.LoadInt64(&s.sendSpansKo))
		s.logger.Printf("     SendSpans retries: %d\n", atomic.LoadInt64(&s.sendSpansRetries))
	})
}
func (s *recorderStats) HasTests() bool {
	return atomic.LoadInt64(&s.totalTestSpans) > 0
}
func (s *recorderStats) HasTestsNotSent() bool {
	return atomic.LoadInt64(&s.testSpansNotSent) > 0
}
func (s *recorderStats) HasTestRejected() bool {
	return atomic.LoadInt64(&s.testSpansRejected) > 0
}
func (s *recorderStats) HasTestSent() bool {
	return atomic.LoadInt64(&s.testSpansSent) > 0
}
