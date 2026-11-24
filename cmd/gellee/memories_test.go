// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/aimd54/gellee/internal/memories"
)

// memoryWorld is one machine with a projects root holding alpha/fern,
// one memory for it, and an empty clone of a bare forge.
type memoryWorld struct {
	flags             []string
	mem, store, forge string
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func newMemoryWorld(t *testing.T) memoryWorld {
	t.Helper()
	for k, v := range map[string]string{
		"GIT_CONFIG_GLOBAL": "/dev/null", "GIT_CONFIG_SYSTEM": "/dev/null",
		"GIT_AUTHOR_NAME": "test", "GIT_AUTHOR_EMAIL": "test@example.invalid",
		"GIT_COMMITTER_NAME": "test", "GIT_COMMITTER_EMAIL": "test@example.invalid",
	} {
		t.Setenv(k, v)
	}
	tmp := t.TempDir()
	w := memoryWorld{forge: filepath.Join(tmp, "forge.git"), store: filepath.Join(tmp, "store")}
	gitIn(t, tmp, "init", "-q", "--bare", "-b", "main", w.forge)
	gitIn(t, tmp, "clone", "-q", w.forge, w.store)
	gitIn(t, w.store, "symbolic-ref", "HEAD", "refs/heads/main")
	stateDir := filepath.Join(tmp, "state")
	projects := filepath.Join(tmp, "projects")
	claude := filepath.Join(tmp, "claude")
	for _, d := range []string{stateDir, filepath.Join(projects, "alpha", "fern")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(stateDir, "config"), []byte("roots=alpha\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(projects)
	if err != nil {
		t.Fatal(err)
	}
	w.mem = filepath.Join(claude, "projects", memories.Encode(filepath.Join(root, "alpha", "fern")), "memory")
	w.write(t, "one.md", "---\nname: one\ndescription: the first\ntype: project\n---\none\n")
	w.write(t, "MEMORY.md", "- [One](one.md) - the first\n")
	w.flags = []string{"--store", w.store, "--state", stateDir, "--projects", projects, "--claude", claude, "--machine", "m1"}
	return w
}

func (w memoryWorld) write(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(w.mem, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.mem, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// gellee runs a memories command with the world's flags after its own.
func (w memoryWorld) gellee(t *testing.T, args ...string) (stdout string) {
	t.Helper()
	var out, errw bytes.Buffer
	if code := run(append(append([]string{"memories"}, args...), w.flags...), &out, &errw); code != 0 {
		t.Fatalf("memories %v: exit code %d\n%s%s", args, code, out.String(), errw.String())
	}
	return out.String()
}

func TestMemoriesAdoptWithoutApplyPrintsThePlanOnly(t *testing.T) {
	w := newMemoryWorld(t)
	out := w.gellee(t, "adopt")
	if !strings.Contains(out, "alpha/fern/one.md") || !strings.Contains(out, "run again with --apply") {
		t.Fatalf("stdout %q", out)
	}
	if _, err := os.Stat(filepath.Join(w.store, "alpha", "fern", "one.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the store already holds one.md: %v", err)
	}
}

func TestMemoriesAdoptApplyOnAJoinedMachineExitsNonZero(t *testing.T) {
	w := newMemoryWorld(t)
	w.gellee(t, "adopt", "--apply")
	var out, errw bytes.Buffer
	code := run(append([]string{"memories", "adopt", "--apply"}, w.flags...), &out, &errw)
	if code != 1 || !strings.Contains(errw.String(), "already joined the memory store: run gellee memories sync") {
		t.Fatalf("code %d stdout %q stderr %q", code, out.String(), errw.String())
	}
	if out := w.gellee(t, "adopt"); !strings.Contains(out, "run again with --apply") {
		t.Fatalf("the dry run printed %q", out)
	}
}

func TestMemoriesAdoptSyncAndList(t *testing.T) {
	w := newMemoryWorld(t)
	w.gellee(t, "adopt", "--apply")
	w.write(t, "two.md", "two\n")
	if out := w.gellee(t, "sync"); !strings.Contains(out, "sync: 1 exported, 0 imported, 0 set aside, 0 deletions overridden") {
		t.Fatalf("sync printed %q", out)
	}
	if data, err := os.ReadFile(filepath.Join(w.store, "alpha", "fern", "two.md")); err != nil || string(data) != "two\n" {
		t.Fatalf("store two.md: %q, %v", data, err)
	}
	var l memories.Listing
	if err := json.Unmarshal([]byte(w.gellee(t, "list", "--json")), &l); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range l.Memories {
		if m.Project == "alpha/fern" && m.File == "one.md" {
			found = m.Type == "project" && m.Description == "the first"
		}
	}
	if !found {
		t.Fatalf("list --json lacks one.md with its frontmatter: %+v", l.Memories)
	}
	if out := w.gellee(t, "list"); !strings.Contains(out, "PROJECT") || !strings.Contains(out, "the first") {
		t.Fatalf("list printed %q", out)
	}
}

func TestMemoriesNoticesSyncAndSayWhatCameIn(t *testing.T) {
	w := newMemoryWorld(t)
	w.gellee(t, "adopt", "--apply")
	other := filepath.Join(t.TempDir(), "other")
	gitIn(t, filepath.Dir(other), "clone", "-q", w.forge, other)
	if err := os.WriteFile(filepath.Join(other, "alpha", "fern", "three.md"), []byte("three\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, other, "add", "alpha/fern/three.md")
	gitIn(t, other, "commit", "-q", "-s", "-m", "elsewhere")
	gitIn(t, other, "push", "-q", "origin", "HEAD:main")
	out := w.gellee(t, "notices", "--sync", "--project", "alpha/fern")
	if out != "memories: 1 memory files imported for alpha/fern, restart to use them\n" {
		t.Fatalf("notices printed %q", out)
	}
	if data, err := os.ReadFile(filepath.Join(w.mem, "three.md")); err != nil || string(data) != "three\n" {
		t.Fatalf("three.md: %q, %v", data, err)
	}
}

func TestMemoriesNoticesReportABrokenStoreInOneLine(t *testing.T) {
	dir, tmp := t.TempDir(), t.TempDir()
	flags := []string{
		"--store", dir, "--state", filepath.Join(tmp, "state"), "--projects", filepath.Join(tmp, "projects"),
		"--claude", filepath.Join(tmp, "claude"), "--machine", "m1", "--project", "alpha/fern",
	}
	for _, sync := range [][]string{nil, {"--sync"}} {
		var out, errw bytes.Buffer
		code := run(append(append([]string{"memories", "notices"}, sync...), flags...), &out, &errw)
		if code != 0 || out.String() != "memories: "+dir+" is not a checkout of the memory store\n" {
			t.Fatalf("%v: code %d stdout %q stderr %q", sync, code, out.String(), errw.String())
		}
	}
}

func TestMemoriesNoticesKeepTheSyncErrorWhenTheNoticesFailToo(t *testing.T) {
	w := newMemoryWorld(t)
	// Not joined, and dirty: the sync refuses the dirty tree, the notices
	// the machine that has not joined.
	if err := os.WriteFile(filepath.Join(w.store, "stray.md"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := w.gellee(t, "notices", "--sync", "--project", "alpha/fern")
	want := "memories: sync failed: the checkout of the memory store has uncommitted changes\n" +
		"memories: this machine has not joined the memory store: run gellee memories adopt\n"
	if out != want {
		t.Fatalf("notices printed %q, want %q", out, want)
	}
}

func TestMemoriesNeedAStore(t *testing.T) {
	t.Setenv("GELLEE_MEMORIES", "")
	var out, errw bytes.Buffer
	if code := run([]string{"memories", "sync"}, &out, &errw); code != 2 ||
		!strings.Contains(errw.String(), "--store or GELLEE_MEMORIES is required") {
		t.Fatalf("code %d stderr %q", code, errw.String())
	}
}

func TestMemoriesUnknownCommand(t *testing.T) {
	var out, errw bytes.Buffer
	if code := run([]string{"memories", "nope"}, &out, &errw); code != 2 ||
		!strings.Contains(errw.String(), `unknown memories command "nope"`) {
		t.Fatalf("code %d stderr %q", code, errw.String())
	}
}

func TestMemoriesSyncWhileLockedDoesNothingAndExitsZero(t *testing.T) {
	w := newMemoryWorld(t)
	w.gellee(t, "adopt", "--apply")
	lockPath := filepath.Join(w.store, ".git", "gellee", "lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	out := w.gellee(t, "sync")
	if out != "sync: another memories command is running; this one did nothing\n" {
		t.Fatalf("sync printed %q", out)
	}
}

func TestMemoriesSyncNamesTheProjectItImported(t *testing.T) {
	w := newMemoryWorld(t)
	w.gellee(t, "adopt", "--apply")
	other := filepath.Join(t.TempDir(), "other")
	gitIn(t, filepath.Dir(other), "clone", "-q", w.forge, other)
	if err := os.WriteFile(filepath.Join(other, "alpha", "fern", "three.md"), []byte("three\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, other, "add", "alpha/fern/three.md")
	gitIn(t, other, "commit", "-q", "-s", "-m", "elsewhere")
	gitIn(t, other, "push", "-q", "origin", "HEAD:main")
	out := w.gellee(t, "sync")
	want := "imported alpha/fern 1\nsync: 0 exported, 1 imported, 0 set aside, 0 deletions overridden\n"
	if out != want {
		t.Fatalf("sync printed %q, want %q", out, want)
	}
}
