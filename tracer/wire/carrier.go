package wire

import (
	"encoding/binary"
	"github.com/google/uuid"
)

// ProtobufCarrier is a DelegatingCarrier that uses protocol buffers as the
// the underlying datastructure. The reason for implementing DelagatingCarrier
// is to allow for end users to serialize the underlying protocol buffers using
// jsonpb or any other serialization forms they want.
type ProtobufCarrier TracerState

// SetState set's the tracer state.
func (p *ProtobufCarrier) SetState(traceID uuid.UUID, spanID uint64, sampled bool) {
	bytes, _ := traceID.MarshalBinary()
	p.TraceIdHi = binary.LittleEndian.Uint64(bytes[:8])
	p.TraceIdLo = binary.LittleEndian.Uint64(bytes[8:])
	p.SpanId = spanID
	p.Sampled = sampled
}

// State returns the tracer state.
func (p *ProtobufCarrier) State() (traceID uuid.UUID, spanID uint64, sampled bool) {
	traceIdBytes := make([]byte, 16)
	binary.LittleEndian.PutUint64(traceIdBytes[:8], p.TraceIdHi)
	binary.LittleEndian.PutUint64(traceIdBytes[8:], p.TraceIdLo)
	tId, _ := uuid.FromBytes(traceIdBytes)
	traceID = tId
	spanID = p.SpanId
	sampled = p.Sampled
	return traceID, spanID, sampled
}

// SetBaggageItem sets a baggage item.
func (p *ProtobufCarrier) SetBaggageItem(key, value string) {
	if p.BaggageItems == nil {
		p.BaggageItems = map[string]string{key: value}
		return
	}

	p.BaggageItems[key] = value
}

// GetBaggage iterates over each baggage item and executes the callback with
// the key:value pair.
func (p *ProtobufCarrier) GetBaggage(f func(k, v string)) {
	for k, v := range p.BaggageItems {
		f(k, v)
	}
}
