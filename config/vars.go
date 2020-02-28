package config

import (
	env "github.com/undefinedlabs/go-env"
)

// Current scope configuration
var Current = loadConfig()

func loadConfig() *ScopeConfig {
	var config ScopeConfig
	//yaml.Unmarshal()
	env.UnmarshalFromEnviron(&config)

	return &config
}
