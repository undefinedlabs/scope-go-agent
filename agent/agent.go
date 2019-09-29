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
	"testing"
)

type Agent struct {
	Tracer opentracing.Tracer

	scopeEndpoint string
	apiKey        string

	agentId     string
	version     string
	metadata    map[string]interface{}
	debugMode   bool
	testingMode bool
	recorder    *SpanRecorder
}

var (
	version            = "0.1.0"
	defaultApiEndpoint = "https://app.scope.dev"

	printReportOnce sync.Once
	gitDataOnce     sync.Once
	gitData         *GitData
)

// Creates a new Scope Agent instance
func NewAgent() (*Agent, error) {
	var apiKey string
	configProfile := GetConfigCurrentProfile()
	if apikey, set := os.LookupEnv("SCOPE_APIKEY"); set && apikey != "" {
		apiKey = apikey
	} else if configProfile != nil {
		apiKey = configProfile.ApiKey
	} else {
		return nil, errors.New("Scope API key could not be autodetected")
	}

	a := new(Agent)
	a.apiKey = apiKey

	if endpoint, set := os.LookupEnv("SCOPE_API_ENDPOINT"); set && endpoint != "" {
		a.scopeEndpoint = endpoint
	} else if configProfile != nil {
		a.scopeEndpoint = configProfile.ApiEndpoint
	} else {
		a.scopeEndpoint = defaultApiEndpoint
	}

	a.debugMode = GetBoolEnv("SCOPE_DEBUG", false)
	a.version = version
	a.agentId = generateAgentID()

	a.metadata = make(map[string]interface{})

	// Agent data
	a.metadata[tags.AgentID] = a.agentId
	a.metadata[tags.AgentVersion] = version
	a.metadata[tags.AgentType] = "go"

	// Platform data
	a.metadata[tags.PlatformName] = runtime.GOOS
	a.metadata[tags.PlatformArchitecture] = runtime.GOARCH
	if runtime.GOARCH == "amd64" {
		a.metadata[tags.ProcessArchitecture] = "X64"
	} else if runtime.GOARCH == "386" {
		a.metadata[tags.ProcessArchitecture] = "X86"
	} else if runtime.GOARCH == "arm" {
		a.metadata[tags.ProcessArchitecture] = "Arm"
	} else if runtime.GOARCH == "arm64" {
		a.metadata[tags.ProcessArchitecture] = "Arm64"
	}

	// Current folder
	wd, _ := os.Getwd()
	a.metadata[tags.CurrentFolder] = wd

	// Hostname
	hostname, _ := os.Hostname()
	a.metadata[tags.Hostname] = hostname

	// Go version
	a.metadata[tags.GoVersion] = runtime.Version()

	// Git data
	autodetectCI(a)
	if repository, set := os.LookupEnv("SCOPE_REPOSITORY"); set {
		a.metadata[tags.Repository] = repository
	}
	if commit, set := os.LookupEnv("SCOPE_COMMIT_SHA"); set {
		a.metadata[tags.Commit] = commit
	}
	if sourceRoot, set := os.LookupEnv("SCOPE_SOURCE_ROOT"); set {
		a.metadata[tags.SourceRoot] = sourceRoot
	}
	if service, set := os.LookupEnv("SCOPE_SERVICE"); set {
		a.metadata[tags.Service] = service
	} else {
		a.metadata[tags.Service] = "default"
	}

	// Fallback to git command
	fillFromGitIfEmpty(a)
	a.metadata[tags.Diff] = GetGitDiff()
	a.metadata[tags.InContainer] = isRunningInContainer()

	a.testingMode = GetBoolEnv("SCOPE_TESTING_MODE", true)
	a.recorder = NewSpanRecorder(a)
	a.Tracer = tracer.NewWithOptions(tracer.Options{
		Recorder: a.recorder,
		ShouldSample: func(traceID uint64) bool {
			return true
		},
		MaxLogsPerSpan: 10000,
		OnSpanFinishPanic: func(rSpan *tracer.RawSpan, r interface{}) {
			// Log the error in the current span
			scopeError.LogErrorInRawSpan(rSpan, r)
		},
	})
	return a, nil
}

// Runs a test suite using the agent
func (a *Agent) Run(m *testing.M) int {
	result := m.Run()
	a.Stop()
	return result
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
