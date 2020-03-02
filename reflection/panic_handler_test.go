package reflection

import (
	"fmt"
	"testing"
	"time"
)

func TestPanicHandler(t *testing.T) {
	panicHandlerVisit := 0

	AddPanicHandler(func(e interface{}) {
		fmt.Println("PANIC HANDLER FOR:", e)
		panicHandlerVisit++
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

	if panicHandlerVisit != 1 {
		t.Fatalf("panic handler should be executed once.")
	}
}
