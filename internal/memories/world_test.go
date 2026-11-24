// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// machine is one working machine: its options and its files.
type machine struct {
	t *testing.T
	o Options
}

// world builds a bare forge, a state directory whose config names the
// roots alpha and beta, and two machines, m1 and m2. Each has a clone of
// the forge, a projects root holding alpha/fern, alpha/pond and
// beta/hill, and an empty Claude directory.
func world(t *testing.T) (a, b machine) {
	t.Helper()
	gitEnv(t)
	tmp := t.TempDir()
	forge := filepath.Join(tmp, "forge.git")
	git(t, tmp, "init", "-q", "--bare", "-b", "main", forge)
	stateDir := filepath.Join(tmp, "state")
	must(t, os.MkdirAll(stateDir, 0o700))
	must(t, os.WriteFile(filepath.Join(stateDir, "config"), []byte("roots=alpha beta\n"), 0o600))
	mk := func(name string) machine {
		home := filepath.Join(tmp, name)
		for _, p := range []string{"alpha/fern", "alpha/pond", "beta/hill"} {
			must(t, os.MkdirAll(filepath.Join(home, "projects", p), 0o700))
		}
		must(t, os.MkdirAll(filepath.Join(home, "claude", "projects"), 0o700))
		store := filepath.Join(home, "store")
		git(t, tmp, "clone", "-q", forge, store)
		git(t, store, "symbolic-ref", "HEAD", "refs/heads/main")
		return machine{t: t, o: Options{
			StoreDir: store, StateDir: stateDir,
			ProjectsRoot: filepath.Join(home, "projects"),
			ClaudeDir:    filepath.Join(home, "claude"),
			Machine:      name, Timeout: 10 * time.Second,
			Now: time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC), Out: io.Discard,
		}}
	}
	return mk("m1"), mk("m2")
}

// dir is the local directory of a unit: a project key, or "global".
func (m machine) dir(key string) string {
	m.t.Helper()
	us, err := m.o.units()
	must(m.t, err)
	for _, u := range us {
		if u.Key == key {
			return u.Local
		}
	}
	m.t.Fatalf("no unit %q", key)
	return ""
}

func (m machine) write(key, name, content string) {
	m.t.Helper()
	must(m.t, os.MkdirAll(m.dir(key), 0o700))
	must(m.t, os.WriteFile(filepath.Join(m.dir(key), name), []byte(content), 0o600))
}

func (m machine) read(key, name string) string {
	m.t.Helper()
	data, err := os.ReadFile(filepath.Join(m.dir(key), name))
	if err != nil {
		m.t.Fatalf("%s: %s/%s: %v", m.o.Machine, key, name, err)
	}
	return string(data)
}

func (m machine) has(key, name string) bool {
	_, err := os.Stat(filepath.Join(m.dir(key), name))
	return err == nil
}

// store is the content of a file of the checkout, or "" when it is not
// there.
func (m machine) store(path string) string {
	data, err := os.ReadFile(filepath.Join(m.o.StoreDir, path))
	if err != nil {
		return ""
	}
	return string(data)
}

func (m machine) adopt() {
	m.t.Helper()
	if _, err := Adopt(m.o, true); err != nil {
		m.t.Fatalf("%s: adopt: %v", m.o.Machine, err)
	}
}

// tree is every file below dir, by path, except git's own.
func tree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	must(t, filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		out[rel] = string(data)
		return nil
	}))
	return out
}

func (m machine) sync() Result {
	m.t.Helper()
	res, err := Sync(m.o)
	if err != nil {
		m.t.Fatalf("%s: sync: %v", m.o.Machine, err)
	}
	return res
}

// hook installs a git hook in the machine's checkout, a shell script
// that git runs at that step of a sync, and returns what removes it.
func (m machine) hook(name, script string) (remove func()) {
	m.t.Helper()
	path := filepath.Join(m.o.StoreDir, ".git", "hooks", name)
	must(m.t, os.MkdirAll(filepath.Dir(path), 0o700))
	must(m.t, os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o700))
	return func() { must(m.t, os.Remove(path)) }
}

// pair returns two machines that have both joined a store holding one
// memory of alpha/fern and its index.
func pair(t *testing.T) (a, b machine) {
	t.Helper()
	a, b = world(t)
	a.write("alpha/fern", "one.md", "one\n")
	a.write("alpha/fern", "MEMORY.md", "- [One](one.md) - first\n")
	a.adopt()
	b.adopt()
	return a, b
}
