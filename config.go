package scopeagent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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
	homeDir, _ := os.UserHomeDir()
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
