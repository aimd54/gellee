// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Notices are the lines the session-start hook shows for the project
// key. res is what the sync the hook just ran did, or nil when it did
// not run.
func Notices(o Options, key string, res *Result) ([]string, error) {
	pushedAt, aside, err := Health(o)
	if err != nil {
		return nil, err
	}
	var lines []string
	if res != nil && res.ImportedBy[key] > 0 {
		lines = append(lines, fmt.Sprintf("%d memory files imported for %s, restart to use them", res.ImportedBy[key], key))
	}
	if aside > 0 {
		lines = append(lines, fmt.Sprintf("%d versions set aside by a conflict, run /memories", aside))
	}
	pushed, err := time.Parse(time.RFC3339, pushedAt)
	switch {
	case err != nil:
		lines = append(lines, "no push to the store yet")
	case o.Now.Sub(pushed) > 24*time.Hour:
		lines = append(lines, "no push to the store since "+pushed.Format(time.RFC3339))
	}
	return lines, nil
}

// countFiles counts the regular files below dir, which may not exist.
func countFiles(dir string) (int, error) {
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	n := 0
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			n++
		}
		return nil
	})
	return n, err
}
