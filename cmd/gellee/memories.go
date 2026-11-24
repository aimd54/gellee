// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aimd54/gellee/internal/memories"
)

const memoriesUsage = `usage: gellee memories <command> [flags]

commands:
  adopt     join this machine to the memory store; --apply to do it
  sync      exchange this machine's memories with the store
  list      list the memories of the store; --json for the skill
  notices   the lines the session-start hook shows for a project
`

func runMemories(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(errw, memoriesUsage)
		return 2
	}
	switch args[0] {
	case "adopt":
		return runAdopt(args[1:], out, errw)
	case "sync":
		return runSync(args[1:], out, errw)
	case "list":
		return runList(args[1:], out, errw)
	case "notices":
		return runNotices(args[1:], out, errw)
	default:
		fmt.Fprintf(errw, "gellee: unknown memories command %q\n\n%s", args[0], memoriesUsage)
		return 2
	}
}

// memoryCmd is a memories subcommand: its flag set, and what turns the
// common flags into options once they are parsed.
type memoryCmd struct {
	fs      *flag.FlagSet
	options func() memories.Options
}

func newMemoryCmd(name string, timeout time.Duration, out, errw io.Writer) memoryCmd {
	fs := flag.NewFlagSet("memories "+name, flag.ContinueOnError)
	fs.SetOutput(errw)
	home, _ := os.UserHomeDir()
	store := fs.String("store", os.Getenv("GELLEE_MEMORIES"), "checkout of the memory store")
	stateDir := fs.String("state", os.Getenv("GELLEE_STATE"), "state repository")
	projects := fs.String("projects", os.Getenv("GELLEE_PROJECTS_ROOT"), "projects root")
	claude := fs.String("claude", filepath.Join(home, ".claude"), "Claude Code's directory")
	machine := fs.String("machine", machineName(), "name of this machine in the store")
	wait := fs.Duration("timeout", timeout, "how long a fetch or a push may take")
	return memoryCmd{fs: fs, options: func() memories.Options {
		return memories.Options{
			StoreDir: *store, StateDir: *stateDir, ProjectsRoot: *projects,
			ClaudeDir: *claude, Machine: *machine, Timeout: *wait,
			Now: time.Now(), Out: out,
		}
	}}
}

// parse reads the arguments and checks what every subcommand needs. A
// code other than 0 is the exit code.
func (c memoryCmd) parse(args []string, errw io.Writer) (memories.Options, int) {
	if err := c.fs.Parse(args); err != nil {
		return memories.Options{}, 2
	}
	o := c.options()
	if o.StoreDir == "" {
		fmt.Fprintln(errw, "gellee: --store or GELLEE_MEMORIES is required")
		return o, 2
	}
	if o.Machine == "" {
		fmt.Fprintln(errw, "gellee: --machine or GELLEE_MACHINE is required")
		return o, 2
	}
	return o, 0
}

func runAdopt(args []string, out, errw io.Writer) int {
	c := newMemoryCmd("adopt", time.Minute, out, errw)
	apply := c.fs.Bool("apply", false, "make the changes; without it, only print them")
	o, code := c.parse(args, errw)
	if code != 0 {
		return code
	}
	actions, err := memories.Adopt(o, *apply)
	if err != nil {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	counts := map[string]int{}
	for _, a := range actions {
		fmt.Fprintf(out, "%-10s  %s/%s\n", a.What, a.Key, a.Name)
		counts[a.What]++
	}
	fmt.Fprintf(out, "adopt: %d to store, %d to machine, %d conflicts, %d orphans\n",
		counts["to store"], counts["to machine"], counts["conflict"], counts["orphan"])
	if !*apply {
		fmt.Fprintln(out, "adopt: nothing changed; run again with --apply to do it")
	}
	return 0
}

func runSync(args []string, out, errw io.Writer) int {
	c := newMemoryCmd("sync", time.Minute, out, errw)
	o, code := c.parse(args, errw)
	if code != 0 {
		return code
	}
	res, err := memories.Sync(o)
	if errors.Is(err, memories.ErrLocked) {
		fmt.Fprintln(out, "sync: another memories command is running; this one did nothing")
		return 0
	}
	if err != nil {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	keys := make([]string, 0, len(res.ImportedBy))
	for k := range res.ImportedBy {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(out, "imported %s %d\n", k, res.ImportedBy[k])
	}
	if res.Unreachable {
		fmt.Fprintln(out, "sync: the remote did not answer; the changes are committed in the checkout")
	}
	fmt.Fprintf(out, "sync: %d exported, %d imported, %d set aside, %d deletions overridden\n",
		res.Exported, res.Imported, res.Conflicts, res.Overridden)
	if res.PushErr != nil {
		fmt.Fprintf(errw, "gellee: push: %v\n", res.PushErr)
		return 1
	}
	return 0
}

func runList(args []string, out, errw io.Writer) int {
	c := newMemoryCmd("list", time.Minute, out, errw)
	asJSON := c.fs.Bool("json", false, "print JSON")
	o, code := c.parse(args, errw)
	if code != 0 {
		return code
	}
	l, err := memories.List(o)
	if err == nil {
		if *asJSON {
			err = l.JSON(out)
		} else {
			err = l.Print(out)
		}
	}
	if err != nil {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	return 0
}

// runNotices never fails once its flags are valid: it runs inside the
// session-start hook, which must not stop a session.
func runNotices(args []string, out, errw io.Writer) int {
	c := newMemoryCmd("notices", 5*time.Second, out, errw)
	project := c.fs.String("project", "", "key of the project the session runs in")
	doSync := c.fs.Bool("sync", false, "sync before reading the notices")
	o, code := c.parse(args, errw)
	if code != 0 {
		return code
	}
	var res *memories.Result
	var syncErr error
	if *doSync {
		r, err := memories.Sync(o)
		switch {
		case err == nil:
			res = &r
		case !errors.Is(err, memories.ErrLocked):
			syncErr = err
		}
	}
	lines, err := memories.Notices(o, *project, res)
	// A store that is not a checkout fails both the same way: one line.
	if syncErr != nil && (err == nil || firstLine(syncErr) != firstLine(err)) {
		fmt.Fprintf(out, "memories: sync failed: %s\n", firstLine(syncErr))
	}
	if err != nil {
		fmt.Fprintf(out, "memories: %s\n", firstLine(err))
		return 0
	}
	for _, line := range lines {
		fmt.Fprintf(out, "memories: %s\n", line)
	}
	return 0
}

// firstLine keeps a notice on one line: some errors list files below.
func firstLine(err error) string {
	line, _, _ := strings.Cut(err.Error(), "\n")
	return strings.TrimSuffix(line, ":")
}
