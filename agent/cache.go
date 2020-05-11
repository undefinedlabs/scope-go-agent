package agent

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mitchellh/go-homedir"
)

const cacheTimeout = 5 * time.Minute

func (a *Agent) getOrSetLocalCacheData(metadata map[string]interface{}, key string, useTimeout bool, fn func(map[string]interface{}) interface{}) interface{} {
	if metadata == nil {
		return nil
	}

	path, err := getLocalCacheFilePath(metadata, key)
	if err != nil {
		a.logger.Printf("Local cache: %v", err)
		return fn(metadata)
	}

	// Loader function
	loaderFunc := func(metadata map[string]interface{}, err error, fn func(map[string]interface{}) interface{}) interface{} {
		if err != nil {
			a.logger.Printf("Local cache: %v", err)
		}
		if fn == nil {
			return nil
		}

		// Call the loader
		resp := fn(metadata)

		if resp != nil {
			// Get local cache file path
			path, err := getLocalCacheFilePath(metadata, key)
			if err != nil {
				return resp
			}

			// Save a local cache for the response
			if data, err := json.Marshal(&resp); err == nil {
				if a.debugMode {
					a.logger.Printf("Local cache saving: %s => %s", path, string(data))
				}
				if err := ioutil.WriteFile(path, data, 0755); err != nil {
					a.logger.Printf("Error writing json file: %v", err)
				}
			}
		}

		return resp
	}

	// We try to load the cached version of the remote configuration
	file, err := os.Open(path)
	if err != nil {
		return loaderFunc(metadata, err, fn)
	}
	defer file.Close()

	// Checks if the cache data is old
	if useTimeout {
		fInfo, err := file.Stat()
		if err != nil {
			return loaderFunc(metadata, err, fn)
		}
		sTime := time.Now().Sub(fInfo.ModTime())
		if sTime > cacheTimeout {
			err = errors.New(fmt.Sprintf("The local cache key '%s' has timeout: %v", path, sTime))
			return loaderFunc(metadata, err, fn)
		}
	}

	// Read the cached value
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return loaderFunc(metadata, err, fn)
	}

	// Unmarshal json data
	var res map[string]interface{}
	if err := json.Unmarshal(fileBytes, &res); err != nil {
		return loaderFunc(metadata, err, fn)
	} else {
		if a.debugMode {
			a.logger.Printf("Local cache loading: %s => %s", path, string(fileBytes))
		} else {
			a.logger.Printf("Local cache loading: %s", path)
		}
		return res
	}
}

// Gets the local cache file path
func getLocalCacheFilePath(metadata map[string]interface{}, key string) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	hash := fmt.Sprintf("%x", sha1.Sum(data))

	var folder string
	if runtime.GOOS == "windows" {
		folder = fmt.Sprintf("%s/AppData/Roaming/scope/cache", homeDir)
	} else {
		folder = fmt.Sprintf("%s/.scope/cache", homeDir)
	}

	if _, err := os.Stat(folder); err == nil {
		return filepath.Join(folder, fmt.Sprintf("%s.%s", hash, key)), nil
	} else if os.IsNotExist(err) {
		err = os.MkdirAll(folder, 0755)
		if err != nil {
			return "", err
		}
		return filepath.Join(folder, fmt.Sprintf("%s.%s", hash, key)), nil
	} else {
		return "", err
	}
}
