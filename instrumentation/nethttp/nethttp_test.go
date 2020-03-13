package nethttp

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"go.undefinedlabs.com/scopeagent"
	"go.undefinedlabs.com/scopeagent/agent"
	"go.undefinedlabs.com/scopeagent/tracer"
)

var r *tracer.InMemorySpanRecorder

func TestMain(m *testing.M) {
	PatchHttpDefaultClient(WithPayloadInstrumentation())

	// Test tracer
	r = tracer.NewInMemoryRecorder()
	os.Exit(scopeagent.Run(m, agent.WithRecorders(r)))
}

func TestHttpClient(t *testing.T) {
	testCtx := scopeagent.GetContextFromTest(t)
	r.Reset()

	req, err := http.NewRequest("GET", "https://www.google.com/", nil)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	req = req.WithContext(testCtx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("server returned %d status code", resp.StatusCode)
	}
	resp.Body.Close()

	spans := r.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("there aren't the right number of spans: %d", len(spans))
	}
	checkTags(t, spans[0].Tags, map[string]string{
		"component":     "net/http",
		"http.method":   "GET",
		"http.url":      "https://www.google.com/",
		"peer.hostname": "www.google.com",
		"peer.port":     "443",
	})
}

func TestHttpServer(t *testing.T) {
	testCtx := scopeagent.GetContextFromTest(t)
	r.Reset()

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Hello world")
		if err != nil {
			w.WriteHeader(500)
			return
		}
	})
	server := httptest.NewServer(Middleware(nil, MWPayloadInstrumentation()))

	url := fmt.Sprintf("%s/hello", server.URL)
	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte("Hello world request")))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	req = req.WithContext(testCtx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("server returned %d status code", resp.StatusCode)
	}
	resp.Body.Close()
	server.Close()

	spans := r.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("there aren't the right number of spans: %d", len(spans))
	}
	checkTags(t, spans[0].Tags, map[string]string{
		"component":             "net/http",
		"http.method":           "POST",
		"http.url":              "/hello",
		"span.kind":             "server",
		"http.status_code":      "200",
		"http.request_payload":  "Hello world request",
		"http.response_payload": "Hello world",
	})
	checkTags(t, spans[1].Tags, map[string]string{
		"component":             "net/http",
		"http.method":           "POST",
		"http.url":              url,
		"peer.ipv4":             "127.0.0.1",
		"span.kind":             "client",
		"http.request_payload":  "Hello world request",
		"http.response_payload": "Hello world",
	})
}

func checkTags(t *testing.T, tags map[string]interface{}, expected map[string]string) {
	for eK, eV := range expected {
		if ok, aV := checkTag(tags, eK, eV); !ok {
			if aV == "" {
				t.Fatalf("the tag with key = '%s' was not found in the span tags", eK)
			} else {
				t.Fatalf("the tag with key = '%s' has a different value in the span tags. Expected = '%s', Actual = '%s'", eK, eV, aV)
			}
		}
	}
}

func checkTag(tags map[string]interface{}, key string, expectedValue string) (bool, string) {
	if val, ok := tags[key]; ok {
		sVal := fmt.Sprint(val)
		return expectedValue == sVal, sVal
	}
	return false, ""
}
