// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Row is the usage of one session, on one day, in one project, with one
// model.
type Row struct {
	Day        string `json:"day"`
	Session    string `json:"session"`
	Project    string `json:"project"`
	Model      string `json:"model"`
	Input      int64  `json:"input"`
	Output     int64  `json:"output"`
	CacheWrite int64  `json:"cache_write"`
	CacheRead  int64  `json:"cache_read"`
	Messages   int64  `json:"messages"`
}

// Merge sums rows that share day, session, project and model, and
// returns them sorted by those fields.
func Merge(rows []Row) []Row {
	type key struct{ day, session, project, model string }
	sums := map[key]Row{}
	for _, r := range rows {
		k := key{r.Day, r.Session, r.Project, r.Model}
		s := sums[k]
		s.Day, s.Session, s.Project, s.Model = r.Day, r.Session, r.Project, r.Model
		s.Input += r.Input
		s.Output += r.Output
		s.CacheWrite += r.CacheWrite
		s.CacheRead += r.CacheRead
		s.Messages += r.Messages
		sums[k] = s
	}
	out := make([]Row, 0, len(sums))
	for _, r := range sums {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch {
		case a.Day != b.Day:
			return a.Day < b.Day
		case a.Session != b.Session:
			return a.Session < b.Session
		case a.Project != b.Project:
			return a.Project < b.Project
		}
		return a.Model < b.Model
	})
	return out
}

// Rename returns a copy of rows with former project names replaced.
func Rename(rows []Row, aliases map[string]string) []Row {
	out := make([]Row, len(rows))
	for i, r := range rows {
		if current, ok := aliases[r.Project]; ok {
			r.Project = current
		}
		out[i] = r
	}
	return out
}

func readRows(path string) ([]Row, error) {
	f, err := os.Open(path) //nolint:gosec // a path below the state directory
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read only
	var rows []Row
	dec := json.NewDecoder(f)
	for {
		var r Row
		err := dec.Decode(&r)
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}
}

// MachineRows reads every month file of one machine, in month order.
func MachineRows(stateDir, machine string) ([]Row, error) {
	files, _ := filepath.Glob(filepath.Join(stateDir, "usage", machine, "*.jsonl"))
	sort.Strings(files)
	var rows []Row
	for _, file := range files {
		r, err := readRows(file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

// AllRows reads the month files of every machine. A file that does not
// parse, or a directory that cannot be listed, is reported and left out.
func AllRows(stateDir string) ([]Row, []Problem) {
	machines, problems := list(stateDir, "usage")
	var rows []Row
	for _, machine := range machines {
		if machine.Type().IsRegular() {
			continue
		}
		dir := "usage/" + machine.Name()
		months, listing := list(stateDir, dir)
		problems = append(problems, listing...)
		for _, month := range months {
			if !strings.HasSuffix(month.Name(), ".jsonl") {
				continue
			}
			rel := dir + "/" + month.Name()
			r, err := readRows(filepath.Join(stateDir, filepath.FromSlash(rel)))
			if err != nil {
				problems = append(problems, Problem{File: rel, Message: err.Error()})
				continue
			}
			rows = append(rows, r...)
		}
	}
	return rows, problems
}

// WriteMachineRows writes one file per month and returns the files whose
// content changed.
func WriteMachineRows(stateDir, machine string, rows []Row) ([]string, error) {
	dir := filepath.Join(stateDir, "usage", machine)
	byMonth := map[string][]Row{}
	for _, r := range Merge(rows) {
		if len(r.Day) < 7 {
			return nil, fmt.Errorf("row with day %q", r.Day)
		}
		byMonth[r.Day[:7]] = append(byMonth[r.Day[:7]], r)
	}
	months := make([]string, 0, len(byMonth))
	for m := range byMonth {
		months = append(months, m)
	}
	sort.Strings(months)
	var changed []string
	for _, m := range months {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		for _, r := range byMonth[m] {
			if err := enc.Encode(r); err != nil {
				return changed, err
			}
		}
		path := filepath.Join(dir, m+".jsonl")
		if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, buf.Bytes()) { //nolint:gosec // a path below the state directory
			continue
		}
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return changed, err
		}
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil { //nolint:gosec // committed to a repository, not a secret
			return changed, err
		}
		changed = append(changed, path)
	}
	return changed, nil
}

// UsageMachines names the machines that have a directory under usage/.
// AllRows reports a directory that cannot be listed.
func UsageMachines(stateDir string) []string {
	entries, _ := list(stateDir, "usage")
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}
