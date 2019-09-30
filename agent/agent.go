package agent

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	scopeError "go.undefinedlabs.com/scopeagent/errors"
	"go.undefinedlabs.com/scopeagent/tags"
	"go.undefinedlabs.com/scopeagent/tracer"
	"os"
	"runtime"
	"sync"
	"time"
)

type (
	Agent struct {
		Tracer opentracing.Tracer

		apiEndpoint string
		apiKey      string

		agentId     string
		version     string
		metadata    map[string]interface{}
		debugMode   bool
		testingMode bool
		recorder    *SpanRecorder
	}

	Option func(*Agent)
)

var (
	version            = "0.1.1"
	defaultApiEndpoint = "https://app.scope.dev"

	printReportOnce sync.Once
	gitDataOnce     sync.Once
	gitData         *GitData

	testingModeFrequency    = time.Second
	nonTestingModeFrequency = time.Minute
)

func WithApiKey(apiKey string) Option {
	return func(agent *Agent) {
		agent.apiKey = apiKey
	}
}

func WithApiEndpoint(apiEndpoint string) Option {
	return func(agent *Agent) {
		agent.apiEndpoint = apiEndpoint
	}
}

func WithServiceName(service string) Option {
	return func(agent *Agent) {
		agent.metadata[tags.Service] = service
	}
}

func WithDebugEnabled() Option {
	return func(agent *Agent) {
		agent.debugMode = true
	}
}

func WithTestingModeEnabled() Option {
	return func(agent *Agent) {
		agent.testingMode = true
	}
}

func WithMetadata(values map[string]interface{}) Option {
	return func(agent *Agent) {
		for k, v := range values {
			agent.metadata[k] = v
		}
	}
}

// Creates a new Scope Agent instance
func NewAgent(options ...Option) (*Agent, error) {
	agent := new(Agent)
	agent.metadata = make(map[string]interface{})
	agent.version = version
	agent.agentId = generateAgentID()

	for _, opt := range options {
		opt(agent)
	}

	agent.debugMode = agent.debugMode || GetBoolEnv("SCOPE_DEBUG", false)

	configProfile := GetConfigCurrentProfile()

	if agent.apiKey == "" {
		if apikey, set := os.LookupEnv("SCOPE_APIKEY"); set && apikey != "" {
			agent.apiKey = apikey
		} else if configProfile != nil {
			agent.apiKey = configProfile.ApiKey
		} else {
			return nil, errors.New("Scope API key could not be autodetected")
		}
	}

	if agent.apiEndpoint == "" {
		if endpoint, set := os.LookupEnv("SCOPE_API_ENDPOINT"); set && endpoint != "" {
			agent.apiEndpoint = endpoint
		} else if configProfile != nil {
			agent.apiEndpoint = configProfile.ApiEndpoint
		} else {
			agent.apiEndpoint = defaultApiEndpoint
		}
	}

	// Agent data
	agent.metadata[tags.AgentID] = agent.agentId
	agent.metadata[tags.AgentVersion] = version
	agent.metadata[tags.AgentType] = "go"

	// Platform data
	agent.metadata[tags.PlatformName] = runtime.GOOS
	agent.metadata[tags.PlatformArchitecture] = runtime.GOARCH
	if runtime.GOARCH == "amd64" {
		agent.metadata[tags.ProcessArchitecture] = "X64"
	} else if runtime.GOARCH == "386" {
		agent.metadata[tags.ProcessArchitecture] = "X86"
	} else if runtime.GOARCH == "arm" {
		agent.metadata[tags.ProcessArchitecture] = "Arm"
	} else if runtime.GOARCH == "arm64" {
		agent.metadata[tags.ProcessArchitecture] = "Arm64"
	}

	// Current folder
	wd, _ := os.Getwd()
	agent.metadata[tags.CurrentFolder] = wd

	// Hostname
	hostname, _ := os.Hostname()
	agent.metadata[tags.Hostname] = hostname

	// Go version
	agent.metadata[tags.GoVersion] = runtime.Version()

	// Git data
	autodetectCI(agent)
	if repository, set := os.LookupEnv("SCOPE_REPOSITORY"); set {
		agent.metadata[tags.Repository] = repository
	}
	if commit, set := os.LookupEnv("SCOPE_COMMIT_SHA"); set {
		agent.metadata[tags.Commit] = commit
	}
	if sourceRoot, set := os.LookupEnv("SCOPE_SOURCE_ROOT"); set {
		agent.metadata[tags.SourceRoot] = sourceRoot
	}
	if _, ok := agent.metadata[tags.Service]; !ok {
		if service, set := os.LookupEnv("SCOPE_SERVICE"); set {
			agent.metadata[tags.Service] = service
		} else {
			agent.metadata[tags.Service] = "default"
		}
	}

	// Fallback to git command
	fillFromGitIfEmpty(agent)
	agent.metadata[tags.Diff] = GetGitDiff()
	agent.metadata[tags.InContainer] = isRunningInContainer()

	agent.recorder = NewSpanRecorder(agent)

	if _, exists := os.LookupEnv("SCOPE_TESTING_MODE"); exists {
		agent.testingMode = GetBoolEnv("SCOPE_TESTING_MODE", false)
	} else {
		agent.testingMode = agent.testingMode || agent.metadata[tags.CI].(bool)
	}
	agent.SetTestingMode(agent.testingMode)

	agent.Tracer = tracer.NewWithOptions(tracer.Options{
		Recorder: agent.recorder,
		ShouldSample: func(traceID uint64) bool {
			return true
		},
		MaxLogsPerSpan: 10000,
		OnSpanFinishPanic: func(rSpan *tracer.RawSpan, r interface{}) {
			// Log the error in the current span
			scopeError.LogErrorInRawSpan(rSpan, r)
		},
	})
	return agent, nil
}

func (a *Agent) SetTestingMode(enabled bool) {
	a.testingMode = enabled
	if a.testingMode {
		a.recorder.ChangeFlushFrequency(testingModeFrequency)
	} else {
		a.recorder.ChangeFlushFrequency(nonTestingModeFrequency)
	}
}

func (a *Agent) SetAsGlobalTracer() {
	opentracing.SetGlobalTracer(a.Tracer)
}

// Stops the agent
func (a *Agent) Stop() {
	if a.debugMode {
		fmt.Println("Scope agent is stopping gracefully...")
	}
	a.recorder.t.Kill(nil)
	_ = a.recorder.t.Wait()

	a.PrintReport()
}

// Flushes the pending payloads to the scope backend
func (a *Agent) Flush() error {
	if a.debugMode {
		fmt.Println("Scope agent is flushing all pending spans manually")
	}
	return a.recorder.SendSpans()
}

func generateAgentID() string {
	agentId, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	return agentId.String()
}
