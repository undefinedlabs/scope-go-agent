package agent

func addToMapIfEmpty(dest map[string]interface{}, source map[string]interface{}) {
	if source == nil {
		return
	}
	for k, newValue := range source {
		if oldValue, ok := dest[k]; !ok || oldValue == "" {
			dest[k] = newValue
		}
	}
}

func addElementToMapIfEmpty(source map[string]interface{}, key string, value interface{}) {
	if val, ok := source[key]; !ok || val == "" {
		source[key] = value
	}
}
