package reflection

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestPanicHandler(t *testing.T) {
	var panicHandlerVisit int32

	AddPanicHandler(func(e interface{}) {
		fmt.Println("PANIC HANDLER FOR:", e)
		atomic.AddInt32(&panicHandlerVisit, 1)
	})

	t.Run("OnPanic", func(t *testing.T) {
		go func() {

			defer func() {
				if r := recover(); r != nil {
					fmt.Println("PANIC RECOVERED")
				}
			}()

			fmt.Println("PANICKING!")
			panic("Panic error")

		}()

		time.Sleep(1 * time.Second)
	})

	if atomic.LoadInt32(&panicHandlerVisit) != 1 {
		t.Fatalf("panic handler should be executed once.")
	}
}
