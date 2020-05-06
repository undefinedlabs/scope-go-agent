package tracer_test

import (
	"bytes"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	opentracing "github.com/opentracing/opentracing-go"
	"go.undefinedlabs.com/scopeagent/tracer"
)

type verbatimCarrier struct {
	tracer.SpanContext
	b map[string]string
}

var _ tracer.DelegatingCarrier = &verbatimCarrier{}

func (vc *verbatimCarrier) SetBaggageItem(k, v string) {
	vc.b[k] = v
}

func (vc *verbatimCarrier) GetBaggage(f func(string, string)) {
	for k, v := range vc.b {
		f(k, v)
	}
}

func (vc *verbatimCarrier) SetState(tID uuid.UUID, sID uint64, sampled bool) {
	vc.SpanContext = tracer.SpanContext{TraceID: tID, SpanID: sID, Sampled: sampled}
}

func (vc *verbatimCarrier) State() (traceID uuid.UUID, spanID uint64, sampled bool) {
	return vc.SpanContext.TraceID, vc.SpanContext.SpanID, vc.SpanContext.Sampled
}

func TestSpanPropagator(t *testing.T) {
	const op = "test"
	recorder := tracer.NewInMemoryRecorder()
	tr := tracer.New(recorder)

	sp := tr.StartSpan(op)
	sp.SetBaggageItem("foo", "bar")

	tmc := opentracing.HTTPHeadersCarrier(http.Header{})
	tests := []struct {
		typ, carrier interface{}
	}{
		{tracer.Delegator, tracer.DelegatingCarrier(&verbatimCarrier{b: map[string]string{}})},
		{opentracing.Binary, &bytes.Buffer{}},
		{opentracing.HTTPHeaders, tmc},
		{opentracing.TextMap, tmc},
	}

	for i, test := range tests {
		if err := tr.Inject(sp.Context(), test.typ, test.carrier); err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		injectedContext, err := tr.Extract(test.typ, test.carrier)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		child := tr.StartSpan(
			op,
			opentracing.ChildOf(injectedContext))
		child.Finish()
	}
	sp.Finish()

	spans := recorder.GetSpans()
	if a, e := len(spans), len(tests)+1; a != e {
		t.Fatalf("expected %d spans, got %d", e, a)
	}

	// The last span is the original one.
	exp, spans := spans[len(spans)-1], spans[:len(spans)-1]
	exp.Duration = time.Duration(123)
	exp.Start = time.Time{}.Add(1)

	for i, sp := range spans {
		if a, e := sp.ParentSpanID, exp.Context.SpanID; a != e {
			t.Fatalf("%d: ParentSpanID %d does not match expectation %d", i, a, e)
		} else {
			// Prepare for comparison.
			sp.Context.SpanID, sp.ParentSpanID = exp.Context.SpanID, 0
			sp.Duration, sp.Start = exp.Duration, exp.Start
		}
		if a, e := sp.Context.TraceID, exp.Context.TraceID; a != e {
			t.Fatalf("%d: TraceID changed from %d to %d", i, e, a)
		}
		if !reflect.DeepEqual(exp, sp) {
			t.Fatalf("%d: wanted %+v, got %+v", i, spew.Sdump(exp), spew.Sdump(sp))
		}
	}
}
