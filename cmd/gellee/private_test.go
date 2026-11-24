// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateStatusPrintsJSON(t *testing.T) {
	for k, v := range map[string]string{
		"GIT_CONFIG_GLOBAL": "/dev/null", "GIT_CONFIG_SYSTEM": "/dev/null",
	} {
		t.Setenv(k, v)
	}
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(base, "state")
	top := filepath.Join(base, "projects", "alpha", "fern")
	for _, d := range []string{stateDir, top} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(stateDir, "config"):      "roots=alpha\n",
		filepath.Join(top, ".gitignore"):       "*.private.md\n",
		filepath.Join(top, "notes.private.md"): "notes\n",
	}
	for p, text := range files {
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.Command("git", "-C", top, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	var out, errw bytes.Buffer
	code := run([]string{"private", "status", "--state", stateDir,
		"--projects", filepath.Join(base, "projects"), "--dir", top, "--json"}, &out, &errw)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	var got struct {
		Key     string   `json:"key"`
		Ready   bool     `json:"ready"`
		Unswept []string `json:"unswept"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	if got.Key != "alpha/fern" || got.Ready || strings.Join(got.Unswept, ",") != "notes.private.md" {
		t.Fatalf("got %+v", got)
	}
}

func TestPrivateOutsideTheRootsFails(t *testing.T) {
	base := t.TempDir()
	stateDir := filepath.Join(base, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "config"), []byte("roots=alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errw bytes.Buffer
	if code := run([]string{"private", "link", "--state", stateDir, "--projects", base, "--dir", base}, &out, &errw); code != 3 {
		t.Fatalf("exit %d, want 3: %s", code, errw.String())
	}
}
