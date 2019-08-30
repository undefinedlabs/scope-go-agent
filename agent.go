package scopeagent

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/undefinedlabs/go-agent/tracer"
	"os"
	"strconv"
	"sync"
)

type Agent struct {
	scopeEndpoint string
	apiKey        string
	version       string
	metadata      map[string]interface{}
	debugMode     bool

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

		if getBoolEnv("SCOPE_SET_GLOBAL_TRACER", true) {
			opentracing.SetGlobalTracer(GlobalAgent.tracer)
		}

		if getBoolEnv("SCOPE_AUTO_INSTRUMENT", true) {
			if err := PatchAll(); err != nil {
				panic(err)
			}
		}
	})
}

func NewAgent() *Agent {
	a := new(Agent)
	a.scopeEndpoint = os.Getenv("SCOPE_API_ENDPOINT")
	a.apiKey = os.Getenv("SCOPE_APIKEY")
	a.debugMode = getBoolEnv("SCOPE_DEBUG", false)
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
	a.tracer = tracer.NewWithOptions(tracer.Options{
		Recorder: a.recorder,
		ShouldSample: func(traceID uint64) bool {
			return true
		},
		MaxLogsPerSpan: 10000,
		OnSpanFinishPanic: func(rSpan *tracer.RawSpan, r interface{}) {
			// When a span finish detect a panic, we set the span tag as error
			if rSpan.Tags == nil {
				rSpan.Tags = opentracing.Tags{}
			}
			rSpan.Tags["error"] = true
		},
	})
	return a
}

func (a *Agent) Stop() {
	if a.debugMode {
		fmt.Println("Scope agent is stopping gracefully...")
	}
	a.recorder.t.Kill(nil)
	_ = a.recorder.t.Wait()
}

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

func autodetectCI(agent *Agent) {
	if _, set := os.LookupEnv("TRAVIS"); set {
		agent.metadata[CI] = true
		agent.metadata[CIProvider] = "Travis"
		agent.metadata[CIBuildId] = os.Getenv("TRAVIS_BUILD_ID")
		agent.metadata[CIBuildNumber] = os.Getenv("TRAVIS_BUILD_NUMBER")
		agent.metadata[CIBuildUrl] = fmt.Sprintf(
			"https://travis-ci.com/%s/builds/%s",
			os.Getenv("TRAVIS_REPO_SLUG"),
			os.Getenv("TRAVIS_BUILD_ID"),
		)
		agent.metadata[Repository] = fmt.Sprintf(
			"https://github.com/%s.git",
			os.Getenv("TRAVIS_REPO_SLUG"),
		)
		agent.metadata[Commit] = os.Getenv("TRAVIS_COMMIT")
		agent.metadata[SourceRoot] = os.Getenv("TRAVIS_BUILD_DIR")
	} else if _, set := os.LookupEnv("CIRCLECI"); set {
		agent.metadata[CI] = true
		agent.metadata[CIProvider] = "CircleCI"
		agent.metadata[CIBuildNumber] = os.Getenv("CIRCLE_BUILD_NUM")
		agent.metadata[CIBuildUrl] = os.Getenv("CIRCLE_BUILD_URL")
		agent.metadata[Repository] = os.Getenv("CIRCLE_REPOSITORY_URL")
		agent.metadata[Commit] = os.Getenv("CIRCLE_SHA1")
		agent.metadata[SourceRoot] = os.Getenv("CIRCLE_WORKING_DIRECTORY")
	} else if _, set := os.LookupEnv("JENKINS_URL"); set {
		agent.metadata[CI] = true
		agent.metadata[CIProvider] = "Jenkins"
		agent.metadata[CIBuildId] = os.Getenv("BUILD_ID")
		agent.metadata[CIBuildNumber] = os.Getenv("BUILD_NUMBER")
		agent.metadata[CIBuildUrl] = os.Getenv("BUILD_URL")
		agent.metadata[Repository] = os.Getenv("GIT_URL")
		agent.metadata[Commit] = os.Getenv("GIT_COMMIT")
		agent.metadata[SourceRoot] = os.Getenv("WORKSPACE")
	} else if _, set := os.LookupEnv("GITLAB_CI"); set {
		agent.metadata[CI] = true
		agent.metadata[CIProvider] = "gitLab"
		agent.metadata[CIBuildId] = os.Getenv("CI_JOB_ID")
		agent.metadata[CIBuildUrl] = os.Getenv("CI_JOB_URL")
		agent.metadata[Repository] = os.Getenv("CI_REPOSITORY_URL")
		agent.metadata[Commit] = os.Getenv("CI_COMMIT_SHA")
		agent.metadata[SourceRoot] = os.Getenv("CI_PROJECT_DIR")
	} else if _, set := os.LookupEnv("APPVEYOR"); set {
		buildId := os.Getenv("APPVEYOR_BUILD_ID")
		agent.metadata[CI] = true
		agent.metadata[CIProvider] = "AppVeyor"
		agent.metadata[CIBuildId] = buildId
		agent.metadata[CIBuildNumber] = os.Getenv("APPVEYOR_BUILD_NUMBER")
		agent.metadata[CIBuildUrl] = fmt.Sprintf(
			"https://ci.appveyor.com/project/%s/builds/%s",
			os.Getenv("APPVEYOR_PROJECT_SLUG"),
			buildId,
		)
		agent.metadata[Repository] = os.Getenv("APPVEYOR_REPO_NAME")
		agent.metadata[Commit] = os.Getenv("APPVEYOR_REPO_COMMIT")
		agent.metadata[SourceRoot] = os.Getenv("APPVEYOR_BUILD_FOLDER")
	} else if _, set := os.LookupEnv("TF_BUILD"); set {
		buildId := os.Getenv("Build.BuildId")
		agent.metadata[CI] = true
		agent.metadata[CIProvider] = "Azure Pipelines"
		agent.metadata[CIBuildId] = buildId
		agent.metadata[CIBuildNumber] = os.Getenv("Build.BuildNumber")
		agent.metadata[CIBuildUrl] = fmt.Sprintf(
			"%s/%s/_build/results?buildId=%s&_a=summary",
			os.Getenv("System.TeamFoundationCollectionUri"),
			os.Getenv("System.TeamProject"),
			buildId,
		)
		agent.metadata[Repository] = os.Getenv("Build.Repository.Uri")
		agent.metadata[Commit] = os.Getenv("Build.SourceVersion")
		agent.metadata[SourceRoot] = os.Getenv("Build.SourcesDirectory")
	} else if sha, set := os.LookupEnv("BITBUCKET_COMMIT"); set {
		agent.metadata[CI] = true
		agent.metadata[CIProvider] = "Bitbucket Pipelines"
		agent.metadata[CIBuildNumber] = os.Getenv("BITBUCKET_BUILD_NUMBER")
		agent.metadata[Repository] = os.Getenv("BITBUCKET_GIT_SSH_ORIGIN")
		agent.metadata[Commit] = sha
		agent.metadata[SourceRoot] = os.Getenv("BITBUCKET_CLONE_DIR")
	} else if sha, set := os.LookupEnv("GITHUB_SHA"); set {
		repo := os.Getenv("GITHUB_REPOSITORY")
		agent.metadata[CI] = true
		agent.metadata[CIProvider] = "GitHub"
		agent.metadata[CIBuildUrl] = fmt.Sprintf(
			"https://github.com/%s/commit/%s/checks",
			repo,
			sha,
		)
		agent.metadata[Repository] = fmt.Sprintf(
			"https://github.com/%s.git",
			repo,
		)
		agent.metadata[Commit] = sha
		agent.metadata[SourceRoot] = os.Getenv("GITHUB_WORKSPACE")
	}
}

func getBoolEnv(key string, fallback bool) bool {
	stringValue, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	value, err := strconv.ParseBool(stringValue)
	if err != nil {
		panic(fmt.Sprintf("unable to parse %s - should be 'true' or 'false'", key))
	}
	return value
}
