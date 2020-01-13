package agent

import (
	"os/exec"
	"strings"

	"go.undefinedlabs.com/scopeagent/tags"
)

// Gets the dependencies map
func getDependencyMap() map[string]string {
	dependencies := map[string]string{}
	if modGraphBytes, err := exec.Command("go", "mod", "graph").Output(); err == nil {
		strGraph := string(modGraphBytes)
		lines := strings.Split(strGraph, "\n")
		for _, v := range lines {
			if len(v) == 0 {
				continue
			}
			lIdx := strings.LastIndex(v, " ") + 1
			arr := strings.Split(v[lIdx:], "@")
			if len(arr) == 2 {
				dependencies[arr[0]] = arr[1]
			}
		}
	}
	return dependencies
}
