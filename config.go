package scopeagent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
)

type Config struct {
	CurrentProfile string             `json:"currentProfile"`
	Profiles       map[string]Profile `json:"profiles"`
}

type Profile struct {
	ApiEndpoint string `json:"apiEndpoint"`
	ApiKey      string `json:"apiKey"`
	OAuthToken  string `json:"oauthToken"`
}

func GetConfig() *Config {
	currentUser, _ := user.Current()
	homeDir := currentUser.HomeDir
	filePath := fmt.Sprintf("%s/.scope/config.json", homeDir)
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()
	fileBytes, _ := ioutil.ReadAll(file)
	var config Config
	if err = json.Unmarshal(fileBytes, &config); err != nil {
		return nil
	}
	return &config
}

func GetConfigCurrentProfile() *Profile {
	if config := GetConfig(); config != nil && config.Profiles != nil && config.CurrentProfile != "" {
		profile := config.Profiles[config.CurrentProfile]
		return &profile
	}
	return nil
}
