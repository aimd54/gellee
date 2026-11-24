// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadPrices(t *testing.T) {
	if p, err := LoadPrices(t.TempDir()); p != nil || err != nil {
		t.Fatalf("a missing file is not an error: %v %v", p, err)
	}
	dir := t.TempDir()
	body := "# model input output cache_write cache_read\nmodel-a 3 15 3.75 0.3\n\n"
	if err := os.WriteFile(filepath.Join(dir, "prices"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPrices(dir)
	want := map[string]Price{"model-a": {Input: 3, Output: 15, CacheWrite: 3.75, CacheRead: 0.3}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v %v", got, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "prices"), []byte("model-a 3 15\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPrices(dir); err == nil {
		t.Fatal("a short line must be an error")
	}
}
