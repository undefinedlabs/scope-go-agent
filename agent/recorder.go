package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack"
	"go.undefinedlabs.com/scopeagent/tracer"
	"gopkg.in/tomb.v2"
	"net/http"
	"sync"
	"time"
)

type SpanRecorder struct {
	sync.RWMutex
	t tomb.Tomb

	apiKey      string
	apiEndpoint string
	version     string
	debugMode   bool
	metadata    map[string]interface{}

	spans          []tracer.RawSpan
	flushFrequency time.Duration
	totalSend      int
	okSend         int
	koSend         int
}

func NewSpanRecorder(agent *Agent) *SpanRecorder {
	r := new(SpanRecorder)
	r.apiEndpoint = agent.apiEndpoint
	r.apiKey = agent.apiKey
	r.version = agent.version
	r.debugMode = agent.debugMode
	r.metadata = agent.metadata
	r.flushFrequency = time.Minute
	r.t.Go(r.loop)
	return r
}

func (r *SpanRecorder) RecordSpan(span tracer.RawSpan) {
	r.Lock()
	defer r.Unlock()
	r.spans = append(r.spans, span)
	if r.debugMode {
		fmt.Printf("record span: %+v\n", span)
	}
}

func (r *SpanRecorder) changeFlushFrequency(frequency time.Duration) {
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
			hasSpans := len(r.spans) > 0
			if hasSpans || time.Now().Sub(cTime) >= r.flushFrequency {
				if r.debugMode {
					if hasSpans {
						fmt.Println("Ticker: Sending by buffer")
					} else {
						fmt.Println("Ticker: Sending by time")
					}
				}
				cTime = time.Now()
				err := r.sendSpans()
				if err != nil {
					fmt.Printf("%v\n", err)
				}
			}
		case <-r.t.Dying():
			err := r.sendSpans()
			if err != nil {
				fmt.Printf("%v\n", err)
			}
			ticker.Stop()
			return nil
		}
	}
}

func (r *SpanRecorder) sendSpans() error {
	r.Lock()
	defer r.Unlock()

	r.totalSend = r.totalSend + 1
	spans := []map[string]interface{}{}
	events := []map[string]interface{}{}
	for _, span := range r.spans {
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
			"start":          span.Start.Format(time.RFC3339Nano),
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
				"timestamp": event.Timestamp.Format(time.RFC3339Nano),
				"fields":    fields,
			})
		}
	}

	payload := map[string]interface{}{
		"metadata": r.metadata,
		"spans":    spans,
		"events":   events,
	}

	if r.debugMode {
		jsonPayLoad, _ := json.Marshal(payload)
		fmt.Printf("Payload: %s\n\n", string(jsonPayLoad))
	}

	binaryPayload, err := msgpack.Marshal(payload)
	if err != nil {
		r.koSend++
		return err
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err = zw.Write(binaryPayload)
	if err != nil {
		r.koSend++
		return err
	}
	if err := zw.Close(); err != nil {
		r.koSend++
		return err
	}
	url := fmt.Sprintf("%s/%s", r.apiEndpoint, "api/agent/ingest")

	retries := 0
	payloadSent := false
	var lastError error
	for {
		if !payloadSent && retries < 3 {
			retries++
			if r.debugMode {
				fmt.Printf("Sending payload [%d try]\n", retries)
			}
			req, err := http.NewRequest("POST", url, &buf)
			if err != nil {
				r.koSend++
				return err
			}
			req.Header.Set("User-Agent", fmt.Sprintf("scope-agent-go/%s", r.version))
			req.Header.Set("Content-Type", "application/msgpack")
			req.Header.Set("Content-Encoding", "gzip")
			req.Header.Set("X-Scope-ApiKey", r.apiKey)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				r.koSend++
				return err
			}
			_ = resp.Body.Close()

			if resp.StatusCode < 400 {
				payloadSent = true
				break
			} else if resp.StatusCode < 500 {
				lastError = errors.New(resp.Status)
				break
			} else {
				lastError = errors.New(resp.Status)
				time.Sleep(500 * time.Millisecond)
			}
		} else {
			break
		}
	}

	r.spans = nil
	if payloadSent {
		r.okSend++
	} else {
		r.koSend++
		return lastError
	}

	return nil
}
