// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Session is one Claude Code session as its machine's collector
// summarised it.
type Session struct {
	ID      string `json:"id"`
	Project string `json:"project"`
	Branch  string `json:"branch"`
	Title   string `json:"title"`
	Remote  string `json:"remote"`
	Started string `json:"started"`
	Last    string `json:"last"`
	Ended   bool   `json:"ended"`
}

// MachineMemories is the memory store as one machine sees it.
type MachineMemories struct {
	Pushed   string `json:"pushed"`
	SetAside int    `json:"set_aside"`
}

// MachineFile is machines/<machine>.json.
type MachineFile struct {
	Collected  string           `json:"collected"`
	ClaudeCode string           `json:"claude_code"`
	Memories   *MachineMemories `json:"memories,omitempty"`
	Sessions   []Session        `json:"sessions"`
}

// WriteMachineFile replaces machines/<machine>.json.
func WriteMachineFile(stateDir, machine string, f MachineFile) error {
	if f.Sessions == nil {
		f.Sessions = []Session{}
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Join(stateDir, "machines")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, machine+".json"), append(data, '\n'), 0o644) //nolint:gosec // committed to a repository, not a secret
}

// ReadMachineFile reads one machine's file. A machine without one reads
// as the zero value.
func ReadMachineFile(stateDir, machine string) (MachineFile, error) {
	var f MachineFile
	data, err := os.ReadFile(filepath.Join(stateDir, "machines", machine+".json")) //nolint:gosec // a path below the state directory
	if errors.Is(err, fs.ErrNotExist) {
		return f, nil
	}
	if err == nil {
		err = json.Unmarshal(data, &f)
	}
	if err != nil {
		return MachineFile{}, fmt.Errorf("machines/%s.json: %w", machine, err)
	}
	return f, nil
}

// MachineFiles reads every machine file, keyed by machine name. Fields
// it does not know are ignored and missing ones read as empty.
func MachineFiles(stateDir string) (map[string]MachineFile, []Problem) {
	entries, problems := list(stateDir, "machines")
	out := map[string]MachineFile{}
	for _, entry := range entries {
		name, ok := strings.CutSuffix(entry.Name(), ".json")
		if !ok {
			continue
		}
		var f MachineFile
		data, err := os.ReadFile(filepath.Join(stateDir, "machines", entry.Name())) //nolint:gosec // a path below the state directory
		if err == nil {
			err = json.Unmarshal(data, &f)
		}
		if err != nil {
			problems = append(problems, Problem{File: "machines/" + entry.Name(), Message: err.Error()})
			continue
		}
		out[name] = f
	}
	return out, problems
}
