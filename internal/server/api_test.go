// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

var ui = fstest.MapFS{
	"index.html":    {Data: []byte("<!doctype html><title>gellee</title>")},
	"assets/app.js": {Data: []byte("console.log('gellee')")},
}

func handler(t *testing.T) http.Handler {
	t.Helper()
	return New(fixtureState, ui, func() time.Time { return noon }, time.Minute).Handler()
}

// get decodes a JSON response into a generic value and returns it with
// the recorder.
func get(t *testing.T, h http.Handler, path string) (map[string]any, *httptest.ResponseRecorder) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var body map[string]any
	if strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: %v\n%s", path, err, rec.Body.String())
		}
	}
	return body, rec
}

func list(v any) []any         { l, _ := v.([]any); return l }
func obj(v any) map[string]any { m, _ := v.(map[string]any); return m }

func TestOverview(t *testing.T) {
	body, rec := get(t, handler(t), "/api/v1/overview")
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("code %d headers %v", rec.Code, rec.Header())
	}
	if body["active"] != 3.0 || body["limit"] != 2.0 || body["over"] != 1.0 || body["today"] != "2026-11-11" {
		t.Fatalf("counts %v %v %v %v", body["active"], body["limit"], body["over"], body["today"])
	}
	var keys []string
	for _, p := range list(body["projects"]) {
		keys = append(keys, obj(p)["key"].(string))
	}
	if strings.Join(keys, " ") != "alpha/fern alpha/renamed beta/tool" {
		t.Fatalf("active projects %q", keys)
	}
	fern := obj(list(body["projects"])[0])
	if obj(fern["usage_7"])["tokens"] != 3330.0 {
		t.Fatalf("fern usage %v", fern["usage_7"])
	}
	days := list(body["days"])
	if len(days) != 30 || obj(days[0])["day"] != "2026-10-13" || obj(days[29])["day"] != "2026-11-11" {
		t.Fatalf("days %v", days)
	}
	last := obj(obj(days[29])["tokens"])
	if last["alpha/renamed"] != 222.0 || last["alpha/fern"] != 2220.0 || last["other"] != 777.0 {
		t.Fatalf("tokens on the last day %v", last)
	}
	if len(obj(obj(days[28])["tokens"])) != 2 || len(obj(obj(days[27])["tokens"])) != 0 {
		t.Fatalf("days 28 and 27: %v %v", days[28], days[27])
	}
	problems := list(body["problems"])
	if len(problems) != 1 || obj(problems[0])["file"] != "alpha/broken.md" {
		t.Fatalf("problems %v", problems)
	}
	if body["collected"] != "2026-11-11T10:00:00+01:00" {
		t.Fatalf("collected %v", body["collected"])
	}
	// The rows of 10 and 11 November: 1110 + 555 + 222 + 777 + 2220.
	week := obj(body["week"])
	if week["tokens"] != 4884.0 || week["cache_read"] != 44000.0 || week["messages"] != 6.0 {
		t.Fatalf("week %v", week)
	}
}

func TestProjectsAndDetail(t *testing.T) {
	h := handler(t)
	body, _ := get(t, h, "/api/v1/projects")
	if len(list(body["projects"])) != 5 {
		t.Fatalf("projects %v", body["projects"])
	}
	body, rec := get(t, h, "/api/v1/projects/alpha/fern")
	if rec.Code != 200 {
		t.Fatalf("code %d", rec.Code)
	}
	project := obj(body["project"])
	if obj(project["sections"])["remaining"] != "- Milestone 2, tasks 18 to 24" {
		t.Fatalf("sections %v", project["sections"])
	}
	if obj(obj(project["git"])["m1"])["branch"] != "m2-task-17" {
		t.Fatalf("git %v", project["git"])
	}
	members := list(body["group_members"])
	if len(members) != 1 || members[0] != "alpha/fern-docs" {
		t.Fatalf("group members %v", members)
	}
	days := list(body["days"])
	if len(days) != 3 {
		t.Fatalf("days %v", days)
	}
	newest := obj(days[0])
	if newest["day"] != "2026-11-11" || newest["model"] != "model-a" || obj(newest["usage"])["tokens"] != 2220.0 {
		t.Fatalf("newest day %v", newest)
	}
	if _, rec := get(t, h, "/api/v1/projects/alpha/nope"); rec.Code != 404 {
		t.Fatalf("code %d for an unknown project", rec.Code)
	}
}

func TestUsage(t *testing.T) {
	h := handler(t)
	body, _ := get(t, h, "/api/v1/usage")
	if body["from"] != "2026-10-13" || body["to"] != "2026-11-11" {
		t.Fatalf("period %v %v", body["from"], body["to"])
	}
	// 3330 + 222 + 555 + 777 + 3
	if obj(body["total"])["tokens"] != 4887.0 {
		t.Fatalf("total %v", body["total"])
	}
	projects := list(body["projects"])
	first := obj(projects[0])
	if len(projects) != 5 || first["key"] != "alpha/fern" || first["tracked"] != true || obj(first["usage"])["tokens"] != 3330.0 {
		t.Fatalf("projects %v", projects)
	}
	untracked := map[string]float64{}
	for _, p := range projects {
		if obj(p)["tracked"] == false {
			untracked[obj(p)["key"].(string)] = obj(obj(p)["usage"])["tokens"].(float64)
		}
	}
	if len(untracked) != 2 || untracked["other"] != 777 || untracked["alpha/gone"] != 3 {
		t.Fatalf("untracked %v", untracked)
	}
	models := list(body["models"])
	if len(models) != 2 || obj(models[0])["model"] != "model-a" || obj(obj(models[0])["usage"])["tokens"] != 4332.0 {
		t.Fatalf("models %v", models)
	}
	if days := list(body["days"]); len(days) != 3 || obj(days[0])["day"] != "2026-11-04" {
		t.Fatalf("days %v", days)
	}

	body, _ = get(t, h, "/api/v1/usage?from=2026-10-01&to=2026-10-31")
	if obj(body["total"])["tokens"] != 11100.0 {
		t.Fatalf("october %v", body["total"])
	}
	if _, rec := get(t, h, "/api/v1/usage?from=yesterday"); rec.Code != 400 {
		t.Fatalf("code %d for a bad date", rec.Code)
	}
}

func TestIdeasAndHealth(t *testing.T) {
	h := handler(t)
	body, _ := get(t, h, "/api/v1/ideas")
	ideas := list(body["ideas"])
	if len(ideas) != 3 {
		t.Fatalf("ideas %v", ideas)
	}
	if obj(ideas[1])["project"] != "alpha/fern" || obj(ideas[1])["text"] != "Write up the soil sampling measurements" {
		t.Fatalf("second idea %v", ideas[1])
	}
	body, _ = get(t, h, "/api/v1/health")
	if body["projects"] != 5.0 || body["rows"] != 7.0 || len(list(body["problems"])) != 1 {
		t.Fatalf("health %v", body)
	}
}

func TestStaticFilesAndHeaders(t *testing.T) {
	h := handler(t)
	for _, path := range []string{"/", "/projects", "/projects/alpha/fern"} {
		_, rec := get(t, h, path)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), "<title>gellee</title>") {
			t.Errorf("%s: code %d body %q", path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src 'self'") {
			t.Errorf("%s: no content security policy", path)
		}
	}
	if _, rec := get(t, h, "/assets/app.js"); !strings.Contains(rec.Body.String(), "console.log") {
		t.Errorf("asset body %q", rec.Body.String())
	}
	if _, rec := get(t, h, "/api/v1/nope"); rec.Code != 404 {
		t.Errorf("unknown API path: code %d", rec.Code)
	}
	for _, path := range []string{"/api/v1/overview", "/api/v1/nope", "/"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s: code %d", path, rec.Code)
		}
	}
}

func TestSnapshotIsRebuiltAfterTheRefreshInterval(t *testing.T) {
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(fixtureState)); err != nil {
		t.Fatal(err)
	}
	clock := noon
	h := New(dir, ui, func() time.Time { return clock }, time.Minute).Handler()
	body, _ := get(t, h, "/api/v1/overview")
	if body["active"] != 3.0 {
		t.Fatalf("active %v", body["active"])
	}
	path := filepath.Join(dir, "beta/tool.md")
	data, _ := os.ReadFile(path)
	parked := strings.Replace(string(data), "state: active", "state: parked\nparked_reason: Later", 1)
	if err := os.WriteFile(path, []byte(parked), 0o600); err != nil {
		t.Fatal(err)
	}
	clock = noon.Add(59 * time.Second)
	if body, _ := get(t, h, "/api/v1/overview"); body["active"] != 3.0 {
		t.Fatalf("rebuilt too early: active %v", body["active"])
	}
	clock = noon.Add(61 * time.Second)
	if body, _ := get(t, h, "/api/v1/overview"); body["active"] != 2.0 || body["over"] != 0.0 {
		t.Fatalf("not rebuilt: active %v over %v", body["active"], body["over"])
	}
}

// The week is the seven days that end today: the 7th day back (5
// November) counts, the 8th (4 November) and a day after today do not.
func TestOverviewWeekBoundaries(t *testing.T) {
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(fixtureState)); err != nil {
		t.Fatal(err)
	}
	rows := `{"day":"2026-11-05","session":"b1","project":"alpha/fern","model":"model-a","input":0,"output":1000,"cache_write":0,"cache_read":7,"messages":1}
{"day":"2026-11-04","session":"b2","project":"alpha/fern","model":"model-a","input":0,"output":20000,"cache_write":0,"cache_read":70000,"messages":1}
{"day":"2026-11-12","session":"b3","project":"alpha/fern","model":"model-a","input":0,"output":300000,"cache_write":0,"cache_read":700000,"messages":1}
`
	if err := os.MkdirAll(filepath.Join(dir, "usage/m3"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "usage/m3/2026-11.jsonl"), []byte(rows), 0o600); err != nil {
		t.Fatal(err)
	}
	h := New(dir, ui, func() time.Time { return noon }, time.Minute).Handler()
	body, _ := get(t, h, "/api/v1/overview")
	// The fixture's 4884 tokens, 44000 cache reads and 6 messages, plus
	// the 5 November row alone.
	week := obj(body["week"])
	if week["tokens"] != 5884.0 || week["cache_read"] != 44007.0 || week["messages"] != 7.0 {
		t.Fatalf("week %v", week)
	}
}

func TestWorkersEndpointServesTheFixture(t *testing.T) {
	body, rec := get(t, handler(t), "/api/v1/workers")
	if rec.Code != 200 || body["today"] != "2026-11-11" {
		t.Fatalf("code %d body %v", rec.Code, body)
	}
	ws := list(body["workers"])
	if len(ws) != 2 || obj(ws[0])["name"] != "m1" || obj(ws[1])["name"] != "m2" {
		t.Fatalf("workers %v", ws)
	}
	m1, m2 := obj(ws[0]), obj(ws[1])
	if obj(m1["memories"])["set_aside"] != 1.0 || obj(m1["memories"])["stale"] != false || m1["behind"] != false {
		t.Fatalf("m1 %v", m1)
	}
	if m2["behind"] != true || obj(m2["memories"])["stale"] != true || len(list(m2["checkouts"])) != 0 {
		t.Fatalf("m2 %v", m2)
	}
	if got := len(list(m1["checkouts"])); got != 3 {
		t.Fatalf("m1 has %d checkouts", got)
	}
	if obj(list(m1["sessions"])[0])["remote"] != "session_01FixtureToolSession" {
		t.Fatalf("m1 sessions %v", m1["sessions"])
	}
	// The fixture's session outside the roots has ended.
	if s := list(m1["sessions"]); obj(s[0])["ended"] != false || obj(s[2])["ended"] != true {
		t.Fatalf("m1 sessions ended %v", s)
	}
}

func TestProjectDetailAndOverviewCarryTheSessions(t *testing.T) {
	h := New(sessionsWorld(t), ui, func() time.Time { return noon }, time.Minute).Handler()
	body, rec := get(t, h, "/api/v1/projects/alpha/fern")
	if rec.Code != 200 {
		t.Fatalf("code %d", rec.Code)
	}
	sessions := list(body["sessions"])
	if len(sessions) != 2 || obj(sessions[0])["machine"] != "m2" || obj(sessions[0])["remote"] != "session_01C" ||
		obj(sessions[0])["title"] != "Second" || obj(sessions[1])["machine"] != "m1" {
		t.Fatalf("sessions %v", sessions)
	}
	if obj(sessions[0])["ended"] != true || obj(sessions[1])["ended"] != false {
		t.Fatalf("ended %v and %v, want true and false", obj(sessions[0])["ended"], obj(sessions[1])["ended"])
	}
	body, _ = get(t, h, "/api/v1/overview")
	last := map[string]any{}
	for _, p := range list(body["projects"]) {
		last[obj(p)["key"].(string)] = obj(p)["last_session"]
	}
	if obj(last["alpha/fern"])["id"] != "c" || obj(last["alpha/fern"])["machine"] != "m2" {
		t.Fatalf("alpha/fern last session %v", last["alpha/fern"])
	}
	if obj(last["alpha/renamed"])["id"] != "b" || last["beta/tool"] != nil {
		t.Fatalf("last sessions %v", last)
	}
	if _, present := last["beta/tool"]; !present {
		t.Fatal("last_session must be present, as null")
	}
}
