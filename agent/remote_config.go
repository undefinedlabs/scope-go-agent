package agent

import (
	"log"
	"net/http"
)

type (
	RemoteConfig struct {
		request     remoteConfigRequest
		response    *remoteConfigResponse
		apiKey      string
		apiEndpoint string
		version     string
		userAgent   string
		debugMode   bool
		url         string
		client      *http.Client
		logger      *log.Logger
	}


	remoteConfigRequest struct {
		Repository   string            `json:"repository" msgpack:"repository"`
		Commit       string            `json:"commit" msgpack:"commit"`
		Service      string            `json:"service" msgpack:"service"`
		Dependencies map[string]string `json:"dependencies" msgpack:"dependencies"`
	}
	remoteConfigResponse struct {
		Cached []CachedTests `json:"cached" msgpack:"cached"`
	}
	CachedTests struct {
		TestSuite string `json:"test_suite" msgpack:"test_suite"`
		TestName  string `json:"test_name" msgpack:"test_name"`
	}
)

func NewRemoteConfig(agent *Agent) *RemoteConfig {
	r := new(RemoteConfig)
	//r.request.Repository = agent.repos
	r.apiEndpoint = agent.apiEndpoint
	r.apiKey = agent.apiKey
	r.version = agent.version
	r.userAgent = agent.userAgent
	r.debugMode = agent.debugMode
	r.logger = agent.logger
	r.url = agent.getUrl("api/agent/config")
	r.client = &http.Client{}
	return r
}
