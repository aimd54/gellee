// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAProjectSeenForTheFirstTimeIsJoinedNotMirrored(t *testing.T) {
	a, b := pair(t)
	must(t, os.MkdirAll(filepath.Join(a.o.ProjectsRoot, "alpha", "new"), 0o700))
	a.write("alpha/new", "fresh.md", "fresh\n")
	a.sync()
	if got := a.store("alpha/new/fresh.md"); got != "fresh\n" {
		t.Fatalf("the store holds %q for the new project", got)
	}
	if got := git(t, a.o.StoreDir, "log", "-1", "--format=%s"); got != "adopt(m1): 1 new project" {
		t.Fatalf("the join commit is %q", got)
	}
	// m2 gets the project later, and Claude Code has already made its
	// memory directory, empty.
	must(t, os.MkdirAll(filepath.Join(b.o.ProjectsRoot, "alpha", "new"), 0o700))
	must(t, os.MkdirAll(b.dir("alpha/new"), 0o700))
	b.sync()
	if got := b.read("alpha/new", "fresh.md"); got != "fresh\n" {
		t.Fatalf("m2 holds %q", got)
	}
	if got := b.store("alpha/new/fresh.md"); got != "fresh\n" {
		t.Fatalf("the store holds %q after m2's sync", got)
	}
	if !b.o.adopted()["alpha/new"] {
		t.Fatal("m2 did not record the project as joined")
	}
}

func TestAFailedJoinCommitLeavesTheMachinesMemoriesAlone(t *testing.T) {
	a, b := pair(t)
	must(t, os.MkdirAll(filepath.Join(a.o.ProjectsRoot, "alpha", "new"), 0o700))
	a.write("alpha/new", "shared.md", "from m1\n")
	a.sync()
	must(t, os.MkdirAll(filepath.Join(b.o.ProjectsRoot, "alpha", "new"), 0o700))
	b.write("alpha/new", "shared.md", "from m2\n")
	b.hook("pre-commit", "exit 1")
	if _, err := Sync(b.o); err == nil {
		t.Fatal("the sync reported success although its join commit failed")
	}
	if got := b.read("alpha/new", "shared.md"); got != "from m2\n" {
		t.Fatalf("m2's memory holds %q after the failed commit, want its own version", got)
	}
}

func TestAJoinedProjectWhoseDirectoryVanishedIsRestored(t *testing.T) {
	a, b := pair(t)
	must(t, os.RemoveAll(b.dir("alpha/fern")))
	b.sync()
	if got := b.read("alpha/fern", "one.md"); got != "one\n" {
		t.Fatalf("m2 holds %q after the restore", got)
	}
	a.sync()
	if got := a.read("alpha/fern", "one.md"); got != "one\n" {
		t.Fatalf("m1 holds %q: the vanished directory was read as a deletion", got)
	}
}

func TestADirectoryHoldingOnlyItsIndexIsRestored(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "two.md", "two\n")
	a.sync()
	b.sync()
	// The directory was lost, and a session wrote an index into a new one.
	must(t, os.RemoveAll(b.dir("alpha/fern")))
	b.write("alpha/fern", "MEMORY.md", "")
	b.sync()
	a.sync()
	for _, m := range []machine{a, b} {
		if got := m.read("alpha/fern", "one.md"); got != "one\n" {
			t.Fatalf("%s holds %q for one.md", m.o.Machine, got)
		}
		if got := m.read("alpha/fern", "two.md"); got != "two\n" {
			t.Fatalf("%s holds %q for two.md", m.o.Machine, got)
		}
	}
	if got := b.read("alpha/fern", "MEMORY.md"); got != "- [One](one.md) - first\n" {
		t.Fatalf("m2's index holds %q, want the store's", got)
	}
	if got := a.store("alpha/fern/two.md"); got != "two\n" {
		t.Fatalf("the store holds %q for two.md", got)
	}
}

func TestAnIndexOnlyDirectoryWithNothingToRestoreIsCarried(t *testing.T) {
	a, b := pair(t)
	// alpha/pond is joined with nothing in the store, and its directory
	// holds only an index: there is nothing to restore it from.
	note := "Notes on the pond, no memory yet.\n"
	b.write("alpha/pond", "MEMORY.md", note)
	b.sync()
	if got := b.read("alpha/pond", "MEMORY.md"); got != note {
		t.Fatalf("m2's index holds %q after the sync", got)
	}
	if got := b.store("alpha/pond/MEMORY.md"); got != note {
		t.Fatalf("the store holds %q for the index", got)
	}
	// Its first memory makes it a directory like any other.
	index := note + "- [Pond](pond.md) - the pond\n"
	b.write("alpha/pond", "pond.md", "pond\n")
	b.write("alpha/pond", "MEMORY.md", index)
	b.sync()
	a.sync()
	if got := a.read("alpha/pond", "MEMORY.md"); got != index {
		t.Fatalf("m1's index holds %q", got)
	}
	if got := a.read("alpha/pond", "pond.md"); got != "pond\n" {
		t.Fatalf("m1 holds %q for pond.md", got)
	}
}

func TestAProjectThatLeftTheMachineAndReturnsIsJoinedAgain(t *testing.T) {
	a, b := pair(t)
	// The checkout goes, to be cloned again later; its memory directory
	// stays behind, holding what m2 last copied back.
	project := filepath.Join(b.o.ProjectsRoot, "alpha", "fern")
	must(t, os.RemoveAll(project))
	b.sync()
	a.write("alpha/fern", "one.md", "one, edited on m1\n")
	a.write("alpha/fern", "two.md", "two from m1\n")
	a.sync()
	b.sync()
	must(t, os.MkdirAll(project, 0o700))
	b.sync()
	a.sync()
	for _, m := range []machine{a, b} {
		if got := m.read("alpha/fern", "one.md"); got != "one, edited on m1\n" {
			t.Fatalf("%s holds %q for one.md", m.o.Machine, got)
		}
		if got := m.read("alpha/fern", "two.md"); got != "two from m1\n" {
			t.Fatalf("%s holds %q for two.md", m.o.Machine, got)
		}
	}
	if got := a.store("alpha/fern/one.md"); got != "one, edited on m1\n" {
		t.Fatalf("the store holds %q for one.md", got)
	}
	if got := a.store("alpha/fern/two.md"); got != "two from m1\n" {
		t.Fatalf("the store holds %q for two.md", got)
	}
}

func TestAnUnreachableRemoteKeepsTheWorkAndDeliversItLater(t *testing.T) {
	a, b := pair(t)
	forge := git(t, a.o.StoreDir, "remote", "get-url", "origin")
	git(t, a.o.StoreDir, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "nowhere.git"))
	a.write("alpha/fern", "two.md", "two\n")
	res := a.sync()
	if !res.Unreachable {
		t.Fatal("the sync did not report the remote as unreachable")
	}
	if got := git(t, a.o.StoreDir, "show", "HEAD:alpha/fern/two.md"); got != "two" {
		t.Fatalf("the local commit holds %q", got)
	}
	if a.o.readState("pushed") != "2026-11-14T10:00:00Z" {
		t.Fatalf("pushed moved to %q without a push", a.o.readState("pushed"))
	}
	git(t, a.o.StoreDir, "remote", "set-url", "origin", forge)
	if res := a.sync(); res.Unreachable {
		t.Fatal("the remote is back and the sync says it is not")
	}
	b.sync()
	if got := b.read("alpha/fern", "two.md"); got != "two\n" {
		t.Fatalf("m2 holds %q", got)
	}
}

func TestASecondSyncAtTheSameTimeStepsAside(t *testing.T) {
	a, b := pair(t)
	unlock, err := a.o.lock()
	must(t, err)
	a.write("alpha/fern", "two.md", "two\n")
	if _, err := Sync(a.o); !errors.Is(err, ErrLocked) {
		t.Fatalf("sync while locked: %v, want ErrLocked", err)
	}
	unlock()
	a.sync()
	b.sync()
	if got := b.read("alpha/fern", "two.md"); got != "two\n" {
		t.Fatalf("m2 holds %q", got)
	}
}

// interrupted runs the tests of a sync that died halfway: two machines
// have joined, m1 has published a new version of one.md, and m2's store
// has been left in the state the dead run produced.

func TestASyncLeftInTheMiddleOfAMergeIsRepeated(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "one.md", "from m1\n")
	a.sync()
	// m2's store holds an unpushed commit that conflicts with the remote,
	// and the merge of the two died before it was settled.
	must(t, os.WriteFile(filepath.Join(b.o.StoreDir, "alpha/fern/one.md"), []byte("from m2\n"), 0o600))
	git(t, b.o.StoreDir, "commit", "-q", "-a", "-m", "local")
	git(t, b.o.StoreDir, "fetch", "-q", "origin")
	cmd := exec.Command("git", "-C", b.o.StoreDir, "merge", "--no-commit", "origin/main")
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("the merge did not conflict:\n%s", out)
	}
	git(t, b.o.StoreDir, "rev-parse", "-q", "--verify", "MERGE_HEAD")

	b.sync()
	if _, err := os.Stat(filepath.Join(b.o.StoreDir, ".git", "MERGE_HEAD")); err == nil {
		t.Fatal("a merge is still in progress")
	}
	if got := b.read("alpha/fern", "one.md"); got != "from m1\n" {
		t.Fatalf("m2 reads %q, want the store's version", got)
	}
	if got := b.store("orphans/m2/conflicts/alpha/fern/one.md"); got != "from m2\n" {
		t.Fatalf("set aside on m2: %q", got)
	}
	a.sync()
	if got := a.store("orphans/m2/conflicts/alpha/fern/one.md"); got != "from m2\n" {
		t.Fatalf("set aside, as m1 sees it: %q", got)
	}
	if got := a.read("alpha/fern", "one.md"); got != "from m1\n" {
		t.Fatalf("m1 reads %q", got)
	}
}

func TestASyncThatDiedAfterSettingAVersionAsideIsRepeated(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "one.md", "from m1\n")
	a.sync()
	// m2's store holds an unpushed commit that conflicts with the remote,
	// and the sync merging them died after it had set m2's version aside,
	// before it committed the merge.
	must(t, os.WriteFile(filepath.Join(b.o.StoreDir, "alpha/fern/one.md"), []byte("from m2\n"), 0o600))
	git(t, b.o.StoreDir, "commit", "-q", "-a", "-m", "local")
	git(t, b.o.StoreDir, "fetch", "-q", "origin")
	if out, err := exec.Command("git", "-C", b.o.StoreDir, "merge", "--no-commit", "origin/main").CombinedOutput(); err == nil {
		t.Fatalf("the merge did not conflict:\n%s", out)
	}
	conflicts := filepath.Join(b.o.StoreDir, "orphans", "m2", "conflicts")
	must(t, os.MkdirAll(filepath.Join(conflicts, "alpha", "fern"), 0o700))
	must(t, os.WriteFile(filepath.Join(conflicts, "alpha", "fern", "one.md"), []byte("from m2\n"), 0o600))
	must(t, os.WriteFile(filepath.Join(conflicts, "alpha", "fern", ".gellee-123"), []byte("from"), 0o600))

	b.sync()
	if got := b.read("alpha/fern", "one.md"); got != "from m1\n" {
		t.Fatalf("m2 reads %q, want the store's version", got)
	}
	aside := tree(t, conflicts)
	if len(aside) != 1 || aside[filepath.Join("alpha", "fern", "one.md")] != "from m2\n" {
		t.Fatalf("set aside on m2: %v, want m2's version once", aside)
	}
	a.sync()
	if got := a.store("orphans/m2/conflicts/alpha/fern/one.md"); got != "from m2\n" {
		t.Fatalf("set aside, as m1 sees it: %q", got)
	}
}

func TestAStoreLeftDetachedWithADirtyTreeIsRepeated(t *testing.T) {
	a, b := pair(t)
	b.sync()
	// The snapshot checks out the synced commit; the run died with files
	// written into it.
	git(t, b.o.StoreDir, "checkout", "-q", "--detach", b.o.readState("synced"))
	must(t, os.WriteFile(filepath.Join(b.o.StoreDir, "alpha/fern/one.md"), []byte("half\n"), 0o600))
	must(t, os.WriteFile(filepath.Join(b.o.StoreDir, "alpha/fern/stray.md"), []byte("stray\n"), 0o600))
	b.write("alpha/fern", "three.md", "three\n")

	b.sync()
	if got := git(t, b.o.StoreDir, "symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Fatalf("the store is on %q", got)
	}
	if got := b.store("alpha/fern/three.md"); got != "three\n" {
		t.Fatalf("the store holds %q for three.md", got)
	}
	if got := b.store("alpha/fern/one.md"); got != "one\n" {
		t.Fatalf("the store holds %q for one.md", got)
	}
	if got := b.store("alpha/fern/stray.md"); got != "" {
		t.Fatalf("the store holds the leftover %q", got)
	}
	a.sync()
	if got := a.read("alpha/fern", "three.md"); got != "three\n" {
		t.Fatalf("m1 holds %q", got)
	}
	if a.has("alpha/fern", "stray.md") {
		t.Fatal("the leftover reached m1")
	}
}

func TestADetachedStoreWithNoBranchToReturnToIsReported(t *testing.T) {
	_, b := pair(t)
	b.sync()
	git(t, b.o.StoreDir, "branch", "other")
	git(t, b.o.StoreDir, "remote", "set-head", "origin", "-d")
	git(t, b.o.StoreDir, "checkout", "-q", "--detach", "HEAD")
	_, err := Sync(b.o)
	if err == nil || !strings.Contains(err.Error(), "detached") {
		t.Fatalf("sync: %v, want an error that names the detached store", err)
	}
}

func TestADirtyTreeOnTheBranchIsRefusedAndLeftAlone(t *testing.T) {
	_, b := pair(t)
	b.sync()
	edit := filepath.Join(b.o.StoreDir, "alpha/fern/one.md")
	must(t, os.WriteFile(edit, []byte("by hand\n"), 0o600))
	_, err := Sync(b.o)
	if err == nil || !strings.Contains(err.Error(), "alpha/fern/one.md") {
		t.Fatalf("sync: %v, want a refusal that names the file", err)
	}
	if got := b.store("alpha/fern/one.md"); got != "by hand\n" {
		t.Fatalf("the edit became %q", got)
	}
}
