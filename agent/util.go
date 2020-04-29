package agent

import (
	"bytes"
	"compress/gzip"
	"github.com/vmihailenco/msgpack"
	"os"
)

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

func getSourceRootFromEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		// We check if is a valid and existing folder
		if fInfo, err := os.Stat(value); err == nil && fInfo.IsDir() {
			return value
		}
	}
	return ""
}

// Encodes `payload` using msgpack and compress it with gzip
func msgPackEncodePayload(payload map[string]interface{}) (*bytes.Buffer, error) {
	binaryPayload, err := msgpack.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err = zw.Write(binaryPayload)
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	return &buf, nil
}
