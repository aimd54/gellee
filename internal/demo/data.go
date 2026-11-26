// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package demo

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/aimd54/gellee/internal/state"
)

const (
	minute = time.Minute
	hour   = time.Hour
	day    = 24 * time.Hour
)

// machine is one invented worker as its collector would report it.
// Every duration is a time before the anchor.
type machine struct {
	name      string
	collected time.Duration
	version   string
	pushed    time.Duration
	setAside  int
	sessions  []session
	trees     map[string]tree
}

type session struct {
	project, branch, title string
	started, last          time.Duration
	ended                  bool
}

type tree struct {
	branch, head string
	commit       time.Duration
	ahead, dirty int
}

var machines = []machine{
	{
		name: "laptop", collected: 10 * minute, version: "2.4.12", pushed: 20 * minute,
		sessions: []session{
			{"studio/loom", "feat/heddles", "Thread the heddle frame", 2 * hour, 5 * minute, false},
			{"studio/loom", "feat/heddles", "Review the heddle tests", 26 * hour, 24 * hour, true},
			{"studio/loom-site", "main", "Draft the pattern gallery", 50 * hour, 48 * hour, true},
			{"garden/seedbank", "main", "Sort germination records", 97 * hour, 96 * hour, true},
			{"studio/loom", "main", "Plan milestone 3", 145 * hour, 143 * hour, true},
		},
		trees: map[string]tree{
			"studio/loom":     {"feat/heddles", "3f9c2a1", 40 * minute, 0, 0},
			"garden/seedbank": {"main", "8b14d07", 96 * hour, 2, 0},
			"studio/weft":     {"main", "5e2d9b0", 41 * day, 0, 0},
		},
	},
	{
		name: "desktop", collected: 3 * hour, version: "2.4.9", pushed: 3*hour + 10*minute, setAside: 2,
		sessions: []session{
			{"studio/loom-site", "main", "Fix the gallery grid", 5 * hour, 3*hour + 15*minute, true},
			{"garden/seedbank", "main", "Import the spring catalogue", 30 * hour, 28 * hour, true},
			{"studio/loom", "feat/heddles", "Sketch the warp solver", 75 * hour, 73 * hour, true},
		},
		trees: map[string]tree{
			"studio/loom-site": {"main", "a71c3e4", 4 * hour, 0, 3},
			"studio/loom":      {"feat/heddles", "1d7e8f2", 27 * hour, 0, 0},
			"garden/almanac":   {"main", "6c4f1a8", 31 * day, 0, 0},
		},
	},
	{
		name: "worker-1", collected: 15 * minute, version: "2.4.12", pushed: 50 * hour,
		sessions: []session{
			{"studio/pathfinder", "route-closures", "Route around closed paths", 90 * minute, 2 * minute, false},
			{"studio/pathfinder", "route-closures", "Benchmark the planner", 48 * hour, 46 * hour, true},
			{"garden/seedbank", "main", "Check the catalogue import", 122 * hour, 121 * hour, true},
		},
		trees: map[string]tree{
			"studio/pathfinder": {"route-closures", "9a0b1c2", 20 * minute, 0, 0},
			"garden/seedbank":   {"main", "8b14d07", 96 * hour, 0, 0},
		},
	},
}

// profile spreads one project's usage on one machine over the days
// from..to before the anchor, from >= to.
type profile struct {
	project, machine string
	from, to         int
	chance           float64 // that a day in the range has work
	tokens           int64   // typical tokens of one session and model
	models           []string
}

var profiles = []profile{
	{"studio/loom", "laptop", 20, 0, 0.8, 60000, []string{"model-large", "model-medium"}},
	{"studio/loom", "desktop", 20, 0, 0.3, 40000, []string{"model-large"}},
	{"studio/shuttle", "laptop", 44, 21, 0.7, 50000, []string{"model-large"}},
	{"studio/loom-site", "desktop", 20, 0, 0.5, 15000, []string{"model-medium"}},
	{"studio/pathfinder", "worker-1", 30, 0, 0.6, 35000, []string{"model-large"}},
	{"garden/seedbank", "laptop", 25, 0, 0.4, 20000, []string{"model-medium"}},
	{"garden/trellis", "laptop", 44, 12, 0.3, 10000, []string{"model-small"}},
	{"garden/almanac", "desktop", 44, 30, 0.5, 8000, []string{"model-medium"}},
	{"studio/weft", "laptop", 44, 40, 0.8, 12000, []string{"model-large"}},
}

// ago is the anchor moved back by d, as RFC 3339 in the anchor's zone.
func ago(anchor time.Time, d time.Duration) string {
	return anchor.Add(-d).Format(time.RFC3339)
}

// usage generates the rows of every profile, keyed by machine. The
// random source has a fixed seed and is drawn the same way whatever the
// anchor, so only the days move.
func usage(anchor time.Time) map[string][]state.Row {
	r := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // invented data, not a secret
	out := map[string][]state.Row{}
	for _, p := range profiles {
		_, name, _ := strings.Cut(p.project, "/")
		for k := p.from; k >= p.to; k-- {
			if r.Float64() >= p.chance {
				continue
			}
			day := anchor.AddDate(0, 0, -k).Format("2006-01-02")
			for s := range 1 + r.IntN(3) {
				id := fmt.Sprintf("%s-%s-%d-%d", p.machine, name, k, s)
				for _, m := range p.models {
					t := p.tokens/2 + r.Int64N(p.tokens)
					out[p.machine] = append(out[p.machine], state.Row{
						Day: day, Session: id, Project: p.project, Model: m,
						Input: t / 20, Output: t * 3 / 20, CacheWrite: t * 4 / 5, CacheRead: t * 12,
						Messages: 4 + r.Int64N(30),
					})
				}
			}
		}
	}
	return out
}

// writeData writes the usage, git and machine files of every machine.
func writeData(dir string, anchor time.Time) error {
	for name, rows := range usage(anchor) {
		if _, err := state.WriteMachineRows(dir, name, rows); err != nil {
			return err
		}
	}
	n := 0
	for _, m := range machines {
		sessions := make([]state.Session, 0, len(m.sessions))
		for _, s := range m.sessions {
			n++
			sessions = append(sessions, state.Session{
				ID: fmt.Sprintf("00000000-0000-4000-8000-%012d", n), Project: s.project,
				Branch: s.branch, Title: s.title,
				Started: ago(anchor, s.started), Last: ago(anchor, s.last), Ended: s.ended,
			})
		}
		file := state.MachineFile{
			Collected: ago(anchor, m.collected), ClaudeCode: m.version,
			Memories: &state.MachineMemories{Pushed: ago(anchor, m.pushed), SetAside: m.setAside},
			Sessions: sessions,
		}
		if err := state.WriteMachineFile(dir, m.name, file); err != nil {
			return err
		}
		git := state.GitFile{Collected: ago(anchor, m.collected), Projects: map[string]state.GitProject{}}
		for key, t := range m.trees {
			git.Projects[key] = state.GitProject{
				Branch: t.branch, Head: t.head, LastCommit: ago(anchor, t.commit),
				Remote: true, Ahead: t.ahead, Dirty: t.dirty,
			}
		}
		if err := state.WriteGitFile(dir, m.name, git); err != nil {
			return err
		}
	}
	return nil
}
