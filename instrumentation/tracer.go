package instrumentation

import (
	"github.com/opentracing/opentracing-go"
	"sync"
)

var (
	tracer opentracing.Tracer = opentracing.NoopTracer{}

	m sync.RWMutex
)

func SetTracer(t opentracing.Tracer) {
	m.Lock()
	defer m.Unlock()

	tracer = t
}

func Tracer() opentracing.Tracer {
	m.RLock()
	defer m.RUnlock()

	return tracer
}
