package nethttp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/opentracing/opentracing-go"

	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/tracer"
)

var r *tracer.InMemorySpanRecorder

func TestMain(m *testing.M) {
	PatchHttpDefaultClient()

	// Test tracer
	r = tracer.NewInMemoryRecorder()
	instrumentation.SetTracer(tracer.New(r))

	os.Exit(m.Run())
}

func TestHttpClient(t *testing.T) {
	r.Reset()
	_, ctx := opentracing.StartSpanFromContextWithTracer(context.Background(), instrumentation.Tracer(), "Test")

	req, err := http.NewRequest("GET", "https://www.google.com/", nil)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	req = req.WithContext(ctx)
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
	r.Reset()
	sp, ctx := opentracing.StartSpanFromContextWithTracer(context.Background(), instrumentation.Tracer(), "Test")
	sp.SetBaggageItem("trace.kind", "test")

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Hello world")
		if err != nil {
			w.WriteHeader(500)
			return
		}
	})
	server := httptest.NewServer(Middleware(nil))

	url := fmt.Sprintf("%s/hello", server.URL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	req = req.WithContext(ctx)
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
