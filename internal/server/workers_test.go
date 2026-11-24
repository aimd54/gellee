// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// stateWith copies the fixture's config and status files and adds files.
// The snapshot is built at noon, 2026-11-11T12:00:00Z.
func stateWith(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "state")
	if err := os.CopyFS(dir, os.DirFS(fixtureState)); err != nil {
		t.Fatal(err)
	}
	for _, generated := range []string{"git", "usage", "machines"} {
		if err := os.RemoveAll(filepath.Join(dir, generated)); err != nil {
			t.Fatal(err)
		}
	}
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const row = `{"day":"2026-11-11","session":"s1","project":"alpha/fern","model":"model-a","input":1,"output":1,"cache_write":1,"cache_read":1,"messages":1}` + "\n"

// workersWorld has one worker of each kind:
//   - m1, all three files, collected 1h59m before noon;
//   - m2, only git/, collected 2h01m before noon;
//   - m3, usage/ and an unreadable machine file;
//   - m4, seen 8 days ago with a newer version;
//   - m5 and m6, collected at 08:00, an older and an unparsed version.
func workersWorld(t *testing.T) Snapshot {
	t.Helper()
	dir := stateWith(t, map[string]string{
		"git/m1.json": `{"collected":"2026-11-11T10:01:00Z","projects":{
			"alpha/fern":{"branch":"m2-task-17","head":"abc1234","last_commit":"2026-11-10T09:00:00Z","remote":true,"ahead":2,"dirty":0},
			"beta/tool":{"branch":"main","head":"0a1b2c3","last_commit":"2026-11-01T09:00:00Z","remote":true,"ahead":0,"dirty":0}}}`,
		"machines/m1.json": `{"collected":"2026-11-11T10:01:00Z","claude_code":"2.1.323",
			"memories":{"pushed":"2026-11-10T10:31:00Z","set_aside":2},
			"sessions":[{"id":"s1","project":"alpha/fern","branch":"m2-task-17","title":"Review","remote":"session_01A","started":"2026-11-11T09:00:00Z","last":"2026-11-11T09:30:00Z"}]}`,
		"usage/m1/2026-11.jsonl": row,
		"git/m2.json": `{"collected":"2026-11-11T09:59:00Z","projects":{
			"alpha/fern":{"branch":"main","head":"abc1234","last_commit":"2026-11-10T09:00:00Z","remote":true,"ahead":0,"dirty":3}}}`,
		"usage/m3/2026-11.jsonl": row,
		"machines/m3.json":       `{`,
		"machines/m4.json":       `{"collected":"2026-11-03T12:00:00Z","claude_code":"2.2.0","sessions":[]}`,
		"machines/m5.json":       `{"collected":"2026-11-11T08:00:00Z","claude_code":"2.1.9","memories":{"pushed":"","set_aside":0},"sessions":[]}`,
		"machines/m6.json":       `{"collected":"2026-11-11T08:00:00Z","claude_code":"dev-build","memories":{"pushed":"2026-11-10T07:00:00Z","set_aside":0},"sessions":[]}`,
	})
	s, err := Build(dir, noon)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func byName(t *testing.T, ws []Worker, name string) Worker {
	t.Helper()
	for _, w := range ws {
		if w.Name == name {
			return w
		}
	}
	t.Fatalf("no worker %s", name)
	return Worker{}
}

func TestWorkersAreEveryMachineTheMostRecentlySeenFirst(t *testing.T) {
	ws := workersWorld(t).Workers()
	var names []string
	for _, w := range ws {
		names = append(names, w.Name)
	}
	if got := strings.Join(names, " "); got != "m1 m2 m5 m6 m4 m3" {
		t.Fatalf("workers %q", got)
	}
}

func TestAnUnreadableMachineFileIsAProblemAndTheRestIsServed(t *testing.T) {
	s := workersWorld(t)
	found := false
	for _, p := range s.Problems {
		found = found || p.File == "machines/m3.json"
	}
	if !found {
		t.Fatalf("problems %+v", s.Problems)
	}
	m3 := byName(t, s.Workers(), "m3")
	if m3.Collected != "" || m3.Up || m3.ClaudeCode != "" || len(m3.Sessions) != 0 || m3.Sessions == nil {
		t.Fatalf("m3 %+v", m3)
	}
}

func TestAWorkerWithAllThreeFiles(t *testing.T) {
	m1 := byName(t, workersWorld(t).Workers(), "m1")
	if !m1.Up || m1.Collected != "2026-11-11T10:01:00Z" || m1.ClaudeCode != "2.1.323" || m1.Behind {
		t.Fatalf("m1 %+v", m1)
	}
	if m1.Memories == nil || *m1.Memories != (MemorySync{Pushed: "2026-11-10T10:31:00Z", SetAside: 2, Stale: false}) {
		t.Fatalf("m1 memories %+v", m1.Memories)
	}
	if len(m1.Sessions) != 1 || m1.Sessions[0].Remote != "session_01A" {
		t.Fatalf("m1 sessions %+v", m1.Sessions)
	}
	want := []Checkout{
		{Project: "alpha/fern", Branch: "m2-task-17", Warnings: []string{"stale", "unpushed"}},
		{Project: "beta/tool", Branch: "main", Warnings: []string{}},
	}
	if !reflect.DeepEqual(m1.Checkouts, want) {
		t.Fatalf("m1 checkouts %+v", m1.Checkouts)
	}
}

func TestAWorkerWithOnlyItsGitFile(t *testing.T) {
	m2 := byName(t, workersWorld(t).Workers(), "m2")
	if m2.Up || m2.Collected != "2026-11-11T09:59:00Z" || m2.ClaudeCode != "" || m2.Behind || m2.Memories != nil {
		t.Fatalf("m2 %+v", m2)
	}
	// Its own figures, not the union with m1's.
	want := []Checkout{{Project: "alpha/fern", Branch: "main", Warnings: []string{"stale", "uncommitted"}}}
	if !reflect.DeepEqual(m2.Checkouts, want) {
		t.Fatalf("m2 checkouts %+v", m2.Checkouts)
	}
}

func TestAVersionIsBehindTheNewestSeenInSevenDays(t *testing.T) {
	ws := workersWorld(t).Workers()
	// 2.1.9 is older than 2.1.323 by number, not by text.
	if !byName(t, ws, "m5").Behind {
		t.Error("m5 must be behind")
	}
	// m4 is newer but was seen 8 days ago, so it sets nothing.
	if byName(t, ws, "m1").Behind || byName(t, ws, "m4").Behind {
		t.Error("m1 and m4 must not be behind")
	}
	if byName(t, ws, "m6").Behind {
		t.Error("a version that does not parse is never marked")
	}
}

func TestTheMemorySyncIsStaleAfterADayOrWithoutAPush(t *testing.T) {
	ws := workersWorld(t).Workers()
	if m := byName(t, ws, "m5").Memories; m == nil || !m.Stale {
		t.Errorf("m5 never pushed: %+v", m)
	}
	if m := byName(t, ws, "m6").Memories; m == nil || !m.Stale {
		t.Errorf("m6 pushed 25 hours before collecting: %+v", m)
	}
	if m := byName(t, ws, "m1").Memories; m == nil || m.Stale {
		t.Errorf("m1 pushed 23h30 before collecting: %+v", m)
	}
}

// sessionsWorld has alpha/fern worked on two machines whose clocks are
// an hour apart: m2's session ended later although its text sorts first.
func sessionsWorld(t *testing.T) string {
	t.Helper()
	return stateWith(t, map[string]string{
		"machines/m1.json": `{"collected":"2026-11-11T11:00:00+01:00","claude_code":"2.1.323","sessions":[
			{"id":"a","project":"alpha/fern","branch":"main","title":"First","remote":"","started":"2026-11-11T09:00:00+01:00","last":"2026-11-11T10:30:00+01:00"},
			{"id":"b","project":"alpha/old-name","branch":"main","title":"Renamed","remote":"","started":"2026-11-10T09:00:00+01:00","last":"2026-11-10T10:00:00+01:00"}]}`,
		"machines/m2.json": `{"collected":"2026-11-11T10:00:00Z","claude_code":"2.1.323","sessions":[
			{"id":"c","project":"alpha/fern","branch":"m2-task-17","title":"Second","remote":"session_01C","started":"2026-11-11T09:00:00Z","last":"2026-11-11T09:45:00Z","ended":true}]}`,
	})
}

func TestSessionsOfAProjectAcrossWorkersMostRecentFirst(t *testing.T) {
	s, err := Build(sessionsWorld(t), noon)
	if err != nil {
		t.Fatal(err)
	}
	got := s.Sessions("alpha/fern")
	if len(got) != 2 || got[0].Machine != "m2" || got[0].ID != "c" || got[1].Machine != "m1" || got[1].ID != "a" {
		t.Fatalf("sessions %+v", got)
	}
	if r := s.Sessions("alpha/renamed"); len(r) != 1 || r[0].ID != "b" {
		t.Fatalf("a session under a former name: %+v", r)
	}
	if none := s.Sessions("beta/tool"); none == nil || len(none) != 0 {
		t.Fatalf("beta/tool %+v", none)
	}
}

func TestAWorkerWhoseClockRunsAheadIsNotUp(t *testing.T) {
	dir := stateWith(t, map[string]string{
		"machines/m1.json": `{"collected":"2026-11-11T11:00:00Z","claude_code":"2.1.323","sessions":[]}`,
		"machines/m9.json": `{"collected":"2026-11-11T15:00:00Z","claude_code":"2.2.0","sessions":[]}`,
		"machines/m8.json": `{"collected":"2026-11-11T12:30:00Z","claude_code":"2.1.323","sessions":[]}`,
	})
	s, err := Build(dir, noon)
	if err != nil {
		t.Fatal(err)
	}
	ws := s.Workers()
	if w := byName(t, ws, "m9"); w.Up {
		t.Errorf("m9 collected 3h ahead is up: %+v", w)
	}
	if w := byName(t, ws, "m1"); w.Behind {
		t.Errorf("m1 is behind a version only a clock running ahead reports: %+v", w)
	}
	if w := byName(t, ws, "m8"); !w.Up {
		t.Errorf("m8 collected 30m ahead is not up: %+v", w)
	}
}

func TestACheckoutWithoutADatedStatusIsNotStale(t *testing.T) {
	dir := stateWith(t, map[string]string{
		"alpha/undated.md": "---\nproject: alpha/undated\nstate: active\n---\n\n## Where I stopped\n\nx\n",
		"git/m1.json": `{"collected":"2026-11-11T10:01:00Z","projects":{
			"alpha/undated":{"branch":"main","head":"abc1234","last_commit":"2026-11-11T09:00:00Z","remote":true,"ahead":0,"dirty":0},
			"beta/untracked":{"branch":"main","head":"0a1b2c3","last_commit":"2026-11-11T09:00:00Z","remote":true,"ahead":0,"dirty":0}}}`,
	})
	s, err := Build(dir, noon)
	if err != nil {
		t.Fatal(err)
	}
	w := byName(t, s.Workers(), "m1")
	if len(w.Checkouts) != 2 {
		t.Fatalf("checkouts %+v", w.Checkouts)
	}
	for _, c := range w.Checkouts {
		if len(c.Warnings) != 0 {
			t.Errorf("%s warns %v, want none", c.Project, c.Warnings)
		}
	}
}

func TestSessionsAtOneInstantOrderByMachineThenID(t *testing.T) {
	dir := stateWith(t, map[string]string{
		"machines/m2.json": `{"collected":"2026-11-11T10:00:00Z","sessions":[
			{"id":"a","project":"alpha/fern","started":"2026-11-11T08:00:00Z","last":"2026-11-11T09:00:00Z"}]}`,
		"machines/m1.json": `{"collected":"2026-11-11T11:00:00+01:00","sessions":[
			{"id":"y","project":"alpha/fern","started":"2026-11-11T09:00:00+01:00","last":"2026-11-11T10:00:00+01:00"},
			{"id":"x","project":"alpha/fern","started":"2026-11-11T09:00:00+01:00","last":"2026-11-11T10:00:00+01:00"}]}`,
	})
	s, err := Build(dir, noon)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range s.Sessions("alpha/fern") {
		got = append(got, m.Machine+"/"+m.ID)
	}
	if want := "m1/x m1/y m2/a"; strings.Join(got, " ") != want {
		t.Fatalf("order %v, want %s", got, want)
	}
}

func TestAWorkerSessionIsTrackedWhenItsProjectHasAStatusFile(t *testing.T) {
	dir := stateWith(t, map[string]string{
		"machines/m1.json": `{"collected":"2026-11-11T10:01:00Z","claude_code":"2.1.323","sessions":[
			{"id":"a","project":"alpha/fern","started":"2026-11-11T09:00:00Z","last":"2026-11-11T09:30:00Z"},
			{"id":"b","project":"beta/scratch","started":"2026-11-11T08:00:00Z","last":"2026-11-11T08:30:00Z"},
			{"id":"c","project":"other","started":"2026-11-11T07:00:00Z","last":"2026-11-11T07:30:00Z"}]}`,
	})
	s, err := Build(dir, noon)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, sess := range byName(t, s.Workers(), "m1").Sessions {
		got[sess.ID] = sess.Tracked
	}
	want := map[string]bool{"a": true, "b": false, "c": false}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tracked %v, want %v", got, want)
	}
}
