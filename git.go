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
	Type			string	`json:"type"`
	Version			string  `json:"version"`
	Uuid			string  `json:"uuid"`
	Files			[]DiffFileItem  `json:"files"`
}
type DiffFileItem struct {
	Path			string  `json:"path"`
	Added			int		`json:"added"`
	Removed 		int		`json:"removed"`
	Status			string  `json:"status"`
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
	var sourceRoot, diff string

	if sourceRootBytes, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		sourceRoot = strings.TrimSuffix(string(sourceRootBytes), "\n")
	}

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

		var path string
		if sourceRoot == "" {
			path = diffItem[2]
		} else {
			path = sourceRoot + "/" + diffItem[2]
		}

		files = append(files, DiffFileItem{
			Path:    path,
			Added:   added,
			Removed: removed,
			Status:  "Modified",
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
