// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Status is one project's status file.
type Status struct {
	Project      string
	State        string
	Group        string
	ParkedReason string
	SupersededBy string
	Updated      string
	Formerly     []string
	Stopped      string
	Next         string
	Remaining    string
	Ideas        string
}

// Problem is a file the state repository holds but that could not be
// used. It is shown, never fatal.
type Problem struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

// ParseStatus reads frontmatter and the four sections. The returned
// Status carries whatever was read even when the error is not nil.
func ParseStatus(data []byte) (Status, error) {
	var s Status
	lines := strings.Split(string(data), "\n")
	if lines[0] != "---" {
		return s, errors.New("no frontmatter")
	}
	i := 1
	for ; i < len(lines) && lines[i] != "---"; i++ {
		key, value, ok := strings.Cut(lines[i], ": ")
		if !ok {
			continue
		}
		switch key {
		case "project":
			s.Project = value
		case "state":
			s.State = value
		case "group":
			s.Group = value
		case "parked_reason":
			s.ParkedReason = value
		case "superseded_by":
			s.SupersededBy = value
		case "updated":
			s.Updated = value
		case "formerly":
			s.Formerly = strings.Fields(value)
		}
	}
	if i == len(lines) {
		return s, errors.New("frontmatter is not closed")
	}
	sections := map[string]*string{
		"## Where I stopped": &s.Stopped,
		"## Next action":     &s.Next,
		"## Remaining":       &s.Remaining,
		"## Ideas":           &s.Ideas,
	}
	var cur *string
	var buf []string
	flush := func() {
		if cur != nil {
			*cur = strings.TrimSpace(strings.Join(buf, "\n"))
		}
		buf = nil
	}
	for _, line := range lines[i+1:] {
		if strings.HasPrefix(line, "## ") {
			flush()
			cur = sections[line]
			continue
		}
		buf = append(buf, line)
	}
	flush()
	if s.Project == "" {
		return s, errors.New("no project")
	}
	if s.Updated == "" {
		return s, errors.New("no updated date")
	}
	if _, err := time.Parse("2006-01-02", s.Updated); err != nil {
		return s, fmt.Errorf("updated %q is not a day written YYYY-MM-DD", s.Updated)
	}
	switch s.State {
	case "active", "parked", "done":
	default:
		return s, fmt.Errorf("state %q is not active, parked or done", s.State)
	}
	return s, nil
}

// LoadStatuses reads <stateDir>/<root>/*.md for every root, sorted by
// project. Files that cannot be used are returned as problems.
func LoadStatuses(stateDir string, roots []string) ([]Status, []Problem) {
	var statuses []Status
	var problems []Problem
	for _, root := range roots {
		entries, listing := list(stateDir, root)
		problems = append(problems, listing...)
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			key := root + "/" + strings.TrimSuffix(entry.Name(), ".md")
			var s Status
			data, err := os.ReadFile(filepath.Join(stateDir, root, entry.Name())) //nolint:gosec // a path below the state directory
			if err == nil {
				s, err = ParseStatus(data)
			}
			if err == nil && s.Project != key {
				err = fmt.Errorf("project is %q, expected %q", s.Project, key)
			}
			if err != nil {
				problems = append(problems, Problem{File: key + ".md", Message: err.Error()})
				continue
			}
			statuses = append(statuses, s)
		}
	}
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].Project < statuses[j].Project })
	return statuses, problems
}

// FirstParagraph joins the wrapped lines of the first paragraph.
func FirstParagraph(text string) string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			if len(out) > 0 {
				break
			}
			continue
		}
		out = append(out, strings.Join(fields, " "))
	}
	return strings.Join(out, " ")
}

// ListItems returns the "- " items of a section, wrapped lines joined.
func ListItems(text string) []string {
	var items []string
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "- "):
			items = append(items, strings.TrimPrefix(t, "- "))
		case t != "" && len(items) > 0:
			items[len(items)-1] += " " + t
		}
	}
	return items
}

// Aliases maps every former project name to the current one.
func Aliases(statuses []Status) map[string]string {
	aliases := map[string]string{}
	for _, s := range statuses {
		for _, old := range s.Formerly {
			aliases[old] = s.Project
		}
	}
	return aliases
}
