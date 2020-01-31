package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/opentracing/opentracing-go"

	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/tracer"
)

var r *tracer.InMemorySpanRecorder

func TestMain(m *testing.M) {
	// Test tracer
	r = tracer.NewInMemoryRecorder()
	instrumentation.SetTracer(tracer.New(r))

	os.Exit(m.Run())
}

func TestProcessContextInjection(t *testing.T) {
	r.Reset()
	_, ctx := opentracing.StartSpanFromContextWithTracer(context.Background(), instrumentation.Tracer(), "Test")

	// Create command and inject the context
	cmd := exec.Command("/usr/local/my-cli", "FirstArg", "SecondArg")
	cmd.Dir = "/home"
	pSpan, _ := InjectToCmdWithSpan(ctx, cmd)

	// Simulate process context extraction
	cCtx, err := Extract(&cmd.Env)
	if err != nil {
		panic(err)
	}
	cSpan := instrumentation.Tracer().StartSpan(getOperationNameFromArgs(cmd.Args), opentracing.ChildOf(cCtx))
	cSpan.Finish()
	pSpan.Finish()

	spans := r.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("there aren't the right number of spans: %d", len(spans))
	}

	if spans[0].Operation != "my-cli FirstArg SecondArg " {
		t.Fatal("the operation name of the Cmd span is invalid")
	}
	checkTags(t, spans[1].Tags, map[string]string{
		"Args": "[/usr/local/my-cli FirstArg SecondArg]",
		"Path": "/usr/local/my-cli",
		"Dir":  "/home",
	})
}

func checkTags(t *testing.T, tags map[string]interface{}, expected map[string]string) {
	for eK, eV := range expected {
		if ok, aV := checkTag(tags, eK, eV); !ok {
			if aV == "" {
				t.Fatalf("the tag with key = '%s' was not found in the span tags", eK)
			} else {
				t.Fatalf("the tag with key = '%s' has a different value in the span tags. Expected = '%s', Actual = '%s'", eK, eV, aV)
			}
		}
	}
}

func checkTag(tags map[string]interface{}, key string, expectedValue string) (bool, string) {
	if val, ok := tags[key]; ok {
		sVal := fmt.Sprint(val)
		return expectedValue == sVal, sVal
	}
	return false, ""
}
