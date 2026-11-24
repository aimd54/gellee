// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAdoptWithoutApplyPrintsThePlanAndChangesNothing(t *testing.T) {
	a, _ := world(t)
	a.write("alpha/fern", "one.md", "one\n")
	storeBefore, claudeBefore := tree(t, a.o.StoreDir), tree(t, a.o.ClaudeDir)
	plan, err := Adopt(a.o, false)
	must(t, err)
	want := []Action{{What: "to store", Key: "alpha/fern", Name: "one.md"}}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("plan %+v, want %+v", plan, want)
	}
	if got := tree(t, a.o.StoreDir); !reflect.DeepEqual(got, storeBefore) {
		t.Fatalf("the store changed: %v", got)
	}
	if got := tree(t, a.o.ClaudeDir); !reflect.DeepEqual(got, claudeBefore) {
		t.Fatalf("the Claude directory changed: %v", got)
	}
	if a.o.readState("synced") != "" || len(a.o.adopted()) != 0 {
		t.Fatal("the machine was recorded as joined")
	}
}

func TestAdoptFillsTheStoreAndPushesIt(t *testing.T) {
	a, _ := world(t)
	a.write("alpha/fern", "one.md", "one\n")
	a.write("global", "CLAUDE.md", "rules\n")
	a.adopt()
	if got := a.store("alpha/fern/one.md"); got != "one\n" {
		t.Fatalf("store holds %q", got)
	}
	if got := a.store("global/CLAUDE.md"); got != "rules\n" {
		t.Fatalf("store holds %q", got)
	}
	if got := a.store(".gitattributes"); got != "MEMORY.md merge=union\n" {
		t.Fatalf(".gitattributes holds %q", got)
	}
	forge := git(t, a.o.StoreDir, "remote", "get-url", "origin")
	if got := git(t, forge, "show", "main:alpha/fern/one.md"); got != "one" {
		t.Fatalf("the forge holds %q", got)
	}
	head := git(t, a.o.StoreDir, "rev-parse", "HEAD")
	if a.o.readState("synced") != head {
		t.Fatalf("synced is %q, head is %q", a.o.readState("synced"), head)
	}
	for _, key := range []string{"global", "alpha/fern", "alpha/pond", "beta/hill"} {
		if !a.o.adopted()[key] {
			t.Fatalf("%s is not recorded as joined", key)
		}
	}
	if a.o.readState("pushed") != "2026-11-14T10:00:00Z" {
		t.Fatalf("pushed is %q", a.o.readState("pushed"))
	}
}

func TestASecondMachineJoinsByUnion(t *testing.T) {
	a, b := world(t)
	a.write("alpha/fern", "one.md", "one\n")
	a.write("alpha/fern", "shared.md", "from m1\n")
	a.write("global", "CLAUDE.md", "rules\n")
	a.adopt()
	b.write("alpha/fern", "two.md", "two\n")
	b.write("alpha/fern", "shared.md", "from m2\n")
	b.write("global", "CLAUDE.md", "other rules\n")
	b.adopt()
	if got := b.read("alpha/fern", "one.md"); got != "one\n" {
		t.Fatalf("m2 received %q", got)
	}
	if got := b.store("alpha/fern/two.md"); got != "two\n" {
		t.Fatalf("the store received %q", got)
	}
	if got := b.read("alpha/fern", "shared.md"); got != "from m1\n" {
		t.Fatalf("m2 reads %q for the conflicting memory", got)
	}
	if got := b.store("orphans/m2/conflicts/alpha/fern/shared.md"); got != "from m2\n" {
		t.Fatalf("set aside: %q", got)
	}
	if got := b.read("global", "CLAUDE.md"); got != "rules\n" {
		t.Fatalf("m2's global file holds %q", got)
	}
	if got := b.store("orphans/m2/CLAUDE.md"); got != "other rules\n" {
		t.Fatalf("set aside: %q", got)
	}
}

func TestAFailedAdoptCommitLeavesTheMachinesMemoriesAlone(t *testing.T) {
	a, b := world(t)
	a.write("alpha/fern", "shared.md", "from m1\n")
	a.write("alpha/fern", "one.md", "one\n")
	a.adopt()
	b.write("alpha/fern", "shared.md", "from m2\n")
	b.hook("pre-commit", "exit 1")
	if _, err := Adopt(b.o, true); err == nil {
		t.Fatal("adopt reported success although its commit failed")
	}
	if got := b.read("alpha/fern", "shared.md"); got != "from m2\n" {
		t.Fatalf("m2's memory holds %q after the failed commit, want its own version", got)
	}
	if b.has("alpha/fern", "one.md") {
		t.Fatal("the store's memory reached m2 before the commit that joins it")
	}
}

func TestAMachineWithoutAGitIdentityIsRefusedBeforeAnythingIsWritten(t *testing.T) {
	a, b := world(t)
	a.write("alpha/fern", "shared.md", "from m1\n")
	a.adopt()
	b.write("alpha/fern", "shared.md", "from m2\n")
	t.Setenv("GIT_COMMITTER_NAME", "")
	before := snapshot(t, b)
	_, err := Adopt(b.o, true)
	if err == nil || !strings.Contains(err.Error(), "no git identity") || !strings.Contains(err.Error(), "user.name and user.email") {
		t.Fatalf("adopt without an identity: %v, want the refusal that names it", err)
	}
	if after := snapshot(t, b); !reflect.DeepEqual(after, before) {
		t.Fatalf("the refused adopt changed the machine:\nbefore %v\nafter  %v", before, after)
	}
	if _, err := Sync(a.o); err == nil || !strings.Contains(err.Error(), "no git identity") {
		t.Fatalf("sync without an identity: %v, want the refusal that names it", err)
	}
}

// linkGlobal makes the machine's global file a symbolic link to a file
// of another directory, holding content, and returns that file. With
// relative set the link names its target relative to its own directory.
func linkGlobal(t *testing.T, m machine, content string, relative bool) string {
	t.Helper()
	target := filepath.Join(filepath.Dir(m.o.ClaudeDir), "dotfiles", "CLAUDE.md")
	must(t, os.MkdirAll(filepath.Dir(target), 0o700))
	must(t, os.WriteFile(target, []byte(content), 0o600))
	name := target
	if relative {
		name = filepath.Join("..", "dotfiles", "CLAUDE.md")
	}
	must(t, os.Symlink(name, filepath.Join(m.o.ClaudeDir, "CLAUDE.md")))
	return target
}

// linkedTo fails unless the machine's global file is still a symbolic
// link and its target holds want.
func linkedTo(t *testing.T, m machine, target, want string) {
	t.Helper()
	fi, err := os.Lstat(filepath.Join(m.o.ClaudeDir, "CLAUDE.md"))
	must(t, err)
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s: the global file is no longer a symbolic link", m.o.Machine)
	}
	data, err := os.ReadFile(target)
	must(t, err)
	if string(data) != want {
		t.Fatalf("%s: the link's target holds %q, want %q", m.o.Machine, data, want)
	}
}

func TestALinkedGlobalFileOnTheFirstMachineIsStoredAndWrittenThrough(t *testing.T) {
	a, b := world(t)
	target := linkGlobal(t, a, "rules from m1\n", false)
	a.adopt()
	if got := a.store("global/CLAUDE.md"); got != "rules from m1\n" {
		t.Fatalf("the store holds %q for the global file", got)
	}
	linkedTo(t, a, target, "rules from m1\n")
	b.adopt()
	b.write("global", "CLAUDE.md", "rules, revised on m2\n")
	b.sync()
	a.sync()
	linkedTo(t, a, target, "rules, revised on m2\n")
	must(t, os.WriteFile(target, []byte("rules, revised on m1\n"), 0o600))
	a.sync()
	if got := a.store("global/CLAUDE.md"); got != "rules, revised on m1\n" {
		t.Fatalf("the store holds %q after an edit through the link", got)
	}
}

func TestALinkedGlobalFileOnASecondMachineIsSetAsideAndWrittenThrough(t *testing.T) {
	a, b := world(t)
	a.write("global", "CLAUDE.md", "rules from m1\n")
	a.adopt()
	target := linkGlobal(t, b, "rules from m2\n", true)
	b.adopt()
	if got := b.store("orphans/m2/CLAUDE.md"); got != "rules from m2\n" {
		t.Fatalf("set aside: %q", got)
	}
	linkedTo(t, b, target, "rules from m1\n")
}

func TestAdoptCopiesTheMemoriesThatMatchNoProject(t *testing.T) {
	a, _ := world(t)
	old := filepath.Join(a.o.ClaudeDir, "projects", "-old-path-fern", "memory")
	must(t, os.MkdirAll(old, 0o700))
	must(t, os.WriteFile(filepath.Join(old, "kept.md"), []byte("kept\n"), 0o600))
	a.adopt()
	if got := a.store("orphans/m1/-old-path-fern/kept.md"); got != "kept\n" {
		t.Fatalf("the store holds %q", got)
	}
	if data, _ := os.ReadFile(filepath.Join(old, "kept.md")); string(data) != "kept\n" {
		t.Fatalf("the local copy holds %q", data)
	}
}

func TestApplyingAdoptOnAMachineThatHasJoinedIsRefused(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "one.md", "one, edited on m1\n")
	a.write("alpha/fern", "two.md", "two from m1\n")
	a.sync()
	before := snapshot(t, b)
	_, err := Adopt(b.o, true)
	if !errors.Is(err, ErrJoined) || !strings.Contains(err.Error(), "run gellee memories sync") {
		t.Fatalf("adopt --apply on a joined machine: %v, want the refusal", err)
	}
	if after := snapshot(t, b); !reflect.DeepEqual(after, before) {
		t.Fatalf("the refused adopt changed the machine:\nbefore %v\nafter  %v", before, after)
	}
	b.sync()
	a.sync()
	for _, m := range []machine{a, b} {
		if got := m.read("alpha/fern", "one.md"); got != "one, edited on m1\n" {
			t.Fatalf("%s holds %q for one.md", m.o.Machine, got)
		}
		if got := m.read("alpha/fern", "two.md"); got != "two from m1\n" {
			t.Fatalf("%s holds %q for two.md", m.o.Machine, got)
		}
		if got := m.store("alpha/fern/one.md"); got != "one, edited on m1\n" {
			t.Fatalf("%s's checkout holds %q for one.md", m.o.Machine, got)
		}
		if got := m.store("alpha/fern/two.md"); got != "two from m1\n" {
			t.Fatalf("%s's checkout holds %q for two.md", m.o.Machine, got)
		}
	}
}

// headOrNone is the head commit, or "none" on a branch with no commit.
func headOrNone(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "-q", "--verify", "HEAD").Output()
	if err != nil {
		return "none"
	}
	return string(out)
}

func snapshot(t *testing.T, m machine) map[string]any {
	t.Helper()
	return map[string]any{
		"head":    headOrNone(m.o.StoreDir),
		"heads":   git(t, m.o.StoreDir, "for-each-ref", "refs/heads"),
		"branch":  git(t, m.o.StoreDir, "symbolic-ref", "HEAD"),
		"status":  git(t, m.o.StoreDir, "status", "--porcelain"),
		"index":   git(t, m.o.StoreDir, "ls-files", "-s"),
		"store":   tree(t, m.o.StoreDir),
		"claude":  tree(t, m.o.ClaudeDir),
		"project": tree(t, m.o.ProjectsRoot),
	}
}

func TestAdoptDryRunOnASecondMachineChangesNothing(t *testing.T) {
	a, b := world(t)
	a.write("alpha/fern", "one.md", "one\n")
	a.adopt()
	b.write("alpha/fern", "two.md", "two\n")
	before := snapshot(t, b)
	plan, err := Adopt(b.o, false)
	must(t, err)
	if after := snapshot(t, b); !reflect.DeepEqual(after, before) {
		t.Fatalf("the dry run changed the machine:\nbefore %v\nafter  %v", before, after)
	}
	want := []Action{
		{What: "to store", Key: "alpha/fern", Name: "two.md"},
		{What: "to machine", Key: "alpha/fern", Name: "one.md"},
	}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("plan %+v, want %+v", plan, want)
	}
}

func TestOrphansReturnsAnUnreadableProjectsDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every directory")
	}
	a, _ := world(t)
	projects := filepath.Join(a.o.ClaudeDir, "projects")
	must(t, os.Chmod(projects, 0))
	t.Cleanup(func() { _ = os.Chmod(projects, 0o700) })
	if _, err := a.o.orphans(nil, false); err == nil {
		t.Fatal("an unreadable projects directory was skipped")
	}
}

func TestAdoptReturnsAnUnreadableMemoryDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every directory")
	}
	a, _ := world(t)
	memory := filepath.Join(a.o.ClaudeDir, "projects", "-old-path-fern", "memory")
	must(t, os.MkdirAll(memory, 0o700))
	must(t, os.Chmod(memory, 0))
	t.Cleanup(func() { _ = os.Chmod(memory, 0o700) })
	if _, err := Adopt(a.o, true); err == nil {
		t.Fatal("an unreadable memory directory was skipped")
	}
}

func TestTheAdoptCommitCountsOnlyWhatLandsInTheStore(t *testing.T) {
	a, b := world(t)
	a.write("alpha/fern", "one.md", "one\n")
	a.write("alpha/fern", "two.md", "two\n")
	a.adopt()
	b.write("alpha/fern", "three.md", "three\n")
	b.adopt()
	if got := git(t, b.o.StoreDir, "log", "-1", "--format=%s"); got != "adopt(m2): 1 files" {
		t.Fatalf("subject %q", got)
	}
}

func TestAdoptDryRunSkipsTheUnitsAMachineAlreadyJoined(t *testing.T) {
	a, b := world(t)
	a.write("alpha/fern", "one.md", "one\n")
	a.adopt()
	b.adopt()
	// A local edit since the join, and a change pushed by the other machine.
	b.write("alpha/fern", "one.md", "edited here\n")
	b.write("alpha/fern", "new.md", "new here\n")
	must(t, os.WriteFile(filepath.Join(a.o.StoreDir, "alpha/fern/other.md"), []byte("other\n"), 0o600))
	git(t, a.o.StoreDir, "add", "alpha/fern/other.md")
	git(t, a.o.StoreDir, "commit", "-q", "-m", "other")
	git(t, a.o.StoreDir, "push", "-q", "origin", "main")
	before := snapshot(t, b)
	plan, err := Adopt(b.o, false)
	must(t, err)
	if len(plan) != 0 {
		t.Fatalf("the dry run plans what apply would skip: %+v", plan)
	}
	if after := snapshot(t, b); !reflect.DeepEqual(after, before) {
		t.Fatal("the dry run changed the machine")
	}
}
