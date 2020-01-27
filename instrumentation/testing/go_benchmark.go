/*
	The purpose with this file is to clone the struct alignment of the testing.B struct so we can assign a *testing.B
	pointer to the *goB to have access to the internal private fields.

	We use this to create a Run clone method to be called from the subtest auto instrumentation
*/

package testing

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

// clone of testing.B struct
type goB struct {
	goCommon
	importPath       string
	context          *goBenchContext
	N                int
	previousN        int
	previousDuration time.Duration
	benchFunc        func(b *testing.B)
	benchTime        goBenchTimeFlag
	bytes            int64
	missingBytes     bool
	timerOn          bool
	showAllocResult  bool
	result           testing.BenchmarkResult
	parallelism      int
	startAllocs      uint64
	startBytes       uint64
	netAllocs        uint64
	netBytes         uint64
	extra            map[string]float64
}

// clone of testing.benchContext struct
type goBenchContext struct {
	match  *goMatcher
	maxLen int
	extLen int
}

// clone of testing.benchTimeFlag struct
type goBenchTimeFlag struct {
	d time.Duration
	n int
}

// Convert *goB to *testing.B
func (b *goB) ToTestingB() *testing.B {
	return *(**testing.B)(unsafe.Pointer(&b))
}

// Convert *testing.B to *goB
func FromTestingB(b *testing.B) *goB {
	return *(**goB)(unsafe.Pointer(&b))
}

//go:linkname benchmarkLock testing.benchmarkLock
var benchmarkLock sync.Mutex

//go:linkname (*goB).run1 testing.(*B).run1
func (b *goB) run1() bool

//go:linkname (*goB).run testing.(*B).run
func (b *goB) run() bool

//go:linkname (*goB).add testing.(*B).add
func (b *goB) add(other testing.BenchmarkResult)

// we clone the same (*testing.B).Run implementation because the Patch
// overwrites the original implementation with the jump
func (b *goB) Run(name string, f func(b *testing.B)) bool {
	atomic.StoreInt32(&b.hasSub, 1)
	benchmarkLock.Unlock()
	defer benchmarkLock.Lock()

	benchName, ok, partial := b.name, true, false
	if b.context != nil {
		benchName, ok, partial = b.context.match.fullName(&b.goCommon, name)
	}
	if !ok {
		return true
	}
	var pc [maxStackLen]uintptr
	n := runtime.Callers(2, pc[:])
	sub := &goB{
		goCommon: goCommon{
			signal:  make(chan bool),
			name:    benchName,
			parent:  &b.goCommon,
			level:   b.level + 1,
			creator: pc[:n],
			w:       b.w,
			chatty:  b.chatty,
		},
		importPath: b.importPath,
		benchFunc:  f,
		benchTime:  b.benchTime,
		context:    b.context,
	}
	if partial {
		atomic.StoreInt32(&sub.hasSub, 1)
	}
	if sub.run1() {
		sub.run()
	}
	b.add(sub.result)
	return !sub.failed
}
