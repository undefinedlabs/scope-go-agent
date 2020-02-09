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
	version            = "0.1.9"
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

	agent.debugMode = agent.debugMode || getBoolEnv("SCOPE_DEBUG", false)

	configProfile := GetConfigCurrentProfile()

	if agent.apiKey == "" || agent.apiEndpoint == "" {
		if dsn, set := os.LookupEnv("SCOPE_DSN"); set && dsn != "" {
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
		if apikey, set := os.LookupEnv("SCOPE_APIKEY"); set && apikey != "" {
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
		if endpoint, set := os.LookupEnv("SCOPE_API_ENDPOINT"); set && endpoint != "" {
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

	// Git data
	addToMapIfEmpty(agent.metadata, getGitInfoFromEnv())
	addToMapIfEmpty(agent.metadata, getCIMetadata())
	addToMapIfEmpty(agent.metadata, getGitInfoFromGitFolder())

	agent.metadata[tags.Diff] = getGitDiff()

	agent.metadata[tags.InContainer] = isRunningInContainer()

	// Dependencies
	agent.metadata[tags.Dependencies] = getDependencyMap()

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
		// Log the error in the current span
		OnSpanFinishPanic: scopeError.LogErrorInRawSpan,
	})

	instrumentation.SetTracer(agent.tracer)
	instrumentation.SetLogger(agent.logger)

	if agent.setGlobalTracer || getBoolEnv("SCOPE_SET_GLOBAL_TRACER", false) {
		opentracing.SetGlobalTracer(agent.Tracer())
	}
	if agent.failRetriesCount == 0 {
		agent.failRetriesCount = getIntEnv("SCOPE_TESTING_FAIL_RETRIES", agent.failRetriesCount)
	}
	agent.panicAsFail = agent.panicAsFail || getBoolEnv("SCOPE_TESTING_PANIC_AS_FAIL", false)

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
	err, _ := a.recorder.SendSpans()
	return err
}

func generateAgentID() string {
	agentId, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	return agentId.String()
}

func getLogPath() (string, error) {
	if logPath, set := os.LookupEnv("SCOPE_LOG_ROOT_PATH"); set {
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
		isOk := true
		// If folder doesn't exist we try to create it
		if _, err := os.Stat(logFolder); err != nil {
			if os.IsNotExist(err) {
				mkErr := os.Mkdir(logFolder, os.ModeDir)
				if mkErr != nil {
					isOk = false
				}
			} else {
				isOk = false
			}
		}
		if isOk {
			return logFolder, nil
		}
	}

	// If the log folder can't be used we return a temporal path, so we don't miss the agent logs
	dir := path.Join(os.TempDir(), "scope-logs")
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			mkErr := os.Mkdir(dir, os.ModeDir)
			if mkErr != nil {
				return "", mkErr
			}
		} else {
			return "", err
		}
	}
	return dir, nil
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
