package wire

// ProtobufCarrier is a DelegatingCarrier that uses protocol buffers as the
// the underlying datastructure. The reason for implementing DelagatingCarrier
// is to allow for end users to serialize the underlying protocol buffers using
// jsonpb or any other serialization forms they want.
type ProtobufCarrier TracerState

// SetState set's the tracer state.
func (p *ProtobufCarrier) SetState(traceIDHi, traceIDLo, spanID uint64, sampled bool) {
	p.TraceIdHi = traceIDHi
	p.TraceIdLo = traceIDLo
	p.SpanId = spanID
	p.Sampled = sampled
}

// State returns the tracer state.
func (p *ProtobufCarrier) State() (traceIDHi, traceIDLo, spanID uint64, sampled bool) {
	traceIDHi = p.TraceIdHi
	traceIDLo = p.TraceIdLo
	spanID = p.SpanId
	sampled = p.Sampled
	return traceIDHi, traceIDLo, spanID, sampled
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
