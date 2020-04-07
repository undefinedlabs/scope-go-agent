package agent

import "testing"

var mp map[string]string

func BenchmarkGetDependencyMap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mp = getDependencyMap()
	}
}
