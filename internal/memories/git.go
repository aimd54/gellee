// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// repo runs git in the checkout of the store.
type repo struct {
	dir     string
	timeout time.Duration
}

func (o Options) repo() repo { return repo{dir: o.StoreDir, timeout: o.Timeout} }

func (r repo) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.dir}, args...)...) //nolint:gosec // fixed binary; arguments are refs and paths of the store
	// A sync runs unattended: a credential prompt must fail, not wait.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// git returns the trimmed standard output.
func (r repo) git(args ...string) (string, error) {
	out, err := r.run(context.Background(), args...)
	return strings.TrimSpace(string(out)), err
}

// raw returns the standard output as written, for file contents.
func (r repo) raw(args ...string) ([]byte, error) {
	return r.run(context.Background(), args...)
}

// net runs a command that talks to the remote, within the timeout.
func (r repo) net(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	_, err := r.run(ctx, args...)
	return err
}

// has reports whether a ref or a commit exists.
func (r repo) has(ref string) bool {
	_, err := r.git("rev-parse", "--verify", "-q", ref)
	return err == nil
}

// identity fails, before anything is written, when git has no name or
// email to sign a commit of the store with: the commit would fail half
// way through an operation.
func (r repo) identity() error {
	for _, v := range []string{"GIT_AUTHOR_IDENT", "GIT_COMMITTER_IDENT"} {
		if _, err := r.git("var", v); err != nil {
			return fmt.Errorf("no git identity to commit to the memory store with: set user.name and user.email: %w", err)
		}
	}
	return nil
}

// clean fails when the working tree holds uncommitted changes.
func (r repo) clean() error {
	out, err := r.git("status", "--porcelain")
	if err != nil {
		return err
	}
	if out != "" {
		return fmt.Errorf("the checkout of the memory store has uncommitted changes:\n%s", out)
	}
	return nil
}

// commit records everything in the working tree and reports whether
// there was anything to record.
func (r repo) commit(msg string) (bool, error) {
	if _, err := r.git("add", "-A"); err != nil {
		return false, err
	}
	if _, err := r.git("diff", "--cached", "--quiet"); err == nil {
		return false, nil
	}
	_, err := r.git("commit", "-q", "-s", "-m", msg)
	return err == nil, err
}

// fetch updates the remote's branches. It reports whether the remote
// answered and whether there is one at all.
func (r repo) fetch() (reachable, remote bool) {
	if out, _ := r.git("remote"); out == "" {
		return false, false
	}
	return r.net("fetch", "-q", "origin") == nil, true
}

// extract writes the regular files of the tree of a ref into dir,
// leaving the checkout alone.
func (r repo) extract(ref, dir string) error {
	archive, err := r.raw("archive", "--format=tar", ref)
	if err != nil {
		return err
	}
	tr := tar.NewReader(bytes.NewReader(archive))
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg || !filepath.IsLocal(h.Name) {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, filepath.FromSlash(h.Name)), data); err != nil {
			return err
		}
	}
}
