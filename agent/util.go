package agent

func addToMapIfEmpty(dest map[string]interface{}, source map[string]interface{}) {
	for k, newValue := range source {
		if oldValue, ok := dest[k]; !ok || oldValue == "" {
			dest[k] = newValue
		}
	}
}
