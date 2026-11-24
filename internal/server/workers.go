// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aimd54/gellee/internal/state"
)

const (
	upWithin   = 2 * time.Hour      // the hourly timer and a margin
	recent     = 7 * 24 * time.Hour // workers whose version counts for the newest
	pushWithin = 24 * time.Hour     // the threshold of the session-start notice
)

// MemorySync is a worker's memory store as its collector saw it.
type MemorySync struct {
	Pushed   string `json:"pushed"`
	SetAside int    `json:"set_aside"`
	Stale    bool   `json:"stale"`
}

// Checkout is one working tree a worker reported, with the warnings its
// own figures raise.
type Checkout struct {
	Project  string   `json:"project"`
	Branch   string   `json:"branch"`
	Warnings []string `json:"warnings"`
}

// WorkerSession is a session with whether a status file tracks its
// project, which is whether the project has a page.
type WorkerSession struct {
	state.Session
	Tracked bool `json:"tracked"`
}

// Worker is one machine that reports to the state repository.
type Worker struct {
	Name       string          `json:"name"`
	Collected  string          `json:"collected"`
	Up         bool            `json:"up"`
	ClaudeCode string          `json:"claude_code"`
	Behind     bool            `json:"behind"`
	Memories   *MemorySync     `json:"memories"`
	Sessions   []WorkerSession `json:"sessions"`
	Checkouts  []Checkout      `json:"checkouts"`
}

// instant reads an RFC 3339 time; ok is false for an empty or broken one.
func instant(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, s)
	return t, err == nil
}

// version reads a dotted version as numbers; ok is false when a part is
// not a number.
func version(v string) ([]int, bool) {
	if v == "" {
		return nil, false
	}
	parts := strings.Split(v, ".")
	out := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil, false
		}
		out[i] = n
	}
	return out, true
}

// staleSync says whether a push is missing or more than a day older than
// the collection that reported it.
func staleSync(pushed, collected string) bool {
	p, ok := instant(pushed)
	if !ok {
		return true
	}
	c, ok := instant(collected)
	return ok && c.Sub(p) > pushWithin
}

// Workers lists every machine found under git/, usage/ or machines/,
// the most recently seen first.
func (s Snapshot) Workers() []Worker {
	names := map[string]bool{}
	for name := range s.Git {
		names[name] = true
	}
	for name := range s.Machines {
		names[name] = true
	}
	for _, name := range s.UsageMachines {
		names[name] = true
	}
	var newest []int
	for _, f := range s.Machines {
		t, seen := instant(f.Collected)
		v, parsed := version(f.ClaudeCode)
		if seen && parsed && s.Built.Sub(t) <= recent && !t.After(s.Built.Add(upWithin)) && slices.Compare(v, newest) > 0 {
			newest = v
		}
	}
	updated := map[string]string{}
	for _, st := range s.Statuses {
		updated[st.Project] = st.Updated
	}
	out := make([]Worker, 0, len(names))
	for name := range names {
		out = append(out, s.worker(name, updated, newest))
	}
	sort.Slice(out, func(i, j int) bool {
		a, aok := instant(out[i].Collected)
		b, bok := instant(out[j].Collected)
		switch {
		case aok != bok:
			return aok
		case aok && !a.Equal(b):
			return a.After(b)
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (s Snapshot) worker(name string, updated map[string]string, newest []int) Worker {
	m, g := s.Machines[name], s.Git[name]
	w := Worker{Name: name, Collected: m.Collected, ClaudeCode: m.ClaudeCode, Sessions: make([]WorkerSession, 0, len(m.Sessions)), Checkouts: []Checkout{}}
	// Without a machine file, the time comes from git/.
	if w.Collected == "" {
		w.Collected = g.Collected
	}
	for _, sess := range m.Sessions {
		_, tracked := updated[sess.Project]
		w.Sessions = append(w.Sessions, WorkerSession{Session: sess, Tracked: tracked})
	}
	if t, ok := instant(w.Collected); ok {
		w.Up = s.Built.Sub(t) <= upWithin && !t.After(s.Built.Add(upWithin))
	}
	if v, ok := version(w.ClaudeCode); ok {
		w.Behind = slices.Compare(v, newest) < 0
	}
	if mm := m.Memories; mm != nil {
		w.Memories = &MemorySync{Pushed: mm.Pushed, SetAside: mm.SetAside, Stale: staleSync(mm.Pushed, m.Collected)}
	}
	for key, tree := range g.Projects {
		w.Checkouts = append(w.Checkouts, Checkout{
			Project: key, Branch: tree.Branch,
			Warnings: warnings([]state.GitProject{tree}, updated[key]),
		})
	}
	sort.Slice(w.Checkouts, func(i, j int) bool { return w.Checkouts[i].Project < w.Checkouts[j].Project })
	return w
}

// MachineSession is a session with the worker it ran on.
type MachineSession struct {
	Machine string `json:"machine"`
	state.Session
}

// Sessions lists the sessions of one project on every worker, the most
// recent first. Times written in different offsets compare as instants.
func (s Snapshot) Sessions(key string) []MachineSession {
	out := []MachineSession{}
	for machine, f := range s.Machines {
		for _, sess := range f.Sessions {
			if sess.Project == key {
				out = append(out, MachineSession{Machine: machine, Session: sess})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, _ := instant(out[i].Last)
		b, _ := instant(out[j].Last)
		switch {
		case !a.Equal(b):
			return a.After(b)
		case out[i].Machine != out[j].Machine:
			return out[i].Machine < out[j].Machine
		}
		return out[i].ID < out[j].ID
	})
	return out
}
