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

func TestWriteMachineFileWritesTheSpecifiedBytes(t *testing.T) {
	dir := t.TempDir()
	f := MachineFile{
		Collected: "2026-11-18T14:00:00+01:00", ClaudeCode: "2.1.323",
		Memories: &MachineMemories{Pushed: "2026-11-18T13:50:00+01:00", SetAside: 0},
		Sessions: []Session{{
			ID: "s1", Project: "alpha/fern", Branch: "main", Title: "Add the workers page",
			Remote: "session_01Example", Started: "2026-11-18T11:35:00+01:00", Last: "2026-11-18T13:52:00+01:00", Ended: true,
		}},
	}
	if err := WriteMachineFile(dir, "m1", f); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "machines", "m1.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "collected": "2026-11-18T14:00:00+01:00",
  "claude_code": "2.1.323",
  "memories": {
    "pushed": "2026-11-18T13:50:00+01:00",
    "set_aside": 0
  },
  "sessions": [
    {
      "id": "s1",
      "project": "alpha/fern",
      "branch": "main",
      "title": "Add the workers page",
      "remote": "session_01Example",
      "started": "2026-11-18T11:35:00+01:00",
      "last": "2026-11-18T13:52:00+01:00",
      "ended": true
    }
  ]
}
`
	if string(data) != want {
		t.Fatalf("file:\n%s", data)
	}
	got, err := ReadMachineFile(dir, "m1")
	if err != nil || !reflect.DeepEqual(got, f) {
		t.Fatalf("read back %+v, err %v", got, err)
	}
}

func TestWriteMachineFileWithoutStoreOrSessions(t *testing.T) {
	dir := t.TempDir()
	if err := WriteMachineFile(dir, "m1", MachineFile{Collected: "2026-11-18T14:00:00+01:00"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "machines", "m1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"sessions": []`) || strings.Contains(string(data), "memories") ||
		!strings.Contains(string(data), `"claude_code": ""`) {
		t.Fatalf("file:\n%s", data)
	}
}

func TestReadMachineFileOfAMachineWithoutOne(t *testing.T) {
	f, err := ReadMachineFile(t.TempDir(), "m1")
	if err != nil || f.Collected != "" || f.Sessions != nil {
		t.Fatalf("file %+v err %v", f, err)
	}
}

func TestReadMachineFileOfAMalformedFileNamesTheFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "machines"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "machines", "m1.json"), []byte(`{"collected":`), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := ReadMachineFile(dir, "m1")
	if err == nil || !strings.HasPrefix(err.Error(), "machines/m1.json:") {
		t.Fatalf("error %v, want one starting with the file's name", err)
	}
	if !reflect.DeepEqual(f, MachineFile{}) {
		t.Fatalf("file %+v, want the zero value", f)
	}
}

func TestMachineFilesReportsAnUnreadableFileAndServesTheRest(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "machines"), 0o750); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"m1.json":   `{"collected":"2026-11-18T14:00:00+01:00","future":{"field":1}}`,
		"m2.json":   `{`,
		"notes.txt": `not a machine`,
	} {
		if err := os.WriteFile(filepath.Join(dir, "machines", name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	files, problems := MachineFiles(dir)
	if len(files) != 1 || files["m1"].Collected != "2026-11-18T14:00:00+01:00" || files["m1"].ClaudeCode != "" {
		t.Fatalf("files %+v", files)
	}
	if len(problems) != 1 || problems[0].File != "machines/m2.json" {
		t.Fatalf("problems %+v", problems)
	}
}

func TestMachineFilesReadsTheFixture(t *testing.T) {
	files, problems := MachineFiles("../../tests/fixtures/state")
	if len(problems) != 0 || len(files) != 2 {
		t.Fatalf("files %+v problems %+v", files, problems)
	}
	m1, m2 := files["m1"], files["m2"]
	if m1.Memories == nil {
		t.Fatalf("m1 has no memories field: %+v", m1)
	}
	if len(m1.Sessions) != 3 || m1.Sessions[0].Remote != "session_01FixtureToolSession" || m1.Memories.SetAside != 1 {
		t.Fatalf("m1 %+v", m1)
	}
	if m2.ClaudeCode != "2.1.300" || m2.Sessions[1].Project != "alpha/old-name" {
		t.Fatalf("m2 %+v", m2)
	}
}
