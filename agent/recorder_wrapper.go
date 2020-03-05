package agent

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack"

	"go.undefinedlabs.com/scopeagent/tags"
	"go.undefinedlabs.com/scopeagent/tracer"
)

type (
	wrapperSpanRecorder struct {
		agentId     string
		apiKey      string
		apiEndpoint string
		version     string
		userAgent   string
		debugMode   bool
		metadata    map[string]interface{}

		url    string
		client *http.Client

		logger *log.Logger

		stats *recorderStats
	}
)
type unixDialer struct {
	net.Dialer
}

// overriding net.Dialer.Dial to force unix socket connection
func (d *unixDialer) Dial(network, address string) (net.Conn, error) {
	path := strings.Replace(os.Getenv("SCOPE_CLI_UNIX_SOCKET"), "unix://", "", -1)
	return d.Dialer.Dial("unix", path)
}
func (d *unixDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	path := strings.Replace(os.Getenv("SCOPE_CLI_UNIX_SOCKET"), "unix://", "", -1)
	return d.Dialer.DialContext(ctx, "unix", path)
}

// copied from http.DefaultTransport with minimal changes
var transport http.RoundTripper = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&unixDialer{net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	},
	}).DialContext,
	TLSHandshakeTimeout: 10 * time.Second,
}

func newWrapperSpanRecorder(agent *Agent) ScopeSpanRecorder {
	r := new(wrapperSpanRecorder)
	r.agentId = agent.agentId
	r.apiEndpoint = agent.apiEndpoint
	r.apiKey = agent.apiKey
	r.version = agent.version
	r.userAgent = agent.userAgent
	r.debugMode = agent.debugMode
	r.metadata = agent.metadata
	r.logger = agent.logger
	r.url = "http://unix/api/agent/ingest"
	r.client = &http.Client{
		Transport: transport,
	}
	r.stats = &recorderStats{logger: r.logger}
	return r
}

func (r *wrapperSpanRecorder) RecordSpan(span tracer.RawSpan) {
	atomic.AddInt64(&r.stats.totalSpans, 1)
	isTest := isTestSpan(span)
	if isTest {
		atomic.AddInt64(&r.stats.totalTestSpans, 1)
	}

	payload := r.getPayload(span)
	buf, err := r.encodePayload(payload)
	if err != nil {
		r.logger.Println(err)
		r.sendError(isTest)
		return
	}
	payloadBytes := buf.Bytes()
	req, err := http.NewRequest("POST", r.url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		r.logger.Println(err)
		r.sendError(isTest)
		return
	}
	req.Header.Set("User-Agent", r.userAgent)
	req.Header.Set("Content-Type", "application/msgpack")
	req.Header.Set("X-Scope-ApiKey", r.apiKey)
	resp, err := r.client.Do(req)
	if err != nil {
		r.logger.Println(err)
		r.sendError(isTest)
		return
	}
	if resp.Body != nil && resp.Body != http.NoBody {
		ioutil.ReadAll(resp.Body)
	}
	if err := resp.Body.Close(); err != nil { // We can't defer inside a for loop
		r.logger.Printf("error: closing the response body. %s", err.Error())
	}
	r.sendOk(isTest)
}

func (r *wrapperSpanRecorder) sendError(isTest bool) {
	atomic.AddInt64(&r.stats.sendSpansKo, 1)
	atomic.AddInt64(&r.stats.spansNotSent, 1)
	if isTest {
		atomic.AddInt64(&r.stats.testSpansNotSent, 1)
	}
}
func (r *wrapperSpanRecorder) sendOk(isTest bool) {
	atomic.AddInt64(&r.stats.sendSpansOk, 1)
	atomic.AddInt64(&r.stats.spansSent, 1)
	if isTest {
		atomic.AddInt64(&r.stats.testSpansSent, 1)
	}

}

func (r *wrapperSpanRecorder) Stop() {
	r.stats.Write()
}

func (r *wrapperSpanRecorder) Stats() RecorderStats {
	return r.stats
}

// Combines `rawSpans` and `metadata` into a payload that the Scope backend can process
func (r *wrapperSpanRecorder) getPayload(span tracer.RawSpan) map[string]interface{} {
	spans := []map[string]interface{}{}
	events := []map[string]interface{}{}
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
	return map[string]interface{}{
		"metadata":   r.metadata,
		"spans":      spans,
		"events":     events,
		tags.AgentID: r.agentId,
	}
}

func (r *wrapperSpanRecorder) encodePayload(payload map[string]interface{}) (*bytes.Buffer, error) {
	binaryPayload, err := msgpack.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(binaryPayload), err
}
