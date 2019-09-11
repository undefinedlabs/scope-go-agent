package scopeagent // import "go.undefinedlabs.com/scopeagent"

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	scopeError "go.undefinedlabs.com/scopeagent/errors"
	"go.undefinedlabs.com/scopeagent/tracer"
	"os"
	"runtime"
	"strconv"
	"sync"
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
	GlobalAgent *Agent

	version = "0.1.0-dev"

	once        sync.Once
	gitDataOnce sync.Once
	gitData     *GitData
)

func init() {
	once.Do(func() {
		GlobalAgent = NewAgent()

		if getBoolEnv("SCOPE_SET_GLOBAL_TRACER", true) {
			opentracing.SetGlobalTracer(GlobalAgent.Tracer)
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
	configProfile := GetConfigCurrentProfile()

	if endpoint, set := os.LookupEnv("SCOPE_API_ENDPOINT"); set && endpoint != "" {
		a.scopeEndpoint = endpoint
	} else if configProfile != nil {
		a.scopeEndpoint = configProfile.ApiEndpoint
	} else {
		panic(errors.New("Api Endpoint is missing"))
	}

	if apikey, set := os.LookupEnv("SCOPE_APIKEY"); set && apikey != "" {
		a.apiKey = apikey
	} else if configProfile != nil {
		a.apiKey = configProfile.ApiKey
	} else {
		panic(errors.New("Api Key is missing"))
	}

	a.debugMode = getBoolEnv("SCOPE_DEBUG", false)
	a.version = version
	a.agentId = generateAgentID()

	a.metadata = make(map[string]interface{})

	// Agent data
	a.metadata[AgentID] = a.agentId
	a.metadata[AgentVersion] = version
	a.metadata[AgentType] = "go"

	// Platform data
	a.metadata[PlatformName] = runtime.GOOS
	a.metadata[PlatformArchitecture] = runtime.GOARCH
	if runtime.GOARCH == "amd64" {
		a.metadata[ProcessArchitecture] = "X64"
	} else if runtime.GOARCH == "386" {
		a.metadata[ProcessArchitecture] = "X86"
	} else if runtime.GOARCH == "arm" {
		a.metadata[ProcessArchitecture] = "Arm"
	} else if runtime.GOARCH == "arm64" {
		a.metadata[ProcessArchitecture] = "Arm64"
	}

	// Current folder
	wd, _ := os.Getwd()
	a.metadata[CurrentFolder] = wd

	// Hostname
	hostname, _ := os.Hostname()
	a.metadata[Hostname] = hostname

	// Go version
	a.metadata[GoVersion] = runtime.Version()

	// Git data
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

	// Failback to git command
	fillFromGitIfEmpty(a)
	a.metadata[Diff] = GetGitDiff()

	a.testingMode = getBoolEnv("SCOPE_TESTING_MODE", true)
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
	return a
}

func (a *Agent) Stop() {
	if a.debugMode {
		fmt.Println("Scope agent is stopping gracefully...")
	}
	a.recorder.t.Kill(nil)
	_ = a.recorder.t.Wait()

	if a.testingMode && a.recorder.totalSend > 0 {
		if a.recorder.koSend == 0 {
			fmt.Printf("\n** Scope Test Report **\n\n")
			fmt.Println("Access the detailed test report for this build at:")
			fmt.Printf("   %s/external/v1/results/%s\n\n", a.scopeEndpoint, a.agentId)
		} else if a.recorder.koSend < a.recorder.totalSend {
			fmt.Printf("\n** Scope Test Report **\n\n")
			fmt.Println("There was a problem sending data to Scope, partial test report for this build at:")
			fmt.Printf("   %s/external/v1/results/%s\n\n", a.scopeEndpoint, a.agentId)
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "\n** Scope Test Report **\n\n")
			_, _ = fmt.Fprintf(os.Stderr, "There was a problem sending data to Scope\n")
		}
	}
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

func getGitData() *GitData {
	gitDataOnce.Do(func() {
		gitData = GetCurrentGitData()
	})
	return gitData
}

func fillFromGitIfEmpty(a *Agent) {
	if a.metadata[Repository] == nil || a.metadata[Repository] == "" {
		if git := getGitData(); git != nil {
			a.metadata[Repository] = git.Repository
		}
	}
	if a.metadata[Commit] == nil || a.metadata[Commit] == "" {
		if git := getGitData(); git != nil {
			a.metadata[Commit] = git.Commit
		}
	}
	if a.metadata[SourceRoot] == nil || a.metadata[SourceRoot] == "" {
		if git := getGitData(); git != nil {
			a.metadata[SourceRoot] = git.SourceRoot
		}
	}
	if a.metadata[Branch] == nil || a.metadata[Branch] == "" {
		if git := getGitData(); git != nil {
			a.metadata[Branch] = git.Branch
		}
	}
}
