package agent

import (
	"bytes"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gopkg.in/tomb.v2"

	"go.undefinedlabs.com/scopeagent/tags"
	"go.undefinedlabs.com/scopeagent/tracer"
)

type (
	inProcSpanRecorder struct {
		sync.RWMutex
		t tomb.Tomb

		agentId     string
		apiKey      string
		apiEndpoint string
		version     string
		userAgent   string
		debugMode   bool
		metadata    map[string]interface{}

		spans []tracer.RawSpan

		flushFrequency time.Duration
		url            string
		client         *http.Client

		logger *log.Logger
		stats  *recorderStats
	}
)

func newInProcSpanRecorder(agent *Agent) ScopeSpanRecorder {
	r := new(inProcSpanRecorder)
	r.agentId = agent.agentId
	r.apiEndpoint = agent.apiEndpoint
	r.apiKey = agent.apiKey
	r.version = agent.version
	r.userAgent = agent.userAgent
	r.debugMode = agent.debugMode
	r.metadata = agent.metadata
	r.logger = agent.logger
	r.flushFrequency = agent.flushFrequency
	r.url = agent.getUrl("api/agent/ingest")
	r.client = &http.Client{}
	r.stats = &recorderStats{logger: r.logger}
	r.t.Go(r.loop)
	return r
}

// Appends a span to the in-memory buffer for async processing
func (r *inProcSpanRecorder) RecordSpan(span tracer.RawSpan) {
	if !r.t.Alive() {
		atomic.AddInt64(&r.stats.totalSpans, 1)
		atomic.AddInt64(&r.stats.spansRejected, 1)
		if isTestSpan(span) {
			atomic.AddInt64(&r.stats.totalTestSpans, 1)
			atomic.AddInt64(&r.stats.testSpansRejected, 1)
		}
		r.logger.Printf("a span has been received but the recorder is not running")
		return
	}
	r.addSpan(span)
}

func (r *inProcSpanRecorder) loop() error {
	ticker := time.NewTicker(1 * time.Second)
	cTime := time.Now()
	for {
		select {
		case <-ticker.C:
			hasSpans := r.hasSpans()
			if hasSpans || time.Now().Sub(cTime) >= r.getFlushFrequency() {
				if r.debugMode {
					if hasSpans {
						r.logger.Println("Ticker: Sending by buffer")
					} else {
						r.logger.Println("Ticker: Sending by time")
					}
				}
				cTime = time.Now()
				err, shouldExit := r.sendSpans()
				if shouldExit {
					r.logger.Printf("stopping recorder due to: %v", err)
					return err // Return so we don't try again in the Dying channel
				} else if err != nil {
					r.logger.Printf("error sending spans: %v\n", err)
				}
			}
		case <-r.t.Dying():
			err, _ := r.sendSpans()
			if err != nil {
				r.logger.Printf("error sending spans: %v\n", err)
			}
			ticker.Stop()
			return nil
		}
	}
}

// Sends the spans in the buffer to Scope
func (r *inProcSpanRecorder) sendSpans() (error, bool) {
	atomic.AddInt64(&r.stats.sendSpansCalls, 1)
	spans := r.popSpans()

	const batchSize = 1000
	batchLength := len(spans) / batchSize

	r.logger.Printf("sending %d spans in %d batches", len(spans), batchLength+1)

	var lastError error
	for b := 0; b <= batchLength; b++ {
		var batch []tracer.RawSpan
		// We extract the batch of spans to be send
		if b == batchLength {
			// If we are in the last batch, we select the remaining spans
			batch = spans[b*batchSize:]
		} else {
			batch = spans[b*batchSize : ((b + 1) * batchSize)]
		}

		payload := r.getPayload(batch)

		buf, err := encodePayload(payload)
		if err != nil {
			atomic.AddInt64(&r.stats.sendSpansKo, 1)
			atomic.AddInt64(&r.stats.spansNotSent, int64(len(spans)))
			return err, false
		}

		var testSpans int64
		for _, span := range batch {
			if isTestSpan(span) {
				testSpans++
			}
		}

		if batchLength > 0 {
			r.logger.Printf("sending batch %d with %d spans", b+1, len(batch))
		}
		statusCode, err := r.callIngest(buf)
		if err != nil {
			atomic.AddInt64(&r.stats.sendSpansKo, 1)
			atomic.AddInt64(&r.stats.spansNotSent, int64(len(spans)))
			atomic.AddInt64(&r.stats.testSpansNotSent, testSpans)
		} else {
			atomic.AddInt64(&r.stats.sendSpansOk, 1)
			atomic.AddInt64(&r.stats.spansSent, int64(len(spans)))
			atomic.AddInt64(&r.stats.testSpansSent, testSpans)
		}
		if statusCode == 401 {
			return err, true
		}
		lastError = err
	}
	return lastError, false
}

// Stop recorder
func (r *inProcSpanRecorder) Stop() {
	if r.debugMode {
		r.logger.Println("Scope recorder is stopping gracefully...")
	}
	r.t.Kill(nil)
	_ = r.t.Wait()
	if r.debugMode {
		r.stats.Write()
	}
}

func (r *inProcSpanRecorder) Stats() RecorderStats {
	return r.stats
}

// Sends the encoded `payload` to the Scope ingest endpoint
func (r *inProcSpanRecorder) callIngest(payload *bytes.Buffer) (statusCode int, err error) {
	payloadBytes := payload.Bytes()
	var lastError error
	for i := 0; i <= numOfRetries; i++ {
		req, err := http.NewRequest("POST", r.url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			return 0, err
		}
		req.Header.Set("User-Agent", r.userAgent)
		req.Header.Set("Content-Type", "application/msgpack")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("X-Scope-ApiKey", r.apiKey)

		if r.debugMode {
			if i == 0 {
				r.logger.Println("sending payload")
			} else {
				r.logger.Printf("sending payload [retry %d]", i)
			}
		}

		resp, err := r.client.Do(req)
		if err != nil {
			if v, ok := err.(*url.Error); ok {
				// Don't retry if the error was due to TLS cert verification failure.
				if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
					return 0, errors.New(fmt.Sprintf("error: http client returns: %s", err.Error()))
				}
			}

			lastError = err
			r.logger.Printf("client error '%s', retrying in %d seconds", err.Error(), retryBackoff/time.Second)
			time.Sleep(retryBackoff)
			atomic.AddInt64(&r.stats.sendSpansRetries, 1)
			continue
		}

		var (
			bodyData []byte
			status   string
		)
		statusCode = resp.StatusCode
		status = resp.Status
		if resp.Body != nil && resp.Body != http.NoBody {
			body, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				bodyData = body
			}
		}
		if err := resp.Body.Close(); err != nil { // We can't defer inside a for loop
			r.logger.Printf("error: closing the response body. %s", err.Error())
		}

		if statusCode == 0 || statusCode >= 400 {
			lastError = errors.New(fmt.Sprintf("error from API [status: %s]: %s", status, string(bodyData)))
		}

		// Check the response code. We retry on 500-range responses to allow
		// the server time to recover, as 500's are typically not permanent
		// errors and may relate to outages on the server side. This will catch
		// invalid response codes as well, like 0 and 999.
		if statusCode == 0 || (statusCode >= 500 && statusCode != 501) {
			r.logger.Printf("error: [status code: %d], retrying in %d seconds", statusCode, retryBackoff/time.Second)
			time.Sleep(retryBackoff)
			atomic.AddInt64(&r.stats.sendSpansRetries, 1)
			continue
		}

		if i > 0 {
			r.logger.Printf("payload was sent successfully after retry.")
		}
		break
	}

	if statusCode != 0 && statusCode < 400 {
		return statusCode, nil
	}
	return statusCode, lastError
}

// Combines `rawSpans` and `metadata` into a payload that the Scope backend can process
func (r *inProcSpanRecorder) getPayload(rawSpans []tracer.RawSpan) map[string]interface{} {
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
		"metadata":   r.metadata,
		"spans":      spans,
		"events":     events,
		tags.AgentID: r.agentId,
	}
}

// Gets the current flush frequency
func (r *inProcSpanRecorder) getFlushFrequency() time.Duration {
	r.RLock()
	defer r.RUnlock()
	return r.flushFrequency
}

// Gets if there any span available to be send
func (r *inProcSpanRecorder) hasSpans() bool {
	r.RLock()
	defer r.RUnlock()
	return len(r.spans) > 0
}

// Gets the spans to be send and clears the buffer
func (r *inProcSpanRecorder) popSpans() []tracer.RawSpan {
	r.Lock()
	defer r.Unlock()
	spans := r.spans
	r.spans = nil
	return spans
}

// Adds a span to the buffer
func (r *inProcSpanRecorder) addSpan(span tracer.RawSpan) {
	r.Lock()
	defer r.Unlock()
	r.spans = append(r.spans, span)
	atomic.AddInt64(&r.stats.totalSpans, 1)
	if isTestSpan(span) {
		atomic.AddInt64(&r.stats.totalTestSpans, 1)
	}
}

// Applies the NTP offset to the given time
func (r *inProcSpanRecorder) applyNTPOffset(t time.Time) time.Time {
	once.Do(func() {
		if r.debugMode {
			r.logger.Println("calculating ntp offset.")
		}
		offset, err := getNTPOffset()
		if err == nil {
			ntpOffset = offset
			r.logger.Printf("ntp offset: %v\n", ntpOffset)
		} else {
			r.logger.Printf("error calculating the ntp offset: %v\n", err)
		}
	})
	return t.Add(ntpOffset)
}
