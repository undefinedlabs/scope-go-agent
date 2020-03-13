package instrumentation

import (
	"github.com/opentracing/opentracing-go"
	"io/ioutil"
	"log"
	"sync"
)

var (
	tracer opentracing.Tracer = opentracing.NoopTracer{}
	logger                    = log.New(ioutil.Discard, "", 0)

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

func SetLogger(l *log.Logger) {
	m.Lock()
	defer m.Unlock()

	logger = l
}

func Logger() *log.Logger {
	m.RLock()
	defer m.RUnlock()

	return logger
}
