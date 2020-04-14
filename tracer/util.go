package tracer

import (
	"runtime"
	"time"

	"go.undefinedlabs.com/scopeagent/rand"
)

var (
	randompool = rand.NewPool(time.Now().UnixNano(), uint64(max(16, runtime.NumCPU())))
)

// max returns the larger value among a and b
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func randomID() uint64 {
	return uint64(randompool.Pick().Int63())
}

func randomID2() (uint64, uint64) {
	n1, n2 := randompool.Pick().TwoInt63()
	return uint64(n1), uint64(n2)
}
