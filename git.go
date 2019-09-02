package scopeagent

import (
	"bufio"
	"github.com/google/uuid"
	"os/exec"
	"strconv"
	"strings"
)

type GitData struct {
	Repository		string
	Commit			string
	SourceRoot		string
	Branch			string
}

type GitDiff struct {
	Type			string	`json:"type" msgpack:"type"`
	Version			string  `json:"version" msgpack:"version"`
	Uuid			string  `json:"uuid" msgpack:"uuid"`
	Files			[]DiffFileItem  `json:"files" msgpack:"files"`
}
type DiffFileItem struct {
	Path			string  `json:"path" msgpack:"path"`
	Added			int		`json:"added" msgpack:"added"`
	Removed 		int		`json:"removed" msgpack:"removed"`
	Status			string  `json:"status" msgpack:"status"`
	PreviousPath	*string  `json:"previousPath" msgpack:"previousPath"`
}

// Gets the current git data
func GetCurrentGitData() *GitData {
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

	return &GitData{
		Repository: repository,
		Commit:     commit,
		SourceRoot: sourceRoot,
		Branch:     branch,
	}
}

func GetGitDiff() *GitDiff {
	var diff string
	if diffBytes, err := exec.Command("git", "diff", "--numstat").Output(); err == nil {
		diff = string(diffBytes)
	} else {
		return nil
	}

	reader := bufio.NewReader(strings.NewReader(diff))
	var files []DiffFileItem
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		diffItem := strings.Split(line, "\t")
		added, _ := strconv.Atoi(diffItem[0])
		removed, _ := strconv.Atoi(diffItem[1])
		path := strings.TrimSuffix(diffItem[2], "\n")

		files = append(files, DiffFileItem{
			Path:    path,
			Added:   added,
			Removed: removed,
			Status:  "Modified",
			PreviousPath: nil,
		})
	}

	id, _ := uuid.NewRandom()
	gitDiff := GitDiff{
		Type:    "com.undefinedlabs.ugdsf",
		Version: "0.1.0",
		Uuid:    id.String(),
		Files:   files,
	}
	return &gitDiff
}
