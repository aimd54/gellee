// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package collect

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// store makes a git checkout that reads as the memory store. joined
// writes the state files a first sync leaves under .git/gellee, with
// the push time given, and aside lists files set aside for m1.
func store(t *testing.T, joined bool, pushed string, aside ...string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "memories")
	git(t, filepath.Dir(dir), "init", "-q", "-b", "main", dir)
	write := func(rel, content string) {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if joined {
		write(".git/gellee/synced", "0123456789abcdef\n")
		write(".git/gellee/pushed", pushed+"\n")
	}
	for _, p := range aside {
		write("orphans/m1/conflicts/"+p, "x\n")
	}
	return dir
}

func TestMachineFileWithoutAStoreHasNoMemoriesField(t *testing.T) {
	o := world(t)
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	f := machineOf(t, o)
	if f.Memories != nil || f.Collected != "2026-11-11T10:00:00Z" {
		t.Fatalf("machine file %+v", f)
	}
	data, err := os.ReadFile(filepath.Join(o.StateDir, "machines/m1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"memories":`) {
		t.Fatalf("machine file holds the memories key:\n%s", data)
	}
}

func TestMachineFileHoldsTheJoinedStoresHealth(t *testing.T) {
	o := world(t)
	o.Memories = store(t, true, "2026-11-11T09:50:00+01:00", "alpha/fern/one.md", "global/CLAUDE.md")
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	f := machineOf(t, o)
	if f.Memories == nil || f.Memories.Pushed != "2026-11-11T09:50:00+01:00" || f.Memories.SetAside != 2 {
		t.Fatalf("memories %+v", f.Memories)
	}
}

func TestAStoreNotJoinedIsReportedAndTheRunCarriesOn(t *testing.T) {
	o := world(t)
	o.Memories = store(t, false, "")
	var out bytes.Buffer
	o.Out = &out
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "collect: memory store: ") {
		t.Fatalf("output %q", out.String())
	}
	f := machineOf(t, o)
	if f.Memories != nil || f.Collected != "2026-11-11T10:00:00Z" {
		t.Fatalf("machine file %+v", f)
	}
}
