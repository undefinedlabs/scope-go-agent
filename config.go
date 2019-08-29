package scopeagent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
)

type Config struct {
	CurrentProfile	string				`json:"currentProfile"`
	Profiles		map[string]Profile	`json:"profiles"`
}

type Profile struct {
	ApiEndpoint		string  `json:"apiEndpoint"`
	ApiKey			string	`json:"apiKey"`
	OAuthToken		string	`json:"oauthToken"`
}

func GetConfig() *Config {
	homeDir, _ := homeDir()
	filePath := fmt.Sprintf("%s/.scope/config.json", homeDir)
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()
	fileBytes, _ := ioutil.ReadAll(file)
	var config Config
	err =  json.Unmarshal(fileBytes, &config)
	if err != nil {
		return nil
	}
	return &config
}

func GetConfigCurrentProfile() *Profile {
	config := GetConfig()
	if config != nil && config.Profiles != nil && config.CurrentProfile != "" {
		profile := config.Profiles[config.CurrentProfile]
		return &profile
	}
	return nil
}


func homeDir() (string, error) {
	env, enverr := "HOME", "$HOME"
	switch runtime.GOOS {
	case "windows":
		env, enverr = "USERPROFILE", "%userprofile%"
	case "plan9":
		env, enverr = "home", "$home"
	case "nacl", "android":
		return "/", nil
	case "darwin":
		if runtime.GOARCH == "arm" || runtime.GOARCH == "arm64" {
			return "/", nil
		}
	}
	if v := os.Getenv(env); v != "" {
		return v, nil
	}
	return "", errors.New(enverr + " is not defined")
}