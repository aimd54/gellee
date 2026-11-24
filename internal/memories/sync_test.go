// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncRefusesAMachineThatHasNotJoined(t *testing.T) {
	a, _ := world(t)
	if _, err := Sync(a.o); !errors.Is(err, ErrNotAdopted) {
		t.Fatalf("sync: %v, want ErrNotAdopted", err)
	}
}

func TestSyncRefusesACheckoutWithUncommittedChanges(t *testing.T) {
	a, _ := pair(t)
	must(t, os.WriteFile(filepath.Join(a.o.StoreDir, "stray.md"), []byte("x"), 0o600))
	_, err := Sync(a.o)
	if err == nil || !strings.Contains(err.Error(), "uncommitted changes") || !strings.Contains(err.Error(), "stray.md") {
		t.Fatalf("sync: %v", err)
	}
}

func TestSyncCarriesAnAddedMemoryToTheOtherMachine(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "two.md", "two\n")
	if res := a.sync(); res.Exported != 1 {
		t.Fatalf("m1 exported %d files, want 1", res.Exported)
	}
	res := b.sync()
	if got := b.read("alpha/fern", "two.md"); got != "two\n" {
		t.Fatalf("m2 holds %q", got)
	}
	if res.Imported != 1 || res.ImportedBy["alpha/fern"] != 1 {
		t.Fatalf("m2 imported %+v", res)
	}
}

func TestSyncCarriesAChangedMemory(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "one.md", "one, revised\n")
	a.sync()
	b.sync()
	if got := b.read("alpha/fern", "one.md"); got != "one, revised\n" {
		t.Fatalf("m2 holds %q", got)
	}
}

func TestSyncCarriesADeletion(t *testing.T) {
	a, b := pair(t)
	// A second memory, so that the deletion is not of the project's
	// last: a directory left with only its index is restored instead.
	a.write("alpha/fern", "two.md", "two\n")
	a.sync()
	b.sync()
	must(t, os.Remove(filepath.Join(a.dir("alpha/fern"), "one.md")))
	a.sync()
	b.sync()
	if b.has("alpha/fern", "one.md") {
		t.Fatal("m2 still holds the deleted memory")
	}
	if !b.has("alpha/fern", "MEMORY.md") {
		t.Fatal("m2 lost the index along with the memory")
	}
	if b.store("alpha/fern/one.md") != "" {
		t.Fatal("the store still holds the deleted memory")
	}
}

func TestSyncKeepsWhatEachMachineAdded(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "from-m1.md", "m1\n")
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
	}
}

func TestSyncKeepsAnEditMadeOnEachMachine(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "two.md", "two\n")
	a.sync()
	b.sync()
	a.write("alpha/fern", "one.md", "one, edited on m1\n")
	b.write("alpha/fern", "two.md", "two, edited on m2\n")
	a.sync()
	b.sync()
	a.sync()
	for _, m := range []machine{a, b} {
		if got := m.read("alpha/fern", "one.md"); got != "one, edited on m1\n" {
			t.Fatalf("%s holds %q for one.md", m.o.Machine, got)
		}
		if got := m.read("alpha/fern", "two.md"); got != "two, edited on m2\n" {
			t.Fatalf("%s holds %q for two.md", m.o.Machine, got)
		}
	}
}

func TestSyncNamesTheMachineInItsMergeCommits(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "from-m1.md", "m1\n")
	b.write("alpha/fern", "from-m2.md", "m2\n")
	a.sync()
	b.sync()
	merges := git(t, b.o.StoreDir, "log", "--merges", "--format=%s%n%b")
	if !strings.Contains(merges, "sync(m2): merge") || !strings.Contains(merges, "Signed-off-by:") {
		t.Fatalf("merge commits of m2: %q", merges)
	}
}

func TestSyncCarriesTheGlobalFile(t *testing.T) {
	a, b := pair(t)
	a.write("global", "CLAUDE.md", "rules\n")
	a.sync()
	b.sync()
	if got := b.read("global", "CLAUDE.md"); got != "rules\n" {
		t.Fatalf("m2's global file holds %q", got)
	}
}

func TestAMemoryWrittenDuringASyncIsKeptAndCarriedByTheNext(t *testing.T) {
	a, b := pair(t)
	b.write("alpha/fern", "from-m2.md", "m2\n")
	b.sync()
	a.write("alpha/fern", "two.md", "two\n")
	// A session writes while the sync is between its snapshot and its
	// copy back: here, while it pushes.
	session := t.TempDir()
	index := "- [One](one.md) - first\n- [Mid](mid.md) - mid\n"
	for name, content := range map[string]string{"mid.md": "mid\n", "one.md": "one, edited\n", "MEMORY.md": index} {
		must(t, os.WriteFile(filepath.Join(session, name), []byte(content), 0o600))
	}
	local := a.dir("alpha/fern")
	remove := a.hook("pre-push", "cp '"+session+"'/* '"+local+"'/ && rm '"+local+"/two.md'")
	a.sync()
	remove()
	if got := a.read("alpha/fern", "mid.md"); got != "mid\n" {
		t.Fatalf("m1 holds %q for the memory written during the sync", got)
	}
	if got := a.read("alpha/fern", "one.md"); got != "one, edited\n" {
		t.Fatalf("m1 holds %q for the memory edited during the sync", got)
	}
	if got := a.read("alpha/fern", "MEMORY.md"); got != index {
		t.Fatalf("m1's index holds %q, want the one written during the sync", got)
	}
	if a.has("alpha/fern", "two.md") {
		t.Fatal("the memory removed during the sync is back on m1")
	}
	if got := a.read("alpha/fern", "from-m2.md"); got != "m2\n" {
		t.Fatalf("m1 holds %q for m2's memory, which the sync imported", got)
	}

	a.sync()
	b.sync()
	for path, want := range map[string]string{
		"alpha/fern/mid.md": "mid\n", "alpha/fern/one.md": "one, edited\n",
		"alpha/fern/MEMORY.md": index, "alpha/fern/two.md": "",
	} {
		if got := a.store(path); got != want {
			t.Fatalf("the store holds %q for %s after the next sync, want %q", got, path, want)
		}
	}
	if got := b.read("alpha/fern", "mid.md"); got != "mid\n" {
		t.Fatalf("m2 holds %q for the memory written during m1's sync", got)
	}
	if got := b.read("alpha/fern", "MEMORY.md"); got != index {
		t.Fatalf("m2's index holds %q", got)
	}
}

// during installs a pre-push hook on m that runs a session's shell
// command in the memory directory of alpha/fern, between the sync's
// snapshot and its copy back.
func during(m machine, cmd string) (remove func()) {
	return m.hook("pre-push", "cd '"+m.dir("alpha/fern")+"' && "+cmd)
}

func TestAMemoryChangedBySessionAndOtherMachineDuringASyncIsSetAside(t *testing.T) {
	a, b := pair(t)
	b.write("alpha/fern", "one.md", "from m2\n")
	b.sync()
	remove := during(a, "printf 'from the session\\n' >one.md")
	res := a.sync()
	remove()
	if res.Conflicts != 1 {
		t.Fatalf("m1 reported %d conflicts, want 1", res.Conflicts)
	}
	if got := a.read("alpha/fern", "one.md"); got != "from m2\n" {
		t.Fatalf("m1 holds %q, want the store's version", got)
	}
	aside := "orphans/m1/conflicts/alpha/fern/one.md"
	if got := a.store(aside); got != "from the session\n" {
		t.Fatalf("set aside on m1: %q", got)
	}
	a.sync()
	b.sync()
	if got := b.read("alpha/fern", "one.md"); got != "from m2\n" {
		t.Fatalf("m2 holds %q after the next syncs, want its own version", got)
	}
	if got := b.store(aside); got != "from the session\n" {
		t.Fatalf("set aside, as m2 sees it: %q", got)
	}
}

func TestAFailedCommitOfASetAsideVersionLeavesTheSessionsVersion(t *testing.T) {
	a, b := pair(t)
	b.write("alpha/fern", "one.md", "from m2\n")
	b.sync()
	hooks := filepath.Join(a.o.StoreDir, ".git", "hooks")
	remove := during(a, "printf 'from the session\\n' >one.md && "+
		"printf '#!/bin/sh\\nexit 1\\n' >'"+hooks+"/pre-commit' && chmod +x '"+hooks+"/pre-commit'")
	_, err := Sync(a.o)
	remove()
	if err == nil {
		t.Fatal("the sync succeeded although the set-aside commit failed")
	}
	if got := a.read("alpha/fern", "one.md"); got != "from the session\n" {
		t.Fatalf("m1 holds %q, want the session's version", got)
	}
	must(t, os.Remove(filepath.Join(hooks, "pre-commit")))
	a.sync()
	if got := a.store("orphans/m1/conflicts/alpha/fern/one.md"); got != "from the session\n" {
		t.Fatalf("set aside on the repeated sync: %q", got)
	}
	if got := a.read("alpha/fern", "one.md"); got != "from m2\n" {
		t.Fatalf("m1 holds %q after the repeated sync, want the store's version", got)
	}
}

func TestAMemoryWrittenAgainDuringTheSetAsideCommitIsSetAsideToo(t *testing.T) {
	a, b := pair(t)
	b.write("alpha/fern", "one.md", "from m2\n")
	b.sync()
	// The session writes once before copy back and once more while the
	// first version set aside is being committed.
	hooks := filepath.Join(a.o.StoreDir, ".git", "hooks")
	local := filepath.Join(a.dir("alpha/fern"), "one.md")
	must(t, os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte("#!/bin/sh\n"+
		"[ -f orphans/m1/conflicts/alpha/fern/one.md ] && printf 'second\\n' >'"+local+"'\nexit 0\n"), 0o700))
	remove := during(a, "printf 'first\\n' >one.md")
	a.sync()
	remove()
	must(t, os.Remove(filepath.Join(hooks, "pre-commit")))
	aside := "orphans/m1/conflicts/alpha/fern/one.md"
	if got := a.store(aside); got != "first\n" {
		t.Fatalf("first version set aside: %q", got)
	}
	if got := a.store(aside + ".2"); got != "second\n" {
		t.Fatalf("second version set aside: %q", got)
	}
	a.sync()
	b.sync()
	if got := b.read("alpha/fern", "one.md"); got != "from m2\n" {
		t.Fatalf("m2 holds %q after the next syncs, want its own version", got)
	}
}

func TestAFailedSetAsideCommitKeepsAVersionSetAsideBefore(t *testing.T) {
	a, b := pair(t)
	b.write("alpha/fern", "one.md", "m2, first\n")
	b.sync()
	remove := during(a, "printf 'mine\\n' >one.md")
	a.sync()
	remove()
	aside := "orphans/m1/conflicts/alpha/fern/one.md"
	if got := a.store(aside); got != "mine\n" {
		t.Fatalf("set aside by the first sync: %q", got)
	}
	// The session writes the same version again, and another conflict
	// makes the commit that follows necessary; it fails.
	b.write("alpha/fern", "one.md", "m2, second\n")
	b.write("global", "CLAUDE.md", "global from m2\n")
	b.sync()
	hooks := filepath.Join(a.o.StoreDir, ".git", "hooks")
	must(t, os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte("#!/bin/sh\n"+
		"[ -f orphans/m1/conflicts/global/CLAUDE.md ] && exit 1\nexit 0\n"), 0o700))
	remove = during(a, "printf 'mine\\n' >one.md && printf 'global, mine\\n' >'"+
		filepath.Join(a.dir("global"), "CLAUDE.md")+"'")
	_, err := Sync(a.o)
	remove()
	must(t, os.Remove(filepath.Join(hooks, "pre-commit")))
	if err == nil {
		t.Fatal("the sync succeeded although the set-aside commit failed")
	}
	a.sync()
	if got := a.store(aside); got != "mine\n" {
		t.Fatalf("the version set aside before holds %q after the repeated sync", got)
	}
	if got := a.store("orphans/m1/conflicts/global/CLAUDE.md"); got != "global, mine\n" {
		t.Fatalf("set aside on the repeated sync: %q", got)
	}
}

func TestAMemoryRestoredAgainstASessionsDeletionKeepsItsIndexLine(t *testing.T) {
	a, b := pair(t)
	b.write("alpha/fern", "one.md", "from m2\n")
	b.sync()
	remove := during(a, "rm one.md && : >MEMORY.md")
	a.sync()
	remove()
	line := "- [One](one.md) - first\n"
	if got := a.read("alpha/fern", "MEMORY.md"); !strings.Contains(got, line) {
		t.Fatalf("m1's index holds %q, want the restored memory's line", got)
	}
	a.sync()
	b.sync()
	if got := b.read("alpha/fern", "MEMORY.md"); !strings.Contains(got, line) {
		t.Fatalf("m2's index holds %q after the next syncs", got)
	}
}

func TestARestoredMemoryInAnIndexTheSessionRemovedKeepsTheOtherLines(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "two.md", "two\n")
	a.write("alpha/fern", "MEMORY.md", "- [One](one.md) - first\n- [Two](two.md) - second\n")
	a.sync()
	b.sync()
	b.write("alpha/fern", "one.md", "from m2\n")
	b.sync()
	remove := during(a, "rm one.md MEMORY.md")
	a.sync()
	remove()
	a.sync()
	b.sync()
	for _, m := range []machine{a, b} {
		index := m.read("alpha/fern", "MEMORY.md")
		for _, line := range []string{"- [One](one.md) - first\n", "- [Two](two.md) - second\n"} {
			if !strings.Contains(index, line) {
				t.Fatalf("%s's index lacks %q: %q", m.o.Machine, line, index)
			}
		}
	}
}

func TestUnionLinesAddsEachMissingLineOnce(t *testing.T) {
	got := string(unionLines([]byte("a\nb"), []byte("c\nc\n\nb\nd\n")))
	if want := "a\nb\nc\nd\n"; got != want {
		t.Fatalf("unionLines = %q, want %q", got, want)
	}
}

func TestAMemoryDeletedBySessionWhileOtherMachineChangedItIsKept(t *testing.T) {
	a, b := pair(t)
	b.write("alpha/fern", "one.md", "from m2\n")
	b.sync()
	remove := during(a, "rm one.md")
	res := a.sync()
	remove()
	if res.Overridden != 1 {
		t.Fatalf("m1 reported %d deletions undone, want 1", res.Overridden)
	}
	if got := a.read("alpha/fern", "one.md"); got != "from m2\n" {
		t.Fatalf("m1 holds %q, want m2's change", got)
	}
	a.sync()
	b.sync()
	if got := b.read("alpha/fern", "one.md"); got != "from m2\n" {
		t.Fatalf("m2 holds %q after the next syncs, want its own version", got)
	}
}

func TestIndexLinesAddedBySessionAndOtherMachineDuringASyncAreBothKept(t *testing.T) {
	a, b := pair(t)
	b.write("alpha/fern", "two.md", "two\n")
	b.write("alpha/fern", "MEMORY.md", "- [One](one.md) - first\n- [Two](two.md) - second\n")
	b.sync()
	remove := during(a, "printf 'mid\\n' >mid.md && printf -- '- [Mid](mid.md) - mid\\n' >>MEMORY.md")
	a.sync()
	remove()
	a.sync()
	b.sync()
	for _, m := range []machine{a, b} {
		index := m.read("alpha/fern", "MEMORY.md")
		for _, line := range []string{"- [Two](two.md) - second\n", "- [Mid](mid.md) - mid\n"} {
			if !strings.Contains(index, line) {
				t.Fatalf("%s's index lacks %q: %q", m.o.Machine, line, index)
			}
		}
	}
}

func TestSyncRecordsWhereItStopped(t *testing.T) {
	a, _ := pair(t)
	a.write("alpha/fern", "two.md", "two\n")
	a.sync()
	head := git(t, a.o.StoreDir, "rev-parse", "HEAD")
	if a.o.readState("synced") != head {
		t.Fatalf("synced is %q, head is %q", a.o.readState("synced"), head)
	}
	if got := git(t, a.o.StoreDir, "log", "-1", "--format=%s%n%b"); !strings.Contains(got, "sync(m1): 1 files") || !strings.Contains(got, "Signed-off-by:") {
		t.Fatalf("last commit: %q", got)
	}
	if got := git(t, a.o.StoreDir, "symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Fatalf("the checkout is on %q", got)
	}
}
