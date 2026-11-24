// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aimd54/gellee/internal/collect"
)

// machineName is GELLEE_MACHINE, else the host name up to its first dot.
func machineName() string {
	if m := os.Getenv("GELLEE_MACHINE"); m != "" {
		return m
	}
	host, _ := os.Hostname()
	short, _, _ := strings.Cut(host, ".")
	return short
}

func runCollect(args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	fs.SetOutput(errw)
	home, _ := os.UserHomeDir()
	stateDir := fs.String("state", os.Getenv("GELLEE_STATE"), "state repository")
	transcripts := fs.String("transcripts", filepath.Join(home, ".claude", "projects"), "directory of session transcripts")
	machine := fs.String("machine", machineName(), "name this machine's files are written under")
	memoryStore := fs.String("memories", os.Getenv("GELLEE_MEMORIES"), "checkout of the memory store, read for this machine's last push")
	noCommit := fs.Bool("no-commit", false, "write the files without committing them")
	push := fs.Bool("push", false, "push the state repository after committing")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *stateDir == "" {
		fmt.Fprintln(errw, "gellee: --state or GELLEE_STATE is required")
		return 2
	}
	if *machine == "" {
		fmt.Fprintln(errw, "gellee: --machine or GELLEE_MACHINE is required")
		return 2
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	err = collect.Run(collect.Options{
		StateDir: *stateDir, TranscriptsDir: *transcripts,
		CacheDir: filepath.Join(cacheRoot, "gellee"), Machine: *machine,
		ProjectsRoot: os.Getenv("GELLEE_PROJECTS_ROOT"), Memories: *memoryStore,
		Commit: !*noCommit, Push: *push,
		Now: time.Now(), Loc: time.Local, Out: out,
		Open: func(p string) (io.ReadCloser, error) { return os.Open(p) }, //nolint:gosec // transcript paths found below the transcripts directory
	})
	if err != nil {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	return 0
}
