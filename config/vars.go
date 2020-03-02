package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/undefinedlabs/go-env"
)

var (
	current *ScopeConfig
	m       sync.RWMutex
)

func Get() *ScopeConfig {
	// We check is already loaded with a reader lock
	m.RLock()
	if current != nil {
		defer m.RUnlock()
		return current
	}
	m.RUnlock()

	// Is not loaded we block to load it
	m.Lock()
	defer m.Unlock()
	if current != nil {
		return current
	}
	var config ScopeConfig
	content, path, err := readConfigurationFile()
	if err == nil {
		config.ConfigPath = path
		_ = yaml.Unmarshal(content, &config)
		if config.Metadata != nil {
			for k, v := range config.Metadata {
				if str, ok := v.(string); ok {
					config.Metadata[k] = os.ExpandEnv(str)
				}
			}
		}
	} else {
		config.LoadError = err
	}
	_, err = env.UnmarshalFromEnviron(&config)
	if err != nil {
		config.LoadError = err
	}
	current = &config
	return current
}

func readConfigurationFile() ([]byte, *string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	for {
		rel, _ := filepath.Rel("/", dir)
		// Exit the loop once we reach the basePath.
		if rel == "." {
			break
		}

		path := fmt.Sprintf("%v/scope.yml", dir)
		dat, err := ioutil.ReadFile(path)
		if err == nil {
			return dat, &path, nil
		}

		// Going up!
		dir += "/.."
	}
	return nil, nil, errors.New("configuration not found")
}
