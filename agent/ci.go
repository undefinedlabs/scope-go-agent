package agent

import (
	"fmt"
	"os"

	"go.undefinedlabs.com/scopeagent/tags"
)

func getCIMetadata() map[string]interface{} {
	ciMetadata := map[string]interface{}{tags.CI: false}

	if _, set := os.LookupEnv("TRAVIS"); set {
		ciMetadata[tags.CI] = true
		ciMetadata[tags.CIProvider] = "Travis"
		ciMetadata[tags.CIBuildId] = os.Getenv("TRAVIS_BUILD_ID")
		ciMetadata[tags.CIBuildNumber] = os.Getenv("TRAVIS_BUILD_NUMBER")
		ciMetadata[tags.CIBuildUrl] = fmt.Sprintf(
			"https://travis-ci.com/%s/builds/%s",
			os.Getenv("TRAVIS_REPO_SLUG"),
			os.Getenv("TRAVIS_BUILD_ID"),
		)
		ciMetadata[tags.Repository] = fmt.Sprintf(
			"https://github.com/%s.git",
			os.Getenv("TRAVIS_REPO_SLUG"),
		)
		ciMetadata[tags.Commit] = os.Getenv("TRAVIS_COMMIT")
		ciMetadata[tags.SourceRoot] = os.Getenv("TRAVIS_BUILD_DIR")
	} else if _, set := os.LookupEnv("CIRCLECI"); set {
		ciMetadata[tags.CI] = true
		ciMetadata[tags.CIProvider] = "CircleCI"
		ciMetadata[tags.CIBuildNumber] = os.Getenv("CIRCLE_BUILD_NUM")
		ciMetadata[tags.CIBuildUrl] = os.Getenv("CIRCLE_BUILD_URL")
		ciMetadata[tags.Repository] = os.Getenv("CIRCLE_REPOSITORY_URL")
		ciMetadata[tags.Commit] = os.Getenv("CIRCLE_SHA1")
		ciMetadata[tags.SourceRoot] = os.Getenv("CIRCLE_WORKING_DIRECTORY")
	} else if _, set := os.LookupEnv("JENKINS_URL"); set {
		ciMetadata[tags.CI] = true
		ciMetadata[tags.CIProvider] = "Jenkins"
		ciMetadata[tags.CIBuildId] = os.Getenv("BUILD_ID")
		ciMetadata[tags.CIBuildNumber] = os.Getenv("BUILD_NUMBER")
		ciMetadata[tags.CIBuildUrl] = os.Getenv("BUILD_URL")
		ciMetadata[tags.Repository] = os.Getenv("GIT_URL")
		ciMetadata[tags.Commit] = os.Getenv("GIT_COMMIT")
		ciMetadata[tags.SourceRoot] = os.Getenv("WORKSPACE")
	} else if _, set := os.LookupEnv("GITLAB_CI"); set {
		ciMetadata[tags.CI] = true
		ciMetadata[tags.CIProvider] = "gitLab"
		ciMetadata[tags.CIBuildId] = os.Getenv("CI_JOB_ID")
		ciMetadata[tags.CIBuildUrl] = os.Getenv("CI_JOB_URL")
		ciMetadata[tags.Repository] = os.Getenv("CI_REPOSITORY_URL")
		ciMetadata[tags.Commit] = os.Getenv("CI_COMMIT_SHA")
		ciMetadata[tags.SourceRoot] = os.Getenv("CI_PROJECT_DIR")
	} else if _, set := os.LookupEnv("APPVEYOR"); set {
		buildId := os.Getenv("APPVEYOR_BUILD_ID")
		ciMetadata[tags.CI] = true
		ciMetadata[tags.CIProvider] = "AppVeyor"
		ciMetadata[tags.CIBuildId] = buildId
		ciMetadata[tags.CIBuildNumber] = os.Getenv("APPVEYOR_BUILD_NUMBER")
		ciMetadata[tags.CIBuildUrl] = fmt.Sprintf(
			"https://ci.appveyor.com/project/%s/builds/%s",
			os.Getenv("APPVEYOR_PROJECT_SLUG"),
			buildId,
		)
		ciMetadata[tags.Repository] = os.Getenv("APPVEYOR_REPO_NAME")
		ciMetadata[tags.Commit] = os.Getenv("APPVEYOR_REPO_COMMIT")
		ciMetadata[tags.SourceRoot] = os.Getenv("APPVEYOR_BUILD_FOLDER")
	} else if _, set := os.LookupEnv("TF_BUILD"); set {
		buildId := os.Getenv("Build.BuildId")
		ciMetadata[tags.CI] = true
		ciMetadata[tags.CIProvider] = "Azure Pipelines"
		ciMetadata[tags.CIBuildId] = buildId
		ciMetadata[tags.CIBuildNumber] = os.Getenv("Build.BuildNumber")
		ciMetadata[tags.CIBuildUrl] = fmt.Sprintf(
			"%s/%s/_build/results?buildId=%s&_a=summary",
			os.Getenv("System.TeamFoundationCollectionUri"),
			os.Getenv("System.TeamProject"),
			buildId,
		)
		ciMetadata[tags.Repository] = os.Getenv("Build.Repository.Uri")
		ciMetadata[tags.Commit] = os.Getenv("Build.SourceVersion")
		ciMetadata[tags.SourceRoot] = os.Getenv("Build.SourcesDirectory")
	} else if sha, set := os.LookupEnv("BITBUCKET_COMMIT"); set {
		ciMetadata[tags.CI] = true
		ciMetadata[tags.CIProvider] = "Bitbucket Pipelines"
		ciMetadata[tags.CIBuildNumber] = os.Getenv("BITBUCKET_BUILD_NUMBER")
		ciMetadata[tags.Repository] = os.Getenv("BITBUCKET_GIT_SSH_ORIGIN")
		ciMetadata[tags.Commit] = sha
		ciMetadata[tags.SourceRoot] = os.Getenv("BITBUCKET_CLONE_DIR")
	} else if sha, set := os.LookupEnv("GITHUB_SHA"); set {
		repo := os.Getenv("GITHUB_REPOSITORY")
		ciMetadata[tags.CI] = true
		ciMetadata[tags.CIProvider] = "GitHub"
		ciMetadata[tags.CIBuildUrl] = fmt.Sprintf(
			"https://github.com/%s/commit/%s/checks",
			repo,
			sha,
		)
		ciMetadata[tags.Repository] = fmt.Sprintf(
			"https://github.com/%s.git",
			repo,
		)
		ciMetadata[tags.Commit] = sha
		ciMetadata[tags.SourceRoot] = os.Getenv("GITHUB_WORKSPACE")
	} else if _, set := os.LookupEnv("TEAMCITY_VERSION"); set {
		buildId := os.Getenv("BUILD_ID")
		ciMetadata[tags.CI] = true
		ciMetadata[tags.CIProvider] = "TeamCity"
		ciMetadata[tags.Repository] = os.Getenv("BUILD_VCS_URL")
		ciMetadata[tags.Commit] = os.Getenv("BUILD_VCS_NUMBER")
		ciMetadata[tags.SourceRoot] = os.Getenv("BUILD_CHECKOUTDIR")
		ciMetadata[tags.CIBuildId] = buildId
		ciMetadata[tags.CIBuildNumber] = os.Getenv("BUILD_NUMBER")
		ciMetadata[tags.CIBuildUrl] = fmt.Sprintf(
			"%s/viewLog.html?buildId=%s",
			os.Getenv("SERVER_URL"),
			buildId,
		)
	} else if _, set := os.LookupEnv("BUILDKITE"); set {
		ciMetadata[tags.CI] = true
		ciMetadata[tags.CIProvider] = "Buildkite"
		ciMetadata[tags.CIBuildId] = os.Getenv("BUILDKITE_BUILD_ID")
		ciMetadata[tags.CIBuildNumber] = os.Getenv("BUILDKITE_BUILD_NUMBER")
		ciMetadata[tags.CIBuildUrl] = os.Getenv("BUILDKITE_BUILD_URL")
		ciMetadata[tags.Repository] = os.Getenv("BUILDKITE_REPO")
		ciMetadata[tags.Commit] = os.Getenv("BUILDKITE_COMMIT")
		ciMetadata[tags.SourceRoot] = os.Getenv("BUILDKITE_BUILD_CHECKOUT_PATH")
	}

	return ciMetadata
}
