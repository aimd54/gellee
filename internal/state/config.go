// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// Package state reads and writes the files of the state repository:
// configuration, status files and the collector's summaries.
package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config is the state repository's config file.
type Config struct {
	Roots        []string
	Limit        int
	ActiveDays   int
	ProjectsRoot string
}

// LoadConfig reads <stateDir>/config, a file of key=value lines.
func LoadConfig(stateDir string) (Config, error) {
	c := Config{Limit: 3, ActiveDays: 10}
	data, err := os.ReadFile(filepath.Join(stateDir, "config")) //nolint:gosec // a path below the state directory
	if err != nil {
		return c, fmt.Errorf("config: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch key {
		case "roots":
			c.Roots = strings.Fields(value)
		case "projects_root":
			c.ProjectsRoot = value
		case "limit":
			if c.Limit, err = strconv.Atoi(value); err != nil {
				return c, fmt.Errorf("config: limit: %w", err)
			}
		case "active_days":
			if c.ActiveDays, err = strconv.Atoi(value); err != nil {
				return c, fmt.Errorf("config: active_days: %w", err)
			}
		}
	}
	if len(c.Roots) == 0 {
		return c, errors.New("config lists no roots")
	}
	return c, nil
}
