package scopeagent

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/google/uuid"
	"github.com/undefinedlabs/go-agent/tracer"
	"github.com/vmihailenco/msgpack"
	"gopkg.in/tomb.v2"
	"net/http"
	"sync"
	"time"
)

type SpanRecorder struct {
	sync.RWMutex
	t tomb.Tomb

	agent *Agent
	spans []tracer.RawSpan
}

func NewSpanRecorder(agent *Agent) *SpanRecorder {
	r := new(SpanRecorder)
	r.agent = agent
	r.t.Go(r.loop)
	return r
}

func (r *SpanRecorder) RecordSpan(span tracer.RawSpan) {
	r.Lock()
	defer r.Unlock()
	r.spans = append(r.spans, span)
}

func (r *SpanRecorder) loop() error {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			err := r.sendSpans()
			if err != nil {
				fmt.Printf("%v", err)
			}
		case <-r.t.Dying():
			err := r.sendSpans()
			if err != nil {
				fmt.Printf("%v", err)
			}
			ticker.Stop()
			return nil
		}
	}
}

func (r *SpanRecorder) sendSpans() error {
	r.Lock()
	defer r.Unlock()

	var spans []map[string]interface{}
	var events []map[string]interface{}
	for _, span := range r.spans {
		spans = append(spans, map[string]interface{}{
			"context": map[string]interface{}{
				"trace_id": fmt.Sprintf("%x", span.Context.TraceID),
				"span_id":  fmt.Sprintf("%x", span.Context.SpanID),
				"baggage":  span.Context.Baggage,
			},
			"parent_span_id": span.ParentSpanID,
			"operation":      span.Operation,
			"start":          span.Start.Format(time.RFC3339),
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
					"event_id": eventId,
				},
				"timestamp": event.Timestamp.Format(time.RFC3339),
				"fields":    fields,
			})
		}
	}

	payload := map[string]interface{}{
		"metadata": r.agent.metadata,
		"spans":    spans,
		"events":   events,
	}

	binaryPayload, err := msgpack.Marshal(payload)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err = zw.Write(binaryPayload)
	if err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s", r.agent.scopeEndpoint, "api/agent/ingest")
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("scope-agent-go/%s", r.agent.version))
	req.Header.Set("Content-Type", "application/msgpack")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("X-Scope-ApiKey", r.agent.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode <= 500 {
		r.spans = nil
	}

	return nil
}
