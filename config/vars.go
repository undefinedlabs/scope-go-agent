package config

import (
	"io/ioutil"
	"os"
	"sync"

	env "github.com/undefinedlabs/go-env"
	"gopkg.in/yaml.v2"
)

var (
	current *ScopeConfig
	m       sync.RWMutex
)

func Get() *ScopeConfig {
	m.RLock()
	defer m.RUnlock()

	return current
}

func Load(filePath string) error {
	m.Lock()
	defer m.Unlock()

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	var config ScopeConfig
	yamlErr := yaml.Unmarshal(content, &config)
	_, envErr := env.UnmarshalFromEnviron(&config)
	if yamlErr != nil && envErr != nil {
		return envErr
	}
	current = &config
	return nil
}
