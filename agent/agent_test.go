package agent_test

import (
	"fmt"
	"go.undefinedlabs.com/scopeagent"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	_ "unsafe"

	"go.undefinedlabs.com/scopeagent/agent"
	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
	"go.undefinedlabs.com/scopeagent/reflection"
	"go.undefinedlabs.com/scopeagent/tags"
)

//go:linkname getDependencyMap go.undefinedlabs.com/scopeagent/agent.getDependencyMap
func getDependencyMap() map[string]string

//go:linkname parseDSN go.undefinedlabs.com/scopeagent/agent.parseDSN
func parseDSN(dsnString string) (apiKey string, apiEndpoint string, err error)

func getAgentMetadata(agent *agent.Agent) map[string]interface{} {
	if ptr, err := reflection.GetFieldPointerOf(agent, "metadata"); err == nil {
		return *(*map[string]interface{})(ptr)
	}
	return nil
}

//

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

	test := scopeagent.GetTest(t)
	for i := 0; i < len(dsnValues); i++ {
		dsnValue := dsnValues[i]
		test.Run(dsnValue[0], func(st *testing.T) {
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

func TestGetDependencies(t *testing.T) {
	deps := getDependencyMap()
	fmt.Printf("Dependency Map: %v\n", deps)
	fmt.Printf("Number of dependencies got: %d\n", len(deps))
	if modGraphBytes, err := exec.Command("go", "mod", "graph").Output(); err == nil {
		strGraph := string(modGraphBytes)
		lines := strings.Split(strGraph, "\n")
		cDeps := map[string]bool{}
		for _, line := range lines {
			if line == "" {
				continue
			}
			lArray := strings.Split(line, " ")
			depName := strings.Split(lArray[1], "@")[0]
			cDeps[depName] = true
		}
		fmt.Printf("Number of dependencies expected: %d\n", len(cDeps))
		if len(cDeps) != len(deps) {
			t.FailNow()
		}
	} else {
		t.FailNow()
	}
}

func TestWithConfigurationKeys(t *testing.T) {
	myKeys := []string{"ConfigKey01", "ConfigKey02", "ConfigKey03"}

	agent, err := agent.NewAgent(agent.WithApiKey("123"), agent.WithConfigurationKeys(myKeys))
	if err != nil {
		t.Fatal(err)
	}
	agent.Stop()

	if agentKeys, ok := getAgentMetadata(agent)[tags.ConfigurationKeys]; ok {
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

	agent, err := agent.NewAgent(agent.WithApiKey("123"), agent.WithConfiguration(myConfiguration))
	if err != nil {
		t.Fatal(err)
	}
	agent.Stop()

	if agentKeys, ok := getAgentMetadata(agent)[tags.ConfigurationKeys]; ok {
		if !sameElements(myKeys, agentKeys.([]string)) {
			t.Fatal("the configuration keys array are different", agentKeys, myKeys)
		}
	} else {
		t.Fatal("agent configuration keys can't be found")
	}

	for k, v := range myConfiguration {
		if mV, ok := getAgentMetadata(agent)[k]; ok {
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
