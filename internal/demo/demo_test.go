// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package demo_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io/fs"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/aimd54/gellee/internal/demo"
	"github.com/aimd54/gellee/internal/server"
	"github.com/aimd54/gellee/internal/state"
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

var update = flag.Bool("update", false, "rewrite the golden files")

func TestEveryWarningIsOnItsProject(t *testing.T) {
	_, snap := build(t, anchor)
	got := map[string]string{}
	active := 0
	for _, p := range snap.Projects() {
		got[p.Key] = strings.Join(p.Warnings, ",")
		if p.State == "active" {
			active++
		}
	}
	want := map[string]string{
		"studio/loom": "", "studio/loom-site": "uncommitted", "studio/pathfinder": "stale",
		"garden/seedbank": "unpushed", "garden/trellis": "", "garden/almanac": "", "studio/weft": "",
	}
	for key, w := range want {
		if g, ok := got[key]; !ok || g != w {
			t.Errorf("%s warnings %q, want %q", key, g, w)
		}
	}
	if active != 4 || snap.Config.Limit != 3 {
		t.Fatalf("%d active against a limit of %d, want 4 against 3", active, snap.Config.Limit)
	}
}

func TestTheWorkersTellTheirStory(t *testing.T) {
	_, snap := build(t, anchor)
	byName := map[string]server.Worker{}
	for _, w := range snap.Workers() {
		byName[w.Name] = w
	}
	running := func(w server.Worker, project string) bool {
		for _, s := range w.Sessions {
			if s.Project == project && !s.Ended {
				return true
			}
		}
		return false
	}
	laptop, desktop, worker := byName["laptop"], byName["desktop"], byName["worker-1"]
	if !laptop.Up || laptop.Behind || !running(laptop, "studio/loom") {
		t.Errorf("laptop %+v", laptop)
	}
	if desktop.Up || !desktop.Behind || desktop.Memories == nil || desktop.Memories.SetAside != 2 {
		t.Errorf("desktop %+v", desktop)
	}
	if !worker.Up || worker.Memories == nil || !worker.Memories.Stale || !running(worker, "studio/pathfinder") {
		t.Errorf("worker-1 %+v", worker)
	}
	n := 0
	for _, w := range byName {
		n += len(w.Sessions)
	}
	if len(byName) != 3 || n != 11 {
		t.Fatalf("%d workers with %d sessions, want 3 with 11", len(byName), n)
	}
}

func TestUsageUnderTheFormerNameIsRenamed(t *testing.T) {
	dir, snap := build(t, anchor)
	raw, err := state.MachineRows(dir, "laptop")
	if err != nil {
		t.Fatal(err)
	}
	written := 0
	for _, r := range raw {
		if r.Project == "studio/shuttle" {
			written++
		}
	}
	old := server.DayOffset(snap.Today, -21)
	renamed := 0
	for _, r := range snap.Rows {
		if r.Project == "studio/shuttle" {
			t.Fatalf("row still under studio/shuttle: %+v", r)
		}
		if r.Project == "studio/loom" && r.Day <= old {
			renamed++
		}
	}
	if written == 0 || renamed == 0 {
		t.Fatalf("%d rows written under studio/shuttle, %d read as studio/loom", written, renamed)
	}
}

func TestOneModelHasNoPrice(t *testing.T) {
	_, snap := build(t, anchor)
	matched := 0
	for _, p := range snap.Projects() {
		switch p.Key {
		case "garden/trellis":
			matched++
			if p.UsageAll.Tokens == 0 || p.UsageAll.Cost != nil {
				t.Errorf("trellis usage %+v, want tokens and no cost", p.UsageAll)
			}
		case "studio/loom":
			matched++
			if p.UsageAll.Tokens == 0 || p.UsageAll.Cost == nil {
				t.Errorf("loom usage %+v, want tokens and a cost", p.UsageAll)
			}
		}
	}
	if matched != 2 {
		t.Fatalf("%d of trellis and loom seen, want both", matched)
	}
}

// digests maps every file below dir to its SHA-256.
func digests(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		sum := sha256.Sum256(data)
		out[rel] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestTheSameAnchorGivesTheSameBytes(t *testing.T) {
	a, _ := build(t, anchor)
	b, _ := build(t, anchor)
	da, db := digests(t, a), digests(t, b)
	if len(da) < 15 {
		t.Fatalf("%d files, want the statuses and the usage, git and machine files", len(da))
	}
	for name, sum := range da {
		if db[name] != sum {
			t.Errorf("%s differs between two writes", name)
		}
	}
	if len(da) != len(db) {
		t.Fatalf("%d files against %d", len(da), len(db))
	}
}

func TestUsageMonthsFollowTheAnchor(t *testing.T) {
	_, nov := build(t, anchor)
	dir, dec := build(t, time.Date(2026, 12, 3, 12, 0, 0, 0, time.UTC))
	entries, err := os.ReadDir(filepath.Join(dir, "usage", "laptop"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if strings.Join(names, " ") != "2026-10.jsonl 2026-11.jsonl 2026-12.jsonl" {
		t.Fatalf("laptop usage files %q", names)
	}
	if len(dec.Rows) == 0 || len(dec.Rows) != len(nov.Rows) {
		t.Fatalf("%d rows in December, %d in November", len(dec.Rows), len(nov.Rows))
	}
}

// roundCosts returns raw with every "cost" number, at any depth,
// rounded to 6 decimal places. Key order and other values are kept.
func roundCosts(t *testing.T, raw json.RawMessage) json.RawMessage {
	t.Helper()
	trimmed := bytes.TrimSpace(raw)
	switch {
	case len(trimmed) > 0 && trimmed[0] == '{':
		var fields []json.RawMessage
		dec := json.NewDecoder(bytes.NewReader(trimmed))
		if _, err := dec.Token(); err != nil {
			t.Fatal(err)
		}
		for dec.More() {
			k, err := dec.Token()
			if err != nil {
				t.Fatal(err)
			}
			var v json.RawMessage
			if err := dec.Decode(&v); err != nil {
				t.Fatal(err)
			}
			key, _ := json.Marshal(k)
			var f float64
			if k == "cost" && json.Unmarshal(v, &f) == nil {
				v = json.RawMessage(strconv.FormatFloat(math.Round(f*1e6)/1e6, 'g', -1, 64))
			} else {
				v = roundCosts(t, v)
			}
			fields = append(fields, append(append(key, ':'), v...))
		}
		return json.RawMessage("{" + string(bytes.Join(toBytes(fields), []byte(","))) + "}")
	case len(trimmed) > 0 && trimmed[0] == '[':
		var items []json.RawMessage
		if err := json.Unmarshal(trimmed, &items); err != nil {
			t.Fatal(err)
		}
		for i := range items {
			items[i] = roundCosts(t, items[i])
		}
		return json.RawMessage("[" + string(bytes.Join(toBytes(items), []byte(","))) + "]")
	}
	return trimmed
}

func toBytes(in []json.RawMessage) [][]byte {
	out := make([][]byte, len(in))
	for i, m := range in {
		out[i] = m
	}
	return out
}

func TestTheOverviewMatchesItsGoldenFile(t *testing.T) {
	dir, _ := build(t, anchor)
	h := server.New(dir, fstest.MapFS{}, func() time.Time { return anchor }, time.Minute).Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/overview", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d: %s", rec.Code, rec.Body.String())
	}
	// Costs are sums of floats, whose last digits depend on fused
	// multiply-add; six places are stable across platforms.
	var got bytes.Buffer
	if err := json.Indent(&got, roundCosts(t, rec.Body.Bytes()), "", "  "); err != nil {
		t.Fatal(err)
	}
	got.WriteByte('\n')
	golden := filepath.Join("testdata", "overview.json")
	if *update {
		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, got.Bytes(), 0o644); err != nil { //nolint:gosec // a test fixture
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("overview differs from %s; rerun with -update after checking the change\n%s", golden, got.String())
	}
}
