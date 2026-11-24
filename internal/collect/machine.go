// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package collect

import (
	"sort"
	"time"

	"github.com/aimd54/gellee/internal/state"
	"github.com/aimd54/gellee/internal/transcript"
)

// window is how far back before the collection the machine file lists
// sessions.
const window = 7 * 24 * time.Hour

// span is what every transcript of one session says together.
type span struct {
	main        transcript.Summary // the session's own transcript
	first, last time.Time          // over its own and its subagents' transcripts
}

// instant reads an RFC 3339 time, or returns the zero time.
func instant(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// machineFile builds machines/<machine>.json from this run's summaries.
// A session that could not be read, or whose transcripts are gone, keeps
// the entry prev had for it while it stays in the window. Memories is
// left for the caller.
func (o Options) machineFile(files []transcript.File, entries map[string]cacheEntry, failed map[string]bool,
	attr transcript.Attributor, aliases map[string]string, prev state.MachineFile) state.MachineFile {
	out := state.MachineFile{Collected: o.Now.Format(time.RFC3339), Sessions: []state.Session{}}
	spans := map[string]*span{}
	var versionAt time.Time
	for _, f := range files {
		e, ok := entries[f.Path]
		if !ok {
			continue
		}
		s := e.Summary
		if s.Version != "" && (out.ClaudeCode == "" || s.VersionAt.After(versionAt)) {
			out.ClaudeCode, versionAt = s.Version, s.VersionAt
		}
		sp := spans[f.Session]
		if sp == nil {
			sp = &span{}
			spans[f.Session] = sp
		}
		if f.Main {
			sp.main = s
		}
		if !s.First.IsZero() && (sp.first.IsZero() || s.First.Before(sp.first)) {
			sp.first = s.First
		}
		if s.Last.After(sp.last) {
			sp.last = s.Last
		}
	}
	from := o.Now.Add(-window)
	for id, sp := range spans {
		if failed[id] || sp.last.Before(from) {
			continue
		}
		project := attr.Project(sp.main.Cwd)
		if current, ok := aliases[project]; ok {
			project = current
		}
		entry := state.Session{
			ID: id, Project: project, Remote: sp.main.Remote(), Ended: sp.main.Ended,
			Started: sp.first.In(o.Loc).Format(time.RFC3339),
			Last:    sp.last.In(o.Loc).Format(time.RFC3339),
		}
		// Outside the roots, the session keeps no title or branch.
		if project != transcript.Other {
			entry.Branch, entry.Title = sp.main.Branch, sp.main.Title()
		}
		out.Sessions = append(out.Sessions, entry)
	}
	for _, s := range prev.Sessions {
		if spans[s.ID] != nil && !failed[s.ID] {
			continue
		}
		if last := instant(s.Last); !last.IsZero() && !last.Before(from) {
			if s.Project == transcript.Other {
				s.Branch, s.Title = "", ""
			}
			out.Sessions = append(out.Sessions, s)
		}
	}
	sort.Slice(out.Sessions, func(i, j int) bool {
		a, b := instant(out.Sessions[i].Last), instant(out.Sessions[j].Last)
		if !a.Equal(b) {
			return a.After(b)
		}
		return out.Sessions[i].ID < out.Sessions[j].ID
	})
	return out
}
