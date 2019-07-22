package nethttp

import (
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"net/http"
)

type Transport struct {
	http.RoundTripper
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Only trace outgoing requests that are inside an active trace
	parent := opentracing.SpanFromContext(req.Context())
	if parent == nil {
		return t.RoundTripper.RoundTrip(req)
	}

	req, ht := nethttp.TraceRequest(opentracing.GlobalTracer(), req)
	defer ht.Finish()

	tr := nethttp.Transport{RoundTripper: t.RoundTripper}
	return tr.RoundTrip(req)
}
