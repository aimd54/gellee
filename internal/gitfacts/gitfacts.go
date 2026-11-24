// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// Package gitfacts reads the state of working trees with the git
// command.
package gitfacts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aimd54/gellee/internal/state"
)

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...) //nolint:gosec // fixed binary; arguments are a directory and constants
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// Collect reads one working tree. It returns false when dir holds no
// repository.
func Collect(dir string) (state.GitProject, bool) {
	var p state.GitProject
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return p, false
	}
	p.Branch, _ = git(dir, "symbolic-ref", "--short", "HEAD")
	if out, err := git(dir, "log", "-1", "--format=%h%x09%cI"); err == nil {
		p.Head, p.LastCommit, _ = strings.Cut(out, "\t")
	}
	if out, _ := git(dir, "remote"); out != "" {
		p.Remote = true
	}
	out, err := git(dir, "rev-list", "--count", "@{upstream}..HEAD")
	if err != nil {
		// No upstream: every commit on the branch is unpublished.
		out, err = git(dir, "rev-list", "--count", "HEAD")
	}
	if err == nil {
		p.Ahead, _ = strconv.Atoi(out)
	}
	if out, err := git(dir, "status", "--porcelain"); err == nil && out != "" {
		p.Dirty = strings.Count(out, "\n") + 1
	}
	return p, true
}

// CollectAll reads every directory one level below each root.
func CollectAll(projectsRoot string, roots []string) map[string]state.GitProject {
	out := map[string]state.GitProject{}
	for _, root := range roots {
		entries, err := os.ReadDir(filepath.Join(projectsRoot, root))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if p, ok := Collect(filepath.Join(projectsRoot, root, e.Name())); ok {
				out[root+"/"+e.Name()] = p
			}
		}
	}
	return out
}
