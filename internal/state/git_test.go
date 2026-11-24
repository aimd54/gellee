// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitFilesReadsTheFixture(t *testing.T) {
	files, problems := GitFiles(filepath.Join(fixtures, "state"))
	if len(problems) != 0 || len(files) != 1 {
		t.Fatalf("files %d problems %+v", len(files), problems)
	}
	got := files["m1"].Projects["alpha/fern"]
	want := GitProject{Branch: "m2-task-17", Head: "abc1234", LastCommit: "2026-11-10T09:00:00+01:00", Remote: true, Ahead: 2}
	if got != want {
		t.Fatalf("got %+v", got)
	}
}

func TestWriteGitFileRoundTrips(t *testing.T) {
	dir := t.TempDir()
	in := GitFile{Collected: "2026-11-11T10:00:00+01:00", Projects: map[string]GitProject{
		"a/b": {Branch: "main", Head: "1234567", LastCommit: "2026-11-11T09:00:00+01:00", Ahead: 1, Dirty: 2},
	}}
	if err := WriteGitFile(dir, "m9", in); err != nil {
		t.Fatal(err)
	}
	files, problems := GitFiles(dir)
	if len(problems) != 0 || files["m9"].Projects["a/b"] != in.Projects["a/b"] || files["m9"].Collected != in.Collected {
		t.Fatalf("files %+v problems %+v", files, problems)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "git/m9.json"))
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Fatal("the file must end with a newline")
	}
}
