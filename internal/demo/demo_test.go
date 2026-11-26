// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package demo_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/aimd54/gellee/internal/demo"
	"github.com/aimd54/gellee/internal/server"
)

var anchor = time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

// build writes the demo at the anchor and reads it as the server does.
func build(t *testing.T, at time.Time) (string, server.Snapshot) {
	t.Helper()
	dir := t.TempDir()
	if err := demo.Write(dir, at); err != nil {
		t.Fatal(err)
	}
	snap, err := server.Build(dir, at)
	if err != nil {
		t.Fatal(err)
	}
	return dir, snap
}

func TestTheStatusFilesReadWithoutProblems(t *testing.T) {
	_, snap := build(t, anchor)
	if len(snap.Problems) != 0 {
		t.Fatalf("problems %v", snap.Problems)
	}
	var got []string
	for _, st := range snap.Statuses {
		got = append(got, st.Project+":"+st.State+":"+st.Updated)
	}
	sort.Strings(got)
	want := []string{
		"garden/almanac:done:2026-10-12",
		"garden/seedbank:active:2026-11-10",
		"garden/trellis:parked:2026-10-30",
		"studio/loom-site:active:2026-11-11",
		"studio/loom:active:2026-11-11",
		"studio/pathfinder:active:2026-11-09",
		"studio/weft:done:2026-10-02",
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("statuses\n got %q\nwant %q", got, want)
	}
	if snap.Config.Limit != 3 || strings.Join(snap.Config.Roots, " ") != "studio garden" {
		t.Fatalf("config %+v", snap.Config)
	}
	if len(snap.Prices) != 2 || snap.Prices["model-large"].Output != 50 {
		t.Fatalf("prices %+v", snap.Prices)
	}
}

func TestNoTemplateIsLeftInTheTree(t *testing.T) {
	dir, _ := build(t, anchor)
	n := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		n++
		if strings.HasSuffix(path, ".tmpl") {
			t.Errorf("%s keeps its template suffix", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "{{") {
			t.Errorf("%s holds a template action", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n < 9 {
		t.Fatalf("%d files written, want at least the config, prices and 7 statuses", n)
	}
}
