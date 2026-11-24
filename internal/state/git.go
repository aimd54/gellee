// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// GitProject is what one machine saw in one working tree.
type GitProject struct {
	Branch     string `json:"branch"`
	Head       string `json:"head"`
	LastCommit string `json:"last_commit"`
	Remote     bool   `json:"remote"`
	Ahead      int    `json:"ahead"`
	Dirty      int    `json:"dirty"`
}

// GitFile is git/<machine>.json.
type GitFile struct {
	Collected string                `json:"collected"`
	Projects  map[string]GitProject `json:"projects"`
}

// WriteGitFile replaces git/<machine>.json.
func WriteGitFile(stateDir, machine string, f GitFile) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Join(stateDir, "git")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, machine+".json"), append(data, '\n'), 0o644) //nolint:gosec // committed to a repository, not a secret
}

// GitFiles reads every machine's git file, keyed by machine name.
func GitFiles(stateDir string) (map[string]GitFile, []Problem) {
	entries, problems := list(stateDir, "git")
	out := map[string]GitFile{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var f GitFile
		data, err := os.ReadFile(filepath.Join(stateDir, "git", entry.Name())) //nolint:gosec // a path below the state directory
		if err == nil {
			err = json.Unmarshal(data, &f)
		}
		if err != nil {
			problems = append(problems, Problem{File: "git/" + entry.Name(), Message: err.Error()})
			continue
		}
		out[strings.TrimSuffix(entry.Name(), ".json")] = f
	}
	return out, problems
}
