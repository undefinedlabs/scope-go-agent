package scopeagent

import (
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/undefinedlabs/go-agent/tracer"
	"os"
	"sync"
)

type Agent struct {
	id      uuid.UUID
	version string

	scopeEndpoint string
	apiKey        string
	service       string
	repository    string
	commit        string
	sourceRoot    string

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
	agentId, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	a.id = agentId
	a.version = version
	a.scopeEndpoint = os.Getenv("SCOPE_API_ENDPOINT")
	a.apiKey = os.Getenv("SCOPE_APIKEY")

	a.repository = os.Getenv("SCOPE_REPOSITORY")
	a.commit = os.Getenv("SCOPE_COMMIT_SHA")
	a.sourceRoot = os.Getenv("SCOPE_SOURCE_ROOT")

	service, set := os.LookupEnv("SCOPE_SERVICE")
	if set {
		a.service = service
	} else {
		a.service = "default"
	}

	a.recorder = NewSpanRecorder(a)
	a.tracer = tracer.New(a.recorder)
	return a
}

func (a *Agent) Stop() {
	a.recorder.t.Kill(nil)
	_ = a.recorder.t.Wait()
}
