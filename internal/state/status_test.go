// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The same table is read by tests/fixtures.bats, so the shell library
// and this package agree on what a status file says.
func TestStatusFixturesMatchExpected(t *testing.T) {
	table, err := os.ReadFile(filepath.Join(fixtures, "expected.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(table)), "\n")[1:]
	if len(lines) != 6 {
		t.Fatalf("%d fixture rows, want 6", len(lines))
	}
	for _, line := range lines {
		f := strings.Split(line, "\t")
		data, err := os.ReadFile(filepath.Join(fixtures, "state", f[0]))
		if err != nil {
			t.Fatal(err)
		}
		s, err := ParseStatus(data)
		if (err == nil) != (f[1] == "yes") {
			t.Errorf("%s: valid=%v, want %s (%v)", f[0], err == nil, f[1], err)
		}
		if s.State != f[2] || s.Updated != f[3] || FirstParagraph(s.Next) != f[4] {
			t.Errorf("%s: got %q %q %q", f[0], s.State, s.Updated, FirstParagraph(s.Next))
		}
	}
}

func TestParseStatusFieldsAndSections(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(fixtures, "state/alpha/fern.md"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := ParseStatus(data)
	if err != nil {
		t.Fatal(err)
	}
	if s.Project != "alpha/fern" || s.Group != "fieldwork" {
		t.Errorf("frontmatter: %+v", s)
	}
	if s.Stopped != "Milestone 2, task 17 paused before its review. Branch `m2-task-17`." {
		t.Errorf("stopped: %q", s.Stopped)
	}
	if s.Remaining != "- Milestone 2, tasks 18 to 24" {
		t.Errorf("remaining: %q", s.Remaining)
	}
	want := []string{
		"Compare two seed mixes on the same field plan",
		"Write up the soil sampling measurements",
	}
	if got := ListItems(s.Ideas); !reflect.DeepEqual(got, want) {
		t.Errorf("ideas: %q", got)
	}
}

func TestParseStatusRejectsMissingFrontmatter(t *testing.T) {
	if _, err := ParseStatus([]byte("## Next action\n\nSomething.\n")); err == nil {
		t.Fatal("want an error")
	}
	if _, err := ParseStatus([]byte("---\nproject: a/b\n")); err == nil {
		t.Fatal("want an error for unclosed frontmatter")
	}
}

func TestLoadStatuses(t *testing.T) {
	statuses, problems := LoadStatuses(filepath.Join(fixtures, "state"), []string{"alpha", "beta"})
	var keys []string
	for _, s := range statuses {
		keys = append(keys, s.Project)
	}
	want := []string{"alpha/fern", "alpha/fern-docs", "alpha/renamed", "beta/legacy", "beta/tool"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("projects: %q", keys)
	}
	if len(problems) != 1 || problems[0].File != "alpha/broken.md" ||
		!strings.Contains(problems[0].Message, `state "sleeping"`) {
		t.Errorf("problems: %+v", problems)
	}
}

func TestLoadStatusesReportsAMisplacedFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "alpha"), 0o750); err != nil {
		t.Fatal(err)
	}
	body := "---\nproject: alpha/other\nstate: active\nupdated: 2026-11-01\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "alpha/here.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	statuses, problems := LoadStatuses(dir, []string{"alpha"})
	if len(statuses) != 0 || len(problems) != 1 ||
		problems[0].Message != `project is "alpha/other", expected "alpha/here"` {
		t.Fatalf("statuses %+v problems %+v", statuses, problems)
	}
}

func TestAliases(t *testing.T) {
	statuses, _ := LoadStatuses(filepath.Join(fixtures, "state"), []string{"alpha", "beta"})
	want := map[string]string{"alpha/old-name": "alpha/renamed", "alpha/older-name": "alpha/renamed"}
	if got := Aliases(statuses); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestParseStatusRejectsAnUpdatedThatIsNotADay(t *testing.T) {
	for _, updated := range []string{"11/09/2026", "soon", "2026-13-01", "2026-11-31", "2026-11-9", "2026-11-09T10:00:00Z"} {
		data := []byte("---\nproject: a/b\nstate: active\nupdated: " + updated + "\n---\n")
		_, err := ParseStatus(data)
		if err == nil {
			t.Errorf("updated %q was accepted", updated)
			continue
		}
		if !strings.Contains(err.Error(), "updated") {
			t.Errorf("updated %q: error %q does not name the field", updated, err)
		}
	}
	good := []byte("---\nproject: a/b\nstate: active\nupdated: 2026-11-09\n---\n")
	s, err := ParseStatus(good)
	if err != nil || s.Updated != "2026-11-09" {
		t.Fatalf("a valid day was refused: %+v %v", s, err)
	}
}
