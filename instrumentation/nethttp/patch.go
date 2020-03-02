package nethttp

import (
	"net/http"
	"sync"
)

type Option func(*Transport)

var once sync.Once

// Enables the payload instrumentation in the transport
func WithPayloadInstrumentation() Option {
	return func(t *Transport) {
		t.PayloadInstrumentation = true
	}
}

// Enables stacktrace
func WithStacktrace() Option {
	return func(t *Transport) {
		t.Stacktrace = true
	}
}

// Patches the default http client with the instrumented transport
func PatchHttpDefaultClient(options ...Option) {
	once.Do(func() {
		transport := &Transport{RoundTripper: http.DefaultTransport}
		for _, option := range options {
			option(transport)
		}
		transport.PayloadInstrumentation = transport.PayloadInstrumentation || (cfg.Instrumentation.Http.Payloads != nil && *cfg.Instrumentation.Http.Payloads)
		transport.Stacktrace = transport.Stacktrace || (cfg.Instrumentation.Http.Stacktrace != nil && *cfg.Instrumentation.Http.Stacktrace)
		http.DefaultClient = &http.Client{Transport: transport}
	})
}
