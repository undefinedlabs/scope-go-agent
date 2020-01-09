package tracer

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"go.undefinedlabs.com/scopeagent/tracer/wire"
)

type textMapPropagator struct {
	tracer *tracerImpl
}
type binaryPropagator struct {
	tracer *tracerImpl
}

const (
	prefixTracerState = "ot-tracer-"
	prefixBaggage     = "ot-baggage-"

	tracerStateFieldCount = 3
	fieldNameTraceID      = prefixTracerState + "traceid"
	fieldNameSpanID       = prefixTracerState + "spanid"
	fieldNameSampled      = prefixTracerState + "sampled"

	traceParentKey = "traceparent"
	traceStateKey  = "tracestate"
)

func (p *textMapPropagator) Inject(
	spanContext opentracing.SpanContext,
	opaqueCarrier interface{},
) error {
	sc, ok := spanContext.(SpanContext)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}
	carrier, ok := opaqueCarrier.(opentracing.TextMapWriter)
	if !ok {
		return opentracing.ErrInvalidCarrier
	}
	carrier.Set(fieldNameTraceID, strconv.FormatUint(sc.TraceID, 16))
	carrier.Set(fieldNameSpanID, strconv.FormatUint(sc.SpanID, 16))
	carrier.Set(fieldNameSampled, strconv.FormatBool(sc.Sampled))

	tpSampled := "00"
	if sc.Sampled {
		tpSampled = "01"
	}
	traceParentValue := fmt.Sprintf("%v-%032x-%016x-%v",
		"00",       // Version 0
		sc.TraceID, // 8bytes TraceId
		sc.SpanID,  // 8bytes SpanId
		tpSampled,  // 00 for not sampled, 01 for sampled
	)
	carrier.Set(traceParentKey, traceParentValue)

	var traceStatePairs []string

	for k, v := range sc.Baggage {
		carrier.Set(prefixBaggage+k, v)
		traceStatePairs = append(traceStatePairs, k+"="+v)
	}

	traceStateValue := strings.Join(traceStatePairs, ",")
	carrier.Set(traceStateKey, traceStateValue)

	return nil
}

func (p *textMapPropagator) Extract(
	opaqueCarrier interface{},
) (opentracing.SpanContext, error) {
	carrier, ok := opaqueCarrier.(opentracing.TextMapReader)
	if !ok {
		return nil, opentracing.ErrInvalidCarrier
	}
	requiredFieldCount := 0
	var traceID, spanID uint64
	var sampled bool
	var err error
	decodedBaggage := make(map[string]string)
	err = carrier.ForeachKey(func(k, v string) error {
		switch strings.ToLower(k) {
		case fieldNameTraceID:
			traceID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
		case fieldNameSpanID:
			spanID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
		case fieldNameSampled:
			sampled, err = strconv.ParseBool(v)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
		case traceParentKey:
			if len(v) < 55 {
				return opentracing.ErrSpanContextCorrupted
			}
			traceParentArray := strings.Split(v, "-")
			if len(traceParentArray) < 4 || traceParentArray[0] != "00" || len(traceParentArray[1]) != 32 || len(traceParentArray[2]) != 16 {
				return opentracing.ErrSpanContextCorrupted
			}

			traceID, err = strconv.ParseUint(traceParentArray[1][16:], 16, 64)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
			spanID, err = strconv.ParseUint(traceParentArray[2], 16, 64)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
			if traceParentArray[3] == "01" {
				sampled = true
			}
		case traceStateKey:
			traceStateArray := strings.Split(v, ",")
			for _, stItem := range traceStateArray {
				stItem = strings.TrimSpace(stItem)
				if strings.IndexRune(stItem, '=') > -1 {
					stItemArray := strings.Split(stItem, "=")
					decodedBaggage[stItemArray[0]] = stItemArray[1]
				}
			}
		default:
			lowercaseK := strings.ToLower(k)
			if strings.HasPrefix(lowercaseK, prefixBaggage) {
				decodedBaggage[strings.TrimPrefix(lowercaseK, prefixBaggage)] = v
			}
			// Balance off the requiredFieldCount++ just below...
			requiredFieldCount--
		}
		requiredFieldCount++
		return nil
	})
	if err != nil {
		return nil, err
	}
	if requiredFieldCount < tracerStateFieldCount {
		if requiredFieldCount == 0 {
			return nil, opentracing.ErrSpanContextNotFound
		}
		return nil, opentracing.ErrSpanContextCorrupted
	}

	return SpanContext{
		TraceID: traceID,
		SpanID:  spanID,
		Sampled: sampled,
		Baggage: decodedBaggage,
	}, nil
}

func (p *binaryPropagator) Inject(
	spanContext opentracing.SpanContext,
	opaqueCarrier interface{},
) error {
	sc, ok := spanContext.(SpanContext)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}
	carrier, ok := opaqueCarrier.(io.Writer)
	if !ok {
		return opentracing.ErrInvalidCarrier
	}

	state := wire.TracerState{}
	state.TraceId = sc.TraceID
	state.SpanId = sc.SpanID
	state.Sampled = sc.Sampled
	state.BaggageItems = sc.Baggage

	b, err := proto.Marshal(&state)
	if err != nil {
		return err
	}

	// Write the length of the marshalled binary to the writer.
	length := uint32(len(b))
	if err := binary.Write(carrier, binary.BigEndian, &length); err != nil {
		return err
	}

	_, err = carrier.Write(b)
	return err
}

func (p *binaryPropagator) Extract(
	opaqueCarrier interface{},
) (opentracing.SpanContext, error) {
	carrier, ok := opaqueCarrier.(io.Reader)
	if !ok {
		return nil, opentracing.ErrInvalidCarrier
	}

	// Read the length of marshalled binary. io.ReadAll isn't that performant
	// since it keeps resizing the underlying buffer as it encounters more bytes
	// to read. By reading the length, we can allocate a fixed sized buf and read
	// the exact amount of bytes into it.
	var length uint32
	if err := binary.Read(carrier, binary.BigEndian, &length); err != nil {
		return nil, opentracing.ErrSpanContextCorrupted
	}
	buf := make([]byte, length)
	if n, err := carrier.Read(buf); err != nil {
		if n > 0 {
			return nil, opentracing.ErrSpanContextCorrupted
		}
		return nil, opentracing.ErrSpanContextNotFound
	}

	ctx := wire.TracerState{}
	if err := proto.Unmarshal(buf, &ctx); err != nil {
		return nil, opentracing.ErrSpanContextCorrupted
	}

	return SpanContext{
		TraceID: ctx.TraceId,
		SpanID:  ctx.SpanId,
		Sampled: ctx.Sampled,
		Baggage: ctx.BaggageItems,
	}, nil
}
