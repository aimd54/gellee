// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package collect

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aimd54/gellee/internal/state"
)

const fixtures = "../../tests/fixtures"

func gitEnv(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"GIT_CONFIG_GLOBAL": "/dev/null", "GIT_CONFIG_SYSTEM": "/dev/null",
		"GIT_AUTHOR_NAME": "test", "GIT_AUTHOR_EMAIL": "test@example.invalid",
		"GIT_COMMITTER_NAME": "test", "GIT_COMMITTER_EMAIL": "test@example.invalid",
	} {
		t.Setenv(k, v)
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// world builds a state repository with the fixture status files, a copy
// of the fixture transcripts, and an empty projects root at /p as far as
// the transcripts are concerned.
func world(t *testing.T) Options {
	t.Helper()
	gitEnv(t)
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	if err := os.CopyFS(stateDir, os.DirFS(filepath.Join(fixtures, "state"))); err != nil {
		t.Fatal(err)
	}
	for _, generated := range []string{"usage", "git", "machines"} {
		if err := os.RemoveAll(filepath.Join(stateDir, generated)); err != nil {
			t.Fatal(err)
		}
	}
	git(t, stateDir, "init", "-q", "-b", "main")
	git(t, stateDir, "add", ".")
	git(t, stateDir, "commit", "-q", "-m", "init")
	transcripts := filepath.Join(tmp, "transcripts")
	if err := os.CopyFS(transcripts, os.DirFS(filepath.Join(fixtures, "transcripts"))); err != nil {
		t.Fatal(err)
	}
	return Options{
		StateDir: stateDir, TranscriptsDir: transcripts, CacheDir: filepath.Join(tmp, "cache"),
		Machine: "m1", ProjectsRoot: "/p", Commit: true,
		Now: time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC), Loc: time.UTC,
		Out:  io.Discard,
		Open: func(p string) (io.ReadCloser, error) { return os.Open(p) },
	}
}

func rowsOf(t *testing.T, o Options) []state.Row {
	t.Helper()
	rows, err := state.MachineRows(o.StateDir, o.Machine)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestRunWritesMergedRenamedRows(t *testing.T) {
	o := world(t)
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	want := []state.Row{
		// The subagent file adds 100/200/300/400 and one message to s1.
		{Day: "2026-11-10", Session: "s1", Project: "alpha/fern", Model: "model-a", Input: 111, Output: 302, CacheWrite: 1303, CacheRead: 10404, Messages: 3},
		{Day: "2026-11-10", Session: "s1", Project: "beta/tool", Model: "model-b", Input: 5, Output: 50, CacheWrite: 500, CacheRead: 5000, Messages: 1},
		{Day: "2026-11-10", Session: "s1", Project: "other", Model: "model-a", Input: 7, Output: 70, CacheWrite: 700, CacheRead: 7000, Messages: 1},
		// Recorded under alpha/old-name, which alpha/renamed lists as a former name.
		{Day: "2026-11-11", Session: "s2", Project: "alpha/renamed", Model: "model-a", Input: 2, Output: 20, CacheWrite: 200, CacheRead: 2000, Messages: 1},
	}
	got := rowsOf(t, o)
	if len(got) != len(want) {
		t.Fatalf("rows:\n%+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: %+v", i, got[i])
		}
	}
	files, problems := state.GitFiles(o.StateDir)
	if len(problems) != 0 || files["m1"].Collected != "2026-11-11T10:00:00Z" {
		t.Fatalf("git file %+v problems %+v", files, problems)
	}
}

func TestRunKeepsRowsOfDeletedTranscripts(t *testing.T) {
	o := world(t)
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	before := rowsOf(t, o)
	if err := os.Remove(filepath.Join(o.TranscriptsDir, "-p-beta-tool/s2.jsonl")); err != nil {
		t.Fatal(err)
	}
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	after := rowsOf(t, o)
	if len(after) != 4 || after[3] != before[3] || after[3].Session != "s2" {
		t.Fatalf("rows after the transcript was removed:\n%+v", after)
	}
}

func TestRunParsesUnchangedFilesOnce(t *testing.T) {
	o := world(t)
	opens := 0
	o.Open = func(p string) (io.ReadCloser, error) { opens++; return os.Open(p) }
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	if opens != 3 {
		t.Fatalf("first run opened %d files, want 3", opens)
	}
	first, _ := os.ReadFile(filepath.Join(o.StateDir, "usage/m1/2026-11.jsonl"))
	firstMachine, _ := os.ReadFile(filepath.Join(o.StateDir, "machines/m1.json"))
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(o.StateDir, "usage/m1/2026-11.jsonl"))
	secondMachine, _ := os.ReadFile(filepath.Join(o.StateDir, "machines/m1.json"))
	if opens != 3 {
		t.Fatalf("second run opened %d more files", opens-3)
	}
	if len(first) == 0 || !bytes.Equal(first, second) {
		t.Fatal("the second run changed the usage file")
	}
	if len(firstMachine) == 0 || !bytes.Equal(firstMachine, secondMachine) {
		t.Fatal("the second run changed the machine file")
	}
}

func TestRunCommitsOnlyItsOwnPaths(t *testing.T) {
	o := world(t)
	stray := filepath.Join(o.StateDir, "alpha/fern.md")
	if err := os.WriteFile(stray, []byte("edited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, o.StateDir, "add", "alpha/fern.md")
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	if got := git(t, o.StateDir, "log", "-1", "--format=%s"); got != "chore(m1): collect" {
		t.Fatalf("subject %q", got)
	}
	if got := git(t, o.StateDir, "log", "-1", "--format=%b"); !strings.Contains(got, "Signed-off-by: test <test@example.invalid>") {
		t.Fatalf("body %q", got)
	}
	if got := git(t, o.StateDir, "show", "--name-only", "--format=", "HEAD"); got != "git/m1.json\nmachines/m1.json\nusage/m1/2026-11.jsonl" {
		t.Fatalf("committed files:\n%s", got)
	}
	if got := git(t, o.StateDir, "status", "--porcelain"); got != "M  alpha/fern.md" {
		t.Fatalf("status %q", got)
	}
}

func TestRunWithoutCommitLeavesTheFilesUncommitted(t *testing.T) {
	o := world(t)
	o.Commit = false
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	if got := git(t, o.StateDir, "log", "--format=%s"); got != "init" {
		t.Fatalf("log %q", got)
	}
	if len(rowsOf(t, o)) != 4 {
		t.Fatal("the rows must be written")
	}
}

func TestRunPushesAfterRebasingOnAnotherMachine(t *testing.T) {
	o := world(t)
	tmp := t.TempDir()
	bare := filepath.Join(tmp, "remote.git")
	git(t, tmp, "init", "-q", "--bare", "-b", "main", bare)
	git(t, o.StateDir, "remote", "add", "origin", bare)
	git(t, o.StateDir, "push", "-q", "-u", "origin", "main")
	other := filepath.Join(tmp, "other")
	git(t, tmp, "clone", "-q", bare, other)
	if err := os.WriteFile(filepath.Join(other, "note"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, other, "add", "note")
	git(t, other, "commit", "-q", "-m", "from another machine")
	git(t, other, "push", "-q")

	o.Push = true
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	if got := git(t, bare, "log", "--format=%s"); got != "chore(m1): collect\nfrom another machine\ninit" {
		t.Fatalf("remote log:\n%s", got)
	}
}

func TestRunReportsAnUnreadableTranscriptAndKeepsItsRows(t *testing.T) {
	o := world(t)
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	before := rowsOf(t, o)
	kept := sessionOf(t, machineOf(t, o), "s2")
	// Touch the file so the cache misses, then make it unreadable.
	path := filepath.Join(o.TranscriptsDir, "-p-beta-tool/s2.jsonl")
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
	o.Open = func(p string) (io.ReadCloser, error) {
		if p == path {
			return nil, os.ErrPermission
		}
		return os.Open(p)
	}
	var out bytes.Buffer
	o.Out = &out
	err := Run(o)
	if err == nil || err.Error() != "1 session could not be read" {
		t.Fatalf("err %v", err)
	}
	if !strings.Contains(out.String(), "unreadable: "+path) {
		t.Fatalf("output %q", out.String())
	}
	after := rowsOf(t, o)
	if len(after) != 4 || after[3] != before[3] {
		t.Fatalf("rows:\n%+v", after)
	}
	if got := sessionOf(t, machineOf(t, o), "s2"); got != kept || got.Last != "2026-11-11T09:00:00Z" {
		t.Fatalf("s2 after a failed read %+v, before %+v", got, kept)
	}
}
