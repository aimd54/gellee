// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// Package memories carries Claude Code's memory files and the global
// instructions file between machines, through a git checkout of a
// private store.
package memories

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/aimd54/gellee/internal/state"
)

// Options says where everything is.
type Options struct {
	StoreDir     string        // checkout of the store
	StateDir     string        // state repository, read for its config
	ProjectsRoot string        // overrides projects_root of the config
	ClaudeDir    string        // Claude Code's directory, ~/.claude
	Machine      string        // name of this machine in the store
	Timeout      time.Duration // time allowed for a fetch or a push
	Now          time.Time
	Out          io.Writer
}

// globalKey is the store directory of the global instructions file.
const globalKey = "global"

// ErrLocked is returned while another memories command holds the lock.
var ErrLocked = errors.New("another memories command is running")

// Encode returns the name Claude Code gives the directory of a project
// path: every slash and every dot becomes a hyphen.
func Encode(path string) string {
	return strings.NewReplacer("/", "-", ".", "-").Replace(path)
}

// unit is one directory of the store and the local directory it mirrors.
type unit struct {
	Key   string   // directory in the store: a project key, or "global"
	Local string   // directory on this machine
	Only  []string // when set, the only file names that belong to it
	Links bool     // a symbolic link in Local is read and written through
}

// units lists the global file, then every project directory one level
// below each configured root, in the order of the configuration. A root
// that does not exist holds no projects.
func (o Options) units() ([]unit, error) {
	cfg, err := state.LoadConfig(o.StateDir)
	if err != nil {
		return nil, err
	}
	root := o.ProjectsRoot
	if root == "" {
		root = cfg.ProjectsRoot
	}
	if root == "" {
		return nil, errors.New("projects root is not set")
	}
	// Claude Code names a memory directory after the resolved path.
	if root, err = filepath.EvalSymlinks(root); err != nil {
		return nil, err
	}
	// The global file is often a link into a repository of dotfiles.
	us := []unit{{Key: globalKey, Local: o.ClaudeDir, Only: []string{"CLAUDE.md"}, Links: true}}
	for _, r := range cfg.Roots {
		entries, err := os.ReadDir(filepath.Join(root, r))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(root, r, e.Name())
			us = append(us, unit{
				Key:   r + "/" + e.Name(),
				Local: filepath.Join(o.ClaudeDir, "projects", Encode(dir), "memory"),
			})
		}
	}
	return us, nil
}

// checkout fails unless StoreDir is a git working tree.
func (o Options) checkout() error {
	if o.StoreDir == "" {
		return errors.New("the memory store is not set")
	}
	if _, err := os.Stat(filepath.Join(o.StoreDir, ".git")); err != nil {
		return fmt.Errorf("%s is not a checkout of the memory store", o.StoreDir)
	}
	return nil
}

func (o Options) statePath(name string) string {
	return filepath.Join(o.StoreDir, ".git", "gellee", name)
}

// readState returns a per-machine state file, or "" when it is absent.
func (o Options) readState(name string) string {
	data, err := os.ReadFile(o.statePath(name)) //nolint:gosec // a fixed name below the checkout
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (o Options) writeState(name, value string) error {
	if err := os.MkdirAll(filepath.Dir(o.statePath(name)), 0o700); err != nil {
		return err
	}
	return os.WriteFile(o.statePath(name), []byte(value+"\n"), 0o600)
}

// adopted is the set of units this machine has joined.
func (o Options) adopted() map[string]bool {
	m := map[string]bool{}
	for _, line := range strings.Split(o.readState("adopted"), "\n") {
		if line != "" {
			m[line] = true
		}
	}
	return m
}

func (o Options) setAdopted(m map[string]bool) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return o.writeState("adopted", strings.Join(keys, "\n"))
}

// flock is the system call lock makes; tests replace it to provoke errors
// a second holder cannot.
var flock = syscall.Flock

// lock takes the per-checkout lock and returns what releases it. The
// kernel drops it when the process ends, so a crash leaves nothing
// behind.
func (o Options) lock() (func(), error) {
	path := o.statePath("lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // a fixed name below the checkout
	if err != nil {
		return nil, err
	}
	if err := flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil { //nolint:gosec // a file descriptor fits an int
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLocked
		}
		return nil, err
	}
	return func() { _ = f.Close() }, nil
}
