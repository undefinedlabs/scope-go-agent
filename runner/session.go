package runner

type (
	testRunnerSession struct {
		Tests []testItem  "json:`tests`"
		Rules runnerRules "json:`rules`"
	}
	testItem struct {
		Fqn                        string       "json:`fqn`"
		Skip                       bool         "json:`skip`"
		RetryOnFailure             bool         "json:`retryOnFailure`"
		IncludeStatusInTestResults bool         "json:`includeStatusInTestResults`"
		Rules                      *runnerRules "json:`rules`"
	}
	runnerRules struct {
		FailRetries  int  "json:`failRetries`"
		PassRetries  int  "json:`passRetries`"
		ErrorRetries int  "json:`errorRetries`"
		ExitOnFail   bool "json:`exitOnFail`"
		ExitOnError  bool "json:`exitOnError`"
	}

	SessionLoader interface {
		// Load session configuration
		LoadSessionConfiguration(repository string, branch string, commit string, serviceName string) *testRunnerSession
	}

	dummySessionLoader struct{}
)

func (l *dummySessionLoader) LoadSessionConfiguration(repository string, branch string, commit string, serviceName string) *testRunnerSession {
	return &testRunnerSession{
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
}
