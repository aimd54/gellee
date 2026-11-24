// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"reflect"
	"testing"
	"time"

	"github.com/aimd54/gellee/internal/state"
)

const fixtureState = "../../tests/fixtures/state"

var noon = time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

func build(t *testing.T) Snapshot {
	t.Helper()
	s, err := Build(fixtureState, noon)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func find(t *testing.T, projects []Project, key string) Project {
	t.Helper()
	for _, p := range projects {
		if p.Key == key {
			return p
		}
	}
	t.Fatalf("no project %s", key)
	return Project{}
}

func TestBuildFailsWithoutAConfig(t *testing.T) {
	if _, err := Build(t.TempDir(), noon); err == nil {
		t.Fatal("want an error")
	}
}

func TestBuildCollectsProblemsAndAppliesAliases(t *testing.T) {
	s := build(t)
	if s.Today != "2026-11-11" {
		t.Fatalf("today %q", s.Today)
	}
	if len(s.Problems) != 1 || s.Problems[0].File != "alpha/broken.md" {
		t.Fatalf("problems %+v", s.Problems)
	}
	for _, r := range s.Rows {
		if r.Project == "alpha/old-name" {
			t.Fatal("a former name survived")
		}
	}
}

func TestProjectsUsageWindows(t *testing.T) {
	projects := build(t).Projects()
	if len(projects) != 5 {
		t.Fatalf("%d projects", len(projects))
	}
	fern := find(t, projects, "alpha/fern")
	if fern.Usage7.Tokens != 3330 || fern.Usage30.Tokens != 3330 || fern.UsageAll.Tokens != 14430 {
		t.Errorf("fern tokens %d %d %d", fern.Usage7.Tokens, fern.Usage30.Tokens, fern.UsageAll.Tokens)
	}
	if fern.Usage7.CacheRead != 30000 || fern.Usage7.Messages != 3 || fern.Usage7.Cost != nil {
		t.Errorf("fern 7 days %+v", fern.Usage7)
	}
	if got := find(t, projects, "alpha/renamed").Usage7.Tokens; got != 222 {
		t.Errorf("renamed %d", got)
	}
	if got := find(t, projects, "alpha/fern-docs").UsageAll.Tokens; got != 0 {
		t.Errorf("fern-docs %d", got)
	}
}

func TestProjectsWarningsAndFields(t *testing.T) {
	projects := build(t).Projects()
	cases := map[string][]string{
		"alpha/fern":      {"stale", "unpushed"},
		"alpha/renamed":   {"unpushed"},
		"beta/tool":       {"uncommitted"},
		"alpha/fern-docs": {},
	}
	for key, want := range cases {
		if got := find(t, projects, key).Warnings; !reflect.DeepEqual(got, want) {
			t.Errorf("%s warnings %q, want %q", key, got, want)
		}
	}
	fern := find(t, projects, "alpha/fern")
	if fern.Next != "Run the task 17 review, then merge the branch." || fern.Group != "fieldwork" ||
		!fern.Tracked || fern.Sections != nil || fern.Git["m1"].Ahead != 2 {
		t.Errorf("fern %+v", fern)
	}
	if got := find(t, projects, "beta/legacy"); got.State != "done" || got.SupersededBy != "alpha/fern" {
		t.Errorf("legacy %+v", got)
	}
	if got := find(t, projects, "alpha/fern-docs"); got.ParkedReason != "Waiting for milestone 2" {
		t.Errorf("fern-docs %+v", got)
	}
	if want := "Milestone 2, task 17 paused before its review. Branch `m2-task-17`."; fern.Stopped != want {
		t.Errorf("fern stopped %q, want %q", fern.Stopped, want)
	}
}

func TestProjectStoppedIsTheFirstParagraph(t *testing.T) {
	s := Snapshot{Today: "2026-11-11", Statuses: []state.Status{{
		Project: "alpha/x", State: "active", Updated: "2026-11-11",
		Stopped: "Line one\nwraps here.\n\nSecond paragraph.",
	}}}
	if got := s.Projects()[0].Stopped; got != "Line one wraps here." {
		t.Fatalf("stopped %q", got)
	}
}

func TestCollectedIsTheMostRecentAcrossMachines(t *testing.T) {
	s := Snapshot{Git: map[string]state.GitFile{
		"m1": {Collected: "2026-11-11T10:00:00+01:00"},
		"m2": {Collected: "2026-11-11T09:30:00Z"},
		"m3": {Collected: "not a time"},
	}}
	// 09:30 UTC is 10:30 at +01:00, so m2 is the later one although its
	// text sorts first.
	if got := s.Collected(); got != "2026-11-11T09:30:00Z" {
		t.Fatalf("collected %q", got)
	}
	if got := (Snapshot{}).Collected(); got != "" {
		t.Fatalf("collected %q without git files", got)
	}
}

func TestSumPricesOnlyWhenEveryModelHasAPrice(t *testing.T) {
	s := build(t)
	s.Prices = map[string]state.Price{"model-a": {Input: 1, Output: 2, CacheWrite: 3, CacheRead: 4}}
	fern := s.Sum(func(r state.Row) bool { return r.Project == "alpha/fern" && r.Day == "2026-11-10" })
	// 10*1 + 100*2 + 1000*3 + 10000*4 = 43210 per million tokens.
	if fern.Cost == nil || *fern.Cost != 0.04321 {
		t.Fatalf("cost %v", fern.Cost)
	}
	mixed := s.Sum(func(r state.Row) bool { return r.Day == "2026-11-10" })
	if mixed.Cost != nil || mixed.Tokens != 1665 {
		t.Fatalf("mixed %+v", mixed)
	}
	if none := s.Sum(func(state.Row) bool { return false }); none.Cost != nil || none.Tokens != 0 {
		t.Fatalf("none %+v", none)
	}
}

func TestDayOffset(t *testing.T) {
	if got := DayOffset("2026-11-11", -29); got != "2026-10-13" {
		t.Fatalf("got %s", got)
	}
	if got := DayOffset("2026-03-01", -1); got != "2026-02-28" {
		t.Fatalf("got %s", got)
	}
}

func TestCollectedBreaksATieByTheSmallerText(t *testing.T) {
	// Both name the same instant, and the answer must not depend on
	// the order a map is read in.
	files := map[string]state.GitFile{
		"m1": {Collected: "2026-11-11T10:00:00+01:00"},
		"m2": {Collected: "2026-11-11T09:00:00Z"},
	}
	for i := 0; i < 50; i++ {
		if got := (Snapshot{Git: files}).Collected(); got != "2026-11-11T09:00:00Z" {
			t.Fatalf("collected %q", got)
		}
	}
}
