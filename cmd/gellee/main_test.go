// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWithoutCommandPrintsUsage(t *testing.T) {
	var out, errw bytes.Buffer
	if code := run(nil, &out, &errw); code != 2 {
		t.Fatalf("exit code %d, want 2", code)
	}
	if !strings.Contains(errw.String(), "usage: gellee <command>") {
		t.Fatalf("stderr: %q", errw.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var out, errw bytes.Buffer
	if code := run([]string{"nope"}, &out, &errw); code != 2 {
		t.Fatalf("exit code %d, want 2", code)
	}
	if !strings.Contains(errw.String(), `unknown command "nope"`) {
		t.Fatalf("stderr: %q", errw.String())
	}
}

func TestCollectCommandWritesTheSummaryFiles(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	if err := os.CopyFS(stateDir, os.DirFS("../../tests/fixtures/state")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(stateDir, "usage")); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", stateDir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))
	t.Setenv("GELLEE_PROJECTS_ROOT", "/p")
	t.Setenv("GELLEE_MEMORIES", "")
	var out, errw bytes.Buffer
	code := run([]string{"collect", "--state", stateDir, "--machine", "m7", "--no-commit",
		"--transcripts", "../../tests/fixtures/transcripts"}, &out, &errw)
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errw.String())
	}
	if !strings.Contains(out.String(), "collect: 3 transcripts, 3 parsed, 1 lines skipped, 1 usage files written") {
		t.Fatalf("stdout %q", out.String())
	}
	data, err := os.ReadFile(filepath.Join(stateDir, "usage/m7/2026-11.jsonl"))
	if err != nil || strings.Count(string(data), "\n") != 4 {
		t.Fatalf("usage file: %v\n%s", err, data)
	}
}

func TestCollectCommandNeedsAStateDirectory(t *testing.T) {
	t.Setenv("GELLEE_STATE", "")
	var out, errw bytes.Buffer
	if code := run([]string{"collect"}, &out, &errw); code != 2 ||
		!strings.Contains(errw.String(), "--state or GELLEE_STATE is required") {
		t.Fatalf("code %d stderr %q", code, errw.String())
	}
}

func TestServeRefusesAMissingStateDirectory(t *testing.T) {
	var out, errw bytes.Buffer
	code := run([]string{"serve", "--state", filepath.Join(t.TempDir(), "nope")}, &out, &errw)
	if code != 1 || !strings.Contains(errw.String(), "gellee: config:") {
		t.Fatalf("code %d stderr %q", code, errw.String())
	}
}

func TestServeRejectsABadDate(t *testing.T) {
	var out, errw bytes.Buffer
	code := run([]string{"serve", "--state", "../../tests/fixtures/state", "--today", "soon"}, &out, &errw)
	if code != 2 || !strings.Contains(errw.String(), "--today") {
		t.Fatalf("code %d stderr %q", code, errw.String())
	}
}

func TestCollectCommandReadsTheMemoryStoreFromTheEnvironment(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	if err := os.CopyFS(stateDir, os.DirFS("../../tests/fixtures/state")); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", stateDir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	nowhere := filepath.Join(tmp, "no-store")
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))
	t.Setenv("GELLEE_PROJECTS_ROOT", "/p")
	t.Setenv("GELLEE_MEMORIES", nowhere)
	var out, errw bytes.Buffer
	code := run([]string{"collect", "--state", stateDir, "--machine", "m7", "--no-commit",
		"--transcripts", "../../tests/fixtures/transcripts"}, &out, &errw)
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errw.String())
	}
	if !strings.Contains(out.String(), "collect: memory store: "+nowhere+" is not a checkout of the memory store") {
		t.Fatalf("stdout %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(stateDir, "machines/m7.json")); err != nil {
		t.Fatal(err)
	}
}

// An address with an invalid port fails after the demo is written and
// read, so the run shows both the demo's startup and its cleanup.
func TestServeDemoIgnoresTheStateAndRemovesItsDirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	t.Setenv("GELLEE_STATE", filepath.Join(tmp, "nope"))
	var out, errw bytes.Buffer
	code := run([]string{"serve", "--demo", "--today", "2026-11-11", "--listen", "127.0.0.1:-1"}, &out, &errw)
	if code != 1 || !strings.Contains(out.String(), "gellee: serving the demo on http://127.0.0.1:-1") {
		t.Fatalf("code %d stdout %q stderr %q", code, out.String(), errw.String())
	}
	if strings.Contains(errw.String(), "config") || !strings.Contains(errw.String(), "invalid port") {
		t.Fatalf("stderr %q, want the listen error and no state error", errw.String())
	}
	left, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Fatalf("left in TMPDIR: %v", left)
	}
}
