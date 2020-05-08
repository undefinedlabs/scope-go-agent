package agent

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"sort"
	"strings"

	"go.undefinedlabs.com/scopeagent/env"
)

type (
	dependency struct {
		Path     string
		Version  string
		Main     bool
		Indirect bool
	}
)

// Gets the dependencies map
func getDependencyMap() map[string]string {
	deps := map[string][]string{}
	if modGraphBytes, err := exec.Command("go", "list", "-m", "-json", "all").Output(); err == nil {
		lIdx := 0
		remain := modGraphBytes
		for {
			// We have to parse this way because the tool returns multiple object but not in array format
			if len(remain) == 0 {
				break
			}
			lIdx = bytes.IndexByte(remain, '}')
			if lIdx == -1 {
				break
			}
			item := remain[:lIdx+1]
			remain = remain[lIdx+1:]

			var depJson dependency
			if err := json.Unmarshal(item, &depJson); err == nil {
				if depJson.Main {
					continue
				}
				if !env.ScopeDependenciesIndirect.Value && depJson.Indirect {
					continue
				}
				if preValue, ok := deps[depJson.Path]; ok {
					// We can have multiple versions of the same dependency by indirection
					deps[depJson.Path] = unique(append(preValue, depJson.Version))
				} else {
					deps[depJson.Path] = []string{depJson.Version}
				}
			}
		}
	}
	dependencies := map[string]string{}
	for k, v := range deps {
		if len(v) > 0 {
			sort.Strings(v)
			dependencies[k] = strings.Join(v, ", ")
		} else {
			dependencies[k] = v[0]
		}
	}
	return dependencies
}

func unique(slice []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
