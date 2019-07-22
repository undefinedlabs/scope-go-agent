package nethttp

import (
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"net/http"
)

func Middleware(h http.Handler, options ...nethttp.MWOption) http.Handler {
	if h == nil {
		h = http.DefaultServeMux
	}
	return MiddlewareFunc(h.ServeHTTP, options...)
}

func MiddlewareFunc(h http.HandlerFunc, options ...nethttp.MWOption) http.Handler {
	// Only trace requests that are part of a test trace
	options = append(options, nethttp.MWSpanFilter(func(r *http.Request) bool {
		ctx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		if err != nil {
			return false
		}
		inTest := false
		ctx.ForeachBaggageItem(func(k, v string) bool {
			if k == "trace.kind" && v == "test" {
				inTest = true
				return false
			}
			return true
		})
		return inTest
	}))

	return nethttp.MiddlewareFunc(opentracing.GlobalTracer(), h, options...)
}
