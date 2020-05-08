package agent

import (
	"fmt"
	"os/exec"
	"testing"
)

var mp map[string]string

func BenchmarkGetDependencyMap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mp = getDependencyMap()
	}
}

func TestGetDependencies(t *testing.T) {
	deps := getDependencyMap()
	fmt.Printf("Dependency Map: %v\n", deps)
	fmt.Printf("Number of dependencies got: %d\n", len(deps))
	if len(deps) == 0 {
		t.FailNow()
	}
	if _, err := exec.Command("go", "list", "-m", "all").Output(); err != nil {
		t.FailNow()
	}
}
