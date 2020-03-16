package agent

import "testing"

func TestRemoteRegex(t *testing.T) {
	tests := map[string]string{
		"[remote \"origin\"]":      "origin",
		"[remote \"upstream\"]":    "upstream",
		"[remote \"a-b-c\"]":       "a-b-c",
		"[remote \"a.b.c\"]":       "a.b.c",
		"[remote \"a_b_c\"]":       "a_b_c",
		"[remote \"a\\b\\c\"]":     "a\\b\\c",
		"[remote \"a$b$c\"]":       "a$b$c",
		"[remote  \"a$b$c\"]":      "a$b$c",
		"[remote  \"a/b/c\"]":      "a/b/c",
		"[remote  \"a b c\"]":      "a b c",
		"[remote    \"a b c\"   ]": "a b c",
	}
	for k, v := range tests {
		match := remoteRegex.FindStringSubmatch(k)
		if match == nil {
			t.Fatalf("match for '%s' is nil", k)
		}
		if match[1] != v {
			t.Fatalf("match for '%s' is '%s', but '%s' was expected", k, match[1], v)
		}
	}
}

func TestBranchRegex(t *testing.T) {
	tests := map[string]string{
		"[branch \"origin\"]":         "origin",
		"[branch \"upstream\"]":       "upstream",
		"[branch \"a-b-c\"]":          "a-b-c",
		"[branch \"a.b.c\"]":          "a.b.c",
		"[branch \"a_b_c\"]":          "a_b_c",
		"[branch \"a\\b\\c\"]":        "a\\b\\c",
		"[branch \"a$b$c\"]":          "a$b$c",
		"[branch  \"a$b$c\"]":         "a$b$c",
		"[branch  \"a/b/c\"]":         "a/b/c",
		"[branch  \"a b c\"]":         "a b c",
		"[branch      \"a b c\"    ]": "a b c",
	}
	for k, v := range tests {
		match := branchRegex.FindStringSubmatch(k)
		if match == nil {
			t.Fatalf("match for '%s' is nil", k)
		}
		if match[1] != v {
			t.Fatalf("match for '%s' is '%s', but '%s' was expected", k, match[1], v)
		}
	}
}

func TestUrlRegex(t *testing.T) {
	tests := map[string]string{
		"	url = git@github.com:undefinedlabs/scope-go-agent.git": "git@github.com:undefinedlabs/scope-go-agent.git",
		"url = git@github.com:undefinedlabs/scope-go-agent.git":  "git@github.com:undefinedlabs/scope-go-agent.git",
		"url =  git@github.com:undefinedlabs/scope-go-agent.git": "git@github.com:undefinedlabs/scope-go-agent.git",
		"	url =    git@github.com:undefinedlabs/scope-go-agent.git": "git@github.com:undefinedlabs/scope-go-agent.git",
		"url=git@github.com:undefinedlabs/scope-go-agent.git":    "git@github.com:undefinedlabs/scope-go-agent.git",
		"url   =git@github.com:undefinedlabs/scope-go-agent.git": "git@github.com:undefinedlabs/scope-go-agent.git",
		"	url = a.b_c:d|e-f%g$h(i)/j\\k": "a.b_c:d|e-f%g$h(i)/j\\k",
	}
	for k, v := range tests {
		match := urlRegex.FindStringSubmatch(k)
		if match == nil {
			t.Fatalf("match for '%s' is nil", k)
		}
		if match[1] != v {
			t.Fatalf("match for '%s' is '%s', but '%s' was expected", k, match[1], v)
		}
	}
}

func TestMergeRegex(t *testing.T) {
	tests := map[string]string{
		"	merge = refs/heads/no-module-package-name": "refs/heads/no-module-package-name",
		"merge = refs/heads/no-module-package-name":           "refs/heads/no-module-package-name",
		"merge       =     refs/heads/no-module-package-name": "refs/heads/no-module-package-name",
		"	merge = a.b_c:d|e-f%g$h(i)/j\\k": "a.b_c:d|e-f%g$h(i)/j\\k",
	}
	for k, v := range tests {
		match := mergeRegex.FindStringSubmatch(k)
		if match == nil {
			t.Fatalf("match for '%s' is nil", k)
		}
		if match[1] != v {
			t.Fatalf("match for '%s' is '%s', but '%s' was expected", k, match[1], v)
		}
	}
}
