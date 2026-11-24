// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMachineRowsReadsEveryMonth(t *testing.T) {
	rows, err := MachineRows(filepath.Join(fixtures, "state"), "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 {
		t.Fatalf("%d rows, want 6", len(rows))
	}
	want := Row{Day: "2026-10-01", Session: "s0", Project: "alpha/fern", Model: "model-a",
		Input: 100, Output: 1000, CacheWrite: 10000, CacheRead: 100000, Messages: 3}
	if rows[0] != want {
		t.Fatalf("first row %+v", rows[0])
	}
}

func TestAllRowsSpansMachinesAndReportsBadFiles(t *testing.T) {
	rows, problems := AllRows(filepath.Join(fixtures, "state"))
	if len(rows) != 7 || len(problems) != 0 {
		t.Fatalf("%d rows, problems %+v", len(rows), problems)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "usage/m1"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "usage/m1/2026-11.jsonl"), []byte("{not json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rows, problems = AllRows(dir)
	if len(rows) != 0 || len(problems) != 1 || problems[0].File != "usage/m1/2026-11.jsonl" {
		t.Fatalf("rows %+v problems %+v", rows, problems)
	}
}

func TestMergeSumsAndSorts(t *testing.T) {
	a := Row{Day: "2026-11-10", Session: "s1", Project: "p/x", Model: "m", Input: 1, Output: 2, CacheWrite: 3, CacheRead: 4, Messages: 1}
	b := a
	b.Input, b.Messages = 10, 2
	c := Row{Day: "2026-11-09", Session: "s1", Project: "p/x", Model: "m", Output: 5, Messages: 1}
	got := Merge([]Row{a, b, c})
	want := []Row{
		c,
		{Day: "2026-11-10", Session: "s1", Project: "p/x", Model: "m", Input: 11, Output: 4, CacheWrite: 6, CacheRead: 8, Messages: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestRename(t *testing.T) {
	rows := []Row{{Project: "a/old"}, {Project: "a/kept"}}
	got := Rename(rows, map[string]string{"a/old": "a/new"})
	if got[0].Project != "a/new" || got[1].Project != "a/kept" || rows[0].Project != "a/old" {
		t.Fatalf("got %+v, input %+v", got, rows)
	}
}

func TestWriteMachineRowsRoundTripsAndSkipsUnchangedFiles(t *testing.T) {
	src := filepath.Join(fixtures, "state")
	rows, err := MachineRows(src, "m1")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	changed, err := WriteMachineRows(dir, "m1", rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 2 {
		t.Fatalf("changed %q, want two month files", changed)
	}
	for _, month := range []string{"2026-10.jsonl", "2026-11.jsonl"} {
		want, _ := os.ReadFile(filepath.Join(src, "usage/m1", month))
		got, _ := os.ReadFile(filepath.Join(dir, "usage/m1", month))
		if len(want) == 0 || string(got) != string(want) {
			t.Errorf("%s differs from the fixture:\n%s", month, got)
		}
	}
	changed, err = WriteMachineRows(dir, "m1", rows)
	if err != nil || len(changed) != 0 {
		t.Fatalf("second write changed %q (%v)", changed, err)
	}
}
