package agent

import (
	"fmt"
	"go.undefinedlabs.com/scopeagent/tags"
	"os"
)

func autodetectCI(agent *Agent) {
	if _, set := os.LookupEnv("TRAVIS"); set {
		agent.metadata[tags.CI] = true
		agent.metadata[tags.CIProvider] = "Travis"
		agent.metadata[tags.CIBuildId] = os.Getenv("TRAVIS_BUILD_ID")
		agent.metadata[tags.CIBuildNumber] = os.Getenv("TRAVIS_BUILD_NUMBER")
		agent.metadata[tags.CIBuildUrl] = fmt.Sprintf(
			"https://travis-ci.com/%s/builds/%s",
			os.Getenv("TRAVIS_REPO_SLUG"),
			os.Getenv("TRAVIS_BUILD_ID"),
		)
		agent.metadata[tags.Repository] = fmt.Sprintf(
			"https://github.com/%s.git",
			os.Getenv("TRAVIS_REPO_SLUG"),
		)
		agent.metadata[tags.Commit] = os.Getenv("TRAVIS_COMMIT")
		agent.metadata[tags.SourceRoot] = os.Getenv("TRAVIS_BUILD_DIR")
	} else if _, set := os.LookupEnv("CIRCLECI"); set {
		agent.metadata[tags.CI] = true
		agent.metadata[tags.CIProvider] = "CircleCI"
		agent.metadata[tags.CIBuildNumber] = os.Getenv("CIRCLE_BUILD_NUM")
		agent.metadata[tags.CIBuildUrl] = os.Getenv("CIRCLE_BUILD_URL")
		agent.metadata[tags.Repository] = os.Getenv("CIRCLE_REPOSITORY_URL")
		agent.metadata[tags.Commit] = os.Getenv("CIRCLE_SHA1")
		agent.metadata[tags.SourceRoot] = os.Getenv("CIRCLE_WORKING_DIRECTORY")
	} else if _, set := os.LookupEnv("JENKINS_URL"); set {
		agent.metadata[tags.CI] = true
		agent.metadata[tags.CIProvider] = "Jenkins"
		agent.metadata[tags.CIBuildId] = os.Getenv("BUILD_ID")
		agent.metadata[tags.CIBuildNumber] = os.Getenv("BUILD_NUMBER")
		agent.metadata[tags.CIBuildUrl] = os.Getenv("BUILD_URL")
		agent.metadata[tags.Repository] = os.Getenv("GIT_URL")
		agent.metadata[tags.Commit] = os.Getenv("GIT_COMMIT")
		agent.metadata[tags.SourceRoot] = os.Getenv("WORKSPACE")
	} else if _, set := os.LookupEnv("GITLAB_CI"); set {
		agent.metadata[tags.CI] = true
		agent.metadata[tags.CIProvider] = "gitLab"
		agent.metadata[tags.CIBuildId] = os.Getenv("CI_JOB_ID")
		agent.metadata[tags.CIBuildUrl] = os.Getenv("CI_JOB_URL")
		agent.metadata[tags.Repository] = os.Getenv("CI_REPOSITORY_URL")
		agent.metadata[tags.Commit] = os.Getenv("CI_COMMIT_SHA")
		agent.metadata[tags.SourceRoot] = os.Getenv("CI_PROJECT_DIR")
	} else if _, set := os.LookupEnv("APPVEYOR"); set {
		buildId := os.Getenv("APPVEYOR_BUILD_ID")
		agent.metadata[tags.CI] = true
		agent.metadata[tags.CIProvider] = "AppVeyor"
		agent.metadata[tags.CIBuildId] = buildId
		agent.metadata[tags.CIBuildNumber] = os.Getenv("APPVEYOR_BUILD_NUMBER")
		agent.metadata[tags.CIBuildUrl] = fmt.Sprintf(
			"https://ci.appveyor.com/project/%s/builds/%s",
			os.Getenv("APPVEYOR_PROJECT_SLUG"),
			buildId,
		)
		agent.metadata[tags.Repository] = os.Getenv("APPVEYOR_REPO_NAME")
		agent.metadata[tags.Commit] = os.Getenv("APPVEYOR_REPO_COMMIT")
		agent.metadata[tags.SourceRoot] = os.Getenv("APPVEYOR_BUILD_FOLDER")
	} else if _, set := os.LookupEnv("TF_BUILD"); set {
		buildId := os.Getenv("Build.BuildId")
		agent.metadata[tags.CI] = true
		agent.metadata[tags.CIProvider] = "Azure Pipelines"
		agent.metadata[tags.CIBuildId] = buildId
		agent.metadata[tags.CIBuildNumber] = os.Getenv("Build.BuildNumber")
		agent.metadata[tags.CIBuildUrl] = fmt.Sprintf(
			"%s/%s/_build/results?buildId=%s&_a=summary",
			os.Getenv("System.TeamFoundationCollectionUri"),
			os.Getenv("System.TeamProject"),
			buildId,
		)
		agent.metadata[tags.Repository] = os.Getenv("Build.Repository.Uri")
		agent.metadata[tags.Commit] = os.Getenv("Build.SourceVersion")
		agent.metadata[tags.SourceRoot] = os.Getenv("Build.SourcesDirectory")
	} else if sha, set := os.LookupEnv("BITBUCKET_COMMIT"); set {
		agent.metadata[tags.CI] = true
		agent.metadata[tags.CIProvider] = "Bitbucket Pipelines"
		agent.metadata[tags.CIBuildNumber] = os.Getenv("BITBUCKET_BUILD_NUMBER")
		agent.metadata[tags.Repository] = os.Getenv("BITBUCKET_GIT_SSH_ORIGIN")
		agent.metadata[tags.Commit] = sha
		agent.metadata[tags.SourceRoot] = os.Getenv("BITBUCKET_CLONE_DIR")
	} else if sha, set := os.LookupEnv("GITHUB_SHA"); set {
		repo := os.Getenv("GITHUB_REPOSITORY")
		agent.metadata[tags.CI] = true
		agent.metadata[tags.CIProvider] = "GitHub"
		agent.metadata[tags.CIBuildUrl] = fmt.Sprintf(
			"https://github.com/%s/commit/%s/checks",
			repo,
			sha,
		)
		agent.metadata[tags.Repository] = fmt.Sprintf(
			"https://github.com/%s.git",
			repo,
		)
		agent.metadata[tags.Commit] = sha
		agent.metadata[tags.SourceRoot] = os.Getenv("GITHUB_WORKSPACE")
	} else {
		agent.metadata[tags.CI] = false
	}
}
