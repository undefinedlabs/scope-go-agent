package agent

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack"
	"gopkg.in/tomb.v2"

	"go.undefinedlabs.com/scopeagent/tags"
	"go.undefinedlabs.com/scopeagent/tracer"
)

const retryBackoff = 1 * time.Second

type (
	SpanRecorder struct {
		sync.RWMutex
		t tomb.Tomb

		apiKey      string
		apiEndpoint string
		version     string
		userAgent   string
		debugMode   bool
		metadata    map[string]interface{}

		spansMutex sync.RWMutex
		spans      []tracer.RawSpan

		flushFrequency time.Duration
		url            string
		client         *http.Client

		logger *log.Logger
		stats  *RecorderStats
	}
	RecorderStats struct {
		totalSpans     int64
		sendSpansCalls int64
		sendSpansOk    int64
		sendSpansKo    int64
		spansSent      int64
		spansNotSent   int64
		spansRejected  int64
		testSpans      int64
	}
)

func NewSpanRecorder(agent *Agent) *SpanRecorder {
	r := new(SpanRecorder)
	r.apiEndpoint = agent.apiEndpoint
	r.apiKey = agent.apiKey
	r.version = agent.version
	r.userAgent = agent.userAgent
	r.debugMode = agent.debugMode
	r.metadata = agent.metadata
	r.logger = agent.logger
	r.flushFrequency = time.Minute
	r.url = agent.getUrl("api/agent/ingest")
	r.client = &http.Client{}
	r.stats = &RecorderStats{}
	r.t.Go(r.loop)
	return r
}

// Appends a span to the in-memory buffer for async processing
func (r *SpanRecorder) RecordSpan(span tracer.RawSpan) {
	r.Lock()
	defer r.Unlock()
	atomic.AddInt64(&r.stats.totalSpans, 1)
	if !r.t.Alive() {
		atomic.AddInt64(&r.stats.spansRejected, 1)
		r.logger.Printf("an span is received but the recorder is already disposed\n")
		return
	}
	r.addSpan(span)
	if span.Tags["span.kind"] == "test" {
		atomic.AddInt64(&r.stats.testSpans, 1)
	}
}

// Change flush frequency
func (r *SpanRecorder) ChangeFlushFrequency(frequency time.Duration) {
	r.Lock()
	defer r.Unlock()
	r.flushFrequency = frequency
}

func (r *SpanRecorder) getFlushFrequency() time.Duration {
	r.RLock()
	defer r.RUnlock()
	return r.flushFrequency
}

func (r *SpanRecorder) loop() error {
	ticker := time.NewTicker(1 * time.Second)
	cTime := time.Now()
	for {
		select {
		case <-ticker.C:
			if r.hasSpans() || time.Now().Sub(cTime) >= r.getFlushFrequency() {
				if r.debugMode {
					if r.hasSpans() {
						r.logger.Println("Ticker: Sending by buffer")
					} else {
						r.logger.Println("Ticker: Sending by time")
					}
				}
				cTime = time.Now()
				err, shouldExit := r.SendSpans()
				if err != nil {
					r.logger.Printf("error sending spans: %v\n", err)
				}
				if shouldExit {
					ticker.Stop()
					r.t.Kill(err)
					return err
				}
			}
		case <-r.t.Dying():
			err, _ := r.SendSpans()
			if err != nil {
				r.logger.Printf("error sending spans: %v\n", err)
			}
			ticker.Stop()
			return err
		}
	}
}

// Sends the spans in the buffer to Scope
func (r *SpanRecorder) SendSpans() (error, bool) {
	atomic.AddInt64(&r.stats.sendSpansCalls, 1)

	spans := r.getSpans()
	payload := r.getPayload(spans, r.metadata)

	buf, err := encodePayload(payload)
	if err != nil {
		atomic.AddInt64(&r.stats.sendSpansKo, 1)
		atomic.AddInt64(&r.stats.spansNotSent, int64(len(spans)))
		return err, false
	}

	payloadSent := false
	shouldExit := false
	var lastError error
	for i := 0; i <= 3; i++ {
		if r.debugMode {
			r.logger.Printf("sending payload [%d try]\n", i)
		}
		statusCode, err := r.callIngest(buf)
		if err != nil {
			if statusCode == 401 {
				shouldExit = true
				lastError = err
				break
			} else if statusCode < 500 {
				lastError = err
				break
			} else {
				lastError = err
				time.Sleep(retryBackoff)
			}
		} else {
			payloadSent = true
			break
		}
	}

	if payloadSent {
		atomic.AddInt64(&r.stats.sendSpansOk, 1)
		atomic.AddInt64(&r.stats.spansSent, int64(len(spans)))
	} else {
		atomic.AddInt64(&r.stats.sendSpansKo, 1)
		atomic.AddInt64(&r.stats.spansNotSent, int64(len(spans)))
	}

	return lastError, shouldExit
}

// Stop recorder
func (r *SpanRecorder) Stop() {
	r.Lock()
	defer r.Unlock()
	if r.debugMode {
		r.logger.Println("Scope recorder is stopping gracefully...")
	}
	r.t.Kill(nil)
	_ = r.t.Wait()
	if r.hasSpans() {
		err, _ := r.SendSpans()
		if err != nil {
			r.logger.Printf("error sending spans: %v\n", err)
		}
	}
	if r.debugMode {
		r.writeStats()
	}
}

// Write statistics
func (r *SpanRecorder) writeStats() {
	r.logger.Printf("** Recorder statistics **\n")
	r.logger.Printf("  Total spans: %d\n", r.stats.totalSpans)
	r.logger.Printf("  Test spans: %d\n", r.stats.testSpans)
	r.logger.Printf("  Spans sent: %d\n", r.stats.spansSent)
	r.logger.Printf("  Spans not sent: %d\n", r.stats.spansNotSent)
	r.logger.Printf("  Spans rejected: %d\n", r.stats.spansRejected)
	r.logger.Printf("  SendSpans calls: %d\n", r.stats.sendSpansCalls)
	r.logger.Printf("  SendSpans OK: %d\n", r.stats.sendSpansOk)
	r.logger.Printf("  SendSpans KO: %d\n", r.stats.sendSpansKo)
}

// Sends the encoded `payload` to the Scope ingest endpoint
func (r *SpanRecorder) callIngest(payload io.Reader) (statusCode int, err error) {
	req, err := http.NewRequest("POST", r.url, payload)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", r.userAgent)
	req.Header.Set("Content-Type", "application/msgpack")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("X-Scope-ApiKey", r.apiKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return resp.StatusCode, err
		}
		return resp.StatusCode, errors.New(fmt.Sprintf("error from API (%s): %s", resp.Status, body))
	}

	return resp.StatusCode, nil
}

// Combines `rawSpans` and `metadata` into a payload that the Scope backend can process
func (r *SpanRecorder) getPayload(rawSpans []tracer.RawSpan, metadata map[string]interface{}) map[string]interface{} {
	spans := []map[string]interface{}{}
	events := []map[string]interface{}{}
	for _, span := range rawSpans {
		var parentSpanID string
		if span.ParentSpanID != 0 {
			parentSpanID = fmt.Sprintf("%x", span.ParentSpanID)
		}
		spans = append(spans, map[string]interface{}{
			"context": map[string]interface{}{
				"trace_id": fmt.Sprintf("%x", span.Context.TraceID),
				"span_id":  fmt.Sprintf("%x", span.Context.SpanID),
				"baggage":  span.Context.Baggage,
			},
			"parent_span_id": parentSpanID,
			"operation":      span.Operation,
			"start":          r.applyNTPOffset(span.Start).Format(time.RFC3339Nano),
			"duration":       span.Duration.Nanoseconds(),
			"tags":           span.Tags,
		})
		for _, event := range span.Logs {
			var fields = make(map[string]interface{})
			for _, field := range event.Fields {
				fields[field.Key()] = field.Value()
			}
			eventId, err := uuid.NewRandom()
			if err != nil {
				panic(err)
			}
			events = append(events, map[string]interface{}{
				"context": map[string]interface{}{
					"trace_id": fmt.Sprintf("%x", span.Context.TraceID),
					"span_id":  fmt.Sprintf("%x", span.Context.SpanID),
					"event_id": eventId.String(),
				},
				"timestamp": r.applyNTPOffset(event.Timestamp).Format(time.RFC3339Nano),
				"fields":    fields,
			})
		}
	}
	return map[string]interface{}{
		"metadata":   metadata,
		"spans":      spans,
		"events":     events,
		tags.AgentID: metadata[tags.AgentID],
	}
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

// Gets if there any span available to be send
func (r *SpanRecorder) hasSpans() bool {
	r.spansMutex.RLock()
	defer r.spansMutex.RUnlock()
	return len(r.spans) > 0
}

// Gets the spans to be send and clears the buffer
func (r *SpanRecorder) getSpans() []tracer.RawSpan {
	r.spansMutex.Lock()
	defer r.spansMutex.Unlock()
	spans := r.spans
	r.spans = nil
	return spans
}

// Adds a span to the buffer
func (r *SpanRecorder) addSpan(span tracer.RawSpan) {
	r.spansMutex.Lock()
	defer r.spansMutex.Unlock()
	r.spans = append(r.spans, span)
}
