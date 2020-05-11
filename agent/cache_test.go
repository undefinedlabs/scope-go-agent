package agent

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

func getTenant() interface{} {
	return map[string]string{
		"key1": "value1",
		"key2": fmt.Sprintf("%v", time.Now()),
	}
}

func TestLocalCache(t *testing.T) {

	tenant := getTenant()

	cache := newLocalCache(tenant, cacheTimeout, true, log.New(os.Stdout, "", 0))
	loader := false
	result := cache.GetOrSet("MyKey01", false, func(i interface{}, s string) interface{} {
		loader = true
		return "hello world"
	})

	if !loader {
		t.Fatal("loader has not been executed.")
	}
	if result.(string) != "hello world" {
		t.Fatal("result was different than expected.")
	}

	cache2 := newLocalCache(tenant, cacheTimeout, true, log.New(os.Stdout, "", 0))
	loader = false
	for i := 0; i < 10; i++ {
		result = cache2.GetOrSet("MyKey01", false, func(i interface{}, s string) interface{} {
			loader = true
			return "hello world"
		})

		if loader {
			t.Fatal("loader has been executed.")
		}
		if result.(string) != "hello world" {
			t.Fatal("result was different than expected.")
		}
	}
}
