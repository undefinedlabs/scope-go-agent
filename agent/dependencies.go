package agent

import (
	"os/exec"
	"regexp"
)

var re = regexp.MustCompile(`(?mi)([a-z./0-9\-_+]*)@([a-z./0-9\-_+]*)$`)

// Gets the dependencies map
func getDependencyMap() map[string]string {
	dependencies := map[string]string{}
	if modGraphBytes, err := exec.Command("go", "mod", "graph").Output(); err == nil {
		strGraph := string(modGraphBytes)
		for _, match := range re.FindAllStringSubmatch(strGraph, -1) {
			dependencies[match[1]] = match[2]
		}
	}
	return dependencies
}
