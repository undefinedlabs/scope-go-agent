package reflection_test

import (
	"sync"
	"sync/atomic"
	"testing"

	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
	"go.undefinedlabs.com/scopeagent/reflection"
)

func TestPanicHandler(t *testing.T) {
	var panicHandlerVisit int32

	reflection.AddPanicHandler(func(e interface{}) {
		t.Log("PANIC HANDLER FOR:", e)
		atomic.AddInt32(&panicHandlerVisit, 1)
	})

	t.Run("OnPanic", func(t2 *testing.T) {
		var wg = new(sync.WaitGroup)
		wg.Add(1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Log("PANIC RECOVERED")
				}
				wg.Done()
			}()

			t.Log("PANICKING!")
			panic("Panic error")

		}()

		wg.Wait()
	})

	if atomic.LoadInt32(&panicHandlerVisit) != 1 {
		t.Fatalf("panic handler should be executed once.")
	}
}
