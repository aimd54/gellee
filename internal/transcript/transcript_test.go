// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package transcript

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aimd54/gellee/internal/state"
)

const fixtures = "../../tests/fixtures/transcripts"

var attr = Attributor{ProjectsRoot: "/p", Roots: []string{"alpha", "beta"}}

func TestProject(t *testing.T) {
	cases := map[string]string{
		"/p/alpha/fern":                      "alpha/fern",
		"/p/alpha/fern/sub/dir":              "alpha/fern",
		"/p/alpha/fern/.claude/worktrees/w1": "alpha/fern",
		"/p/beta/tool":                       "beta/tool",
		"/p/alpha":                           Other,
		"/p/gamma/thing":                     Other,
		"/elsewhere/x":                       Other,
		"/pp/alpha/fern":                     Other,
		"":                                   Other,
	}
	for cwd, want := range cases {
		if got := attr.Project(cwd); got != want {
			t.Errorf("Project(%q) = %q, want %q", cwd, got, want)
		}
	}
}

func TestFindNamesSubagentFilesAfterTheirSession(t *testing.T) {
	files, err := Find(fixtures)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range files {
		rel, _ := filepath.Rel(fixtures, f.Path)
		got[filepath.ToSlash(rel)] = fmt.Sprintf("%s %t", f.Session, f.Main)
	}
	want := map[string]string{
		"-p-alpha-fern/s1.jsonl":                    "s1 true",
		"-p-alpha-fern/s1/subagents/agent-a1.jsonl": "s1 false",
		"-p-beta-tool/s2.jsonl":                     "s2 true",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestParseDeduplicatesAttributesAndSkips(t *testing.T) {
	f, err := os.Open(filepath.Join(fixtures, "-p-alpha-fern/s1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint:errcheck // read only
	rows, _, skipped, err := Parse(f, "s1", attr, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	want := []state.Row{
		// m1 once with its last counts, plus m2 from a subdirectory.
		{Day: "2026-11-10", Session: "s1", Project: "alpha/fern", Model: "model-a", Input: 11, Output: 102, CacheWrite: 1003, CacheRead: 10004, Messages: 2},
		{Day: "2026-11-10", Session: "s1", Project: "beta/tool", Model: "model-b", Input: 5, Output: 50, CacheWrite: 500, CacheRead: 5000, Messages: 1},
		{Day: "2026-11-10", Session: "s1", Project: "other", Model: "model-a", Input: 7, Output: 70, CacheWrite: 700, CacheRead: 7000, Messages: 1},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows:\n%+v", rows)
	}
	if skipped != 1 {
		t.Fatalf("skipped %d lines, want 1", skipped)
	}
}

func TestParseUsesTheLocalDay(t *testing.T) {
	line := `{"type":"assistant","timestamp":"2026-11-10T23:30:00.000Z","cwd":"/p/alpha/fern","message":{"id":"m1","model":"model-a","usage":{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":1,"cache_read_input_tokens":1}}}`
	rows, _, _, err := Parse(strings.NewReader(line), "s1", attr, time.FixedZone("plus-one", 3600))
	if err != nil || len(rows) != 1 || rows[0].Day != "2026-11-11" {
		t.Fatalf("rows %+v err %v", rows, err)
	}
}

func TestParseReadsALineLongerThanTheBuffer(t *testing.T) {
	pad := strings.Repeat("x", 3<<20)
	line := `{"type":"assistant","timestamp":"2026-11-10T10:00:00.000Z","cwd":"/p/alpha/fern","pad":"` + pad +
		`","message":{"id":"m1","model":"model-a","usage":{"input_tokens":4,"output_tokens":5,"cache_creation_input_tokens":6,"cache_read_input_tokens":7}}}` + "\n"
	rows, _, skipped, err := Parse(strings.NewReader(line), "s1", attr, time.UTC)
	if err != nil || skipped != 0 || len(rows) != 1 || rows[0].Output != 5 {
		t.Fatalf("rows %+v skipped %d err %v", rows, skipped, err)
	}
}

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseSummarisesTheSession(t *testing.T) {
	lines := strings.Join([]string{
		`{"type":"user","timestamp":"2026-11-10T08:00:00.000Z","cwd":"/p/alpha/fern","gitBranch":"main","version":"2.1.300","message":{"role":"user","content":"a prompt"}}`,
		`{"type":"custom-title","customTitle":"Named by hand"}`,
		`{"type":"ai-title","aiTitle":"Generated title"}`,
		`{"type":"bridge-session","bridgeSessionId":"cse_01Abc","accountUuid":"acct-1"}`,
		`{"type":"assistant","timestamp":"2026-11-10T08:05:00.000Z","cwd":"/p/alpha/fern/sub","gitBranch":"feature","version":"2.1.10","message":{"id":"m1","model":"model-a","usage":{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":1,"cache_read_input_tokens":1}}}`,
		`{"type":"bridge-session","bridgeSessionId":""}`,
		`{"type":"ai-title","aiTitle":"Later generated title"}`,
		// Written out of order: the earliest line, with the highest version.
		`{"type":"system","timestamp":"2026-11-10T07:59:00.000Z","version":"2.1.400"}`,
		`not json`,
	}, "\n")
	_, sum, skipped, err := Parse(strings.NewReader(lines), "s1", attr, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 {
		t.Errorf("skipped %d lines, want 1", skipped)
	}
	if !sum.First.Equal(at("2026-11-10T07:59:00Z")) || !sum.Last.Equal(at("2026-11-10T08:05:00Z")) {
		t.Errorf("first %v last %v", sum.First, sum.Last)
	}
	if sum.Cwd != "/p/alpha/fern/sub" || sum.Branch != "feature" {
		t.Errorf("cwd %q branch %q", sum.Cwd, sum.Branch)
	}
	if sum.CustomTitle != "Named by hand" || sum.AITitle != "Later generated title" || sum.Title() != "Named by hand" {
		t.Errorf("titles %q %q, title %q", sum.CustomTitle, sum.AITitle, sum.Title())
	}
	if sum.Bridge != "cse_01Abc" {
		t.Errorf("bridge %q: the empty record must not replace it", sum.Bridge)
	}
	if sum.Version != "2.1.10" || !sum.VersionAt.Equal(at("2026-11-10T08:05:00Z")) {
		t.Errorf("version %q at %v, want the most recent line's", sum.Version, sum.VersionAt)
	}
}

func TestParseSkipsAMalformedTitleOrBridgeRecord(t *testing.T) {
	lines := strings.Join([]string{
		`{"type":"custom-title","customTitle":"Named by hand"}`,
		`{"type":"bridge-session","bridgeSessionId":"cse_01Abc"}`,
		`{"type":"custom-title","customTitle":5}`,
		`{"type":"bridge-session","bridgeSessionId":7}`,
		`{"type":"ai-title","aiTitle":["x"]}`,
	}, "\n")
	_, sum, skipped, err := Parse(strings.NewReader(lines), "s1", attr, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 3 {
		t.Errorf("skipped %d lines, want the 3 malformed records", skipped)
	}
	if sum.CustomTitle != "Named by hand" || sum.AITitle != "" || sum.Bridge != "cse_01Abc" {
		t.Errorf("summary %+v: a malformed record must change nothing", sum)
	}
}

func TestParseTellsWhetherTheSessionEnded(t *testing.T) {
	turn := `{"type":"user","timestamp":"2026-11-10T08:00:00.000Z","cwd":"/p/alpha/fern","message":{"role":"user","content":"a prompt"}}`
	reply := `{"type":"assistant","timestamp":"2026-11-10T08:01:00.000Z","cwd":"/p/alpha/fern","message":{"id":"m1","model":"model-a"}}`
	end := `{"type":"cost-state","sessionId":"s1","totalCostUSD":0.5}`
	cases := []struct {
		name  string
		lines []string
		want  bool
	}{
		{"running", []string{turn, reply}, false},
		{"ended", []string{turn, reply, end}, true},
		{"ended, then a background record", []string{turn, reply, end, `{"type":"artifact-comment-monitor","sessionId":"s1"}`}, true},
		{"resumed after it ended", []string{turn, reply, end, `{"type":"bridge-session","bridgeSessionId":"cse_01Abc"}`, turn}, false},
		{"resumed and ended again", []string{turn, end, reply, end}, true},
		{"no turn at all", []string{end}, true},
		{"nothing", nil, false},
	}
	for _, c := range cases {
		_, sum, _, err := Parse(strings.NewReader(strings.Join(c.lines, "\n")), "s1", attr, time.UTC)
		if err != nil {
			t.Fatal(err)
		}
		if sum.Ended != c.want {
			t.Errorf("%s: ended %v, want %v", c.name, sum.Ended, c.want)
		}
	}
}

func TestTitleFallsBackToTheGeneratedOne(t *testing.T) {
	if got := (Summary{AITitle: "Generated"}).Title(); got != "Generated" {
		t.Fatalf("title %q", got)
	}
}

func TestRemote(t *testing.T) {
	cases := map[string]string{
		"cse_01Abc": "session_01Abc",
		"":          "",
		"cse_":      "",
		"xyz_01Abc": "",
		"01Abc":     "",
	}
	for bridge, want := range cases {
		if got := Remote(bridge); got != want {
			t.Errorf("Remote(%q) = %q, want %q", bridge, got, want)
		}
	}
}

func TestSummaryRemoteFallsBackToTheSpawnedWorktree(t *testing.T) {
	wt := "/p/alpha/fern/.claude/worktrees/"
	cases := []struct {
		bridge, cwd, want string
	}{
		{"cse_01Abc", wt + "bridge-cse_01Other", "session_01Abc"},
		{"", wt + "bridge-cse_01Abc", "session_01Abc"},
		{"", wt + "bridge-cse_01Abc/sub/dir", "session_01Abc"},
		{"xyz_01Abc", wt + "bridge-cse_01Other", ""},
		{"", wt + "feature", ""},
		{"", wt + "bridge-cse_", ""},
		{"", wt + "bridge-cse_01A?b", ""},
		{"", wt + "bridge-cse_01Ábc", ""},
		{"", wt + "bridge-cse_01A/.claude/worktrees/bridge-cse_01B", "session_01B"},
		{"", wt + "bridge-cse_01A/sub/.claude/worktrees/feature", "session_01A"},
		{"", "/p/alpha/fern/bridge-cse_01Abc", ""},
		{"", "/p/alpha/fern", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		s := Summary{Bridge: c.bridge, Cwd: c.cwd}
		if got := s.Remote(); got != c.want {
			t.Errorf("Summary{Bridge: %q, Cwd: %q}.Remote() = %q, want %q", c.bridge, c.cwd, got, c.want)
		}
	}
}
