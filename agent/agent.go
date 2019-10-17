package agent

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"

	scopeError "go.undefinedlabs.com/scopeagent/errors"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/tags"
	"go.undefinedlabs.com/scopeagent/tracer"
)

type (
	Agent struct {
		tracer opentracing.Tracer

		apiEndpoint string
		apiKey      string

		agentId         string
		version         string
		metadata        map[string]interface{}
		debugMode       bool
		testingMode     bool
		setGlobalTracer bool

		recorder         *SpanRecorder
		recorderFilename string

		logger *log.Logger
	}

	Option func(*Agent)
)

var (
	version            = "0.1.4"
	defaultApiEndpoint = "https://app.scope.dev"

	printReportOnce sync.Once

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

func WithSetGlobalTracer() Option {
	return func(agent *Agent) {
		agent.setGlobalTracer = true
	}
}

func WithMetadata(values map[string]interface{}) Option {
	return func(agent *Agent) {
		for k, v := range values {
			agent.metadata[k] = v
		}
	}
}

func WithGitInfo(repository string, commitSha string, sourceRoot string) Option {
	return func(agent *Agent) {
		agent.metadata[tags.Repository] = repository
		agent.metadata[tags.Commit] = commitSha
		agent.metadata[tags.SourceRoot] = sourceRoot
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

	if err := agent.setupLogging(); err != nil {
		agent.logger = log.New(ioutil.Discard, "", 0)
	}

	agent.debugMode = agent.debugMode || getBoolEnv("SCOPE_DEBUG", false)

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
	addToMapIfEmpty(agent.metadata, getGitInfoFromEnv())
	addToMapIfEmpty(agent.metadata, getCIMetadata())
	addToMapIfEmpty(agent.metadata, getGitInfoFromGitFolder())

	agent.metadata[tags.Diff] = getGitDiff()

	agent.metadata[tags.InContainer] = isRunningInContainer()

	agent.recorder = NewSpanRecorder(agent)

	if _, set := os.LookupEnv("SCOPE_TESTING_MODE"); set {
		agent.testingMode = getBoolEnv("SCOPE_TESTING_MODE", false)
	} else {
		agent.testingMode = agent.testingMode || agent.metadata[tags.CI].(bool)
	}
	agent.SetTestingMode(agent.testingMode)

	agent.tracer = tracer.NewWithOptions(tracer.Options{
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

	instrumentation.SetTracer(agent.tracer)
	instrumentation.SetLogger(agent.logger)

	if agent.setGlobalTracer || getBoolEnv("SCOPE_SET_GLOBAL_TRACER", false) {
		opentracing.SetGlobalTracer(agent.Tracer())
	}

	return agent, nil
}

func (a *Agent) setupLogging() error {
	filename := fmt.Sprintf("scope-go-%s-%s.log", time.Now().Format("20060102150405"), a.agentId)
	dir, err := ioutil.TempDir("", "scope")
	if err != nil {
		return err
	}
	a.recorderFilename = path.Join(dir, filename)

	file, err := os.OpenFile(a.recorderFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	a.logger = log.New(file, "", log.LstdFlags|log.Lshortfile)
	return nil
}

func (a *Agent) SetTestingMode(enabled bool) {
	a.testingMode = enabled
	if a.testingMode {
		a.recorder.ChangeFlushFrequency(testingModeFrequency)
	} else {
		a.recorder.ChangeFlushFrequency(nonTestingModeFrequency)
	}
}

func (a *Agent) Tracer() opentracing.Tracer {
	return a.tracer
}

func (a *Agent) Logger() *log.Logger {
	return a.logger
}

// Stops the agent
func (a *Agent) Stop() {
	if a.debugMode {
		a.logger.Println("Scope agent is stopping gracefully...")
	}
	a.recorder.t.Kill(nil)
	_ = a.recorder.t.Wait()

	a.PrintReport()
}

// Flushes the pending payloads to the scope backend
func (a *Agent) Flush() error {
	if a.debugMode {
		a.logger.Println("Scope agent is flushing all pending spans manually")
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
