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
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack"
	"gopkg.in/tomb.v2"

	"go.undefinedlabs.com/scopeagent/tags"
	"go.undefinedlabs.com/scopeagent/tracer"
)

const retryBackoff = 1 * time.Second

type SpanRecorder struct {
	sync.RWMutex
	t tomb.Tomb

	apiKey      string
	apiEndpoint string
	version     string
	userAgent   string
	debugMode   bool
	metadata    map[string]interface{}

	spans          []tracer.RawSpan
	flushFrequency time.Duration
	totalSend      int
	okSend         int
	koSend         int
	url            string
	client         *http.Client

	logger *log.Logger
}

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
	r.t.Go(r.loop)
	return r
}

// Appends a span to the in-memory buffer for async processing
func (r *SpanRecorder) RecordSpan(span tracer.RawSpan) {
	r.Lock()
	defer r.Unlock()
	if !r.t.Alive() {
		r.logger.Printf("an span is received but the recorder is already disposed.\n")
		return
	}
	r.spans = append(r.spans, span)
	if r.debugMode {
		r.logger.Printf("record span: %+v\n", span)
	}
}

func (r *SpanRecorder) ChangeFlushFrequency(frequency time.Duration) {
	r.Lock()
	defer r.Unlock()
	r.flushFrequency = frequency
}

func (r *SpanRecorder) loop() error {
	ticker := time.NewTicker(1 * time.Second)
	cTime := time.Now()
	for {
		select {
		case <-ticker.C:
			r.Lock()
			hasSpans := len(r.spans) > 0
			r.Unlock()
			if hasSpans || time.Now().Sub(cTime) >= r.flushFrequency {
				if r.debugMode {
					if hasSpans {
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
					return nil
				}
			}
		case <-r.t.Dying():
			err, _ := r.SendSpans()
			if err != nil {
				r.logger.Printf("error sending spans: %v\n", err)
			}
			ticker.Stop()
			return nil
		}
	}
}

// Sends the spans in the buffer to Scope
func (r *SpanRecorder) SendSpans() (error, bool) {
	r.Lock()
	spans := r.spans
	r.spans = nil
	r.Unlock()

	r.totalSend = r.totalSend + 1

	payload := r.getPayload(spans, r.metadata)

	if r.debugMode {
		r.logger.Printf("payload: %+v\n\n", payload)
	}

	buf, err := encodePayload(payload)
	if err != nil {
		r.koSend++
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
		r.okSend++
	} else {
		r.koSend++
	}

	return lastError, shouldExit
}

// Sends the encoded `payload` to the Scope ingest endpoint
func (r *SpanRecorder) callIngest(payload io.Reader) (statusCode int, err error) {
	req, err := http.NewRequest("POST", r.url, payload)
	if err != nil {
		r.koSend++
		return 0, err
	}
	req.Header.Set("User-Agent", r.userAgent)
	req.Header.Set("Content-Type", "application/msgpack")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("X-Scope-ApiKey", r.apiKey)

	resp, err := r.client.Do(req)
	if err != nil {
		r.koSend++
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
