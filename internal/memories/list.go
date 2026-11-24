// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"unicode/utf8"
)

// Memory is one memory file of a project in the store.
type Memory struct {
	Project     string `json:"project"`
	File        string `json:"file"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Size        int    `json:"size"`
	Changed     string `json:"changed"` // YYYY-MM-DD, or "" when never committed
}

// Listing is what the store holds, for a review.
type Listing struct {
	Memories   []Memory            `json:"memories"`
	Conflicts  []string            `json:"conflicts"`  // paths of versions set aside
	Orphans    []string            `json:"orphans"`    // paths of memories that match no project
	Duplicates map[string][]string `json:"duplicates"` // a name and the projects holding it
}

// List reads the working tree of the checkout. It changes nothing and
// takes no lock.
func List(o Options) (Listing, error) {
	l := Listing{Memories: []Memory{}, Conflicts: []string{}, Orphans: []string{}, Duplicates: map[string][]string{}}
	if err := o.checkout(); err != nil {
		return l, err
	}
	changed, err := o.repo().changed()
	if err != nil {
		return l, err
	}
	holders := map[string]map[string]bool{}
	err = filepath.WalkDir(o.StoreDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != o.StoreDir && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(o.StoreDir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		parts := strings.Split(rel, "/")
		switch {
		case parts[0] == "orphans" && len(parts) > 3 && parts[2] == "conflicts":
			l.Conflicts = append(l.Conflicts, rel)
		case parts[0] == "orphans":
			l.Orphans = append(l.Orphans, rel)
		case parts[0] == globalKey, len(parts) != 3, parts[2] == "MEMORY.md":
			// Not a memory of a project.
		default:
			data, err := os.ReadFile(p) //nolint:gosec // a file below the checkout
			if err != nil {
				return err
			}
			m := Memory{Project: path.Dir(rel), File: parts[2], Size: len(data), Changed: changed[rel]}
			m.Name, m.Type, m.Description = frontmatter(data)
			if m.Name == "" {
				m.Name = strings.TrimSuffix(m.File, ".md")
			}
			l.Memories = append(l.Memories, m)
			if holders[m.Name] == nil {
				holders[m.Name] = map[string]bool{}
			}
			holders[m.Name][m.Project] = true
		}
		return nil
	})
	for name, projects := range holders {
		if len(projects) < 2 {
			continue
		}
		for p := range projects {
			l.Duplicates[name] = append(l.Duplicates[name], p)
		}
		sort.Strings(l.Duplicates[name])
	}
	return l, err
}

// frontmatter reads the name, type and description of a memory. A file
// without frontmatter, or one that is not valid UTF-8, gives three empty
// strings.
func frontmatter(data []byte) (name, typ, desc string) {
	if !utf8.Valid(data) {
		return "", "", ""
	}
	lines := strings.Split(string(data), "\n")
	if strings.TrimSpace(lines[0]) != "---" {
		return "", "", ""
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch {
		case key == "name":
			name = value
		case key == "description":
			desc = value
		case strings.TrimSpace(key) == "type" && typ == "":
			typ = value
		}
	}
	return name, typ, desc
}

// changed is the date of the last commit that touched each file.
func (r repo) changed() (map[string]string, error) {
	out := map[string]string{}
	if !r.has("HEAD") {
		return out, nil
	}
	log, err := r.git("-c", "core.quotePath=false", "log", "--format=@%cs", "--name-only")
	if err != nil {
		return nil, err
	}
	date := ""
	for _, line := range strings.Split(log, "\n") {
		switch {
		case strings.HasPrefix(line, "@"):
			date = line[1:]
		case line != "" && out[line] == "":
			out[line] = date
		}
	}
	return out, nil
}

// Print writes the memories as a table, then the three groups.
func (l Listing) Print(w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "PROJECT\tTYPE\tNAME\tSIZE\tCHANGED\tDESCRIPTION")
	for _, m := range l.Memories {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\n", m.Project, dash(m.Type), m.Name, m.Size, dash(m.Changed), m.Description)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	group(w, "conflicts set aside", l.Conflicts)
	group(w, "orphans", l.Orphans)
	dups := make([]string, 0, len(l.Duplicates))
	for name, projects := range l.Duplicates {
		dups = append(dups, name+": "+strings.Join(projects, ", "))
	}
	sort.Strings(dups)
	group(w, "names in more than one project", dups)
	return nil
}

// JSON writes the listing for the curation skill.
func (l Listing) JSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(l)
}

func group(w io.Writer, title string, items []string) {
	fmt.Fprintf(w, "\n%s: %d\n", title, len(items))
	for _, item := range items {
		fmt.Fprintf(w, "  %s\n", item)
	}
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
