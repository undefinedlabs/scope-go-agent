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

func GetGitDiff() *GitDiff {

	sourceRootBytes, _ := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	sourceRoot := strings.TrimSuffix(string(sourceRootBytes), "\n")

	diffBytes, _ := exec.Command("git", "diff", "--numstat").Output()
	diff := string(diffBytes)

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
		files = append(files, DiffFileItem{
			Path:    sourceRoot + "/" + diffItem[2],
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
