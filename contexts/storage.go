package contexts

var (
	routineData map[int]map[string]interface{}
)

// Sets local data to the current go routine
func SetGoRoutineData(key string, value interface{}) {
	if routineData == nil {
		routineData = map[int]map[string]interface{}{}
	}
	_, goId := GetGoRoutineStackMap()
	if routineData[goId] == nil {
		routineData[goId] = map[string]interface{}{}
	}
	routineData[goId][key] = value
}

// Gets local data from the current and parents go routines
func GetGoRoutineData(key string) interface{} {
	if routineData == nil {
		routineData = map[int]map[string]interface{}{}
	}
	stackMap, goId := GetGoRoutineStackMap()
	if routineData[goId] == nil {
		routineData[goId] = map[string]interface{}{}
	}
	value := routineData[goId][key]
	if value == nil {
		currentStack := stackMap[goId]
		for {
			if currentStack == nil || currentStack.ParentId == 0 {
				break
			}
			value = routineData[currentStack.ParentId][key]
			if value != nil {
				break
			}
			currentStack = stackMap[currentStack.ParentId]
		}
	}
	return value
}
