package scopeagent

import (
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/undefinedlabs/go-agent/tracer"
	"os"
	"sync"
)

type Agent struct {
	scopeEndpoint string
	apiKey        string
	version       string
	metadata      map[string]interface{}

	recorder *SpanRecorder
	tracer   opentracing.Tracer
}

var (
	once        sync.Once
	GlobalAgent *Agent
	version     = "0.1.0-dev"
)

func init() {
	once.Do(func() {
		GlobalAgent = NewAgent()
		opentracing.SetGlobalTracer(GlobalAgent.tracer)
	})
}

func NewAgent() *Agent {
	a := new(Agent)
	a.scopeEndpoint = os.Getenv("SCOPE_API_ENDPOINT")
	a.apiKey = os.Getenv("SCOPE_APIKEY")
	a.version = version

	a.metadata = make(map[string]interface{})
	a.metadata[AgentID] = generateAgentID()
	a.metadata[AgentVersion] = version
	a.metadata[AgentType] = "go"

	autodetectCI(a)
	if repository, set := os.LookupEnv("SCOPE_REPOSITORY"); set {
		a.metadata[Repository] = repository
	}
	if commit, set := os.LookupEnv("SCOPE_COMMIT_SHA"); set {
		a.metadata[Commit] = commit
	}
	if sourceRoot, set := os.LookupEnv("SCOPE_SOURCE_ROOT"); set {
		a.metadata[SourceRoot] = sourceRoot
	}
	if service, set := os.LookupEnv("SCOPE_SERVICE"); set {
		a.metadata[Service] = service
	} else {
		a.metadata[Service] = "default"
	}

	a.recorder = NewSpanRecorder(a)
	a.tracer = tracer.New(a.recorder)
	return a
}

func (a *Agent) Stop() {
	a.recorder.t.Kill(nil)
	_ = a.recorder.t.Wait()
}

func generateAgentID() string {
	agentId, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	return agentId.String()
}

func autodetectCI(agent *Agent) {
	if _, set := os.LookupEnv("CIRCLECI"); set {
		agent.metadata[CI] = true
		agent.metadata[CIProvider] = "CircleCI"
		agent.metadata[CIBuildNumber] = os.Getenv("CIRCLE_BUILD_NUM")
		agent.metadata[CIBuildUrl] = os.Getenv("CIRCLE_BUILD_URL")
		agent.metadata[Repository] = os.Getenv("CIRCLE_REPOSITORY_URL")
		agent.metadata[Commit] = os.Getenv("CIRCLE_SHA1")
		agent.metadata[SourceRoot] = os.Getenv("CIRCLE_WORKING_DIRECTORY")
	}
}
