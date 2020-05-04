package gocheck

import (
	"fmt"

	"go.undefinedlabs.com/scopeagent/reflection"

	chk "gopkg.in/check.v1"
)

func startTest(method *methodType, c *chk.C) {
	fmt.Println("*** Start", c)
}

func endTest(method *methodType, c *chk.C) {
	var status uint32
	fmt.Println("*** End", c)

	r := recover()
	if r != nil {
		status = 3
	} else {
		if ptr, err := reflection.GetFieldPointerOf(c, "_status"); err == nil {
			status = *(*uint32)(ptr)
		}
	}

	if status == 0 {
		fmt.Println("Success")
	} else if status == 1 {
		fmt.Println("Failed")
	} else if status == 2 {
		fmt.Println("Skipped")
	} else if status == 3 {
		fmt.Println("Panicked")
	} else if status == 4 {
		fmt.Println("Fixture Panicked")
	} else if status == 5 {
		fmt.Println("Missed")
	}

	if r != nil {
		panic(r)
	}
}
