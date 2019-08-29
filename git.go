package scopeagent

import (
	"os/exec"
	"strings"
)

type GitData struct {
	Repository		string
	Commit			string
	SourceRoot		string
	Branch			string
}

// Gets the current git data
func GetCurrentGitData() *GitData {
	repoBytes, _ := exec.Command("git", "remote", "get-url", "origin").Output()
	repository := strings.TrimSuffix(string(repoBytes), "\n")

	commitBytes, _ := exec.Command("git", "rev-parse", "HEAD").Output()
	commit := strings.TrimSuffix(string(commitBytes), "\n")

	sourceRootBytes, _ := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	sourceRoot := strings.TrimSuffix(string(sourceRootBytes), "\n")

	branchBytes, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	branch := strings.TrimSuffix(string(branchBytes), "\n")

	return &GitData{
		Repository: repository,
		Commit:     commit,
		SourceRoot: sourceRoot,
		Branch:     branch,
	}
}

