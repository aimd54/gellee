// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// Package server builds a read-only view of the state repository and
// serves it over HTTP.
package server

import (
	"time"

	"github.com/aimd54/gellee/internal/state"
)

// Usage is a sum of rows. Tokens leaves cache reads out. Cost is set
// only when every model in the sum has a price.
type Usage struct {
	Input      int64    `json:"input"`
	Output     int64    `json:"output"`
	CacheWrite int64    `json:"cache_write"`
	CacheRead  int64    `json:"cache_read"`
	Messages   int64    `json:"messages"`
	Tokens     int64    `json:"tokens"`
	Cost       *float64 `json:"cost"`
}

// Sections is the text of a status file.
type Sections struct {
	Stopped   string `json:"stopped"`
	Next      string `json:"next"`
	Remaining string `json:"remaining"`
	Ideas     string `json:"ideas"`
}

// Project is one status file with what the snapshot derives for it.
type Project struct {
	Key          string                      `json:"key"`
	Tracked      bool                        `json:"tracked"`
	State        string                      `json:"state"`
	Group        string                      `json:"group"`
	ParkedReason string                      `json:"parked_reason"`
	SupersededBy string                      `json:"superseded_by"`
	Updated      string                      `json:"updated"`
	Next         string                      `json:"next"`
	Stopped      string                      `json:"stopped"`
	Sections     *Sections                   `json:"sections,omitempty"`
	Git          map[string]state.GitProject `json:"git"`
	Warnings     []string                    `json:"warnings"`
	LastSession  *MachineSession             `json:"last_session"`
	Usage7       Usage                       `json:"usage_7"`
	Usage30      Usage                       `json:"usage_30"`
	UsageAll     Usage                       `json:"usage_all"`
}

// Snapshot is everything read from the state directory at one moment.
type Snapshot struct {
	Built         time.Time
	Today         string
	Config        state.Config
	Statuses      []state.Status
	Rows          []state.Row
	Git           map[string]state.GitFile
	Prices        map[string]state.Price
	Problems      []state.Problem
	Machines      map[string]state.MachineFile
	UsageMachines []string
}

// Build reads the state directory. Only a missing or unusable config is
// an error; every other unreadable file becomes a problem.
func Build(stateDir string, now time.Time) (Snapshot, error) {
	cfg, err := state.LoadConfig(stateDir)
	if err != nil {
		return Snapshot{}, err
	}
	s := Snapshot{Built: now, Today: now.Format("2006-01-02"), Config: cfg, Problems: []state.Problem{}}
	var problems []state.Problem
	s.Statuses, problems = state.LoadStatuses(stateDir, cfg.Roots)
	s.Problems = append(s.Problems, problems...)
	rows, problems := state.AllRows(stateDir)
	s.Problems = append(s.Problems, problems...)
	s.Rows = state.Rename(rows, state.Aliases(s.Statuses))
	s.Git, problems = state.GitFiles(stateDir)
	s.Problems = append(s.Problems, problems...)
	s.Machines, problems = state.MachineFiles(stateDir)
	s.Problems = append(s.Problems, problems...)
	s.Machines = renameSessions(s.Machines, state.Aliases(s.Statuses))
	s.UsageMachines = state.UsageMachines(stateDir)
	if s.Prices, err = state.LoadPrices(stateDir); err != nil {
		s.Problems = append(s.Problems, state.Problem{File: "prices", Message: err.Error()})
	}
	return s, nil
}

// renameSessions applies former project names to the sessions of every
// machine file.
func renameSessions(files map[string]state.MachineFile, aliases map[string]string) map[string]state.MachineFile {
	for name, f := range files {
		sessions := make([]state.Session, len(f.Sessions))
		for i, s := range f.Sessions {
			if current, ok := aliases[s.Project]; ok {
				s.Project = current
			}
			sessions[i] = s
		}
		f.Sessions = sessions
		files[name] = f
	}
	return files
}

// DayOffset returns day moved by n days.
func DayOffset(day string, n int) string {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return day
	}
	return t.AddDate(0, 0, n).Format("2006-01-02")
}

// Collected returns the most recent collection time among the git
// files, as it was written, or an empty string.
func (s Snapshot) Collected() string {
	var latest time.Time
	out := ""
	for _, file := range s.Git {
		t, err := time.Parse(time.RFC3339, file.Collected)
		if err != nil {
			continue
		}
		// Equal instants are settled by the text, so that the answer
		// does not depend on map order.
		if out == "" || t.After(latest) || (t.Equal(latest) && file.Collected < out) {
			latest, out = t, file.Collected
		}
	}
	return out
}

// Sum adds the rows keep accepts.
func (s Snapshot) Sum(keep func(state.Row) bool) Usage {
	var u Usage
	cost, priced, n := 0.0, true, 0
	for _, r := range s.Rows {
		if !keep(r) {
			continue
		}
		n++
		u.Input += r.Input
		u.Output += r.Output
		u.CacheWrite += r.CacheWrite
		u.CacheRead += r.CacheRead
		u.Messages += r.Messages
		p, ok := s.Prices[r.Model]
		if !ok {
			priced = false
			continue
		}
		cost += (float64(r.Input)*p.Input + float64(r.Output)*p.Output +
			float64(r.CacheWrite)*p.CacheWrite + float64(r.CacheRead)*p.CacheRead) / 1e6
	}
	u.Tokens = u.Input + u.CacheWrite + u.Output
	if n > 0 && priced {
		u.Cost = &cost
	}
	return u
}

// Projects returns every status file as a Project, sorted by key.
func (s Snapshot) Projects() []Project {
	out := make([]Project, 0, len(s.Statuses))
	for _, st := range s.Statuses {
		out = append(out, s.project(st))
	}
	return out
}

func (s Snapshot) project(st state.Status) Project {
	p := Project{
		Key: st.Project, Tracked: true, State: st.State, Group: st.Group,
		ParkedReason: st.ParkedReason, SupersededBy: st.SupersededBy,
		Updated: st.Updated, Next: state.FirstParagraph(st.Next),
		Stopped: state.FirstParagraph(st.Stopped),
		Git:     map[string]state.GitProject{},
	}
	var trees []state.GitProject
	for machine, file := range s.Git {
		g, ok := file.Projects[st.Project]
		if !ok {
			continue
		}
		p.Git[machine] = g
		trees = append(trees, g)
	}
	p.Warnings = warnings(trees, st.Updated)
	if sessions := s.Sessions(st.Project); len(sessions) > 0 {
		p.LastSession = &sessions[0]
	}
	since := func(from string) func(state.Row) bool {
		return func(r state.Row) bool { return r.Project == st.Project && r.Day >= from && r.Day <= s.Today }
	}
	p.Usage7 = s.Sum(since(DayOffset(s.Today, -6)))
	p.Usage30 = s.Sum(since(DayOffset(s.Today, -29)))
	p.UsageAll = s.Sum(func(r state.Row) bool { return r.Project == st.Project })
	return p
}

// warnings names what working trees show against a status updated on
// that day, in the order stale, unpushed, uncommitted. Without a day,
// as for a checkout no status file tracks, nothing is stale.
func warnings(trees []state.GitProject, updated string) []string {
	stale, unpushed, uncommitted := false, false, false
	for _, g := range trees {
		// The date part of an RFC 3339 timestamp compares as a string.
		if updated != "" && len(g.LastCommit) >= 10 && g.LastCommit[:10] > updated {
			stale = true
		}
		if g.Ahead > 0 || !g.Remote {
			unpushed = true
		}
		if g.Dirty > 0 {
			uncommitted = true
		}
	}
	out := []string{}
	for _, w := range []struct {
		on   bool
		name string
	}{{stale, "stale"}, {unpushed, "unpushed"}, {uncommitted, "uncommitted"}} {
		if w.on {
			out = append(out, w.name)
		}
	}
	return out
}
