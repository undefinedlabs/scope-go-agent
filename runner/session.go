package runner

type (
	testRunnerSession struct {
		Tests []testItem  "json:`tests`"
		Rules runnerRules "json:`rules`"
	}
	testItem struct {
		Suite          string "json:`suite`"
		Name           string "json:`name`"
		Skip           bool   "json:`skip`"
		RetryOnFailure bool   "json:`retryOnFailure`"
	}
	runnerRules struct {
		FailureRetries int "json:`failureRetries`"
		PassRetries    int "json:`passRetries`"
	}

	sessionLoader interface {
		// Load session configuration
		LoadSessionConfiguration(repository string, branch string, commit string, serviceName string) *testRunnerSession
	}

	dummySessionLoader struct{}
)

func (l *dummySessionLoader) LoadSessionConfiguration(repository string, branch string, commit string, serviceName string) *testRunnerSession {
	return &testRunnerSession{
		Tests: []testItem{
			{
				Suite:          "go.undefinedlabs.com/scopeagent/agent",
				Name:           "TestDsnParser",
				Skip:           false,
				RetryOnFailure: true,
			},
		},
		Rules: runnerRules{
			FailureRetries: 4,
			PassRetries:    0,
		},
	}
}
