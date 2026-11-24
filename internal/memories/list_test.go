// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var isoDay = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func TestListShowsEveryMemoryWithItsFrontmatter(t *testing.T) {
	a, _ := pair(t)
	rule := "---\nname: same-rule\ndescription: one rule\nmetadata:\n  type: feedback\n---\nbody\n"
	a.write("alpha/fern", "rule.md", rule)
	a.write("alpha/pond", "rule.md", rule)
	a.write("alpha/pond", "who.md", "---\nname: who\ntype: user\n---\nme\n")
	a.sync()
	l, err := List(a.o)
	must(t, err)
	got := map[string]Memory{}
	for _, m := range l.Memories {
		got[m.Project+"/"+m.File] = m
	}
	if len(got) != 4 {
		t.Fatalf("memories %+v, want one.md and rule.md of fern, rule.md and who.md of pond", l.Memories)
	}
	one := got["alpha/fern/one.md"]
	if one.Name != "one" || one.Type != "" || one.Size != 4 || !isoDay.MatchString(one.Changed) {
		t.Fatalf("one.md: %+v", one)
	}
	r := got["alpha/pond/rule.md"]
	if r.Name != "same-rule" || r.Type != "feedback" || r.Description != "one rule" || r.Size != len(rule) {
		t.Fatalf("rule.md: %+v", r)
	}
	if w := got["alpha/pond/who.md"]; w.Type != "user" {
		t.Fatalf("who.md: %+v", w)
	}
	if d := l.Duplicates["same-rule"]; len(d) != 2 || d[0] != "alpha/fern" || d[1] != "alpha/pond" {
		t.Fatalf("duplicates %v", l.Duplicates)
	}
	if len(l.Duplicates) != 1 {
		t.Fatalf("duplicates %v, want same-rule only", l.Duplicates)
	}
}

func TestListSeparatesConflictsAndOrphans(t *testing.T) {
	a, _ := pair(t)
	for p, c := range map[string]string{
		"orphans/m1/conflicts/alpha/fern/one.md": "mine\n",
		"orphans/m1/-home-u-old/a.md":            "---\ntype: user\n---\n",
		"global/CLAUDE.md":                       "rules\n",
	} {
		must(t, os.MkdirAll(filepath.Dir(filepath.Join(a.o.StoreDir, p)), 0o700))
		must(t, os.WriteFile(filepath.Join(a.o.StoreDir, p), []byte(c), 0o600))
	}
	l, err := List(a.o)
	must(t, err)
	if len(l.Conflicts) != 1 || l.Conflicts[0] != "orphans/m1/conflicts/alpha/fern/one.md" {
		t.Fatalf("conflicts %v", l.Conflicts)
	}
	if len(l.Orphans) != 1 || l.Orphans[0] != "orphans/m1/-home-u-old/a.md" {
		t.Fatalf("orphans %v", l.Orphans)
	}
	if len(l.Memories) != 1 || l.Memories[0].File != "one.md" {
		t.Fatalf("memories %+v, want alpha/fern/one.md alone", l.Memories)
	}
	var out bytes.Buffer
	must(t, l.Print(&out))
	for _, want := range []string{"PROJECT", "alpha/fern", "conflicts set aside: 1", "orphans: 1", "names in more than one project: 0"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("printed listing lacks %q:\n%s", want, out.String())
		}
	}
}

func TestListOfAnEmptyStoreGivesEmptyGroups(t *testing.T) {
	a, _ := world(t)
	l, err := List(a.o)
	must(t, err)
	var out bytes.Buffer
	must(t, l.JSON(&out))
	for _, want := range []string{`"memories": []`, `"conflicts": []`, `"orphans": []`, `"duplicates": {}`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("JSON lacks %s:\n%s", want, out.String())
		}
	}
}

func TestListReportsTheAdoptTimeGlobalOrphanAsAnOrphan(t *testing.T) {
	a, _ := pair(t)
	p := "orphans/m1/CLAUDE.md"
	must(t, os.MkdirAll(filepath.Dir(filepath.Join(a.o.StoreDir, p)), 0o700))
	must(t, os.WriteFile(filepath.Join(a.o.StoreDir, p), []byte("mine\n"), 0o600))
	l, err := List(a.o)
	must(t, err)
	if len(l.Orphans) != 1 || l.Orphans[0] != p {
		t.Fatalf("orphans %v, want %s", l.Orphans, p)
	}
	if len(l.Conflicts) != 0 {
		t.Fatalf("conflicts %v, want none", l.Conflicts)
	}
}

func TestListShowsAFileThatIsNotUTF8WithoutAType(t *testing.T) {
	a, _ := pair(t)
	raw := "---\nname: odd\ntype: user\ndescription: caf\xe9\n---\n\xff\xfe\n"
	a.write("alpha/fern", "odd.md", raw)
	a.sync()
	l, err := List(a.o)
	must(t, err)
	var odd *Memory
	for i, m := range l.Memories {
		if m.File == "odd.md" {
			odd = &l.Memories[i]
		}
	}
	if odd == nil {
		t.Fatalf("odd.md missing from %+v", l.Memories)
	}
	if odd.Type != "" || odd.Size != len(raw) {
		t.Fatalf("odd.md: %+v, want an empty type and size %d", *odd, len(raw))
	}
	var out bytes.Buffer
	must(t, l.JSON(&out))
	var back Listing
	if err := json.Unmarshal(out.Bytes(), &back); err != nil {
		t.Fatalf("JSON is not valid: %v\n%s", err, out.String())
	}
	if len(back.Memories) != len(l.Memories) {
		t.Fatalf("round trip kept %d memories, want %d", len(back.Memories), len(l.Memories))
	}
}
