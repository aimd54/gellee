// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package memories

import "path/filepath"

// Health returns when this machine last pushed to the store, empty
// before its first push, and how many versions a conflict set aside for
// it. It reads nothing else from the checkout and never memory text.
func Health(o Options) (pushed string, setAside int, err error) {
	if err := o.checkout(); err != nil {
		return "", 0, err
	}
	if o.readState("synced") == "" {
		return "", 0, ErrNotAdopted
	}
	setAside, err = countFiles(filepath.Join(o.StoreDir, "orphans", o.Machine, "conflicts"))
	if err != nil {
		return "", 0, err
	}
	return o.readState("pushed"), setAside, nil
}
