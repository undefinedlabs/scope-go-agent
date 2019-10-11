package agent

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"go.undefinedlabs.com/scopeagent/tags"
)

type gitData struct {
	Repository string
	Commit     string
	SourceRoot string
	Branch     string
}

type gitDiff struct {
	Type    string         `json:"type" msgpack:"type"`
	Version string         `json:"version" msgpack:"version"`
	Uuid    string         `json:"uuid" msgpack:"uuid"`
	Files   []diffFileItem `json:"files" msgpack:"files"`
}

type diffFileItem struct {
	Path         string  `json:"path" msgpack:"path"`
	Added        int     `json:"added" msgpack:"added"`
	Removed      int     `json:"removed" msgpack:"removed"`
	Status       string  `json:"status" msgpack:"status"`
	PreviousPath *string `json:"previousPath" msgpack:"previousPath"`
}

// Gets the current git data
func getGitData() *gitData {
	var repository, commit, sourceRoot, branch string

	if repoBytes, err := exec.Command("git", "remote", "get-url", "origin").Output(); err == nil {
		repository = strings.TrimSuffix(string(repoBytes), "\n")
	}

	if commitBytes, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		commit = strings.TrimSuffix(string(commitBytes), "\n")
	}

	if sourceRootBytes, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		sourceRoot = strings.TrimSuffix(string(sourceRootBytes), "\n")
	}

	if branchBytes, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		branch = strings.TrimSuffix(string(branchBytes), "\n")
	}

	return &gitData{
		Repository: repository,
		Commit:     commit,
		SourceRoot: sourceRoot,
		Branch:     branch,
	}
}

func getGitDiff() *gitDiff {
	var diff string
	if diffBytes, err := exec.Command("git", "diff", "--numstat").Output(); err == nil {
		diff = string(diffBytes)
	} else {
		return nil
	}

	reader := bufio.NewReader(strings.NewReader(diff))
	var files []diffFileItem
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		diffItem := strings.Split(line, "\t")
		added, _ := strconv.Atoi(diffItem[0])
		removed, _ := strconv.Atoi(diffItem[1])
		path := strings.TrimSuffix(diffItem[2], "\n")

		files = append(files, diffFileItem{
			Path:         path,
			Added:        added,
			Removed:      removed,
			Status:       "Modified",
			PreviousPath: nil,
		})
	}

	id, _ := uuid.NewRandom()
	gitDiff := gitDiff{
		Type:    "com.undefinedlabs.ugdsf",
		Version: "0.1.0",
		Uuid:    id.String(),
		Files:   files,
	}
	return &gitDiff
}

func getGitInfoFromGitFolder() map[string]interface{} {
	gitData := getGitData()

	if gitData == nil {
		return nil
	}

	gitInfo := map[string]interface{}{}

	if gitData.Repository != "" {
		gitInfo[tags.Repository] = gitData.Repository
	}
	if gitData.Commit != "" {
		gitInfo[tags.Commit] = gitData.Commit
	}
	if gitData.SourceRoot != "" {
		gitInfo[tags.SourceRoot] = gitData.SourceRoot
	}
	if gitData.Branch != "" {
		gitInfo[tags.Branch] = gitData.Branch
	}

	return gitInfo
}

func getGitInfoFromEnv() map[string]interface{} {
	gitInfo := map[string]interface{}{}

	if repository, set := os.LookupEnv("SCOPE_REPOSITORY"); set {
		gitInfo[tags.Repository] = repository
	}
	if commit, set := os.LookupEnv("SCOPE_COMMIT_SHA"); set {
		gitInfo[tags.Commit] = commit
	}
	if sourceRoot, set := os.LookupEnv("SCOPE_SOURCE_ROOT"); set {
		gitInfo[tags.SourceRoot] = sourceRoot
	}
	if service, set := os.LookupEnv("SCOPE_SERVICE"); set {
		gitInfo[tags.Service] = service
	} else {
		gitInfo[tags.Service] = "default"
	}

	return gitInfo
}
