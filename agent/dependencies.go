package agent

import (
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

var re = regexp.MustCompile(`(?mi)([A-Za-z./0-9\-_+]*) ([A-Za-z./0-9\-_+]*)$`)

// Gets the dependencies map
func getDependencyMap() map[string]string {
	deps := map[string][]string{}
	if modGraphBytes, err := exec.Command("go", "list", "-m", "all").Output(); err == nil {
		strGraph := string(modGraphBytes)
		for _, match := range re.FindAllStringSubmatch(strGraph, -1) {
			if preValue, ok := deps[match[1]]; ok {
				// We can have multiple versions of the same dependency by indirection
				deps[match[1]] = unique(append(preValue, match[2]))
			} else {
				deps[match[1]] = []string{match[2]}
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
