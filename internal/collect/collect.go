// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// Package collect summarises transcripts and working trees into the
// state repository.
package collect

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aimd54/gellee/internal/gitfacts"
	"github.com/aimd54/gellee/internal/memories"
	"github.com/aimd54/gellee/internal/state"
	"github.com/aimd54/gellee/internal/transcript"
)

// Options configures one run.
type Options struct {
	StateDir       string
	TranscriptsDir string
	CacheDir       string
	Machine        string
	ProjectsRoot   string
	Memories       string // checkout of the memory store; empty without one
	Commit         bool
	Push           bool
	Now            time.Time
	Loc            *time.Location
	Out            io.Writer
	Open           func(string) (io.ReadCloser, error)
}

type cacheEntry struct {
	Size    int64              `json:"size"`
	ModTime int64              `json:"mod_time"`
	Rows    []state.Row        `json:"rows"`
	Skipped int                `json:"skipped"`
	Summary transcript.Summary `json:"summary"`
}

// cache holds the rows computed for each transcript. Key records what
// attribution depended on, so a changed configuration discards it.
type cache struct {
	Key   string                `json:"key"`
	Files map[string]cacheEntry `json:"files"`
}

func loadCache(path, key string) cache {
	c := cache{Key: key, Files: map[string]cacheEntry{}}
	data, err := os.ReadFile(path) //nolint:gosec // the collector's own cache file
	if err != nil {
		return c
	}
	var stored cache
	if json.Unmarshal(data, &stored) == nil && stored.Key == key && stored.Files != nil {
		return stored
	}
	return c
}

func saveCache(path string, c cache) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Run collects once.
func Run(o Options) error {
	cfg, err := state.LoadConfig(o.StateDir)
	if err != nil {
		return err
	}
	root := o.ProjectsRoot
	if root == "" {
		root = cfg.ProjectsRoot
	}
	if root == "" {
		return errors.New("projects root is not set")
	}
	statuses, _ := state.LoadStatuses(o.StateDir, cfg.Roots)
	attr := transcript.Attributor{ProjectsRoot: root, Roots: cfg.Roots}

	cachePath := filepath.Join(o.CacheDir, "transcripts.json")
	// The first field names the layout of an entry.
	key := "ended|" + root + "|" + strings.Join(cfg.Roots, " ") + "|" + o.Loc.String()
	old := loadCache(cachePath, key)
	next := cache{Key: key, Files: map[string]cacheEntry{}}

	files, err := transcript.Find(o.TranscriptsDir)
	if err != nil {
		return err
	}
	onDisk := map[string]bool{}
	failed := map[string]bool{}
	var computed []state.Row
	parsed, skipped := 0, 0
	for _, f := range files {
		onDisk[f.Session] = true
		entry, err := o.entry(f, old, attr)
		if err != nil {
			failed[f.Session] = true
			fmt.Fprintf(o.Out, "unreadable: %s: %v\n", f.Path, err)
			continue
		}
		if _, cached := old.Files[f.Path]; !cached || old.Files[f.Path].ModTime != entry.ModTime || old.Files[f.Path].Size != entry.Size {
			parsed++
		}
		next.Files[f.Path] = entry
		computed = append(computed, entry.Rows...)
		skipped += entry.Skipped
	}

	existing, err := state.MachineRows(o.StateDir, o.Machine)
	if err != nil {
		return err
	}
	// Rows of a session are replaced only when every one of its
	// transcripts was read. Anything else keeps what was recorded.
	var all []state.Row
	for _, r := range existing {
		if !onDisk[r.Session] || failed[r.Session] {
			all = append(all, r)
		}
	}
	for _, r := range computed {
		if !failed[r.Session] {
			all = append(all, r)
		}
	}
	changed, err := state.WriteMachineRows(o.StateDir, o.Machine, state.Rename(all, state.Aliases(statuses)))
	if err != nil {
		return err
	}
	gitFile := state.GitFile{Collected: o.Now.Format(time.RFC3339), Projects: gitfacts.CollectAll(root, cfg.Roots)}
	if err := state.WriteGitFile(o.StateDir, o.Machine, gitFile); err != nil {
		return err
	}
	prev, err := state.ReadMachineFile(o.StateDir, o.Machine)
	if err != nil {
		fmt.Fprintf(o.Out, "collect: %v, its sessions are listed anew\n", err)
	}
	summary := o.machineFile(files, next.Files, failed, attr, state.Aliases(statuses), prev)
	summary.Memories = o.memoryState()
	if err := state.WriteMachineFile(o.StateDir, o.Machine, summary); err != nil {
		return err
	}
	if err := saveCache(cachePath, next); err != nil {
		return err
	}
	fmt.Fprintf(o.Out, "collect: %d transcripts, %d parsed, %d lines skipped, %d usage files written, %d working trees\n",
		len(files), parsed, skipped, len(changed), len(gitFile.Projects))

	if o.Commit {
		if err := o.commit(); err != nil {
			return err
		}
	}
	if o.Push {
		if err := o.push(); err != nil {
			return err
		}
	}
	switch len(failed) {
	case 0:
		return nil
	case 1:
		return errors.New("1 session could not be read")
	}
	return fmt.Errorf("%d sessions could not be read", len(failed))
}

// memoryState is this machine's memory store health, or nil without a
// store. A store that cannot be read is reported and left out.
func (o Options) memoryState() *state.MachineMemories {
	if o.Memories == "" {
		return nil
	}
	pushed, aside, err := memories.Health(memories.Options{StoreDir: o.Memories, Machine: o.Machine})
	if err != nil {
		fmt.Fprintf(o.Out, "collect: memory store: %v\n", err)
		return nil
	}
	return &state.MachineMemories{Pushed: pushed, SetAside: aside}
}

// entry returns the cached rows of a transcript, or parses it.
func (o Options) entry(f transcript.File, old cache, attr transcript.Attributor) (cacheEntry, error) {
	info, err := os.Stat(f.Path)
	if err != nil {
		return cacheEntry{}, err
	}
	if e, ok := old.Files[f.Path]; ok && e.Size == info.Size() && e.ModTime == info.ModTime().UnixNano() {
		return e, nil
	}
	rc, err := o.Open(f.Path)
	if err != nil {
		return cacheEntry{}, err
	}
	defer rc.Close() //nolint:errcheck // read only
	rows, sum, skipped, err := transcript.Parse(rc, f.Session, attr, o.Loc)
	if err != nil {
		return cacheEntry{}, err
	}
	return cacheEntry{Size: info.Size(), ModTime: info.ModTime().UnixNano(), Rows: rows, Skipped: skipped, Summary: sum}, nil
}

func (o Options) git(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", o.StateDir}, args...)...) //nolint:gosec // fixed binary; arguments are constants and the machine name
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// commit stages and commits the collector's own paths and nothing else.
func (o Options) commit() error {
	var paths []string
	for _, p := range []string{
		filepath.Join("usage", o.Machine),
		filepath.Join("git", o.Machine+".json"),
		filepath.Join("machines", o.Machine+".json"),
	} {
		if _, err := os.Stat(filepath.Join(o.StateDir, p)); err == nil {
			paths = append(paths, p)
		}
	}
	if _, err := o.git(append([]string{"add", "--"}, paths...)...); err != nil {
		return err
	}
	if _, err := o.git(append([]string{"diff", "--cached", "--quiet", "--"}, paths...)...); err == nil {
		fmt.Fprintln(o.Out, "collect: nothing to commit")
		return nil
	}
	_, err := o.git(append([]string{"commit", "-q", "-s", "-m", "chore(" + o.Machine + "): collect", "--"}, paths...)...)
	return err
}

// push publishes the state repository, rebasing once when the remote has
// commits from another machine.
func (o Options) push() error {
	remotes, err := o.git("remote")
	if err != nil {
		return err
	}
	if remotes == "" {
		fmt.Fprintln(o.Out, "collect: the state repository has no remote, nothing pushed")
		return nil
	}
	if _, err := o.git("push", "-q"); err == nil {
		return nil
	}
	if _, err := o.git("pull", "-q", "--rebase", "--autostash"); err != nil {
		return err
	}
	_, err = o.git("push", "-q")
	return err
}
