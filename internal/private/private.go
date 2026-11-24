// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// Package private carries a project's private files, the untracked
// files git ignores whose names match *.private.*, through the state
// repository, and links them back into each checkout.
package private

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/aimd54/gellee/internal/state"
)

// ErrOutside is returned for a checkout outside the configured roots.
var ErrOutside = errors.New("outside the configured roots")

// Checkout is a working tree of a project and the directory of the
// state repository that keeps its private files.
type Checkout struct {
	Key   string // project key, <root>/<name>
	Top   string // top of the working tree, resolved
	State string // the state repository, absolute
	Store string // <state>/private/<root>/<name>
}

// Report lists paths relative to the top of the checkout.
type Report struct {
	Linked     []string `json:"linked"`
	Swept      []string `json:"swept"`
	Replaced   []string `json:"replaced"`
	Unswept    []string `json:"unswept"`
	InTheWay   []string `json:"in_the_way"`
	Missing    []string `json:"missing"`
	Broken     []string `json:"broken"`
	NotIgnored []string `json:"not_ignored"`
}

// Ready reports whether nothing keeps the checkout's private files from
// being found on another machine.
func (r Report) Ready() bool {
	return len(r.Unswept)+len(r.InTheWay)+len(r.Missing)+len(r.Broken)+len(r.NotIgnored) == 0
}

// Summary counts each non-empty list, in a fixed order.
func (r Report) Summary() string {
	var parts []string
	for _, g := range r.groups() {
		if len(g.paths) > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", len(g.paths), g.what))
		}
	}
	if len(parts) == 0 {
		return "nothing to report"
	}
	return strings.Join(parts, ", ")
}

// Group is one list of a report and what it holds.
type Group struct {
	What  string
	Paths []string
}

// Groups returns the lists of the report in the order Summary uses.
func (r Report) Groups() []Group {
	var out []Group
	for _, g := range r.groups() {
		out = append(out, Group{What: g.what, Paths: g.paths})
	}
	return out
}

func (r Report) groups() []struct {
	what  string
	paths []string
} {
	return []struct {
		what  string
		paths []string
	}{
		{"linked", r.Linked}, {"swept", r.Swept}, {"replaced", r.Replaced},
		{"unswept", r.Unswept},
		{"in the way", r.InTheWay}, {"missing", r.Missing},
		{"broken", r.Broken}, {"not ignored", r.NotIgnored},
	}
}

// Locate finds the working tree that holds dir and its project key: the
// first two directories of the tree's path below the projects root, the
// first being a configured root. A worktree inside a checkout belongs
// to that checkout's project; a repository nested in a project does not.
func Locate(stateDir, projectsRoot, dir string) (Checkout, error) {
	cfg, err := state.LoadConfig(stateDir)
	if err != nil {
		return Checkout{}, err
	}
	if projectsRoot == "" {
		projectsRoot = cfg.ProjectsRoot
	}
	if projectsRoot == "" {
		return Checkout{}, errors.New("projects root is not set")
	}
	root, err := filepath.EvalSymlinks(projectsRoot)
	if err != nil {
		return Checkout{}, err
	}
	top, err := gitOut(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return Checkout{}, fmt.Errorf("%s is not in a git working tree: %w", dir, ErrOutside)
	}
	if top, err = filepath.EvalSymlinks(top); err != nil {
		return Checkout{}, err
	}
	rel, err := filepath.Rel(root, top)
	if err != nil || !filepath.IsLocal(rel) {
		return Checkout{}, fmt.Errorf("%s: %w", top, ErrOutside)
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 2 || !slices.Contains(cfg.Roots, parts[0]) {
		return Checkout{}, fmt.Errorf("%s: %w", top, ErrOutside)
	}
	owner := filepath.Join(root, parts[0], parts[1])
	common, err := gitOut(top, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return Checkout{}, err
	}
	if dir, err := filepath.EvalSymlinks(filepath.Dir(common)); err != nil || dir != owner {
		return Checkout{}, fmt.Errorf("%s: %w", top, ErrOutside)
	}
	stateAbs, err := filepath.Abs(stateDir)
	if err != nil {
		return Checkout{}, err
	}
	return Checkout{
		Key:   parts[0] + "/" + parts[1],
		Top:   top,
		State: stateAbs,
		Store: filepath.Join(stateAbs, "private", parts[0], parts[1]),
	}, nil
}

func gitOut(dir string, args ...string) (string, error) {
	out, err := gitBytes(dir, args...)
	return strings.TrimRight(string(out), "\n"), err
}

func gitBytes(dir string, args ...string) ([]byte, error) {
	return exec.Command("git", append([]string{"-C", dir}, args...)...).Output() //nolint:gosec // fixed binary; arguments are a directory and constants
}

// isPrivate reports whether a name matches *.private.* and is not one of
// the temporary files this package writes.
func isPrivate(p string) bool {
	ok, _ := filepath.Match("*.private.*", filepath.Base(p))
	return ok && !strings.HasSuffix(p, ".gellee-tmp") && !strings.HasSuffix(p, ".gellee-link")
}

// stored is where the state repository keeps the file at rel.
func (c Checkout) stored(rel string) (string, error) {
	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("%s leaves the checkout", rel)
	}
	return filepath.Join(c.Store, rel), nil
}

// candidates lists the checkout's untracked private files, split by
// whether git ignores them, including inside ignored directories.
func (c Checkout) candidates() (ignored, plain []string, err error) {
	list := func(args ...string) ([]string, error) {
		args = append(append([]string{"ls-files", "-z", "--others", "--exclude-standard"}, args...), "--", "*.private.*")
		out, err := gitOut(c.Top, args...)
		if err != nil {
			return nil, err
		}
		var paths []string
		for _, p := range strings.Split(out, "\x00") {
			if p != "" && !strings.HasSuffix(p, "/") && isPrivate(p) {
				paths = append(paths, filepath.FromSlash(p))
			}
		}
		slices.Sort(paths)
		return paths, nil
	}
	if ignored, err = list("--ignored"); err != nil {
		return nil, nil, err
	}
	plain, err = list()
	return ignored, plain, err
}

// replaceWithLink puts a link to target at path in one rename, so path
// is never missing.
func replaceWithLink(path, target string) error {
	tmp := path + ".gellee-link"
	_ = os.Remove(tmp)
	if err := os.Symlink(target, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func syncDir(dir string) error {
	d, err := os.Open(dir) //nolint:gosec // a directory of the checkout or the store
	if err != nil {
		return err
	}
	defer d.Close() //nolint:errcheck // read only
	return d.Sync()
}

// walkStore checks every stored file against the checkout. With create,
// a missing link is made, and a link that is dangling or points
// elsewhere in the state repository's private directory is replaced;
// without create, they are reported.
func (c Checkout) walkStore(r *Report, create bool) error {
	err := filepath.WalkDir(c.Store, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == c.Store && errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if !d.Type().IsRegular() || !isPrivate(p) {
			return nil
		}
		rel, err := filepath.Rel(c.Store, p)
		if err != nil {
			return err
		}
		at := filepath.Join(c.Top, rel)
		info, err := os.Lstat(at)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if !create {
				r.Missing = append(r.Missing, rel)
				return nil
			}
			if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
				return err
			}
			if err := os.Symlink(p, at); err != nil { //nolint:gosec // the checkout is the user's own working tree
				return err
			}
			r.Linked = append(r.Linked, rel)
		case err != nil:
			return err
		case info.Mode()&fs.ModeSymlink != 0:
			target, err := os.Readlink(at)
			if err == nil && target == p {
				return nil
			}
			_, statErr := os.Stat(at)
			stray := err == nil && strings.HasPrefix(target, filepath.Join(c.State, "private")+string(filepath.Separator))
			if !create || (statErr == nil && !stray) {
				r.Broken = append(r.Broken, rel)
				return nil
			}
			if err := replaceWithLink(at, p); err != nil {
				return err
			}
			r.Linked = append(r.Linked, rel)
		default:
			r.InTheWay = append(r.InTheWay, rel)
		}
		return nil
	})
	return err
}

// Link makes the links that are missing from the checkout.
func Link(c Checkout) (Report, error) {
	var r Report
	err := c.walkStore(&r, true)
	return r, err
}

// Sweep moves each ignored private file that is still a regular file
// into the state repository and leaves a link in its place, then links
// what is missing. A file whose stored copy differs is left as it is,
// unless the stored copy is the committed one and the file is newer,
// as after an editor saved over its link.
func Sweep(c Checkout) (Report, error) {
	var r Report
	ignored, plain, err := c.candidates()
	if err != nil {
		return r, err
	}
	r.NotIgnored = plain
	for _, rel := range ignored {
		src := filepath.Join(c.Top, rel)
		info, err := os.Lstat(src)
		if err != nil {
			return r, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		dst, err := c.stored(rel)
		if err != nil {
			return r, err
		}
		data, err := os.ReadFile(src) //nolint:gosec // a file of the checkout
		if err != nil {
			return r, err
		}
		dinfo, err := os.Lstat(dst)
		replaced := false
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if err := writeDurable(dst, data, info.Mode().Perm()); err != nil {
				return r, err
			}
		case err != nil:
			return r, err
		case !dinfo.Mode().IsRegular() || os.SameFile(info, dinfo):
			r.InTheWay = append(r.InTheWay, rel)
			continue
		default:
			have, err := os.ReadFile(dst) //nolint:gosec // a file below the store
			if err != nil {
				return r, err
			}
			if !bytes.Equal(have, data) {
				if !c.newer(rel, have, info, dinfo) {
					continue // reported as in the way by walkStore
				}
				if err := writeDurable(dst, data, info.Mode().Perm()); err != nil {
					return r, err
				}
				replaced = true
			}
		}
		if err := replaceWithLink(src, dst); err != nil {
			return r, err
		}
		if replaced {
			r.Replaced = append(r.Replaced, rel)
		} else {
			r.Swept = append(r.Swept, rel)
		}
	}
	if err := c.walkStore(&r, true); err != nil {
		return r, err
	}
	slices.Sort(r.InTheWay)
	r.InTheWay = slices.Compact(r.InTheWay)
	return r, c.dangling(&r)
}

// newer reports whether the checkout's file replaces the stored one: the
// stored copy is the one the state repository last committed, and the
// checkout's file was written after it.
func (c Checkout) newer(rel string, have []byte, src, dst fs.FileInfo) bool {
	committed, err := gitBytes(c.State, "show", "HEAD:"+filepath.ToSlash(filepath.Join("private", c.Key, rel)))
	return err == nil && bytes.Equal(committed, have) && src.ModTime().After(dst.ModTime())
}

// dangling adds the checkout's private links whose target is gone.
func (c Checkout) dangling(r *Report) error {
	ignored, _, err := c.candidates()
	if err != nil {
		return err
	}
	for _, rel := range ignored {
		at := filepath.Join(c.Top, rel)
		info, err := os.Lstat(at)
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink == 0 || slices.Contains(r.Broken, rel) {
			continue
		}
		if _, err := os.Stat(at); err != nil {
			r.Broken = append(r.Broken, rel)
		}
	}
	return nil
}

// writeDurable writes data to a temporary file beside dst, flushes it to
// disk, checks it, and renames it into place.
func writeDurable(dst string, data []byte, perm fs.FileMode) (err error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	tmp := dst + ".gellee-tmp"
	defer func() {
		if err != nil {
			_ = os.Remove(tmp)
		}
	}()
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm) //nolint:gosec // dst is checked to stay below the store
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	back, err := os.ReadFile(tmp) //nolint:gosec // a file below the store
	if err != nil {
		return err
	}
	if !bytes.Equal(back, data) {
		return fmt.Errorf("%s: the copy differs from the original", dst)
	}
	if err := os.Rename(tmp, dst); err != nil {
		return err
	}
	return syncDir(filepath.Dir(dst))
}

// Status reports without changing anything.
func Status(c Checkout) (Report, error) {
	var r Report
	ignored, plain, err := c.candidates()
	if err != nil {
		return r, err
	}
	r.NotIgnored = plain
	for _, rel := range ignored {
		info, err := os.Lstat(filepath.Join(c.Top, rel))
		if err != nil {
			return r, err
		}
		dst, err := c.stored(rel)
		if err != nil {
			return r, err
		}
		if _, err := os.Lstat(dst); info.Mode().IsRegular() && err != nil {
			r.Unswept = append(r.Unswept, rel)
		}
	}
	if err := c.walkStore(&r, false); err != nil {
		return r, err
	}
	return r, c.dangling(&r)
}
