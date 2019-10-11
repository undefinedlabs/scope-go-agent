package agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"runtime"
)

type config struct {
	CurrentProfile string             `json:"currentProfile"`
	Profiles       map[string]profile `json:"profiles"`
}

type profile struct {
	ApiEndpoint string `json:"apiEndpoint"`
	ApiKey      string `json:"apiKey"`
	OAuthToken  string `json:"oauthToken"`
}

func getDesktopConfig() (*config, error) {
	currentUser, _ := user.Current()
	homeDir := currentUser.HomeDir
	var filePath string
	if runtime.GOOS == "windows" {
		filePath = fmt.Sprintf("%s/AppData/Roaming/scope/config.json", homeDir)
	} else {
		filePath = fmt.Sprintf("%s/.scope/config.json", homeDir)
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	fileBytes, _ := ioutil.ReadAll(file)
	var config config
	if err = json.Unmarshal(fileBytes, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *config) getCurrentProfile() *profile {
	if c.Profiles != nil && c.CurrentProfile != "" {
		profile := c.Profiles[c.CurrentProfile]
		return &profile
	}
	return nil
}
