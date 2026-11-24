// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const fixtures = "../../tests/fixtures"

func TestLoadConfig(t *testing.T) {
	c, err := LoadConfig(filepath.Join(fixtures, "state"))
	if err != nil {
		t.Fatal(err)
	}
	want := Config{Roots: []string{"alpha", "beta"}, Limit: 2, ActiveDays: 10}
	if !reflect.DeepEqual(c, want) {
		t.Fatalf("got %+v, want %+v", c, want)
	}
}

func TestLoadConfigDefaultsAndProjectsRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config"), []byte("roots=one\nprojects_root=/p\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := Config{Roots: []string{"one"}, Limit: 3, ActiveDays: 10, ProjectsRoot: "/p"}
	if !reflect.DeepEqual(c, want) {
		t.Fatalf("got %+v, want %+v", c, want)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	if _, err := LoadConfig(t.TempDir()); err == nil {
		t.Fatal("a missing config file must be an error")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config"), []byte("limit=3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(dir); err == nil || err.Error() != "config lists no roots" {
		t.Fatalf("got %v", err)
	}
}
