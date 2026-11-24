// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package gitfacts

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/aimd54/gellee/internal/state"
)

// run executes git in dir with a fixed identity and date.
func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.invalid",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.invalid",
		"GIT_AUTHOR_DATE=2026-11-10T12:00:00+01:00", "GIT_COMMITTER_DATE=2026-11-10T12:00:00+01:00")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func repo(t *testing.T, dir string, commits int) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "init", "-q", "-b", "main")
	for i := 0; i < commits; i++ {
		run(t, dir, "commit", "-q", "--allow-empty", "-m", "c")
	}
}

func TestCollectWithoutARemote(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "w")
	repo(t, dir, 2)
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, ok := Collect(dir)
	if !ok {
		t.Fatal("a repository must be collected")
	}
	if len(got.Head) < 7 {
		t.Fatalf("head %q", got.Head)
	}
	got.Head = ""
	want := state.GitProject{Branch: "main", LastCommit: "2026-11-10T12:00:00+01:00", Remote: false, Ahead: 2, Dirty: 2}
	if got != want {
		t.Fatalf("got %+v", got)
	}
}

func TestCollectCountsCommitsAheadOfTheUpstream(t *testing.T) {
	tmp := t.TempDir()
	bare := filepath.Join(tmp, "remote.git")
	run(t, tmp, "init", "-q", "--bare", "-b", "main", bare)
	dir := filepath.Join(tmp, "w")
	repo(t, dir, 1)
	run(t, dir, "remote", "add", "origin", bare)
	run(t, dir, "push", "-q", "-u", "origin", "main")
	run(t, dir, "commit", "-q", "--allow-empty", "-m", "local")
	got, _ := Collect(dir)
	if !got.Remote || got.Ahead != 1 || got.Dirty != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestCollectHandlesARepositoryWithoutCommits(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "w")
	repo(t, dir, 0)
	got, ok := Collect(dir)
	want := state.GitProject{Branch: "main"}
	if !ok || got != want {
		t.Fatalf("got %+v ok %v", got, ok)
	}
}

func TestCollectAllSkipsWhatIsNotARepository(t *testing.T) {
	root := t.TempDir()
	repo(t, filepath.Join(root, "alpha/fern"), 1)
	repo(t, filepath.Join(root, "ignored/thing"), 1)
	if err := os.MkdirAll(filepath.Join(root, "alpha/plain"), 0o750); err != nil {
		t.Fatal(err)
	}
	got := CollectAll(root, []string{"alpha", "missing"})
	if len(got) != 1 || got["alpha/fern"].Branch != "main" || got["alpha/fern"].Ahead != 1 {
		t.Fatalf("got %+v", got)
	}
}
