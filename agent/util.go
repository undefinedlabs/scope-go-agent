package agent

import (
	"fmt"
	"os"
	"strconv"
)

func getBoolEnv(key string, fallback bool) bool {
	stringValue, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	value, err := strconv.ParseBool(stringValue)
	if err != nil {
		panic(fmt.Sprintf("unable to parse %s - should be 'true' or 'false'", key))
	}
	return value
}

func addToMapIfEmpty(dest map[string]interface{}, source map[string]interface{}) {
	for k, newValue := range source {
		if oldValue, ok := dest[k]; !ok || oldValue == "" {
			dest[k] = newValue
		}
	}
}
