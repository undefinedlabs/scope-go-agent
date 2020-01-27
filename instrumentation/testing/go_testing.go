/*
	The purpose with this file is to clone the struct alignment of the testing.T struct so we can assign a *testing.T
	pointer to the *goT to have access to the internal private fields.

	We use this to create a Run clone method to be called from the subtest auto instrumentation
*/
package testing

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

// clone of testing.T struct
type goT struct {
	goCommon
	isParallel bool
	context    *goTestContext
}

// clone of testing.testContext struct
type goTestContext struct {
	match         *goMatcher
	mu            sync.Mutex
	startParallel chan bool
	running       int
	numWaiting    int
	maxParallel   int
}

// clone of testing.matcher struct
type goMatcher struct {
	filter    []string
	matchFunc func(pat, str string) (bool, error)
	mu        sync.Mutex
	subNames  map[string]int64
}

// clone of testing.indenter struct
type goIndenter struct {
	c *goCommon
}

// Convert *goT to *testing.T
func (t *goT) ToTestingT() *testing.T {
	return *(**testing.T)(unsafe.Pointer(&t))
}

// Convert *testing.T to *goT
func FromTestingT(t *testing.T) *goT {
	return *(**goT)(unsafe.Pointer(&t))
}

const maxStackLen = 50

//go:linkname matchMutex testing.matchMutex
var matchMutex sync.Mutex

//go:linkname goTRunner testing.tRunner
func goTRunner(t *testing.T, fn func(t *testing.T))

//go:linkname rewrite testing.rewrite
func rewrite(s string) string

//go:linkname shouldFailFast testing.shouldFailFast
func shouldFailFast() bool

//go:linkname (*goMatcher).fullName testing.(*matcher).fullName
func (m *goMatcher) fullName(c *goCommon, subname string) (name string, ok, partial bool)

// this method calls the original testing.tRunner by converting *goT to *testing.T
func tRunner(t *goT, fn func(t *goT)) {
	goTRunner(t.ToTestingT(), func(t *testing.T) { fn(FromTestingT(t)) })
}

// we clone the same (*testing.T).Run implementation because the Patch
// overwrites the original implementation with the jump
func (t *goT) Run(name string, f func(t *goT)) bool {
	atomic.StoreInt32(&t.hasSub, 1)
	testName, ok, _ := t.context.match.fullName(&t.goCommon, name)
	if !ok || shouldFailFast() {
		return true
	}
	var pc [maxStackLen]uintptr
	n := runtime.Callers(2, pc[:])
	t = &goT{
		goCommon: goCommon{
			barrier: make(chan bool),
			signal:  make(chan bool),
			name:    testName,
			parent:  &t.goCommon,
			level:   t.level + 1,
			creator: pc[:n],
			chatty:  t.chatty,
		},
		context: t.context,
	}
	t.w = goIndenter{&t.goCommon}

	if t.chatty {
		root := t.parent
		for ; root.parent != nil; root = root.parent {
		}
		root.mu.Lock()
		fmt.Fprintf(root.w, "=== RUN   %s\n", t.name)
		root.mu.Unlock()
	}
	go tRunner(t, f)
	if !<-t.signal {
		runtime.Goexit()
	}
	return !t.failed
}

// we can't link an instance method without a struct pointer
func (w goIndenter) Write(b []byte) (n int, err error) {
	n = len(b)
	for len(b) > 0 {
		end := bytes.IndexByte(b, '\n')
		if end == -1 {
			end = len(b)
		} else {
			end++
		}
		const indent = "    "
		w.c.output = append(w.c.output, indent...)
		w.c.output = append(w.c.output, b[:end]...)
		b = b[end:]
	}
	return
}
