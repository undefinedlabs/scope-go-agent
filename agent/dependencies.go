package agent

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

type (
	dependency struct {
		Path     string
		Version  string
		Main     bool
		Indirect bool
	}
)

var re = regexp.MustCompile(`(?mi)([A-Za-z./0-9\-_+]*) ([A-Za-z./0-9\-_+]*)$`)

// Gets the dependencies map
func getDependencyMap() map[string]string {
	deps := map[string][]string{}
	if modGraphBytes, err := exec.Command("go", "list", "-m", "-json", "all").Output(); err == nil {
		depsItems := bytes.Split(modGraphBytes, []byte("}"))
		for _, depItem := range depsItems {
			depItem = append(depItem, byte('}'))
			var depJson dependency
			if err := json.Unmarshal(depItem, &depJson); err == nil {
				if depJson.Main || depJson.Indirect {
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
		sort.Strings(v)
		dependencies[k] = strings.Join(v, ", ")
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
