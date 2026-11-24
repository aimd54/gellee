// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestEncodeReplacesSlashesAndDots(t *testing.T) {
	got := Encode("/home/u/projects/alpha/site.example")
	if got != "-home-u-projects-alpha-site-example" {
		t.Fatalf("got %q", got)
	}
}

func TestUnitsNameTheMemoryDirectoryOfEveryProject(t *testing.T) {
	a, _ := world(t)
	us, err := a.o.units()
	must(t, err)
	var keys []string
	for _, u := range us {
		keys = append(keys, u.Key)
	}
	want := []string{"global", "alpha/fern", "alpha/pond", "beta/hill"}
	if len(keys) != len(want) {
		t.Fatalf("keys %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys %v, want %v", keys, want)
		}
	}
	root, err := filepath.EvalSymlinks(a.o.ProjectsRoot)
	must(t, err)
	fern := filepath.Join(a.o.ClaudeDir, "projects", Encode(filepath.Join(root, "alpha", "fern")), "memory")
	if us[1].Local != fern {
		t.Fatalf("alpha/fern is mirrored from %q, want %q", us[1].Local, fern)
	}
	if us[0].Local != a.o.ClaudeDir || len(us[0].Only) != 1 || us[0].Only[0] != "CLAUDE.md" {
		t.Fatalf("global unit %+v", us[0])
	}
}

func TestUnitsSkipARootThatDoesNotExist(t *testing.T) {
	a, _ := world(t)
	must(t, os.RemoveAll(filepath.Join(a.o.ProjectsRoot, "alpha")))
	us, err := a.o.units()
	must(t, err)
	var keys []string
	for _, u := range us {
		keys = append(keys, u.Key)
	}
	if len(keys) != 2 || keys[0] != "global" || keys[1] != "beta/hill" {
		t.Fatalf("keys %v, want [global beta/hill]", keys)
	}
}

func skipAsRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("running as root: a directory without read permission is still readable, so the error cannot be provoked")
	}
}

func TestUnitsReturnAnErrorForARootThatCannotBeListed(t *testing.T) {
	skipAsRoot(t)
	a, _ := world(t)
	alpha := filepath.Join(a.o.ProjectsRoot, "alpha")
	must(t, os.Chmod(alpha, 0o000))
	t.Cleanup(func() { _ = os.Chmod(alpha, 0o700) })
	us, err := a.o.units()
	if err == nil {
		t.Fatalf("units listed %d entries and no error for an unreadable root", len(us))
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v, want a permission error", err)
	}
}

func TestFilesMirrorAndCopyBackReturnAnErrorForADirectoryThatCannotBeRead(t *testing.T) {
	skipAsRoot(t)
	dst := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dst, "b.md"), []byte("y"), 0o600))
	must(t, os.Chmod(dst, 0o000))
	t.Cleanup(func() { _ = os.Chmod(dst, 0o700) })
	if _, err := (unit{Key: "k"}).files(dst); err == nil {
		t.Fatal("files: no error for an unreadable directory")
	}
	want := map[string][]byte{"a.md": []byte("x")}
	if _, err := (unit{Key: "k"}).mirror(want, dst); err == nil {
		t.Fatal("mirror: no error for an unreadable destination")
	}
	if _, err := (unit{Key: "k", Local: dst}).copyBack(want, nil); err == nil {
		t.Fatal("copyBack: no error for an unreadable local directory")
	}
	must(t, os.Chmod(dst, 0o700))
	if data, _ := os.ReadFile(filepath.Join(dst, "b.md")); string(data) != "y" {
		t.Fatalf("b.md holds %q: the destination was changed despite the error", data)
	}
}

func TestFilesOfAMissingDirectoryIsEmpty(t *testing.T) {
	got, err := (unit{Key: "k"}).files(filepath.Join(t.TempDir(), "absent"))
	must(t, err)
	if got == nil || len(got) != 0 {
		t.Fatalf("got %v, want an empty map", got)
	}
}

func TestMirrorCopiesWhatDiffersAndRemovesWhatIsGone(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	must(t, os.WriteFile(filepath.Join(src, "a.md"), []byte("new"), 0o600))
	must(t, os.WriteFile(filepath.Join(src, "same.md"), []byte("same"), 0o600))
	must(t, os.WriteFile(filepath.Join(src, ".hidden"), []byte("x"), 0o600))
	must(t, os.WriteFile(filepath.Join(dst, "a.md"), []byte("old"), 0o600))
	must(t, os.WriteFile(filepath.Join(dst, "same.md"), []byte("same"), 0o600))
	must(t, os.WriteFile(filepath.Join(dst, "gone.md"), []byte("x"), 0o600))
	want, err := unit{Key: "k"}.files(src)
	must(t, err)
	n, err := unit{Key: "k"}.mirror(want, dst)
	must(t, err)
	if n != 2 {
		t.Fatalf("%d files changed, want 2", n)
	}
	if data, _ := os.ReadFile(filepath.Join(dst, "a.md")); string(data) != "new" {
		t.Fatalf("a.md holds %q", data)
	}
	if _, err := os.Stat(filepath.Join(dst, "gone.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("gone.md: %v", err)
	}
	entries, err := os.ReadDir(dst)
	must(t, err)
	if len(entries) != 2 {
		t.Fatalf("%d entries in the destination, want a.md and same.md", len(entries))
	}
}

func TestMirrorOfARestrictedUnitLeavesOtherFilesAlone(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	must(t, os.WriteFile(filepath.Join(src, "CLAUDE.md"), []byte("rules"), 0o600))
	must(t, os.WriteFile(filepath.Join(dst, "settings.json"), []byte("{}"), 0o600))
	global := unit{Key: globalKey, Only: []string{"CLAUDE.md"}}
	want, err := global.files(src)
	must(t, err)
	n, err := global.mirror(want, dst)
	must(t, err)
	if n != 1 {
		t.Fatalf("%d files changed, want 1", n)
	}
	if data, _ := os.ReadFile(filepath.Join(dst, "settings.json")); string(data) != "{}" {
		t.Fatalf("settings.json holds %q", data)
	}
	if data, _ := os.ReadFile(filepath.Join(dst, "CLAUDE.md")); string(data) != "rules" {
		t.Fatalf("CLAUDE.md holds %q", data)
	}
}

func TestLockRefusesASecondHolder(t *testing.T) {
	a, _ := world(t)
	unlock, err := a.o.lock()
	must(t, err)
	if _, err := a.o.lock(); !errors.Is(err, ErrLocked) {
		t.Fatalf("second lock: %v, want ErrLocked", err)
	}
	unlock()
	again, err := a.o.lock()
	must(t, err)
	again()
}

func TestLockReturnsAFlockErrorOtherThanContention(t *testing.T) {
	a, _ := world(t)
	real := flock
	flock = func(int, int) error { return syscall.EBADF }
	t.Cleanup(func() { flock = real })
	_, err := a.o.lock()
	if !errors.Is(err, syscall.EBADF) || errors.Is(err, ErrLocked) {
		t.Fatalf("lock: %v, want EBADF and not ErrLocked", err)
	}
}

func TestLockReturnsOpenErrorsAsTheyAre(t *testing.T) {
	skipAsRoot(t)
	a, _ := world(t)
	dir := filepath.Join(a.o.StoreDir, ".git", "gellee")
	must(t, os.MkdirAll(dir, 0o700))
	must(t, os.Chmod(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	_, err := a.o.lock()
	if !errors.Is(err, os.ErrPermission) || errors.Is(err, ErrLocked) {
		t.Fatalf("lock in a read-only directory: %v, want a permission error", err)
	}
}
