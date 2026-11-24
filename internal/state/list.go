// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// list returns the entries of <stateDir>/<rel>, sorted by name. A
// directory that does not exist is empty. One that exists and cannot be
// listed is a problem: a checkout the server's user may not read would
// otherwise look like a state repository with nothing in it.
func list(stateDir, rel string) ([]os.DirEntry, []Problem) {
	entries, err := os.ReadDir(filepath.Join(stateDir, filepath.FromSlash(rel)))
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return entries, nil
	}
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	return nil, []Problem{{File: rel, Message: "cannot list the directory: " + err.Error()}}
}
