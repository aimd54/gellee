// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrJoined is returned when adopt is applied on a machine that has
// joined the store already.
var ErrJoined = errors.New("this machine has already joined the memory store: run gellee memories sync")

// Action is one line of an adoption plan.
type Action struct {
	What string // "to store", "to machine", "conflict" or "orphan"
	Key  string
	Name string
}

// Adopt joins this machine to the store. Without apply it only returns
// what it would do. It never deletes: every unit is joined by union, and
// a memory whose content differs from the store's is set aside.
func Adopt(o Options, apply bool) ([]Action, error) {
	r, us, branch, unlock, err := o.begin(false)
	if err != nil {
		return nil, err
	}
	defer unlock()
	// A joined machine syncs instead: adopt would move the checkout to
	// the remote without copying it back, and the next sync would read
	// the memory directories it left behind as this machine's edits.
	if apply && o.readState("synced") != "" {
		return nil, ErrJoined
	}
	// Read the joined set while StoreDir still names the checkout.
	adopted := o.adopted()
	// Join against what the other machines already pushed.
	reachable, remote := r.fetch()
	if reachable && r.has("origin/"+branch) {
		if apply {
			if _, err := r.git("merge", "-q", "--ff-only", "origin/"+branch); err != nil {
				return nil, err
			}
		} else {
			// A plan reads the remote's tree without touching the checkout.
			tmp, err := os.MkdirTemp("", "gellee-adopt-")
			if err != nil {
				return nil, err
			}
			defer os.RemoveAll(tmp) //nolint:errcheck // a temporary copy
			if err := r.extract("origin/"+branch, tmp); err != nil {
				return nil, err
			}
			o.StoreDir = tmp
		}
	}

	var plan []Action
	views := map[string]map[string][]byte{}
	for _, u := range us {
		if adopted[u.Key] {
			continue
		}
		acts, local, err := o.union(u, apply)
		if err != nil {
			return plan, err
		}
		plan = append(plan, acts...)
		views[u.Key] = local
	}
	acts, err := o.orphans(us, apply)
	if err != nil {
		return plan, err
	}
	plan = append(plan, acts...)
	if !apply {
		return plan, nil
	}

	attrs := filepath.Join(o.StoreDir, ".gitattributes")
	if _, err := os.Stat(attrs); err != nil {
		// Two machines each add a line to an index: keep both.
		if err := writeFile(attrs, []byte("MEMORY.md merge=union\n")); err != nil {
			return plan, err
		}
	}
	if _, err := r.commit(fmt.Sprintf("adopt(%s): %d files", o.Machine, stored(plan))); err != nil {
		return plan, err
	}
	// The machine's versions are committed: the store's may now replace
	// them.
	if err := o.copyBack(r, us, views, &Result{ImportedBy: map[string]int{}}); err != nil {
		return plan, err
	}
	if remote {
		pushErr, err := o.push(r, branch)
		if err != nil {
			return plan, err
		}
		if pushErr != nil {
			fmt.Fprintln(o.Out, "adopt: not pushed, the remote did not answer")
		}
	}
	for _, u := range us {
		adopted[u.Key] = true
	}
	if err := o.setAdopted(adopted); err != nil {
		return plan, err
	}
	head, err := r.git("rev-parse", "HEAD")
	if err != nil {
		return plan, err
	}
	return plan, o.writeState("synced", head)
}

// stored counts the actions that add or change a file of the store.
func stored(plan []Action) int {
	n := 0
	for _, a := range plan {
		if a.What != "to machine" {
			n++
		}
	}
	return n
}

// union plans how a unit's local directory and its store directory
// reach the same content without deleting anything. Where both hold a
// file of the same name with different content, the store's version
// goes to the machine and the machine's is set aside under orphans.
// With apply it writes the store's side; it returns what it read from
// the local directory, for the caller to copy the store's files back
// once that is committed, so no version of the machine's is replaced
// before the store holds it.
func (o Options) union(u unit, apply bool) ([]Action, map[string][]byte, error) {
	storeDir := filepath.Join(o.StoreDir, filepath.FromSlash(u.Key))
	local, err := u.files(u.Local)
	if err != nil {
		return nil, nil, err
	}
	stored, err := u.files(storeDir)
	if err != nil {
		return nil, nil, err
	}
	var acts []Action
	for _, name := range names(local) {
		s, ok := stored[name]
		switch {
		case !ok:
			acts = append(acts, Action{"to store", u.Key, name})
			if apply {
				if err := writeFile(filepath.Join(storeDir, name), local[name]); err != nil {
					return acts, local, err
				}
			}
		case !bytes.Equal(s, local[name]):
			acts = append(acts, Action{"conflict", u.Key, name})
			if apply {
				// The global file has no project directory to set aside under.
				dir := filepath.Join(o.StoreDir, "orphans", o.Machine)
				if u.Key != globalKey {
					dir = filepath.Join(dir, "conflicts", filepath.FromSlash(u.Key))
				}
				aside := freeName(filepath.Join(dir, name), local[name])
				if err := writeFile(aside, local[name]); err != nil {
					return acts, local, err
				}
			}
		}
	}
	for _, name := range names(stored) {
		if _, ok := local[name]; !ok {
			acts = append(acts, Action{"to machine", u.Key, name})
		}
	}
	return acts, local, nil
}

// freeName returns path, or path with a numbered suffix when path
// already holds different content: a version set aside earlier is never
// overwritten by a later one.
func freeName(path string, data []byte) string {
	candidate := path
	for i := 2; ; i++ {
		old, err := os.ReadFile(candidate) //nolint:gosec // a path below the checkout
		if err != nil || bytes.Equal(old, data) {
			return candidate
		}
		candidate = fmt.Sprintf("%s.%d", path, i)
	}
}

// orphans copies into the store every memory directory of this machine
// whose name is the encoding of no project. The local copies stay.
func (o Options) orphans(us []unit, apply bool) ([]Action, error) {
	known := map[string]bool{}
	for _, u := range us {
		if u.Key != globalKey {
			known[filepath.Base(filepath.Dir(u.Local))] = true
		}
	}
	entries, err := os.ReadDir(filepath.Join(o.ClaudeDir, "projects"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil // no Claude projects directory: no orphans
	}
	if err != nil {
		return nil, err
	}
	var acts []Action
	for _, e := range entries {
		if !e.IsDir() || known[e.Name()] {
			continue
		}
		ou := unit{
			Key:   "orphans/" + o.Machine + "/" + e.Name(),
			Local: filepath.Join(o.ClaudeDir, "projects", e.Name(), "memory"),
		}
		storeDir := filepath.Join(o.StoreDir, filepath.FromSlash(ou.Key))
		local, err := ou.files(ou.Local)
		if err != nil {
			return acts, err
		}
		stored, err := ou.files(storeDir)
		if err != nil {
			return acts, err
		}
		for _, name := range names(local) {
			if s, ok := stored[name]; ok && bytes.Equal(s, local[name]) {
				continue
			}
			acts = append(acts, Action{"orphan", ou.Key, name})
			if apply {
				if err := writeFile(filepath.Join(storeDir, name), local[name]); err != nil {
					return acts, err
				}
			}
		}
	}
	return acts, nil
}
