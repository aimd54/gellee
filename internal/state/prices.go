// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Price is the cost of one million tokens of each kind for one model.
type Price struct {
	Input      float64
	Output     float64
	CacheWrite float64
	CacheRead  float64
}

// LoadPrices reads the optional prices file: one model per line followed
// by four numbers. A missing file returns nil and no error.
func LoadPrices(stateDir string) (map[string]Price, error) {
	data, err := os.ReadFile(filepath.Join(stateDir, "prices")) //nolint:gosec // a path below the state directory
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	prices := map[string]Price{}
	for n, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		if len(fields) != 5 {
			return nil, fmt.Errorf("prices line %d: want a model and four numbers", n+1)
		}
		var v [4]float64
		for i := range v {
			if v[i], err = strconv.ParseFloat(fields[i+1], 64); err != nil {
				return nil, fmt.Errorf("prices line %d: %w", n+1, err)
			}
		}
		prices[fields[0]] = Price{Input: v[0], Output: v[1], CacheWrite: v[2], CacheRead: v[3]}
	}
	return prices, nil
}
