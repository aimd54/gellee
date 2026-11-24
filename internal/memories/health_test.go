// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHealthReadsThePushAndThisMachinesVersionsSetAside(t *testing.T) {
	a, _ := pair(t)
	for _, p := range []string{
		"orphans/m1/conflicts/alpha/fern/one.md",
		"orphans/m1/conflicts/global/CLAUDE.md",
		"orphans/m2/conflicts/alpha/fern/one.md",
	} {
		must(t, os.MkdirAll(filepath.Dir(filepath.Join(a.o.StoreDir, p)), 0o700))
		must(t, os.WriteFile(filepath.Join(a.o.StoreDir, p), []byte("x\n"), 0o600))
	}
	pushed, aside, err := Health(Options{StoreDir: a.o.StoreDir, Machine: "m1"})
	must(t, err)
	if pushed != "2026-11-14T10:00:00Z" || aside != 2 {
		t.Fatalf("pushed %q, %d set aside", pushed, aside)
	}
}

func TestHealthNeedsAMachineThatHasJoined(t *testing.T) {
	a, _ := world(t)
	if _, _, err := Health(a.o); !errors.Is(err, ErrNotAdopted) {
		t.Fatalf("health: %v, want ErrNotAdopted", err)
	}
}

func TestHealthNeedsACheckout(t *testing.T) {
	_, _, err := Health(Options{StoreDir: t.TempDir(), Machine: "m1"})
	if err == nil || !strings.Contains(err.Error(), "is not a checkout of the memory store") {
		t.Fatalf("health: %v", err)
	}
}
