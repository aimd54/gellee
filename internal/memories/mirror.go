// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// files reads the regular files of dir that belong to the unit, by
// name. A missing directory holds none. Names that start with a dot and
// subdirectories are not memories and are left out. In the local
// directory of a unit that follows links, a symbolic link to a regular
// file reads as that file, and a link to nothing as no file.
func (u unit) files(dir string) (map[string][]byte, error) {
	out := map[string][]byte{}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if len(u.Only) > 0 && !slices.Contains(u.Only, e.Name()) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if e.Type()&fs.ModeSymlink != 0 && u.Links && dir == u.Local {
			fi, err := os.Stat(path)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if !fi.Mode().IsRegular() {
				continue
			}
		} else if !e.Type().IsRegular() {
			continue
		}
		data, err := os.ReadFile(path) //nolint:gosec // a name read from the directory itself
		if err != nil {
			return nil, err
		}
		out[e.Name()] = data
	}
	return out, nil
}

// through returns the file that path names once every symbolic link on
// the way is followed, so that a write replaces the file a link points
// to and leaves the link in place. A path that does not exist is the
// file to create.
func through(path string) (string, error) {
	p := path
	for range 40 {
		fi, err := os.Lstat(p)
		if errors.Is(err, fs.ErrNotExist) {
			return p, nil
		}
		if err != nil {
			return "", err
		}
		if fi.Mode()&fs.ModeSymlink == 0 {
			return p, nil
		}
		target, err := os.Readlink(p)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(p), target)
		}
		p = target
	}
	return "", fmt.Errorf("%s: too many symbolic links", path)
}

// mirror makes the unit's files in dst equal to want and returns how
// many it wrote or removed.
func (u unit) mirror(want map[string][]byte, dst string) (int, error) {
	have, err := u.files(dst)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, name := range names(want) {
		if old, ok := have[name]; ok && bytes.Equal(old, want[name]) {
			continue
		}
		if err := writeFile(filepath.Join(dst, name), want[name]); err != nil {
			return n, err
		}
		n++
	}
	for _, name := range names(have) {
		if _, ok := want[name]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(dst, name)); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// copyBack makes the unit's local files equal to want, the files of its
// store directory, and returns how many it wrote or removed. It changes
// a local file only while the file still holds what view says the run
// read from it, a missing file counting as a state of its own: a file a
// session wrote, changed or removed since is left as it is, and the next
// sync carries it. A unit that follows links is written through them; a
// removal removes the link and leaves its target alone.
func (u unit) copyBack(want, view map[string][]byte) (int, error) {
	have, err := u.files(u.Local)
	if err != nil {
		return 0, err
	}
	untouched := func(name string) bool {
		now, isNow := have[name]
		then, wasThen := view[name]
		return isNow == wasThen && bytes.Equal(now, then)
	}
	n := 0
	for _, name := range names(want) {
		if old, ok := have[name]; ok && bytes.Equal(old, want[name]) {
			continue
		}
		if !untouched(name) {
			continue
		}
		path := filepath.Join(u.Local, name)
		if u.Links {
			if path, err = through(path); err != nil {
				return n, err
			}
		}
		if err := writeFile(path, want[name]); err != nil {
			return n, err
		}
		n++
	}
	for _, name := range names(have) {
		if _, ok := want[name]; ok || !untouched(name) {
			continue
		}
		if err := os.Remove(filepath.Join(u.Local, name)); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// reconcile settles the files a session changed since the run read them
// (view) against want, the store's files, and returns what copy back
// works from: target, the files the local directory should hold, and
// expect, what a local file must still hold for copy back to change it.
// A file the store left as the run read it stays the session's, carried
// by the next sync, and so does one the session changed and the store
// deleted. A memory the session deleted and the store changed is
// written back (overridden). A memory both changed keeps the store's
// version, and the session's is returned in aside, to be set aside
// before copy back overwrites it. An index both changed gets the lines
// of both.
func (u unit) reconcile(want, view map[string][]byte) (target, expect, aside map[string][]byte, overridden int, err error) {
	have, err := u.files(u.Local)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	target, expect, aside = maps.Clone(want), maps.Clone(view), map[string][]byte{}
	same := func(a, b map[string][]byte, name string) bool {
		x, inA := a[name]
		y, inB := b[name]
		return inA == inB && bytes.Equal(x, y)
	}
	all := maps.Clone(have)
	maps.Copy(all, view)
	var restored []string
	for _, name := range names(all) {
		if same(have, view, name) || same(want, view, name) || same(have, want, name) {
			continue
		}
		now, isNow := have[name]
		theirs, isTheirs := want[name]
		switch {
		case !isNow:
			delete(expect, name)
			overridden++
			restored = append(restored, name)
		case !isTheirs:
		case name == "MEMORY.md":
			target[name] = unionLines(theirs, now)
			expect[name] = now
		default:
			aside[name] = now
			expect[name] = now
		}
	}
	// A session removes a memory along with its index line, and Claude
	// Code loads no memory its index does not list: a memory written
	// back gets its line back too.
	var lines []byte
	for _, name := range restored {
		if l, ok := linkTo(string(want["MEMORY.md"]), name); ok {
			lines = append(lines, l...)
		}
	}
	if len(lines) > 0 {
		const index = "MEMORY.md"
		base := target[index]
		if !same(have, expect, index) {
			// The session changed the index and copy back leaves it:
			// add the lines to the session's version instead. An index
			// the session removed is the store's again, since a line
			// alone would drop every other memory from it.
			if now, ok := have[index]; ok {
				base, expect[index] = now, now
			} else {
				delete(expect, index)
			}
		}
		target[index] = unionLines(base, lines)
	}
	return target, expect, aside, overridden, nil
}

// unionLines is base followed by the lines of extra that base lacks,
// each once; blank lines of extra are left out.
func unionLines(base, extra []byte) []byte {
	out := slices.Clone(base)
	seen := map[string]bool{}
	for _, l := range strings.SplitAfter(string(base), "\n") {
		seen[strings.TrimSuffix(l, "\n")] = true
	}
	for _, l := range strings.SplitAfter(string(extra), "\n") {
		key := strings.TrimSuffix(l, "\n")
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		if len(out) > 0 && out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		out = append(out, l...)
	}
	return out
}

func names(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for name := range m {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// writeFile replaces path in one step, so a session that reads the
// directory never sees half a memory.
func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".gellee-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}
