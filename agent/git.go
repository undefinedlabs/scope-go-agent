package agent

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/src-d/go-git.v4"

	"go.undefinedlabs.com/scopeagent/env"
	"go.undefinedlabs.com/scopeagent/tags"
)

type GitData struct {
	Repository string
	Commit     string
	SourceRoot string
	Branch     string
}

type GitDiff struct {
	Type    string         `json:"type" msgpack:"type"`
	Version string         `json:"version" msgpack:"version"`
	Uuid    string         `json:"uuid" msgpack:"uuid"`
	Files   []DiffFileItem `json:"files" msgpack:"files"`
}
type DiffFileItem struct {
	Path         string  `json:"path" msgpack:"path"`
	Added        int     `json:"added" msgpack:"added"`
	Removed      int     `json:"removed" msgpack:"removed"`
	Status       string  `json:"status" msgpack:"status"`
	PreviousPath *string `json:"previousPath" msgpack:"previousPath"`
}

// Gets the current git data
func getGitData() *GitData {
	var repository, commit, sourceRoot, branch string

	wd, err := os.Getwd()
	if err != nil {
		return nil
	}
	repo, err := git.PlainOpenWithOptions(wd, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil
	}

	if remote, err := repo.Remote("origin"); err == nil {
		urls := remote.Config().URLs
		if len(urls) > 0 {
			repository = urls[0]
		}
	}

	if head, err := repo.Head(); err == nil {
		commit = head.Hash().String()
		branch = head.Name().Short()
	}

	if tree, err := repo.Worktree(); err == nil {
		sourceRoot = tree.Filesystem.Root()
	}

	return &GitData{
		Repository: repository,
		Commit:     commit,
		SourceRoot: sourceRoot,
		Branch:     branch,
	}
}

func getGitDiff() *GitDiff {
	var diff string
	// Git diff with numstat is not supported by "gopkg.in/src-d/go-git.v4"
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
			Path:         path,
			Added:        added,
			Removed:      removed,
			Status:       "Modified",
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

	if repository, set := env.ScopeRepository.Tuple(); set && repository != "" {
		gitInfo[tags.Repository] = repository
	}
	if commit, set := env.ScopeCommitSha.Tuple(); set && commit != "" {
		gitInfo[tags.Commit] = commit
	}
	if sourceRoot, set := env.ScopeSourceRoot.Tuple(); set && sourceRoot != "" {
		// We check if is a valid and existing folder
		if fInfo, err := os.Stat(sourceRoot); err == nil && fInfo.IsDir() {
			gitInfo[tags.SourceRoot] = sourceRoot
		}
	}
	if branch, set := env.ScopeBranch.Tuple(); set && branch != "" {
		gitInfo[tags.Branch] = branch
	}

	return gitInfo
}
