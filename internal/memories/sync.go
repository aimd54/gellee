// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ErrNotAdopted is returned on a machine that has not joined the store.
var ErrNotAdopted = errors.New("this machine has not joined the memory store: run gellee memories adopt")

// Result says what a sync did.
type Result struct {
	Exported    int            // files this machine changed since its last sync
	Imported    int            // files written to or removed from this machine
	Conflicts   int            // versions newly set aside under orphans
	Overridden  int            // deletions undone because the other side had changed the file
	ImportedBy  map[string]int // Imported, per unit
	Unreachable bool           // the remote did not answer
	PushErr     error          // the push failed although the remote answered
}

// side says which side of a merge holds the store's version.
type side int

const (
	storeIsOurs   side = iota // merging this machine's snapshot into the branch
	storeIsTheirs             // merging the remote into a branch that holds unpushed snapshots
)

// begin opens an operation on the store: it checks the checkout out,
// takes the lock, refuses a machine that cannot commit and a dirty tree,
// and names the units and the branch. With repair set it first undoes
// what an interrupted sync left behind. The caller releases the lock
// with unlock.
func (o Options) begin(repair bool) (r repo, us []unit, branch string, unlock func(), err error) {
	if err = o.checkout(); err != nil {
		return r, nil, "", nil, err
	}
	release, err := o.lock()
	if err != nil {
		return r, nil, "", nil, err
	}
	r = o.repo()
	err = r.identity()
	if err == nil && repair {
		err = o.recover(r)
	}
	if err == nil {
		err = r.clean()
	}
	if err == nil {
		if us, err = o.units(); err == nil {
			branch, err = r.git("symbolic-ref", "--short", "HEAD")
		}
	}
	if err != nil {
		release()
		return r, nil, "", nil, err
	}
	return r, us, branch, release, nil
}

// recover makes a sync that died halfway repeatable:
// `synced` is written last, so everything an interrupted run left in the
// checkout is rebuilt from it and from the memory directories. A merge in
// progress is aborted, along with the versions it had set aside, which
// are untracked and set aside again by the repeated run. A detached
// HEAD, which the snapshot leaves, is reset and put back on the store's
// branch.
func (o Options) recover(r repo) error {
	if r.has("MERGE_HEAD") {
		if _, err := r.git("merge", "--abort"); err != nil {
			return err
		}
		if _, err := r.git("clean", "-fdq", "--", "orphans/"+o.Machine+"/"); err != nil {
			return err
		}
	}
	if _, err := r.git("symbolic-ref", "-q", "HEAD"); err == nil {
		return nil
	}
	branch, err := o.storeBranch(r)
	if err != nil {
		return err
	}
	if _, err := r.git("reset", "-q", "--hard"); err != nil {
		return err
	}
	if _, err := r.git("clean", "-fdq"); err != nil {
		return err
	}
	_, err = r.git("checkout", "-q", branch)
	return err
}

// storeBranch is the branch of the store when HEAD is detached: the
// only local branch, or else the one the remote's HEAD names.
func (o Options) storeBranch(r repo) (string, error) {
	out, err := r.git("for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return "", err
	}
	if out != "" && !strings.Contains(out, "\n") {
		return out, nil
	}
	if head, err := r.git("symbolic-ref", "-q", "--short", "refs/remotes/origin/HEAD"); err == nil {
		if b, ok := strings.CutPrefix(head, "origin/"); ok && r.has("refs/heads/"+b) {
			return b, nil
		}
	}
	return "", errors.New("the checkout of the memory store is detached and holds no single branch to return to")
}

// push sends HEAD to the branch of the remote and records the time when
// it arrives. pushErr is the push failing; err is the record failing.
func (o Options) push(r repo, branch string) (pushErr, err error) {
	if pushErr = r.net("push", "-q", "origin", "HEAD:refs/heads/"+branch); pushErr != nil {
		return pushErr, nil
	}
	return nil, o.writeState("pushed", o.Now.Format(time.RFC3339))
}

// Sync exchanges this machine's memories with the store.
func Sync(o Options) (Result, error) {
	res := Result{ImportedBy: map[string]int{}}
	r, us, branch, unlock, err := o.begin(true)
	if err != nil {
		return res, err
	}
	defer unlock()
	synced := o.readState("synced")
	if synced == "" {
		return res, ErrNotAdopted
	}
	adopted := o.adopted()
	// What the run read from each unit's local directory: copy back
	// changes only the files that still hold it.
	views := map[string]map[string][]byte{}

	snap, err := o.snapshot(r, us, adopted, views, synced, branch, &res)
	if err != nil {
		return res, err
	}
	// The second round only runs when the remote moved between the
	// fetch and the push.
	for attempt := 0; attempt < 2; attempt++ {
		reachable, remote := r.fetch()
		res.Unreachable = remote && !reachable
		if reachable && r.has("origin/"+branch) {
			if err := o.merge(r, "origin/"+branch, storeIsTheirs, &res); err != nil {
				return res, err
			}
		}
		if attempt == 0 && snap != "" {
			if err := o.merge(r, snap, storeIsOurs, &res); err != nil {
				return res, err
			}
		}
		if attempt == 0 {
			if err := o.join(r, us, adopted, views); err != nil {
				return res, err
			}
		}
		if err := o.prune(r); err != nil {
			return res, err
		}
		if !reachable {
			break
		}
		res.PushErr, err = o.push(r, branch)
		if err != nil {
			return res, err
		}
		if res.PushErr == nil {
			break
		}
	}

	if err := o.copyBack(r, us, views, &res); err != nil {
		return res, err
	}
	// Only the units of this run stay joined. A project that leaves the
	// machine and comes back is then joined again by union, which never
	// deletes, rather than having the directory it left behind read as
	// this machine's edits.
	present := map[string]bool{}
	for _, u := range us {
		present[u.Key] = true
	}
	if err := o.setAdopted(present); err != nil {
		return res, err
	}
	head, err := r.git("rev-parse", "HEAD")
	if err != nil {
		return res, err
	}
	return res, o.writeState("synced", head)
}

// join brings in, by union, every unit of this machine that it has not
// joined yet: a project cloned after the machine adopted. The union
// writes only to the store; copy back brings the store's files to the
// machine once they are committed.
func (o Options) join(r repo, us []unit, adopted map[string]bool, views map[string]map[string][]byte) error {
	joined := 0
	for _, u := range us {
		if adopted[u.Key] {
			continue
		}
		_, local, err := o.union(u, true)
		if err != nil {
			return err
		}
		views[u.Key] = local
		adopted[u.Key] = true
		joined++
	}
	if joined == 0 {
		return nil
	}
	what := "new projects"
	if joined == 1 {
		what = "new project"
	}
	_, err := r.commit(fmt.Sprintf("adopt(%s): %d %s", o.Machine, joined, what))
	return err
}

// snapshot commits what this machine changed since its last sync, on
// top of the commit it last synced, and returns that commit, or "" when
// nothing changed. It records in views what it read from each unit, and
// leaves the checkout on branch.
func (o Options) snapshot(r repo, us []unit, adopted map[string]bool, views map[string]map[string][]byte, synced, branch string, res *Result) (string, error) {
	if _, err := r.git("checkout", "-q", "--detach", synced); err != nil {
		return "", err
	}
	back := func(cause error) (string, error) {
		for _, args := range [][]string{{"reset", "-q", "--hard"}, {"clean", "-fdq"}, {"checkout", "-q", branch}} {
			if _, err := r.git(args...); err != nil && cause == nil {
				cause = err
			}
		}
		return "", cause
	}
	for _, u := range us {
		if !adopted[u.Key] {
			continue
		}
		local, err := u.files(u.Local)
		if err != nil {
			return back(err)
		}
		views[u.Key] = local
		storeDir := filepath.Join(o.StoreDir, filepath.FromSlash(u.Key))
		if empty(local) {
			// A missing or empty directory is restored from the store.
			// Read as a deletion, a new clone or a cleared cache would
			// erase a project's memories everywhere. So is a directory
			// holding only its index, which a session may write into a
			// directory it lost, unless the store holds no memory to
			// restore either: then the index is all the project has.
			stored, err := u.files(storeDir)
			if err != nil {
				return back(err)
			}
			if len(local) == 0 || !empty(stored) {
				continue
			}
		}
		n, err := u.mirror(local, storeDir)
		if err != nil {
			return back(err)
		}
		res.Exported += n
	}
	if res.Exported == 0 {
		return back(nil)
	}
	if _, err := r.commit(fmt.Sprintf("sync(%s): %d files", o.Machine, res.Exported)); err != nil {
		return back(err)
	}
	snap, err := r.git("rev-parse", "HEAD")
	if err != nil {
		return back(err)
	}
	if _, err := r.git("checkout", "-q", branch); err != nil {
		return "", err
	}
	return snap, nil
}

// empty reports whether a unit's files hold no memory: none at all, or
// only the index, which a session may write into a directory it lost.
func empty(files map[string][]byte) bool {
	for name := range files {
		if name != "MEMORY.md" {
			return false
		}
	}
	return true
}

// merge brings ref into the current branch and settles every conflict,
// so a sync never stops on one. --ff overrides a merge.ff setting that
// would refuse the merge commit.
func (o Options) merge(r repo, ref string, s side, res *Result) error {
	msg := fmt.Sprintf("sync(%s): merge", o.Machine)
	_, err := r.git("merge", "-q", "--ff", "--signoff", "-X", "no-renames", "-m", msg, ref)
	if err == nil {
		return nil
	}
	return o.settle(r, s, res, err)
}

// settle resolves the unmerged paths of a merge in progress and
// concludes it. cause is returned when the merge failed for a reason
// other than conflicts.
func (o Options) settle(r repo, s side, res *Result, cause error) error {
	out, err := r.raw("status", "--porcelain=v1", "-z")
	if err != nil {
		return err
	}
	settled := 0
	var restored []restoredFile
	for _, entry := range strings.Split(string(out), "\x00") {
		if len(entry) < 4 {
			continue
		}
		code, p := entry[:2], entry[3:]
		switch code {
		case "UU", "AA":
			if err := o.setAside(r, s, p); err != nil {
				return err
			}
			res.Conflicts++
		case "UD", "DU":
			// The changed version is in the working tree: a change is a
			// sign the memory is still in use.
			if _, err := r.git("add", "--", p); err != nil {
				return err
			}
			res.Overridden++
			// UD: this branch changed it. DU: the merged ref did.
			ref := "HEAD"
			if code == "DU" {
				ref = "MERGE_HEAD"
			}
			restored = append(restored, restoredFile{path: p, changedIn: ref})
		case "DD":
			if _, err := r.git("rm", "-q", "--", p); err != nil {
				return err
			}
		default:
			continue
		}
		settled++
	}
	if settled == 0 {
		_, _ = r.git("merge", "--abort")
		return cause
	}
	for _, f := range restored {
		if err := o.reindex(r, f); err != nil {
			return err
		}
	}
	if _, err := r.git("add", "-A"); err != nil {
		return err
	}
	_, err = r.git("commit", "-q", "-s", "-m", fmt.Sprintf("sync(%s): merge", o.Machine))
	return err
}

// restoredFile is a memory kept because one side changed it while the
// other deleted it. changedIn is the ref of the side that changed it.
type restoredFile struct{ path, changedIn string }

// setAside keeps the store's version of path and writes this machine's
// under orphans/<machine>/conflicts/.
func (o Options) setAside(r repo, s side, p string) error {
	stage, keep := "3", "--ours"
	if s == storeIsTheirs {
		stage, keep = "2", "--theirs"
	}
	if !strings.HasPrefix(p, "orphans/") {
		mine, err := r.raw("show", ":"+stage+":"+p)
		if err != nil {
			return err
		}
		aside := filepath.Join(o.StoreDir, "orphans", o.Machine, "conflicts", filepath.FromSlash(p))
		if err := writeFile(freeName(aside, mine), mine); err != nil {
			return err
		}
	}
	if _, err := r.git("checkout", "-q", keep, "--", p); err != nil {
		return err
	}
	_, err := r.git("add", "--", p)
	return err
}

// linkTo reports whether a line of an index links to the memory file.
func linkTo(index, file string) (line string, ok bool) {
	for _, l := range strings.SplitAfter(index, "\n") {
		if m := link.FindStringSubmatch(l); m != nil && m[1] == file {
			return l, true
		}
	}
	return "", false
}

// reindex gives a memory kept against a deletion a line in the merged
// index of its project: the other side usually deleted the line along
// with the file, and Claude Code loads no memory the index does not
// list. The line comes from the index of the side that changed the
// memory, or is built from the memory's name.
func (o Options) reindex(r repo, f restoredFile) error {
	dir, file := path.Split(f.path)
	// Only <root>/<project>/<file>.md has an index: not global/, not orphans/.
	if strings.Count(dir, "/") != 2 || strings.HasPrefix(dir, "orphans/") ||
		file == "MEMORY.md" || !strings.HasSuffix(file, ".md") {
		return nil
	}
	indexPath := filepath.Join(o.StoreDir, filepath.FromSlash(dir), "MEMORY.md")
	merged, err := os.ReadFile(indexPath) //nolint:gosec // a path below the checkout
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if _, ok := linkTo(string(merged), file); ok {
		return nil
	}
	var line string
	listed, err := r.git("ls-tree", "--name-only", f.changedIn, "--", dir+"MEMORY.md")
	if err != nil {
		return err
	}
	if listed != "" {
		theirs, err := r.raw("show", f.changedIn+":"+dir+"MEMORY.md")
		if err != nil {
			return err
		}
		line, _ = linkTo(string(theirs), file)
	}
	if line == "" {
		body, err := os.ReadFile(filepath.Join(o.StoreDir, filepath.FromSlash(f.path))) //nolint:gosec // a path below the checkout
		if err != nil {
			return err
		}
		line = fmt.Sprintf("- [%s](%s)\n", memoryName(body, file), file)
	}
	index := string(merged)
	if index != "" && !strings.HasSuffix(index, "\n") {
		index += "\n"
	}
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	return writeFile(indexPath, []byte(index+line))
}

// memoryName is the name in a memory's frontmatter, or its file name
// without the extension.
func memoryName(body []byte, file string) string {
	lines := strings.Split(string(body), "\n")
	if lines[0] == "---" {
		for _, l := range lines[1:] {
			if l == "---" {
				break
			}
			if v, ok := strings.CutPrefix(l, "name:"); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	return strings.TrimSuffix(file, ".md")
}

// link matches the target of a Markdown link to a file of the same
// directory.
var link = regexp.MustCompile(`\]\(([^)/]+\.md)\)`)

// prune drops, from every project's index, the lines that link to a
// memory that is not there and the repeats a union merge can leave.
func (o Options) prune(r repo) error {
	indexes, err := filepath.Glob(filepath.Join(o.StoreDir, "*", "*", "MEMORY.md"))
	if err != nil {
		return err
	}
	for _, index := range indexes {
		data, err := os.ReadFile(index) //nolint:gosec // a path found below the checkout
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		var kept []string
		for _, line := range strings.SplitAfter(string(data), "\n") {
			if m := link.FindStringSubmatch(line); m != nil {
				if _, err := os.Stat(filepath.Join(filepath.Dir(index), m[1])); err != nil { //nolint:gosec // a link target inside a store directory
					if !errors.Is(err, fs.ErrNotExist) {
						return err
					}
					continue
				}
				if seen[line] {
					continue
				}
				seen[line] = true
			}
			kept = append(kept, line)
		}
		if pruned := strings.Join(kept, ""); pruned != string(data) {
			if err := writeFile(index, []byte(pruned)); err != nil {
				return err
			}
		}
	}
	_, err = r.commit(fmt.Sprintf("sync(%s): drop index lines that point nowhere", o.Machine))
	return err
}

// copyBack makes the local directory of every unit the run read, by
// views, equal to its directory in the store, except for the files a
// session changed since the run read them, which reconcile settles. The
// versions it sets aside are committed before any local file is
// overwritten, and the next sync pushes them. A session may write again
// while they are committed, so reconcile runs again until a round sets
// nothing new aside.
func (o Options) copyBack(r repo, us []unit, views map[string]map[string][]byte, res *Result) error {
	type plan struct {
		u              unit
		target, expect map[string][]byte
	}
	var plans []plan
	merged := res.Overridden
	for round := 0; round < 10; round++ {
		plans = plans[:0]
		var written []string
		overridden := 0
		for _, u := range us {
			view, ok := views[u.Key]
			if !ok {
				continue
			}
			want, err := u.files(filepath.Join(o.StoreDir, filepath.FromSlash(u.Key)))
			if err != nil {
				return err
			}
			target, expect, aside, n, err := u.reconcile(want, view)
			if err != nil {
				return err
			}
			overridden += n
			for _, name := range names(aside) {
				p := filepath.Join(o.StoreDir, "orphans", o.Machine, "conflicts", filepath.FromSlash(u.Key), name)
				p = freeName(p, aside[name])
				if _, err := os.Lstat(p); err == nil {
					continue // set aside already, by this run or an earlier one
				}
				written = append(written, p)
				if err := writeFile(p, aside[name]); err != nil {
					return o.unwrite(r, written, err)
				}
			}
			plans = append(plans, plan{u, target, expect})
		}
		res.Overridden = merged + overridden
		if len(written) == 0 {
			break
		}
		if _, err := r.commit(fmt.Sprintf("sync(%s): set aside versions changed during the sync", o.Machine)); err != nil {
			return o.unwrite(r, written, err)
		}
		res.Conflicts += len(written)
	}
	for _, p := range plans {
		n, err := p.u.copyBack(p.target, p.expect)
		if err != nil {
			return err
		}
		if n > 0 {
			res.Imported += n
			res.ImportedBy[p.u.Key] = n
		}
	}
	return nil
}

// unwrite takes back the versions copy back set aside and could not
// commit, so the next sync does not find the tree dirty: the local
// files still hold them, and that sync sets them aside again.
func (o Options) unwrite(r repo, paths []string, cause error) error {
	if len(paths) == 0 {
		return cause
	}
	if _, err := r.git(append([]string{"reset", "-q", "--"}, paths...)...); err != nil {
		return errors.Join(cause, err)
	}
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return errors.Join(cause, err)
		}
	}
	return cause
}
