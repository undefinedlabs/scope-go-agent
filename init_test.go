package scopeagent_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/opentracing/opentracing-go"

	"go.undefinedlabs.com/scopeagent"
	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestFromGoroutineRace(t *testing.T) {
	ctx := scopeagent.GetContextFromTest(t)
	span := opentracing.SpanFromContext(ctx)

	for x := 0; x<10;x++ {
		go func() {
			for i := 0; i < 100; i++ {
				span.SetTag(fmt.Sprintf("Key%v", i), i)
			}
		}()
	}
}
