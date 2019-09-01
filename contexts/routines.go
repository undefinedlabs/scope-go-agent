package contexts

import (
	"runtime"
	"strconv"
	"strings"
)

type GoRoutineInfo struct {
	// Go routine id
	Id 					int
	// Calculated parent go routine id
	ParentId			int
	// Current running go routine
	Current				bool
	// Name of the parent func of the go routine
	parentFuncName		string
	// Stack trace funcs
	stackFuncs			[]string
}

// Gets the current go routine stack map
func GetGoRoutineStackMap() (map[int]*GoRoutineInfo, int) {
	stackMap := map[int]*GoRoutineInfo{}
	sMap := getRoutineStackData()
	for _, item := range sMap {
		stackMap[item.Id] = item
	}
	return stackMap, sMap[0].Id
}

// Creates all go routines info and relationship based on the stacktrace data
func getRoutineStackData() []*GoRoutineInfo {
	stackSlice := make([]byte, 8120)
	s := runtime.Stack(stackSlice, true)
	stackString := string(stackSlice[0:s])

	routines := make([]*GoRoutineInfo, 0)
	var currentRoutine *GoRoutineInfo

	stackArray := strings.Split(stackString, "\n")
	for _, line := range stackArray {
		if strings.Index(line, ".go:") > -1 {
			continue
		}
		if strings.Index(line, "goroutine ") > -1 {
			sArr := strings.Split(line, " ")
			goId, _ := strconv.Atoi(sArr[1])
			if currentRoutine != nil {
				routines = append(routines, currentRoutine)
			}
			currentRoutine = &GoRoutineInfo{
				Id:             goId,
				ParentId:       0,
				Current:		currentRoutine == nil,
				parentFuncName: "",
				stackFuncs:     make([]string, 0),
			}
		} else if strings.Index(line, "created by") > -1 {
			sArr := strings.Split(line, " ")
			currentRoutine.parentFuncName = sArr[2]
		} else if line != "" {
			currentRoutine.stackFuncs = append(currentRoutine.stackFuncs, line)
		}
	}
	if currentRoutine != nil {
		routines = append(routines, currentRoutine)
	}

	for _, item := range routines {
		if item.parentFuncName == "" {
			continue
		}
		for _, parent := range routines {
			for _, stackLine := range parent.stackFuncs {
				if strings.Contains(stackLine, item.parentFuncName) {
					item.ParentId = parent.Id
					break
				}
			}
			if item.ParentId > 0 {
				break;
			}
		}
	}

	return routines
}