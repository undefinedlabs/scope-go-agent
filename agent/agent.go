package agent

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/user"
	"path"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"

	"go.undefinedlabs.com/scopeagent/env"
	scopeError "go.undefinedlabs.com/scopeagent/errors"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/runner"
	"go.undefinedlabs.com/scopeagent/tags"
	"go.undefinedlabs.com/scopeagent/tracer"
)

type (
	Agent struct {
		tracer opentracing.Tracer

		apiEndpoint string
		apiKey      string

		agentId          string
		version          string
		metadata         map[string]interface{}
		debugMode        bool
		testingMode      bool
		setGlobalTracer  bool
		panicAsFail      bool
		failRetriesCount int

		recorder         *SpanRecorder
		recorderFilename string

		userAgent string
		agentType string

		logger *log.Logger
	}

	Option func(*Agent)
)

var (
	version            = "0.1.11"
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

func WithUserAgent(userAgent string) Option {
	return func(agent *Agent) {
		userAgent = strings.TrimSpace(userAgent)
		if userAgent != "" {
			agent.userAgent = userAgent
		}
	}
}

func WithAgentType(agentType string) Option {
	return func(agent *Agent) {
		agentType = strings.TrimSpace(agentType)
		if agentType != "" {
			agent.agentType = agentType
		}
	}
}

func WithConfigurationKeys(keys []string) Option {
	return func(agent *Agent) {
		if keys != nil && len(keys) > 0 {
			agent.metadata[tags.ConfigurationKeys] = keys
		}
	}
}

func WithConfiguration(values map[string]interface{}) Option {
	return func(agent *Agent) {
		if values == nil {
			return
		}
		var keys []string
		for k, v := range values {
			agent.metadata[k] = v
			keys = append(keys, k)
		}
		agent.metadata[tags.ConfigurationKeys] = keys
	}
}

func WithRetriesOnFail(retriesCount int) Option {
	return func(agent *Agent) {
		agent.failRetriesCount = retriesCount
	}
}

func WithHandlePanicAsFail() Option {
	return func(agent *Agent) {
		agent.panicAsFail = true
	}
}

// Creates a new Scope Agent instance
func NewAgent(options ...Option) (*Agent, error) {
	agent := new(Agent)
	agent.metadata = make(map[string]interface{})
	agent.version = version
	agent.agentId = generateAgentID()
	agent.userAgent = fmt.Sprintf("scope-agent-go/%s", agent.version)
	agent.panicAsFail = false
	agent.failRetriesCount = 0

	for _, opt := range options {
		opt(agent)
	}

	if err := agent.setupLogging(); err != nil {
		agent.logger = log.New(ioutil.Discard, "", 0)
	}

	agent.debugMode = env.IfFalse(agent.debugMode, env.SCOPE_DEBUG, false)

	configProfile := GetConfigCurrentProfile()

	if agent.apiKey == "" || agent.apiEndpoint == "" {
		if dsn, set := env.SCOPE_DSN.AsTuple(); set && dsn != "" {
			dsnApiKey, dsnApiEndpoint, dsnErr := parseDSN(dsn)
			if dsnErr != nil {
				agent.logger.Printf("Error parsing dsn value: %v\n", dsnErr)
			} else {
				agent.apiKey = dsnApiKey
				agent.apiEndpoint = dsnApiEndpoint
			}
		} else {
			agent.logger.Println("environment variable $SCOPE_DSN not found")
		}
	}

	if agent.apiKey == "" {
		if apikey, set := env.SCOPE_APIKEY.AsTuple(); set && apikey != "" {
			agent.apiKey = apikey
		} else if configProfile != nil {
			agent.logger.Println("API key found in the native app configuration")
			agent.apiKey = configProfile.ApiKey
		} else {
			agent.logger.Println("API key not found, agent can't be started")
			return nil, errors.New(fmt.Sprintf("There was a problem initializing Scope.\n"+
				"Check the agent logs at %s for more information.\n", agent.recorderFilename))
		}
	}

	if agent.apiEndpoint == "" {
		if endpoint, set := env.SCOPE_API_ENDPOINT.AsTuple(); set && endpoint != "" {
			agent.apiEndpoint = endpoint
		} else if configProfile != nil {
			agent.logger.Println("API endpoint found in the native app configuration")
			agent.apiEndpoint = configProfile.ApiEndpoint
		} else {
			agent.logger.Printf("using default endpoint: %v\n", defaultApiEndpoint)
			agent.apiEndpoint = defaultApiEndpoint
		}
	}

	// Agent data
	if agent.agentType == "" {
		agent.agentType = "go"
	}
	agent.metadata[tags.AgentID] = agent.agentId
	agent.metadata[tags.AgentVersion] = version
	agent.metadata[tags.AgentType] = agent.agentType
	agent.metadata[tags.TestingMode] = agent.testingMode

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

	// Service name
	env.AddStringToMapIfEmpty(agent.metadata, tags.Service, env.SCOPE_SERVICE, "default")

	// Git data
	addToMapIfEmpty(agent.metadata, getGitInfoFromEnv())
	addToMapIfEmpty(agent.metadata, getCIMetadata())
	addToMapIfEmpty(agent.metadata, getGitInfoFromGitFolder())

	agent.metadata[tags.Diff] = getGitDiff()

	agent.metadata[tags.InContainer] = isRunningInContainer()

	// Dependencies
	agent.metadata[tags.Dependencies] = getDependencyMap()

	agent.recorder = NewSpanRecorder(agent)

	agent.testingMode = env.IfFalse(agent.testingMode, env.SCOPE_TESTING_MODE, agent.metadata[tags.CI].(bool))

	agent.SetTestingMode(agent.testingMode)

	agent.tracer = tracer.NewWithOptions(tracer.Options{
		Recorder: agent.recorder,
		ShouldSample: func(traceID uint64) bool {
			return true
		},
		MaxLogsPerSpan: 10000,
		// Log the error in the current span
		OnSpanFinishPanic: scopeError.LogErrorInRawSpan,
	})

	instrumentation.SetTracer(agent.tracer)
	instrumentation.SetLogger(agent.logger)

	if env.IfFalse(agent.setGlobalTracer, env.SCOPE_SET_GLOBAL_TRACER, false) {
		opentracing.SetGlobalTracer(agent.Tracer())
	}
	agent.failRetriesCount = env.IfIntZero(agent.failRetriesCount, env.SCOPE_TESTING_FAIL_RETRIES, 0)
	agent.panicAsFail = env.IfFalse(agent.panicAsFail, env.SCOPE_TESTING_PANIC_AS_FAIL, false)
	if agent.debugMode {
		agent.logMetadata()
	}
	return agent, nil
}

func (a *Agent) setupLogging() error {
	filename := fmt.Sprintf("scope-go-%s-%s.log", time.Now().Format("20060102150405"), a.agentId)
	dir, err := getLogPath()
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

// Runs the test suite
func (a *Agent) Run(m *testing.M) int {
	defer a.Stop()
	if a.panicAsFail || a.failRetriesCount > 0 {
		return runner.Run(m, a.panicAsFail, a.failRetriesCount, a.logger)
	}
	return m.Run()
}

// Stops the agent
func (a *Agent) Stop() {
	a.logger.Println("Scope agent is stopping gracefully...")
	a.recorder.Stop()
	a.PrintReport()
}

func generateAgentID() string {
	agentId, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	return agentId.String()
}

func getLogPath() (string, error) {
	if logPath, set := env.SCOPE_LOG_ROOT_PATH.AsTuple(); set {
		return logPath, nil
	}

	logFolder := ""
	if runtime.GOOS == "linux" {
		logFolder = "/var/log/scope"
	} else {
		currentUser, err := user.Current()
		if err != nil {
			return "", err
		}
		homeDir := currentUser.HomeDir
		if runtime.GOOS == "windows" {
			logFolder = fmt.Sprintf("%s/AppData/Roaming/scope/logs", homeDir)
		} else if runtime.GOOS == "darwin" {
			logFolder = fmt.Sprintf("%s/Library/Logs/Scope", homeDir)
		}
	}

	if logFolder != "" {
		if _, err := os.Stat(logFolder); err == nil {
			return logFolder, nil
		} else if os.IsNotExist(err) && os.Mkdir(logFolder, os.ModeDir) == nil {
			return logFolder, nil
		}
	}

	// If the log folder can't be used we return a temporal path, so we don't miss the agent logs
	logFolder = path.Join(os.TempDir(), "scope")
	if _, err := os.Stat(logFolder); err == nil {
		return logFolder, nil
	} else if os.IsNotExist(err) && os.Mkdir(logFolder, os.ModeDir) == nil {
		return logFolder, nil
	} else {
		return "", err
	}
}

func parseDSN(dsnString string) (apiKey string, apiEndpoint string, err error) {
	uri, err := url.Parse(dsnString)
	if err != nil {
		return "", "", err
	}
	if uri.User != nil {
		apiKey = uri.User.Username()
	}
	uri.User = nil
	apiEndpoint = uri.String()
	return
}

func (a *Agent) getUrl(pathValue string) string {
	uri, err := url.Parse(a.apiEndpoint)
	if err != nil {
		a.logger.Fatal(err)
	}
	uri.Path = path.Join(uri.Path, pathValue)
	return uri.String()
}
