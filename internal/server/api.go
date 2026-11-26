// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aimd54/gellee/internal/state"
)

// Server serves one state directory.
type Server struct {
	stateDir string
	ui       fs.FS
	now      func() time.Time
	refresh  time.Duration

	// Demo marks the overview and health answers, for a server over the
	// invented state of internal/demo.
	Demo bool

	mu   sync.Mutex
	snap Snapshot
	has  bool
}

// New returns a server that rebuilds its snapshot when a request finds
// it older than refresh.
func New(stateDir string, ui fs.FS, now func() time.Time, refresh time.Duration) *Server {
	return &Server{stateDir: stateDir, ui: ui, now: now, refresh: refresh}
}

func (s *Server) snapshot() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if s.has && now.Sub(s.snap.Built) <= s.refresh {
		return s.snap, nil
	}
	snap, err := Build(s.stateDir, now)
	if err != nil {
		return Snapshot{}, err
	}
	s.snap, s.has = snap, true
	return snap, nil
}

// Handler returns the API and the embedded interface.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	api := func(pattern string, fn func(Snapshot, *http.Request) (any, int)) {
		mux.HandleFunc("GET /api/v1/"+pattern, func(w http.ResponseWriter, r *http.Request) {
			snap, err := s.snapshot()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			body, code := fn(snap, r)
			writeJSON(w, code, body)
		})
	}
	// marked adds "demo": true to a map answer when the server is a demo.
	marked := func(fn func(Snapshot, *http.Request) (any, int)) func(Snapshot, *http.Request) (any, int) {
		return func(snap Snapshot, r *http.Request) (any, int) {
			body, code := fn(snap, r)
			if m, ok := body.(map[string]any); ok && s.Demo {
				m["demo"] = true
			}
			return body, code
		}
	}
	api("overview", marked(overview))
	api("projects", func(snap Snapshot, _ *http.Request) (any, int) {
		return map[string]any{"projects": snap.Projects()}, http.StatusOK
	})
	api("projects/{root}/{name}", detail)
	api("usage", usage)
	api("ideas", ideas)
	api("workers", func(snap Snapshot, _ *http.Request) (any, int) {
		return map[string]any{
			"built": snap.Built.Format(time.RFC3339), "today": snap.Today,
			"workers": snap.Workers(), "problems": snap.Problems,
		}, http.StatusOK
	})
	api("health", marked(func(snap Snapshot, _ *http.Request) (any, int) {
		return map[string]any{
			"built": snap.Built.Format(time.RFC3339), "today": snap.Today,
			"projects": len(snap.Statuses), "rows": len(snap.Rows), "problems": snap.Problems,
		}, http.StatusOK
	}))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "the API is read-only"})
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	})
	// Registered without a method: "GET /" and "/api/" would overlap
	// with neither being the more specific pattern.
	mux.HandleFunc("/", s.static)
	return headers(mux)
}

func headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

// static serves a built file when the path names one, and the
// interface's entry page for every other path, which it routes itself.
func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "read-only", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	if info, err := fs.Stat(s.ui, name); name == "" || err != nil || info.IsDir() {
		name = "index.html"
	}
	data, err := fs.ReadFile(s.ui, name)
	if err != nil {
		http.Error(w, "the interface is not built into this binary", http.StatusNotFound)
		return
	}
	if name == "index.html" {
		w.Header().Set("Cache-Control", "no-store")
	}
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
}

type dayTokens struct {
	Day    string           `json:"day"`
	Tokens map[string]int64 `json:"tokens"`
}

func overview(snap Snapshot, _ *http.Request) (any, int) {
	active := []Project{}
	for _, p := range snap.Projects() {
		if p.State == "active" {
			active = append(active, p)
		}
	}
	from := DayOffset(snap.Today, -29)
	index := map[string]int{}
	days := make([]dayTokens, 30)
	for i := range days {
		days[i] = dayTokens{Day: DayOffset(from, i), Tokens: map[string]int64{}}
		index[days[i].Day] = i
	}
	for _, r := range snap.Rows {
		if i, ok := index[r.Day]; ok {
			days[i].Tokens[r.Project] += r.Input + r.CacheWrite + r.Output
		}
	}
	week := DayOffset(snap.Today, -6)
	return map[string]any{
		"built": snap.Built.Format(time.RFC3339), "today": snap.Today,
		"limit": snap.Config.Limit, "active": len(active),
		"over":     max(0, len(active)-snap.Config.Limit),
		"projects": active, "days": days, "problems": snap.Problems,
		"collected": snap.Collected(),
		"week":      snap.Sum(func(r state.Row) bool { return r.Day >= week && r.Day <= snap.Today }),
	}, http.StatusOK
}

type dayModel struct {
	Day   string `json:"day"`
	Model string `json:"model"`
	Usage Usage  `json:"usage"`
}

func detail(snap Snapshot, r *http.Request) (any, int) {
	key := r.PathValue("root") + "/" + r.PathValue("name")
	for _, st := range snap.Statuses {
		if st.Project != key {
			continue
		}
		p := snap.project(st)
		p.Sections = &Sections{Stopped: st.Stopped, Next: st.Next, Remaining: st.Remaining, Ideas: st.Ideas}
		members := []string{}
		for _, other := range snap.Statuses {
			if st.Group != "" && other.Group == st.Group && other.Project != key {
				members = append(members, other.Project)
			}
		}
		seen := map[[2]string]bool{}
		days := []dayModel{}
		for _, row := range snap.Rows {
			k := [2]string{row.Day, row.Model}
			if row.Project != key || seen[k] {
				continue
			}
			seen[k] = true
			days = append(days, dayModel{Day: row.Day, Model: row.Model, Usage: snap.Sum(func(x state.Row) bool {
				return x.Project == key && x.Day == row.Day && x.Model == row.Model
			})})
		}
		sort.Slice(days, func(i, j int) bool {
			if days[i].Day != days[j].Day {
				return days[i].Day > days[j].Day
			}
			return days[i].Model < days[j].Model
		})
		return map[string]any{
			"project": p, "group_members": members, "days": days,
			"sessions": snap.Sessions(key),
		}, http.StatusOK
	}
	return map[string]string{"error": "no project " + key}, http.StatusNotFound
}

type projectUsage struct {
	Key     string `json:"key"`
	Tracked bool   `json:"tracked"`
	Usage   Usage  `json:"usage"`
}

type modelUsage struct {
	Model string `json:"model"`
	Usage Usage  `json:"usage"`
}

type dayUsage struct {
	Day   string `json:"day"`
	Usage Usage  `json:"usage"`
}

func usage(snap Snapshot, r *http.Request) (any, int) {
	from, to := DayOffset(snap.Today, -29), snap.Today
	for name, target := range map[string]*string{"from": &from, "to": &to} {
		v := r.URL.Query().Get(name)
		if v == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", v); err != nil {
			return map[string]string{"error": name + " must be a date like 2026-11-11"}, http.StatusBadRequest
		}
		*target = v
	}
	in := func(row state.Row) bool { return row.Day >= from && row.Day <= to }
	tracked := map[string]bool{}
	for _, st := range snap.Statuses {
		tracked[st.Project] = true
	}
	keys, models, dayset := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, row := range snap.Rows {
		if in(row) {
			keys[row.Project], models[row.Model], dayset[row.Day] = true, true, true
		}
	}
	projects := []projectUsage{}
	for k := range keys {
		projects = append(projects, projectUsage{Key: k, Tracked: tracked[k], Usage: snap.Sum(func(x state.Row) bool { return in(x) && x.Project == k })})
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Usage.Tokens != projects[j].Usage.Tokens {
			return projects[i].Usage.Tokens > projects[j].Usage.Tokens
		}
		return projects[i].Key < projects[j].Key
	})
	byModel := []modelUsage{}
	for m := range models {
		byModel = append(byModel, modelUsage{Model: m, Usage: snap.Sum(func(x state.Row) bool { return in(x) && x.Model == m })})
	}
	sort.Slice(byModel, func(i, j int) bool {
		if byModel[i].Usage.Tokens != byModel[j].Usage.Tokens {
			return byModel[i].Usage.Tokens > byModel[j].Usage.Tokens
		}
		return byModel[i].Model < byModel[j].Model
	})
	days := []dayUsage{}
	for d := range dayset {
		days = append(days, dayUsage{Day: d, Usage: snap.Sum(func(x state.Row) bool { return x.Day == d })})
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Day < days[j].Day })
	return map[string]any{
		"from": from, "to": to, "total": snap.Sum(in),
		"projects": projects, "models": byModel, "days": days,
	}, http.StatusOK
}

type idea struct {
	Project string `json:"project"`
	Text    string `json:"text"`
}

func ideas(snap Snapshot, _ *http.Request) (any, int) {
	out := []idea{}
	for _, st := range snap.Statuses {
		for _, text := range state.ListItems(st.Ideas) {
			out = append(out, idea{Project: st.Project, Text: text})
		}
	}
	return map[string]any{"ideas": out}, http.StatusOK
}
