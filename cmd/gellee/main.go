// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// Command gellee collects usage and git facts into the state repository
// and serves the dashboard that reads them.
package main

import (
	"fmt"
	"io"
	"os"
)

const usage = `usage: gellee <command> [flags]

commands:
  collect   summarise transcripts and working trees into the state repository
  memories  carry Claude Code's memories between machines through a git store
  private   carry a project's private files through the state repository
  serve     serve the dashboard from a state repository checkout
`

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(errw, usage)
		return 2
	}
	switch args[0] {
	case "collect":
		return runCollect(args[1:], out, errw)
	case "memories":
		return runMemories(args[1:], out, errw)
	case "private":
		return runPrivate(args[1:], out, errw)
	case "serve":
		return runServe(args[1:], out, errw)
	default:
		fmt.Fprintf(errw, "gellee: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}
