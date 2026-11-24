// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// Package transcript reads Claude Code session transcripts and turns
// them into usage rows.
package transcript

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/aimd54/gellee/internal/state"
)

// Other is the project given to usage outside the configured roots.
const Other = "other"

// Attributor maps a working directory to a project.
type Attributor struct {
	ProjectsRoot string
	Roots        []string
}

// Project returns <root>/<name> for a directory at any depth below a
// configured root, and Other for anything else.
func (a Attributor) Project(cwd string) string {
	rel, ok := strings.CutPrefix(cwd, strings.TrimSuffix(a.ProjectsRoot, "/")+"/")
	if !ok {
		return Other
	}
	parts := strings.SplitN(rel, "/", 3)
	if len(parts) < 2 || parts[1] == "" || !slices.Contains(a.Roots, parts[0]) {
		return Other
	}
	return parts[0] + "/" + parts[1]
}

// File is one transcript and the session it belongs to. Main is set for
// the session's own transcript and unset for a subagent's.
type File struct {
	Path    string
	Session string
	Main    bool
}

// Find lists the transcripts below dir, which is laid out as
// <encoded path>/<session>.jsonl with subagent files under
// <encoded path>/<session>/.
func Find(dir string) ([]File, error) {
	var files []File
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 2 {
			return nil
		}
		files = append(files, File{Path: path, Session: strings.TrimSuffix(parts[1], ".jsonl"), Main: len(parts) == 2})
		return nil
	})
	return files, err
}

type record struct {
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	Cwd         string `json:"cwd"`
	GitBranch   string `json:"gitBranch"`
	Version     string `json:"version"`
	AITitle     string `json:"aiTitle"`
	CustomTitle string `json:"customTitle"`
	Bridge      string `json:"bridgeSessionId"`
	Message     struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Usage *struct {
			Input      int64 `json:"input_tokens"`
			Output     int64 `json:"output_tokens"`
			CacheWrite int64 `json:"cache_creation_input_tokens"`
			CacheRead  int64 `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// Summary is what one transcript says about its session. It holds no
// prompt text and no account identifier.
type Summary struct {
	First       time.Time `json:"first"`        // earliest timestamp
	Last        time.Time `json:"last"`         // latest timestamp
	Cwd         string    `json:"cwd"`          // last working directory
	Branch      string    `json:"branch"`       // last gitBranch
	CustomTitle string    `json:"custom_title"` // last title set with /rename
	AITitle     string    `json:"ai_title"`     // last generated title
	Bridge      string    `json:"bridge"`       // last non-empty bridgeSessionId
	Ended       bool      `json:"ended"`        // a cost-state follows the last turn
	Version     string    `json:"version"`      // version of the most recent line
	VersionAt   time.Time `json:"version_at"`   // that line's timestamp
}

// Title is the title set by hand, else the generated one.
func (s Summary) Title() string {
	if s.CustomTitle != "" {
		return s.CustomTitle
	}
	return s.AITitle
}

// Remote returns the address segment of a session connected through
// Remote Control, session_<id> for a bridge ID cse_<id>, or "".
func Remote(bridge string) string {
	id, ok := strings.CutPrefix(bridge, "cse_")
	if !ok || id == "" {
		return ""
	}
	return "session_" + id
}

// Remote returns session_<id> from the bridge ID, or without one from a
// last cwd inside the innermost .claude/worktrees/bridge-cse_<id>.
func (s Summary) Remote() string {
	if s.Bridge != "" {
		return Remote(s.Bridge)
	}
	const marker = "/.claude/worktrees/bridge-cse_"
	i := strings.LastIndex(s.Cwd, marker)
	if i < 0 {
		return ""
	}
	id, _, _ := strings.Cut(s.Cwd[i+len(marker):], "/")
	if id == "" || strings.ContainsFunc(id, func(r rune) bool {
		return (r < '0' || r > '9') && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z')
	}) {
		return ""
	}
	return "session_" + id
}

// add folds one line into the summary. Lines without a readable
// timestamp still carry titles, a bridge ID, a directory and a branch.
func (s *Summary) add(rec record) {
	if ts, err := time.Parse(time.RFC3339Nano, rec.Timestamp); err == nil {
		if s.First.IsZero() || ts.Before(s.First) {
			s.First = ts
		}
		if ts.After(s.Last) {
			s.Last = ts
		}
		if rec.Version != "" && (s.Version == "" || !ts.Before(s.VersionAt)) {
			s.Version, s.VersionAt = rec.Version, ts
		}
	}
	if rec.Cwd != "" {
		s.Cwd = rec.Cwd
	}
	if rec.GitBranch != "" {
		s.Branch = rec.GitBranch
	}
	switch rec.Type {
	case "user", "assistant":
		s.Ended = false
	case "cost-state":
		s.Ended = true
	}
	switch {
	case rec.Type == "custom-title" && rec.CustomTitle != "":
		s.CustomTitle = rec.CustomTitle
	case rec.Type == "ai-title" && rec.AITitle != "":
		s.AITitle = rec.AITitle
	case rec.Type == "bridge-session" && rec.Bridge != "":
		s.Bridge = rec.Bridge
	}
}

// Parse reads one transcript. Each message id counts once, with the
// usage of its last line. skipped counts lines that are not valid JSON
// and messages without a readable timestamp. The summary covers every
// valid line.
func Parse(r io.Reader, session string, a Attributor, loc *time.Location) ([]state.Row, Summary, int, error) {
	br := bufio.NewReaderSize(r, 1<<20)
	last := map[string]record{}
	var sum Summary
	skipped := 0
	for {
		line, err := br.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			var rec record
			if jerr := json.Unmarshal(line, &rec); jerr != nil {
				skipped++
			} else {
				sum.add(rec)
				if rec.Type == "assistant" && rec.Message.ID != "" &&
					rec.Message.Usage != nil && rec.Message.Model != "<synthetic>" {
					last[rec.Message.ID] = rec
				}
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, sum, skipped, err
		}
	}
	rows := make([]state.Row, 0, len(last))
	for _, rec := range last {
		ts, err := time.Parse(time.RFC3339Nano, rec.Timestamp)
		if err != nil {
			skipped++
			continue
		}
		rows = append(rows, state.Row{
			Day:        ts.In(loc).Format("2006-01-02"),
			Session:    session,
			Project:    a.Project(rec.Cwd),
			Model:      rec.Message.Model,
			Input:      rec.Message.Usage.Input,
			Output:     rec.Message.Usage.Output,
			CacheWrite: rec.Message.Usage.CacheWrite,
			CacheRead:  rec.Message.Usage.CacheRead,
			Messages:   1,
		})
	}
	return state.Merge(rows), sum, skipped, nil
}
