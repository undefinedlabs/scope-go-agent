package agent

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"go.undefinedlabs.com/scopeagent/env"
	"go.undefinedlabs.com/scopeagent/tags"
)

func TestDsnParser(t *testing.T) {
	dsnValues := [][]string{
		{"https://4432432432432423@shared.scope.dev", "4432432432432423", "https://shared.scope.dev"},
		{"http://4432432432432423@shared.scope.dev", "4432432432432423", "http://shared.scope.dev"},
		{"https://4432432432432423:ignored@shared.scope.dev", "4432432432432423", "https://shared.scope.dev"},
		{"https://4432432432432423:ignored@shared.scope.dev/custom/path", "4432432432432423", "https://shared.scope.dev/custom/path"},
		{"https://4432432432432423:ignored@scope.dev", "4432432432432423", "https://scope.dev"},

		{"4432432432432423@shared.scope.dev", "", "4432432432432423@shared.scope.dev"},
		{"noise", "", "noise"},
	}

	for i := 0; i < len(dsnValues); i++ {
		dsnValue := dsnValues[i]
		t.Run(dsnValue[0], func(st *testing.T) {
			apiKey, apiEndpoint, err := parseDSN(dsnValue[0])
			if apiKey != dsnValue[1] || apiEndpoint != dsnValue[2] {
				if err != nil {
					st.Error(err)
				} else {
					fmt.Println(dsnValue, apiKey, apiEndpoint)
				}
				st.FailNow()
			}
		})
	}
}

func TestWithConfigurationKeys(t *testing.T) {
	myKeys := []string{"ConfigKey01", "ConfigKey02", "ConfigKey03"}

	agent, err := NewAgent(WithApiKey("123"), WithConfigurationKeys(myKeys))
	if err != nil {
		t.Fatal(err)
	}
	agent.Stop()

	if agentKeys, ok := agent.metadata[tags.ConfigurationKeys]; ok {
		if !reflect.DeepEqual(myKeys, agentKeys) {
			t.Fatal("the configuration keys array are different")
		}
	} else {
		t.Fatal("agent configuration keys can't be found")
	}
}

func TestWithConfiguration(t *testing.T) {
	myKeys := []string{"ConfigKey01", "ConfigKey02", "ConfigKey03"}
	myConfiguration := map[string]interface{}{
		myKeys[0]: 101,
		myKeys[1]: "Value 2",
		myKeys[2]: true,
	}

	agent, err := NewAgent(WithApiKey("123"), WithConfiguration(myConfiguration))
	if err != nil {
		t.Fatal(err)
	}
	agent.Stop()

	if agentKeys, ok := agent.metadata[tags.ConfigurationKeys]; ok {
		if !sameElements(myKeys, agentKeys.([]string)) {
			t.Fatal("the configuration keys array are different", agentKeys, myKeys)
		}
	} else {
		t.Fatal("agent configuration keys can't be found")
	}

	for k, v := range myConfiguration {
		if mV, ok := agent.metadata[k]; ok {
			if !reflect.DeepEqual(v, mV) {
				t.Fatal("the configuration values are different")
			}
		} else {
			t.Fatal("the configuration maps are different")
		}
	}
}

func sameElements(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, v := range a {
		found := false
		for _, v2 := range b {
			found = found || v == v2
		}
		if !found {
			return false
		}
	}
	return true
}

func TestTildeExpandRaceMetadata(t *testing.T) {
	env.ScopeSourceRoot.Value = "~/scope"
	agent, err := NewAgent(WithApiKey("123"), WithTestingModeEnabled())
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(5 * time.Second)
	agent.Stop()
}

var a *Agent

func BenchmarkNewAgent(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var err error
		a, err = NewAgent(WithTestingModeEnabled(),
			WithHandlePanicAsFail(),
			WithRetriesOnFail(3),
			WithSetGlobalTracer())
		if err != nil {
			b.Fatal(err)
		}
		span := a.Tracer().StartSpan("Test")
		span.SetTag("span.kind", "test")
		span.SetTag("test.name", "BenchNewAgent")
		span.SetTag("test.suite", "root")
		span.SetTag("test.status", tags.TestStatus_PASS)
		span.SetBaggageItem("trace.kind", "test")
		span.Finish()
		once = sync.Once{}
		a.Stop()
	}
}
