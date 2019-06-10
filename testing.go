package scopeagent

import (
	"github.com/opentracing/opentracing-go"
	"testing"
)

func InstrumentTest(t *testing.T, f func(t *testing.T)) {
	span := opentracing.StartSpan(t.Name(), opentracing.Tags{"span.kind": "test"})
	defer span.Finish()
	f(t)
}
