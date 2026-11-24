// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package collect

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aimd54/gellee/internal/state"
)

// writeTranscript writes a transcript below the transcripts directory,
// one line per argument.
func writeTranscript(t *testing.T, o Options, rel string, lines ...string) {
	t.Helper()
	path := filepath.Join(o.TranscriptsDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func machineOf(t *testing.T, o Options) state.MachineFile {
	t.Helper()
	f, err := state.ReadMachineFile(o.StateDir, o.Machine)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func sessionOf(t *testing.T, f state.MachineFile, id string) state.Session {
	t.Helper()
	for _, s := range f.Sessions {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no session %s in %+v", id, f.Sessions)
	return state.Session{}
}

func ids(f state.MachineFile) string {
	var out []string
	for _, s := range f.Sessions {
		out = append(out, s.ID)
	}
	return strings.Join(out, " ")
}

// The world's clock is 2026-11-11T10:00:00Z.
func addSessions(t *testing.T, o Options) {
	t.Helper()
	writeTranscript(t, o, "-p-alpha-fern/s3.jsonl",
		`{"type":"user","timestamp":"2026-11-10T08:00:00.000Z","cwd":"/p/alpha/fern","gitBranch":"main","version":"2.1.300","sessionId":"s3","message":{"role":"user","content":"a prompt"}}`,
		`{"type":"custom-title","customTitle":"Named by hand","sessionId":"s3"}`,
		`{"type":"ai-title","aiTitle":"Generated title","sessionId":"s3"}`,
		`{"type":"bridge-session","bridgeSessionId":"cse_01Fern","accountUuid":"acct-1","organizationUuid":"org-1"}`,
		`{"type":"bridge-session","bridgeSessionId":""}`,
		`{"type":"assistant","timestamp":"2026-11-10T08:10:00.000Z","cwd":"/p/alpha/fern","message":{"id":"x1","model":"model-a","usage":{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":1,"cache_read_input_tokens":1}}}`)
	writeTranscript(t, o, "-p-alpha-fern/s3/subagents/agent-b1.jsonl",
		`{"type":"assistant","timestamp":"2026-11-10T07:55:00.000Z","cwd":"/p/alpha/fern","message":{"id":"x2","model":"model-a","usage":{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":1,"cache_read_input_tokens":1}}}`,
		`{"type":"assistant","timestamp":"2026-11-10T09:30:00.000Z","cwd":"/p/alpha/fern","message":{"id":"x3","model":"model-a","usage":{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":1,"cache_read_input_tokens":1}}}`)
	writeTranscript(t, o, "-p-beta-tool/s4.jsonl",
		`{"type":"user","timestamp":"2026-11-11T09:30:00.000Z","cwd":"/p/beta/tool","gitBranch":"main","version":"2.1.10","message":{"role":"user","content":"a prompt"}}`,
		`{"type":"ai-title","aiTitle":"Tool title"}`,
		`{"type":"bridge-session","bridgeSessionId":"xyz_01Tool"}`)
	writeTranscript(t, o, "-p-beta-tool/s5.jsonl",
		`{"type":"user","timestamp":"2026-11-05T10:30:00.000Z","cwd":"/p/beta/tool","message":{"role":"user","content":"a prompt"}}`)
	writeTranscript(t, o, "-p-beta-tool/s6.jsonl",
		`{"type":"user","timestamp":"2026-11-03T10:00:00.000Z","cwd":"/p/beta/tool","message":{"role":"user","content":"a prompt"}}`)
}

func TestMachineFileSummarisesTheSessions(t *testing.T) {
	o := world(t)
	addSessions(t, o)
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	f := machineOf(t, o)
	if f.Collected != "2026-11-11T10:00:00Z" {
		t.Errorf("collected %q", f.Collected)
	}
	// The most recent line on the machine, not the highest version.
	if f.ClaudeCode != "2.1.10" {
		t.Errorf("claude_code %q", f.ClaudeCode)
	}
	// s5 is 6 days old and listed; s6 is 8 days old and is not.
	if got := ids(f); got != "s4 s2 s1 s3 s5" {
		t.Errorf("sessions %q", got)
	}
	want := state.Session{
		ID: "s3", Project: "alpha/fern", Branch: "main", Title: "Named by hand",
		Remote: "session_01Fern", Started: "2026-11-10T07:55:00Z", Last: "2026-11-10T09:30:00Z",
	}
	if got := sessionOf(t, f, "s3"); got != want {
		t.Errorf("s3 %+v", got)
	}
	if got := sessionOf(t, f, "s4"); got.Remote != "" || got.Title != "Tool title" {
		t.Errorf("s4 %+v: a prefix other than cse_ gives no remote", got)
	}
	// Recorded under alpha/old-name, which alpha/renamed lists as a former name.
	if got := sessionOf(t, f, "s2"); got.Project != "alpha/renamed" {
		t.Errorf("s2 %+v", got)
	}
}

func TestMachineFileWritesTimesInTheMachinesZone(t *testing.T) {
	o := world(t)
	o.Loc = time.FixedZone("east", 2*60*60)
	addSessions(t, o)
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	got := sessionOf(t, machineOf(t, o), "s3")
	if got.Started != "2026-11-10T09:55:00+02:00" || got.Last != "2026-11-10T11:30:00+02:00" {
		t.Errorf("s3 started %q last %q, want the instants written at +02:00", got.Started, got.Last)
	}
}

func TestMachineFileKeepsOtherSessionsOnTheMachine(t *testing.T) {
	o := world(t)
	secrets := []string{"Secret other title", "secret-branch", "secret prompt text", "secret last prompt",
		"acct-0000-secret", "org-0000-secret", "fern prompt text", "acct-0000-fern"}
	writeTranscript(t, o, "-elsewhere-secret/s8.jsonl",
		`{"type":"user","timestamp":"2026-11-11T08:00:00.000Z","cwd":"/elsewhere/secret","gitBranch":"secret-branch","message":{"role":"user","content":"secret prompt text"}}`,
		`{"type":"ai-title","aiTitle":"Secret other title"}`,
		`{"type":"bridge-session","bridgeSessionId":"cse_01Other","accountUuid":"acct-0000-secret","organizationUuid":"org-0000-secret"}`,
		`{"type":"last-prompt","lastPrompt":"secret last prompt"}`,
		`{"type":"system","timestamp":"2026-11-11T08:20:00.000Z","cwd":"/elsewhere/secret"}`)
	writeTranscript(t, o, "-p-alpha-fern/s9.jsonl",
		`{"type":"user","timestamp":"2026-11-11T08:30:00.000Z","cwd":"/p/alpha/fern","gitBranch":"main","message":{"role":"user","content":"fern prompt text"}}`,
		`{"type":"bridge-session","bridgeSessionId":"cse_01Fern","accountUuid":"acct-0000-fern"}`,
		`{"type":"last-prompt","lastPrompt":"fern prompt text"}`)
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	f := machineOf(t, o)
	want := state.Session{ID: "s8", Project: "other", Remote: "session_01Other",
		Started: "2026-11-11T08:00:00Z", Last: "2026-11-11T08:20:00Z"}
	if got := sessionOf(t, f, "s8"); got != want {
		t.Errorf("s8 %+v", got)
	}
	if got := sessionOf(t, f, "s9"); got.Branch != "main" || got.Remote != "session_01Fern" {
		t.Errorf("s9 %+v", got)
	}
	noFileHolds(t, o.StateDir, secrets...)
	noCommitHolds(t, o, secrets...)
	// The cache stays on the machine with titles and branches, never
	// prompt text or account UUIDs.
	noFileHolds(t, cacheOf(t, o), "secret prompt text", "secret last prompt",
		"acct-0000-secret", "org-0000-secret", "fern prompt text", "acct-0000-fern")
}

// noFileHolds fails the test for each file below dir, .git aside, that
// holds one of the secrets.
func noFileHolds(t *testing.T, dir string, secrets ...string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, s := range secrets {
			if strings.Contains(string(data), s) {
				t.Errorf("%s holds %q", path, s)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// noCommitHolds checks that the collector's commit holds the machine
// file and none of the secrets.
func noCommitHolds(t *testing.T, o Options, secrets ...string) {
	t.Helper()
	committed := git(t, o.StateDir, "show", "HEAD")
	if !strings.Contains(committed, "machines/m1.json") {
		t.Fatalf("the commit does not hold the machine file:\n%s", committed)
	}
	for _, s := range secrets {
		if strings.Contains(committed, s) {
			t.Errorf("the commit holds %q", s)
		}
	}
}

// cacheOf returns the cache directory once its transcripts file exists.
func cacheOf(t *testing.T, o Options) string {
	t.Helper()
	if _, err := os.Stat(filepath.Join(o.CacheDir, "transcripts.json")); err != nil {
		t.Fatalf("no cache to scan: %v", err)
	}
	return o.CacheDir
}

// A session Remote Control spawns into a worktree writes no
// bridge-session record; its worktree's name carries the bridge ID.
func TestMachineFileLinksASessionSpawnedIntoAWorktree(t *testing.T) {
	o := world(t)
	secrets := []string{"spawned prompt text", "queued prompt text", "acct-0000-spawn", "org-0000-spawn", "atis-0000-spawn",
		"Out title", "out-branch"}
	writeTranscript(t, o, "-elsewhere--claude-worktrees-bridge-cse-01Out/s11.jsonl",
		`{"type":"user","timestamp":"2026-11-11T08:30:00.000Z","cwd":"/elsewhere/.claude/worktrees/bridge-cse_01Out","gitBranch":"out-branch","message":{"role":"user","content":"spawned prompt text"}}`,
		`{"type":"custom-title","customTitle":"Out title","sessionId":"s11"}`)
	wt := "/p/alpha/fern/.claude/worktrees/bridge-cse_01Spawn"
	writeTranscript(t, o, "-p-alpha-fern--claude-worktrees-bridge-cse-01Spawn/s10.jsonl",
		`{"type":"queue-operation","operation":"enqueue","timestamp":"2026-11-11T08:00:00.000Z","sessionId":"s10","content":"queued prompt text"}`,
		`{"type":"session-owner","accountUuid":"acct-0000-spawn","organizationUuid":"org-0000-spawn","sessionId":"s10"}`,
		`{"type":"atis-latch","atis":"atis-0000-spawn","sessionId":"s10"}`,
		`{"type":"user","timestamp":"2026-11-11T08:00:01.000Z","cwd":"`+wt+`","gitBranch":"worktree-bridge-cse_01Spawn","entrypoint":"sdk-cli","message":{"role":"user","content":"spawned prompt text"}}`,
		`{"type":"custom-title","customTitle":"Worker check","sessionId":"s10"}`,
		`{"type":"agent-name","agentName":"Worker check","sessionId":"s10"}`,
		`{"type":"last-prompt","lastPrompt":"spawned prompt text","sessionId":"s10"}`,
		`{"type":"assistant","timestamp":"2026-11-11T08:00:20.000Z","cwd":"`+wt+`","gitBranch":"worktree-bridge-cse_01Spawn","message":{"id":"x10","model":"model-a","usage":{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":1,"cache_read_input_tokens":1}}}`)
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	want := state.Session{
		ID: "s10", Project: "alpha/fern", Branch: "worktree-bridge-cse_01Spawn", Title: "Worker check",
		Remote: "session_01Spawn", Started: "2026-11-11T08:00:00Z", Last: "2026-11-11T08:00:20Z",
	}
	f := machineOf(t, o)
	if got := sessionOf(t, f, "s10"); got != want {
		t.Errorf("s10 %+v", got)
	}
	// Outside the roots: the link and times, no title or branch.
	wantOut := state.Session{ID: "s11", Project: "other", Remote: "session_01Out",
		Started: "2026-11-11T08:30:00Z", Last: "2026-11-11T08:30:00Z"}
	if got := sessionOf(t, f, "s11"); got != wantOut {
		t.Errorf("s11 %+v", got)
	}
	noFileHolds(t, o.StateDir, secrets...)
	noCommitHolds(t, o, secrets...)
	noFileHolds(t, cacheOf(t, o), secrets[:5]...)
}

func TestMachineFileOfAMachineWithoutTranscripts(t *testing.T) {
	o := world(t)
	o.TranscriptsDir = t.TempDir()
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	f := machineOf(t, o)
	if f.Collected != "2026-11-11T10:00:00Z" || f.ClaudeCode != "" || f.Sessions == nil || len(f.Sessions) != 0 {
		t.Fatalf("machine file %+v", f)
	}
}

func TestRunReparsesACacheOfAnEarlierLayout(t *testing.T) {
	o := world(t)
	opens := 0
	o.Open = func(p string) (io.ReadCloser, error) { opens++; return os.Open(p) }
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(o.CacheDir, "transcripts.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stored := cache{}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored.Key, "ended|") {
		t.Fatalf("stored key %q does not start with ended|", stored.Key)
	}
	// The keys of the layouts before summaries, and before ended.
	rest := strings.TrimPrefix(stored.Key, "ended|")
	for i, old := range []string{rest, "sessions|" + rest} {
		stored.Key = old
		if err := saveCache(path, stored); err != nil {
			t.Fatal(err)
		}
		if err := Run(o); err != nil {
			t.Fatal(err)
		}
		if want := 3 * (i + 2); opens != want {
			t.Fatalf("after a cache under %q: opened %d files in all, want %d", old, opens, want)
		}
		if err := json.Unmarshal(mustRead(t, path), &stored); err != nil {
			t.Fatal(err)
		}
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestMachineFileTellsWhetherASessionEnded(t *testing.T) {
	o := world(t)
	turn := func(ts string) string {
		return `{"type":"user","timestamp":"` + ts + `","cwd":"/p/alpha/fern","gitBranch":"main","message":{"role":"user","content":"a prompt"}}`
	}
	end := `{"type":"cost-state","sessionId":"x","totalCostUSD":0.123456,"modelUsage":{"model-a":{"costUSD":0.654321}}}`
	writeTranscript(t, o, "-p-alpha-fern/s12.jsonl", turn("2026-11-11T08:00:00.000Z"), end)
	writeTranscript(t, o, "-p-alpha-fern/s13.jsonl", turn("2026-11-11T08:00:00.000Z"))
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	f := machineOf(t, o)
	if got := sessionOf(t, f, "s12"); !got.Ended {
		t.Errorf("s12 %+v: a cost-state after its last turn ends it", got)
	}
	if got := sessionOf(t, f, "s13"); got.Ended {
		t.Errorf("s13 %+v: without a cost-state it has not ended", got)
	}
	if !strings.Contains(string(mustRead(t, filepath.Join(o.StateDir, "machines", "m1.json"))), `"ended": true`) {
		t.Error(`the machine file holds no "ended": true`)
	}
	// Only the fact that the session ended leaves its cost-state.
	noFileHolds(t, o.StateDir, "0.123456", "0.654321", "totalCostUSD", "modelUsage")
	noFileHolds(t, cacheOf(t, o), "0.123456", "0.654321", "totalCostUSD", "modelUsage")
}

func TestMachineFileKeepsNoTitleOrBranchForAnOtherSessionKeptFromBefore(t *testing.T) {
	o := world(t)
	prev := state.MachineFile{Collected: "2026-11-11T08:00:00Z", Sessions: []state.Session{
		{ID: "gone-other", Project: "other", Remote: "session_01Gone", Title: "Carried secret title", Branch: "carried-secret-branch",
			Started: "2026-11-10T08:00:00Z", Last: "2026-11-10T09:00:00Z"},
		{ID: "gone-fern", Project: "alpha/fern", Title: "Kept fern title", Branch: "fern-branch",
			Started: "2026-11-10T08:00:00Z", Last: "2026-11-10T09:30:00Z"},
	}}
	if err := state.WriteMachineFile(o.StateDir, o.Machine, prev); err != nil {
		t.Fatal(err)
	}
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	f := machineOf(t, o)
	want := state.Session{ID: "gone-other", Project: "other", Remote: "session_01Gone",
		Started: "2026-11-10T08:00:00Z", Last: "2026-11-10T09:00:00Z"}
	if got := sessionOf(t, f, "gone-other"); got != want {
		t.Errorf("kept other session %+v, want %+v", got, want)
	}
	if got := sessionOf(t, f, "gone-fern"); got.Title != "Kept fern title" || got.Branch != "fern-branch" {
		t.Errorf("kept project session %+v", got)
	}
	noFileHolds(t, o.StateDir, "Carried secret title", "carried-secret-branch")
}

func TestAnUnreadablePreviousMachineFileIsReportedAndReplaced(t *testing.T) {
	o := world(t)
	addSessions(t, o)
	if err := os.MkdirAll(filepath.Join(o.StateDir, "machines"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(o.StateDir, "machines", "m1.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	o.Out = &out
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "its sessions are listed anew") {
		t.Fatalf("output %q", out.String())
	}
	f := machineOf(t, o)
	if f.Collected != "2026-11-11T10:00:00Z" || sessionOf(t, f, "s3").Title != "Named by hand" || sessionOf(t, f, "s4").Title != "Tool title" || len(f.Sessions) != 5 {
		t.Fatalf("machine file %+v", f)
	}
}
