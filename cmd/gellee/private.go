// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/aimd54/gellee/internal/private"
)

const privateUsage = `usage: gellee private <command> [flags]

commands:
  link    link the project's private files from the state repository
  sweep   move private files into the state repository and link them back
  status  list what keeps private files from travelling; --json for the skill

Exit status 3: the directory is not a checkout of a project in the roots.
`

func runPrivate(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(errw, privateUsage)
		return 2
	}
	name := args[0]
	act := map[string]func(private.Checkout) (private.Report, error){
		"link": private.Link, "sweep": private.Sweep, "status": private.Status,
	}[name]
	if act == nil {
		fmt.Fprintf(errw, "gellee: unknown private command %q\n\n%s", name, privateUsage)
		return 2
	}
	fs := flag.NewFlagSet("private "+name, flag.ContinueOnError)
	fs.SetOutput(errw)
	stateDir := fs.String("state", os.Getenv("GELLEE_STATE"), "state repository")
	projects := fs.String("projects", os.Getenv("GELLEE_PROJECTS_ROOT"), "projects root")
	dir := fs.String("dir", ".", "a directory inside the checkout")
	asJSON := fs.Bool("json", false, "print the report as JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *stateDir == "" {
		fmt.Fprintln(errw, "gellee: --state or GELLEE_STATE is required")
		return 2
	}
	c, err := private.Locate(*stateDir, *projects, *dir)
	if errors.Is(err, private.ErrOutside) {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 3
	}
	if err != nil {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	r, err := act(c)
	if err != nil {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	if *asJSON {
		err := json.NewEncoder(out).Encode(struct {
			Key   string `json:"key"`
			Ready bool   `json:"ready"`
			private.Report
		}{c.Key, r.Ready(), r})
		if err != nil {
			fmt.Fprintf(errw, "gellee: %v\n", err)
			return 1
		}
		return 0
	}
	for _, g := range r.Groups() {
		for _, p := range g.Paths {
			fmt.Fprintf(out, "%-11s  %s\n", g.What, p)
		}
	}
	fmt.Fprintf(out, "private: %s\n", r.Summary())
	return 0
}
