package agent

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
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
