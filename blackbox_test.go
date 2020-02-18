/*
This file contains complete black box tests of the agent, avoiding any cycle reference
*/

package scopeagent_test

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/vmihailenco/msgpack"

	"go.undefinedlabs.com/scopeagent"
	"go.undefinedlabs.com/scopeagent/agent"
	"go.undefinedlabs.com/scopeagent/env"
	"go.undefinedlabs.com/scopeagent/tracer"
)

var (
	server             *httptest.Server
	onPayloadError     = func(error) {}
	onPayloadWithSpans = func(map[string]interface{}) {}
)

func TestMain(m *testing.M) {
	url := configureServer()
	env.ScopeDsn.Value = url
	env.ScopeDsn.IsSet = true
	os.Exit(scopeagent.Run(m, agent.WithSetGlobalTracer()))
}

func TestComplete(t *testing.T) {
	myPayload := runAndGetPayload(func() {
		sp, _ := opentracing.StartSpanFromContext(scopeagent.GetContextFromTest(t), "MySpan")
		sp.SetTag("Demo", "Value")
		sp.LogFields(log.String("Hello", "World"), log.Int("Num", 42))
		sp.Finish()
	})
	fmt.Println(getSpans(myPayload))
}

func runAndGetPayload(f func()) map[string]interface{} {
	pChan := make(chan map[string]interface{}, 1)
	onPayloadWithSpans = func(payload map[string]interface{}) { pChan <- payload }
	f()
	pload := <-pChan
	onPayloadWithSpans = nil
	return pload
}

func getSpans(payload map[string]interface{}) []tracer.RawSpan {
	var spans []tracer.RawSpan
	payloadSpans := payload["spans"].([]interface{})
	for _, item := range payloadSpans {
		pSpan := item.(map[string]interface{})
		pSpanContext := pSpan["context"].(map[string]interface{})
		traceId, _ := strconv.ParseUint(pSpanContext["trace_id"].(string), 16, 64)
		spanId, _ := strconv.ParseUint(pSpanContext["span_id"].(string), 16, 64)
		baggage := pSpanContext["baggage"].(map[string]interface{})
		parentSpanId, _ := strconv.ParseUint(pSpan["parent_span_id"].(string), 16, 64)
		startTime, _ := time.Parse(time.RFC3339Nano, pSpan["start"].(string))
		duration := time.Duration(pSpan["duration"].(int64))

		var tags map[string]interface{}
		if pTags, ok := pSpan["tags"].(map[string]interface{}); ok {
			tags = pTags
		}
		cBaggage := map[string]string{}
		for k, v := range baggage {
			cBaggage[k] = fmt.Sprintf("%v", v)
		}

		rSpan := tracer.RawSpan{
			Context: tracer.SpanContext{
				TraceID: traceId,
				SpanID:  spanId,
				Sampled: true,
				Baggage: cBaggage,
			},
			ParentSpanID: parentSpanId,
			Operation:    pSpan["operation"].(string),
			Start:        startTime,
			Duration:     duration,
			Tags:         opentracing.Tags(tags),
			Logs:         nil,
		}
		spans = append(spans, rSpan)
	}
	payloadEvents := payload["events"].([]interface{})
	for _, item := range payloadEvents {
		pEvent := item.(map[string]interface{})
		pEventContext := pEvent["context"].(map[string]interface{})
		traceId, _ := strconv.ParseUint(pEventContext["trace_id"].(string), 16, 64)
		spanId, _ := strconv.ParseUint(pEventContext["span_id"].(string), 16, 64)
		timestamp, _ := time.Parse(time.RFC3339Nano, pEvent["timestamp"].(string))
		var recordFields []log.Field
		if pFields, ok := pEvent["fields"].(map[string]interface{}); ok {
			for k, v := range pFields {
				switch v.(type) {
				case string:
					recordFields = append(recordFields, log.String(k, v.(string)))
				case bool:
					recordFields = append(recordFields, log.Bool(k, v.(bool)))
				case int:
					recordFields = append(recordFields, log.Int(k, v.(int)))
				case int32:
					recordFields = append(recordFields, log.Int32(k, v.(int32)))
				case int64:
					recordFields = append(recordFields, log.Int64(k, v.(int64)))
				case uint32:
					recordFields = append(recordFields, log.Uint32(k, v.(uint32)))
				case uint64:
					recordFields = append(recordFields, log.Uint64(k, v.(uint64)))
				case float32:
					recordFields = append(recordFields, log.Float32(k, v.(float32)))
				case float64:
					recordFields = append(recordFields, log.Float64(k, v.(float64)))
				case error:
					recordFields = append(recordFields, log.Error(v.(error)))
				default:
					recordFields = append(recordFields, log.Object(k, v))
				}
			}
		}

		for idx := range spans {
			if spans[idx].Context.TraceID != traceId || spans[idx].Context.SpanID != spanId {
				continue
			}
			spans[idx].Logs = append(spans[idx].Logs, opentracing.LogRecord{
				Timestamp: timestamp,
				Fields:    recordFields,
			})
		}
	}
	return spans
}

func configureServer() string {
	http.HandleFunc("/api/agent/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			if onPayloadError != nil {
				onPayloadError(errors.New("invalid method: " + r.Method))
			}
			w.WriteHeader(400)
			return
		}
		zipBytes, err := gzip.NewReader(r.Body)
		if err != nil {
			if onPayloadError != nil {
				onPayloadError(err)
			}
			w.WriteHeader(500)
			return
		}
		bodyBytes, err := ioutil.ReadAll(zipBytes)
		if err != nil {
			if onPayloadError != nil {
				onPayloadError(err)
			}
			w.WriteHeader(500)
			return
		}
		var payload map[string]interface{}
		err = msgpack.Unmarshal(bodyBytes, &payload)
		if err != nil {
			if onPayloadError != nil {
				onPayloadError(err)
			}
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		if val, ok := payload["spans"].([]interface{}); ok && len(val) > 0 && onPayloadWithSpans != nil {
			onPayloadWithSpans(payload)
		}
	})
	server = httptest.NewServer(nil)
	return server.URL
}
