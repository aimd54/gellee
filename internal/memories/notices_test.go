// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func notices(t *testing.T, m machine, key string, res *Result) []string {
	t.Helper()
	lines, err := Notices(m.o, key, res)
	must(t, err)
	return lines
}

func TestNoticesSayWhatTheSyncImportedForThisProject(t *testing.T) {
	a, b := pair(t)
	a.write("alpha/fern", "two.md", "two\n")
	a.sync()
	res := b.sync()
	want := "1 memory files imported for alpha/fern, restart to use them"
	if got := notices(t, b, "alpha/fern", &res); len(got) != 1 || got[0] != want {
		t.Fatalf("notices %q, want %q", got, want)
	}
	if got := notices(t, b, "alpha/pond", &res); len(got) != 0 {
		t.Fatalf("notices for alpha/pond: %q", got)
	}
}

func TestNoticesCountThisMachinesVersionsSetAside(t *testing.T) {
	_, b := pair(t)
	for _, p := range []string{
		"orphans/m2/conflicts/alpha/fern/one.md",
		"orphans/m2/conflicts/global/CLAUDE.md",
		"orphans/m1/conflicts/alpha/fern/one.md",
	} {
		must(t, os.MkdirAll(filepath.Dir(filepath.Join(b.o.StoreDir, p)), 0o700))
		must(t, os.WriteFile(filepath.Join(b.o.StoreDir, p), []byte("x\n"), 0o600))
	}
	want := "2 versions set aside by a conflict, run /memories"
	if got := notices(t, b, "alpha/fern", nil); len(got) != 1 || got[0] != want {
		t.Fatalf("notices %q, want %q", got, want)
	}
}

func TestNoticesSayWhenTheStoreWasNotPushedForADay(t *testing.T) {
	a, _ := pair(t)
	a.o.Now = a.o.Now.Add(23 * time.Hour)
	if got := notices(t, a, "alpha/fern", nil); len(got) != 0 {
		t.Fatalf("notices after 23 hours: %q", got)
	}
	a.o.Now = a.o.Now.Add(2 * time.Hour)
	want := "no push to the store since 2026-11-14T10:00:00Z"
	if got := notices(t, a, "alpha/fern", nil); len(got) != 1 || got[0] != want {
		t.Fatalf("notices after 25 hours %q, want %q", got, want)
	}
	must(t, a.o.writeState("pushed", ""))
	want = "no push to the store yet"
	if got := notices(t, a, "alpha/fern", nil); len(got) != 1 || got[0] != want {
		t.Fatalf("notices without a push %q, want %q", got, want)
	}
}

func TestNoticesNeedAMachineThatHasJoined(t *testing.T) {
	a, _ := world(t)
	if _, err := Notices(a.o, "alpha/fern", nil); !errors.Is(err, ErrNotAdopted) {
		t.Fatalf("notices: %v, want ErrNotAdopted", err)
	}
}

func TestNoticesDoNotCountTheAdoptTimeGlobalOrphan(t *testing.T) {
	a, _ := pair(t)
	p := "orphans/m1/CLAUDE.md"
	must(t, os.MkdirAll(filepath.Dir(filepath.Join(a.o.StoreDir, p)), 0o700))
	must(t, os.WriteFile(filepath.Join(a.o.StoreDir, p), []byte("mine\n"), 0o600))
	for _, line := range notices(t, a, "alpha/fern", nil) {
		if strings.Contains(line, "set aside") {
			t.Fatalf("notice %q counts the orphan", line)
		}
	}
}
