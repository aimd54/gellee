// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"os"
	"path/filepath"
	"testing"
)

// A regular file where a directory is expected cannot be listed, the
// same way a directory without read permission cannot, and it does not
// depend on who runs the test.
func unlistable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
}

func problemFiles(problems []Problem) []string {
	out := make([]string, 0, len(problems))
	for _, p := range problems {
		if p.Message == "" {
			out = append(out, p.File+" (no message)")
			continue
		}
		out = append(out, p.File)
	}
	return out
}

func TestLoadersReportADirectoryTheyCannotList(t *testing.T) {
	dir := t.TempDir()
	unlistable(t, filepath.Join(dir, "alpha"))
	unlistable(t, filepath.Join(dir, "usage"))
	unlistable(t, filepath.Join(dir, "git"))

	_, problems := LoadStatuses(dir, []string{"alpha", "beta"})
	if got := problemFiles(problems); len(got) != 1 || got[0] != "alpha" {
		t.Errorf("statuses: problems %v, want [alpha]", got)
	}
	_, problems = AllRows(dir)
	if got := problemFiles(problems); len(got) != 1 || got[0] != "usage" {
		t.Errorf("rows: problems %v, want [usage]", got)
	}
	_, problems = GitFiles(dir)
	if got := problemFiles(problems); len(got) != 1 || got[0] != "git" {
		t.Errorf("git: problems %v, want [git]", got)
	}
}

func TestAllRowsReportsAMachineDirectoryItCannotList(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "usage/m1"), 0o750); err != nil {
		t.Fatal(err)
	}
	row := `{"day":"2026-11-01","session":"s","project":"alpha/fern","model":"model-a","input":1,"output":2,"messages":1}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "usage/m1/2026-11.jsonl"), []byte(row), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "nowhere", "deeper"), filepath.Join(dir, "usage/m2")); err != nil {
		t.Fatal(err)
	}
	unlistable(t, filepath.Join(dir, "nowhere"))

	rows, problems := AllRows(dir)
	if len(rows) != 1 || rows[0].Project != "alpha/fern" {
		t.Errorf("rows %+v, want the one row of m1", rows)
	}
	if got := problemFiles(problems); len(got) != 1 || got[0] != "usage/m2" {
		t.Errorf("problems %v, want [usage/m2]", got)
	}
}

func TestLoadersTreatAMissingDirectoryAsEmpty(t *testing.T) {
	dir := t.TempDir()
	if s, problems := LoadStatuses(dir, []string{"alpha"}); len(s) != 0 || len(problems) != 0 {
		t.Errorf("statuses %+v problems %+v", s, problems)
	}
	if r, problems := AllRows(dir); len(r) != 0 || len(problems) != 0 {
		t.Errorf("rows %+v problems %+v", r, problems)
	}
	if g, problems := GitFiles(dir); len(g) != 0 || len(problems) != 0 {
		t.Errorf("git %+v problems %+v", g, problems)
	}
}
