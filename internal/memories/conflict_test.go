// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTheSameLinesChangedOnBothMachinesSetsOneVersionAside(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "one.md", "from m1\n")
	b.write("alpha/fern", "one.md", "from m2\n")
	a.sync()
	res := b.sync()
	if res.Conflicts != 1 {
		t.Fatalf("m2 reported %d conflicts, want 1", res.Conflicts)
	}
	if got := b.read("alpha/fern", "one.md"); got != "from m1\n" {
		t.Fatalf("m2 reads %q, want the store's version", got)
	}
	aside := "orphans/m2/conflicts/alpha/fern/one.md"
	if got := b.store(aside); got != "from m2\n" {
		t.Fatalf("set aside on m2: %q", got)
	}
	a.sync()
	if got := a.store(aside); got != "from m2\n" {
		t.Fatalf("set aside, as m1 sees it: %q", got)
	}
	if got := a.read("alpha/fern", "one.md"); got != "from m1\n" {
		t.Fatalf("m1 reads %q", got)
	}
}

func TestSyncMergesInACheckoutThatAllowsOnlyFastForwards(t *testing.T) {
	a, b := pair(t)
	git(t, b.o.StoreDir, "config", "merge.ff", "only")
	a.write("alpha/fern", "one.md", "from m1\n")
	a.write("alpha/fern", "from-m1.md", "m1\n")
	b.write("alpha/fern", "one.md", "from m2\n")
	b.write("alpha/fern", "from-m2.md", "m2\n")
	a.sync()
	b.sync()
	a.sync()
	for _, m := range []machine{a, b} {
		if got := m.read("alpha/fern", "from-m1.md"); got != "m1\n" {
			t.Fatalf("%s holds %q for m1's memory", m.o.Machine, got)
		}
		if got := m.read("alpha/fern", "from-m2.md"); got != "m2\n" {
			t.Fatalf("%s holds %q for m2's memory", m.o.Machine, got)
		}
		if got := m.read("alpha/fern", "one.md"); got != "from m1\n" {
			t.Fatalf("%s reads %q, want the store's version", m.o.Machine, got)
		}
	}
	if got := a.store("orphans/m2/conflicts/alpha/fern/one.md"); got != "from m2\n" {
		t.Fatalf("set aside: %q", got)
	}
}

func TestDifferentLinesChangedOnBothMachinesAreBothKept(t *testing.T) {
	a, b := pair(t)
	base := "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\n"
	a.write("alpha/fern", "one.md", base)
	a.sync()
	b.sync()
	a.write("alpha/fern", "one.md", strings.Replace(base, "l1\n", "l1 from m1\n", 1))
	b.write("alpha/fern", "one.md", strings.Replace(base, "l9\n", "l9 from m2\n", 1))
	a.sync()
	res := b.sync()
	if res.Conflicts != 0 {
		t.Fatalf("m2 reported %d conflicts, want 0", res.Conflicts)
	}
	a.sync()
	want := strings.Replace(strings.Replace(base, "l1\n", "l1 from m1\n", 1), "l9\n", "l9 from m2\n", 1)
	for _, m := range []machine{a, b} {
		if got := m.read("alpha/fern", "one.md"); got != want {
			t.Fatalf("%s reads %q, want %q", m.o.Machine, got, want)
		}
	}
	if got := b.store("orphans/m2/conflicts/alpha/fern/one.md"); got != "" {
		t.Fatalf("something was set aside: %q", got)
	}
}

func TestAConflictOnTheGlobalFileIsSetAsideUnderItsOwnPath(t *testing.T) {
	a, b := pair(t)
	a.write("global", "CLAUDE.md", "global from m1\n")
	b.write("global", "CLAUDE.md", "global from m2\n")
	a.sync()
	res := b.sync()
	if res.Conflicts != 1 {
		t.Fatalf("m2 reported %d conflicts, want 1", res.Conflicts)
	}
	if got := b.read("global", "CLAUDE.md"); got != "global from m1\n" {
		t.Fatalf("m2 reads %q, want the store's version", got)
	}
	if got := b.store("orphans/m2/conflicts/global/CLAUDE.md"); got != "global from m2\n" {
		t.Fatalf("set aside: %q", got)
	}
}

func TestAChangeWinsOverADeletion(t *testing.T) {
	a, b := pair(t)
	// A second memory, so that the deletion is not of the project's
	// last: a directory left with only its index is restored instead.
	a.write("alpha/fern", "two.md", "two\n")
	a.sync()
	b.sync()
	must(t, os.Remove(filepath.Join(a.dir("alpha/fern"), "one.md")))
	b.write("alpha/fern", "one.md", "one, still in use\n")
	a.sync()
	res := b.sync()
	if res.Overridden != 1 {
		t.Fatalf("m2 reported %d overridden deletions, want 1", res.Overridden)
	}
	if got := b.read("alpha/fern", "one.md"); got != "one, still in use\n" {
		t.Fatalf("m2 reads %q", got)
	}
	a.sync()
	if got := a.read("alpha/fern", "one.md"); got != "one, still in use\n" {
		t.Fatalf("m1 reads %q after the change came back", got)
	}
}

func TestAMemoryKeptAgainstADeletionKeepsItsIndexLine(t *testing.T) {
	a, b := pair(t)
	// A second memory, so that the deletion is not of the project's
	// last: a directory left with only its index is restored instead.
	a.write("alpha/fern", "two.md", "two\n")
	a.sync()
	b.sync()
	must(t, os.Remove(filepath.Join(a.dir("alpha/fern"), "one.md")))
	a.write("alpha/fern", "MEMORY.md", "")
	b.write("alpha/fern", "one.md", "one, still in use\n")
	a.sync()
	b.sync()
	a.sync()
	for _, m := range []machine{a, b} {
		if got := m.read("alpha/fern", "one.md"); got != "one, still in use\n" {
			t.Fatalf("%s reads %q", m.o.Machine, got)
		}
		if got := m.read("alpha/fern", "MEMORY.md"); got != "- [One](one.md) - first\n" {
			t.Fatalf("%s: the index holds %q, want the line of the memory kept", m.o.Machine, got)
		}
	}
}

func TestAMemoryKeptAgainstADeletionWithoutAnIndexLineGetsOne(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "two.md", "---\nname: Two things\n---\ntwo\n")
	a.sync()
	b.sync()
	must(t, os.Remove(filepath.Join(a.dir("alpha/fern"), "two.md")))
	b.write("alpha/fern", "two.md", "---\nname: Two things\n---\ntwo, changed\n")
	a.sync()
	b.sync()
	want := "- [One](one.md) - first\n- [Two things](two.md)\n"
	if got := b.read("alpha/fern", "MEMORY.md"); got != want {
		t.Fatalf("m2's index holds %q, want %q", got, want)
	}
}

func TestBothMachinesAddingALineToTheIndexKeepsBoth(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "two.md", "two\n")
	a.write("alpha/fern", "MEMORY.md", "- [One](one.md) - first\n- [Two](two.md) - second\n")
	b.write("alpha/fern", "three.md", "three\n")
	b.write("alpha/fern", "MEMORY.md", "- [One](one.md) - first\n- [Three](three.md) - third\n")
	a.sync()
	if res := b.sync(); res.Conflicts != 0 {
		t.Fatalf("m2 reported %d conflicts on the index", res.Conflicts)
	}
	a.sync()
	for _, m := range []machine{a, b} {
		index := m.read("alpha/fern", "MEMORY.md")
		for _, line := range []string{"- [One](one.md) - first", "- [Two](two.md) - second", "- [Three](three.md) - third"} {
			if strings.Count(index, line+"\n") != 1 {
				t.Fatalf("%s: the index holds %q, want %q once", m.o.Machine, index, line)
			}
		}
	}
}

func TestAnIndexLineThatPointsNowhereIsDropped(t *testing.T) {
	a, _ := pair(t)
	a.write("alpha/fern", "MEMORY.md", "- [One](one.md) - first\n- [Lost](lost.md) - gone\nA note without a link.\n")
	a.sync()
	want := "- [One](one.md) - first\nA note without a link.\n"
	if got := a.read("alpha/fern", "MEMORY.md"); got != want {
		t.Fatalf("m1's index holds %q, want %q", got, want)
	}
	if got := a.store("alpha/fern/MEMORY.md"); got != want {
		t.Fatalf("the store's index holds %q, want %q", got, want)
	}
}

func TestASecondVersionSetAsideDoesNotReplaceTheFirst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "one.md")
	if got := freeName(path, []byte("first")); got != path {
		t.Fatalf("free name of a new path: %q", got)
	}
	must(t, os.WriteFile(path, []byte("first"), 0o600))
	if got := freeName(path, []byte("first")); got != path {
		t.Fatalf("the same content should reuse the path, got %q", got)
	}
	if got := freeName(path, []byte("second")); got != path+".2" {
		t.Fatalf("different content should get a new name, got %q", got)
	}
}

func TestAGlobalFileKeptAgainstADeletionGetsNoIndex(t *testing.T) {
	a, b := pair(t)
	a.write("global", "CLAUDE.md", "global\n")
	a.sync()
	b.sync()
	// The file is the whole unit, so deleting it on m1 would read as an
	// empty directory and be restored: the deletion reaches the remote
	// as a commit of the store instead.
	git(t, a.o.StoreDir, "rm", "-q", "global/CLAUDE.md")
	git(t, a.o.StoreDir, "commit", "-q", "-s", "-m", "drop the global file")
	git(t, a.o.StoreDir, "push", "-q", "origin", "HEAD:main")
	b.write("global", "CLAUDE.md", "global, still in use\n")
	if res := b.sync(); res.Overridden != 1 {
		t.Fatalf("m2 reported %d overridden deletions, want 1", res.Overridden)
	}
	if got := b.read("global", "CLAUDE.md"); got != "global, still in use\n" {
		t.Fatalf("m2 reads %q", got)
	}
	if got := b.store("global/MEMORY.md"); got != "" {
		t.Fatalf("the store holds a global index: %q", got)
	}
	if b.has("global", "MEMORY.md") {
		t.Fatal("m2 holds a global index")
	}
}

func TestAConflictWithAnUnpushedSnapshotSetsThisMachinesVersionAside(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "one.md", "from m1\n")
	b.write("alpha/fern", "one.md", "from m2\n")
	forge := git(t, b.o.StoreDir, "remote", "get-url", "origin")
	git(t, b.o.StoreDir, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
	if res := b.sync(); !res.Unreachable {
		t.Fatal("the missing remote answered")
	}
	a.sync()
	git(t, b.o.StoreDir, "remote", "set-url", "origin", forge)
	res := b.sync()
	if res.Conflicts != 1 {
		t.Fatalf("m2 reported %d conflicts, want 1", res.Conflicts)
	}
	if got := b.read("alpha/fern", "one.md"); got != "from m1\n" {
		t.Fatalf("m2 reads %q, want the store's version", got)
	}
	if got := b.store("orphans/m2/conflicts/alpha/fern/one.md"); got != "from m2\n" {
		t.Fatalf("set aside on m2: %q", got)
	}
}

func TestTheMergeCommitOfAConflictIsOnlyTheSubjectAndTheSignOff(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "one.md", "from m1\n")
	b.write("alpha/fern", "one.md", "from m2\n")
	a.sync()
	b.sync()
	msg := git(t, b.o.StoreDir, "log", "--merges", "-1", "--format=%B")
	lines := strings.Split(msg, "\n")
	if len(lines) != 3 || lines[0] != "sync(m2): merge" || lines[1] != "" ||
		!strings.HasPrefix(lines[2], "Signed-off-by: ") {
		t.Fatalf("the merge commit message is %q", msg)
	}
}
