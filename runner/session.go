package runner

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
)

type (
	testRunnerSession struct {
		Tests []testItem  `json:"tests,omitempty"`
		Rules runnerRules `json:"rules,omitempty"`
	}
	testItem struct {
		Fqn                        string       `json:"fqn"`
		Skip                       bool         `json:"skip,omitempty"`
		RetryOnFailure             bool         `json:"retryOnFailure,omitempty"`
		IncludeStatusInTestResults bool         `json:"includeStatusInTestResults,omitempty"`
		Rules                      *runnerRules `json:"rules,omitempty"`
	}
	runnerRules struct {
		FailRetries  int  `json:"failRetries,omitempty"`
		PassRetries  int  `json:"passRetries,omitempty"`
		ErrorRetries int  `json:"errorRetries,omitempty"`
		ExitOnFail   bool `json:"exitOnFail,omitempty"`
		ExitOnError  bool `json:"exitOnError,omitempty"`
	}

	SessionLoader interface {
		// Load session configuration
		LoadSessionConfiguration(repository string, branch string, commit string, serviceName string) *testRunnerSession
	}
)

type dummySessionLoader struct{}

func (l *dummySessionLoader) LoadSessionConfiguration(repository string, branch string, commit string, serviceName string) *testRunnerSession {
	session := &testRunnerSession{
		Tests: []testItem{
			{
				Fqn:                        "go.undefinedlabs.com/scopeagent.TestFirstTest",
				Skip:                       false,
				RetryOnFailure:             true,
				IncludeStatusInTestResults: true,
				Rules: &runnerRules{
					FailRetries:  0,
					PassRetries:  0,
					ErrorRetries: 0,
					ExitOnError:  false,
				},
			},
			{
				Fqn:                        "go.undefinedlabs.com/scopeagent.TestSkipped",
				Skip:                       true,
				RetryOnFailure:             true,
				IncludeStatusInTestResults: true,
			},
			{
				Fqn:                        "go.undefinedlabs.com/scopeagent.TestFlaky",
				Skip:                       false,
				RetryOnFailure:             true,
				IncludeStatusInTestResults: false,
			},
			{
				Fqn:                        "go.undefinedlabs.com/scopeagent.TestFail",
				Skip:                       false,
				RetryOnFailure:             true,
				IncludeStatusInTestResults: false,
				Rules: &runnerRules{
					FailRetries:  4,
					PassRetries:  0,
					ErrorRetries: 0,
					ExitOnError:  false,
				},
			},
			{
				Fqn:                        "go.undefinedlabs.com/scopeagent.TestError",
				Skip:                       false,
				RetryOnFailure:             true,
				IncludeStatusInTestResults: false,
			},
		},
		Rules: runnerRules{
			FailRetries:  3,
			PassRetries:  1,
			ErrorRetries: 1,
			ExitOnFail:   false,
			ExitOnError:  false,
		},
	}
	return session
}

type fileSessionLoader struct{}

func (l *fileSessionLoader) LoadSessionConfiguration(repository string, branch string, commit string, serviceName string) *testRunnerSession {
	if file, err := os.OpenFile("session.json", os.O_RDONLY, os.ModeType); err == nil {
		if bytes, err := ioutil.ReadAll(bufio.NewReader(file)); err == nil {
			session := &testRunnerSession{}
			json.Unmarshal(bytes, session)
			return session
		}
	}
	return nil
}
