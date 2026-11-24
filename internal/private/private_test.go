// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package private

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

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

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func isLink(t *testing.T, path string) bool {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode()&os.ModeSymlink != 0
}

// fixture is a state directory naming the roots alpha and beta, and a
// checkout of alpha/fern that ignores *.private.md.
type fixture struct{ state, projects, top string }

func setup(t *testing.T) fixture {
	t.Helper()
	gitEnv(t)
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{
		state:    filepath.Join(base, "state"),
		projects: filepath.Join(base, "projects"),
		top:      filepath.Join(base, "projects", "alpha", "fern"),
	}
	write(t, filepath.Join(f.state, "config"), "roots=alpha beta\n")
	write(t, filepath.Join(f.top, ".gitignore"), "*.private.md\n")
	git(t, f.top, "init", "-q", "-b", "main")
	git(t, f.top, "add", ".gitignore")
	git(t, f.top, "commit", "-q", "-m", "init")
	return f
}

func (f fixture) locate(t *testing.T, dir string) Checkout {
	t.Helper()
	c, err := Locate(f.state, f.projects, dir)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLocateFindsTheKeyFromASubdirectory(t *testing.T) {
	f := setup(t)
	write(t, filepath.Join(f.top, "docs", "x"), "x")
	c := f.locate(t, filepath.Join(f.top, "docs"))
	if c.Key != "alpha/fern" || c.Top != f.top {
		t.Fatalf("got %+v", c)
	}
	if c.Store != filepath.Join(f.state, "private", "alpha", "fern") {
		t.Fatalf("store %s", c.Store)
	}
}

func TestLocateFindsTheKeyFromAWorktree(t *testing.T) {
	f := setup(t)
	wt := filepath.Join(f.top, ".claude", "worktrees", "w1")
	git(t, f.top, "worktree", "add", "-q", wt)
	c := f.locate(t, wt)
	if c.Key != "alpha/fern" || c.Top != wt {
		t.Fatalf("got %+v", c)
	}
}

func TestLocateRefusesADirectoryOutsideTheRoots(t *testing.T) {
	f := setup(t)
	other := filepath.Join(f.projects, "gamma", "pond")
	write(t, filepath.Join(other, "f"), "f")
	git(t, other, "init", "-q")
	if _, err := Locate(f.state, f.projects, other); !errors.Is(err, ErrOutside) {
		t.Fatalf("err %v, want ErrOutside", err)
	}
}

func TestSweepMovesIgnoredPrivateFilesAndLinksThem(t *testing.T) {
	f := setup(t)
	write(t, filepath.Join(f.top, "docs", "new", "a.private.md"), "plan\n")
	write(t, filepath.Join(f.top, "kept.private.md"), "tracked\n")
	git(t, f.top, "add", "-f", "kept.private.md")
	git(t, f.top, "commit", "-q", "-m", "tracked")
	write(t, filepath.Join(f.top, "c.private.txt"), "loose\n")
	c := f.locate(t, f.top)

	r, err := Sweep(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Swept, []string{"docs/new/a.private.md"}) {
		t.Fatalf("swept %v", r.Swept)
	}
	moved := filepath.Join(f.top, "docs", "new", "a.private.md")
	if !isLink(t, moved) || read(t, moved) != "plan\n" {
		t.Fatal("a.private.md is not a link to its content")
	}
	if read(t, filepath.Join(c.Store, "docs", "new", "a.private.md")) != "plan\n" {
		t.Fatal("the stored copy is wrong")
	}
	if isLink(t, filepath.Join(f.top, "kept.private.md")) {
		t.Fatal("a tracked private file was moved")
	}
	if !slices.Equal(r.NotIgnored, []string{"c.private.txt"}) {
		t.Fatalf("not ignored %v", r.NotIgnored)
	}
	if isLink(t, filepath.Join(f.top, "c.private.txt")) {
		t.Fatal("a file git does not ignore was moved")
	}
}

func TestLinkInANewWorktreeLinksOnce(t *testing.T) {
	f := setup(t)
	write(t, filepath.Join(f.top, "docs", "a.private.md"), "plan\n")
	if _, err := Sweep(f.locate(t, f.top)); err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(f.top, ".claude", "worktrees", "w1")
	git(t, f.top, "worktree", "add", "-q", wt)
	c := f.locate(t, wt)

	r, err := Link(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Linked, []string{"docs/a.private.md"}) {
		t.Fatalf("linked %v", r.Linked)
	}
	if read(t, filepath.Join(wt, "docs", "a.private.md")) != "plan\n" {
		t.Fatal("the worktree's link does not reach the content")
	}
	again, err := Link(c)
	if err != nil {
		t.Fatal(err)
	}
	if again.Summary() != "nothing to report" {
		t.Fatalf("second link: %s", again.Summary())
	}
}

func TestLinkLeavesARegularFileInTheWay(t *testing.T) {
	f := setup(t)
	c := f.locate(t, f.top)
	write(t, filepath.Join(c.Store, "a.private.md"), "stored\n")
	write(t, filepath.Join(f.top, "a.private.md"), "local\n")

	r, err := Link(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.InTheWay, []string{"a.private.md"}) || isLink(t, filepath.Join(f.top, "a.private.md")) {
		t.Fatalf("link: %+v", r)
	}
	r, err = Sweep(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.InTheWay, []string{"a.private.md"}) || read(t, filepath.Join(f.top, "a.private.md")) != "local\n" {
		t.Fatalf("sweep with different bytes: %+v", r)
	}
	write(t, filepath.Join(f.top, "a.private.md"), "stored\n")
	r, err = Sweep(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Swept, []string{"a.private.md"}) || !isLink(t, filepath.Join(f.top, "a.private.md")) {
		t.Fatalf("sweep with equal bytes: %+v", r)
	}
}

func TestStatusListsWhatIsNotInOrder(t *testing.T) {
	f := setup(t)
	c := f.locate(t, f.top)
	write(t, filepath.Join(f.top, "loose.private.md"), "loose\n")
	write(t, filepath.Join(c.Store, "docs", "far.private.md"), "far\n")
	if err := os.Symlink(filepath.Join(c.Store, "gone.private.md"), filepath.Join(f.top, "gone.private.md")); err != nil {
		t.Fatal(err)
	}

	r, err := Status(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Unswept, []string{"loose.private.md"}) ||
		!slices.Equal(r.Missing, []string{"docs/far.private.md"}) ||
		!slices.Equal(r.Broken, []string{"gone.private.md"}) {
		t.Fatalf("status %+v", r)
	}
	if r.Ready() {
		t.Fatal("ready with problems")
	}
	if _, err := os.Lstat(filepath.Join(f.top, "docs", "far.private.md")); err == nil {
		t.Fatal("status made a link")
	}
	if !strings.Contains(r.Summary(), "1 unswept") {
		t.Fatalf("summary %q", r.Summary())
	}
}

func TestStoredRefusesAPathLeavingTheStore(t *testing.T) {
	f := setup(t)
	c := f.locate(t, f.top)
	if _, err := c.stored("../other/x.private.md"); err == nil {
		t.Fatal("a path leaving the store was accepted")
	}
}

// commitState makes the state directory a repository holding its files.
func commitState(t *testing.T, f fixture) {
	t.Helper()
	git(t, f.state, "init", "-q", "-b", "main")
	git(t, f.state, "add", ".")
	git(t, f.state, "commit", "-q", "-m", "state")
}

func TestSweepRefusesAStoredCopyThatIsALink(t *testing.T) {
	f := setup(t)
	c := f.locate(t, f.top)
	src := filepath.Join(f.top, "a.private.md")
	write(t, src, "only copy\n")
	if err := os.MkdirAll(c.Store, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(src, filepath.Join(c.Store, "a.private.md")); err != nil {
		t.Fatal(err)
	}
	r, err := Sweep(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Swept) != 0 || !slices.Equal(r.InTheWay, []string{"a.private.md"}) {
		t.Fatalf("sweep %+v", r)
	}
	if isLink(t, src) || read(t, src) != "only copy\n" {
		t.Fatal("the only copy was replaced")
	}
}

func TestSweepFindsFilesInIgnoredDirectories(t *testing.T) {
	f := setup(t)
	write(t, filepath.Join(f.top, ".gitignore"), "*.private.md\nnotes/\n")
	write(t, filepath.Join(f.top, "notes", "deep", "n.private.md"), "n\n")
	r, err := Sweep(f.locate(t, f.top))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Swept, []string{"notes/deep/n.private.md"}) {
		t.Fatalf("swept %v", r.Swept)
	}
}

func TestLinkRepairsALinkIntoAnotherStoreLocation(t *testing.T) {
	f := setup(t)
	c := f.locate(t, f.top)
	write(t, filepath.Join(c.Store, "a.private.md"), "a\n")
	at := filepath.Join(f.top, "a.private.md")
	if err := os.Symlink(filepath.Join(f.state, "private", "alpha", "old", "a.private.md"), at); err != nil {
		t.Fatal(err)
	}
	r, err := Link(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Linked, []string{"a.private.md"}) || read(t, at) != "a\n" {
		t.Fatalf("link %+v", r)
	}
}

func TestSweepTakesANewerCopySavedOverItsLink(t *testing.T) {
	f := setup(t)
	c := f.locate(t, f.top)
	at := filepath.Join(f.top, "a.private.md")
	write(t, at, "v1\n")
	if _, err := Sweep(c); err != nil {
		t.Fatal(err)
	}
	commitState(t, f)
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(c.Store, "a.private.md"), old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(at); err != nil {
		t.Fatal(err)
	}
	write(t, at, "v2\n")
	r, err := Sweep(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Replaced, []string{"a.private.md"}) || len(r.Swept) != 0 ||
		read(t, filepath.Join(c.Store, "a.private.md")) != "v2\n" || !isLink(t, at) {
		t.Fatalf("sweep %+v", r)
	}
}

func TestSweepKeepsTheStoreOverAnOlderLocalCopy(t *testing.T) {
	f := setup(t)
	c := f.locate(t, f.top)
	at := filepath.Join(f.top, "a.private.md")
	write(t, at, "stale\n")
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(at, old, old); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(c.Store, "a.private.md"), "current\n")
	commitState(t, f)
	r, err := Sweep(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.InTheWay, []string{"a.private.md"}) || read(t, filepath.Join(c.Store, "a.private.md")) != "current\n" {
		t.Fatalf("sweep %+v", r)
	}
}

func TestLocateRefusesANestedRepository(t *testing.T) {
	f := setup(t)
	nested := filepath.Join(f.top, "vendor", "x")
	write(t, filepath.Join(nested, "f"), "f")
	git(t, nested, "init", "-q")
	if _, err := Locate(f.state, f.projects, nested); !errors.Is(err, ErrOutside) {
		t.Fatalf("err %v, want ErrOutside", err)
	}
}

func TestSweepReportsABrokenLink(t *testing.T) {
	f := setup(t)
	c := f.locate(t, f.top)
	if err := os.Symlink(filepath.Join(f.top, "nowhere"), filepath.Join(f.top, "gone.private.md")); err != nil {
		t.Fatal(err)
	}
	r, err := Sweep(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Broken, []string{"gone.private.md"}) {
		t.Fatalf("sweep %+v", r)
	}
}

func TestSweepReportsAHardLinkedCopyOnce(t *testing.T) {
	f := setup(t)
	c := f.locate(t, f.top)
	src := filepath.Join(f.top, "a.private.md")
	write(t, src, "a\n")
	if err := os.MkdirAll(c.Store, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(src, filepath.Join(c.Store, "a.private.md")); err != nil {
		t.Fatal(err)
	}
	r, err := Sweep(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.InTheWay, []string{"a.private.md"}) || isLink(t, src) {
		t.Fatalf("sweep %+v", r)
	}
}

func TestLinkSkipsLeftoverTemporaryFiles(t *testing.T) {
	f := setup(t)
	c := f.locate(t, f.top)
	write(t, filepath.Join(c.Store, "a.private.md.gellee-tmp"), "half\n")
	write(t, filepath.Join(f.top, "b.private.md.gellee-link"), "x\n")
	r, err := Sweep(c)
	if err != nil {
		t.Fatal(err)
	}
	if r.Summary() != "nothing to report" {
		t.Fatalf("sweep %+v", r)
	}
}
