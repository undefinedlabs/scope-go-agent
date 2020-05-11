package agent

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/mitchellh/go-homedir"
)

const cacheTimeout = 5 * time.Minute

type (
	localCache struct {
		m         sync.Mutex
		tenant    interface{}
		basePath  string
		timeout   time.Duration
		debugMode bool
		logger    *log.Logger
	}
	cacheItem struct {
		Value interface{}
	}
)

// Create a new local cache
func newLocalCache(tenant map[string]interface{}, timeout time.Duration, debugMode bool, logger *log.Logger) *localCache {
	lc := &localCache{
		timeout:   timeout,
		debugMode: debugMode,
		logger:    logger,
	}
	lc.SetTenant(tenant)
	return lc
}

// Gets or sets a local cache value
func (c *localCache) GetOrSet(key string, useTimeout bool, fn func(interface{}, string) interface{}) interface{} {
	c.m.Lock()
	defer c.m.Unlock()

	// Loader function
	loaderFunc := func(key string, err error, fn func(interface{}, string) interface{}) interface{} {
		if err != nil {
			c.logger.Printf("Local cache: %v", err)
		}
		if fn == nil {
			return nil
		}

		// Call the loader
		resp := fn(c.tenant, key)

		if resp != nil {
			path := fmt.Sprintf("%s.%s", c.basePath, key)
			cItem := &cacheItem{Value: resp}
			// Save a local cache for the response
			if data, err := json.Marshal(cItem); err == nil {
				if c.debugMode {
					c.logger.Printf("Local cache saving: %s => %s", path, string(data))
				}
				if err := ioutil.WriteFile(path, data, 0755); err != nil {
					c.logger.Printf("Error writing json file: %v", err)
				}
			}
		}

		return resp
	}

	path := fmt.Sprintf("%s.%s", c.basePath, key)

	// We try to load the cached version of the remote configuration
	file, err := os.Open(path)
	if err != nil {
		return loaderFunc(key, err, fn)
	}
	defer file.Close()

	// Checks if the cache data is old
	if useTimeout {
		fInfo, err := file.Stat()
		if err != nil {
			return loaderFunc(key, err, fn)
		}
		sTime := time.Now().Sub(fInfo.ModTime())
		if sTime > cacheTimeout {
			err = errors.New(fmt.Sprintf("The local cache key '%s' has timeout: %v", path, sTime))
			return loaderFunc(key, err, fn)
		}
	}

	// Read the cached value
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return loaderFunc(key, err, fn)
	}

	// Unmarshal json data
	var cItem cacheItem
	if err := json.Unmarshal(fileBytes, &cItem); err != nil {
		return loaderFunc(key, err, fn)
	} else {
		if c.debugMode {
			c.logger.Printf("Local cache loading: %s => %s", path, string(fileBytes))
		} else {
			c.logger.Printf("Local cache loading: %s", path)
		}
		return cItem.Value
	}
}

// Sets the local cache tenant
func (c *localCache) SetTenant(tenant interface{}) {
	homeDir, err := homedir.Dir()
	if err != nil {
		c.logger.Printf("local cache error: %v", err)
		return
	}
	data, err := json.Marshal(tenant)
	if err != nil {
		c.logger.Printf("local cache error: %v", err)
		return
	}
	hash := fmt.Sprintf("%x", sha1.Sum(data))

	var folder string
	if runtime.GOOS == "windows" {
		folder = fmt.Sprintf("%s/AppData/Roaming/scope/cache", homeDir)
	} else {
		folder = fmt.Sprintf("%s/.scope/cache", homeDir)
	}

	if _, err := os.Stat(folder); err == nil {
		c.tenant = tenant
		c.basePath = filepath.Join(folder, hash)
	} else if os.IsNotExist(err) {
		err = os.MkdirAll(folder, 0755)
		if err != nil {
			c.logger.Printf("local cache error: %v", err)
			return
		}
		c.tenant = tenant
		c.basePath = filepath.Join(folder, hash)
	} else {
		c.logger.Printf("local cache error: %v", err)
	}
}
