// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// Package web holds the built interface, embedded into the binary. Vite
// builds it into dist/app; before make web has run, it is empty.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS returns the built interface, rooted at its index.html.
func FS() (fs.FS, error) { return fs.Sub(dist, "dist/app") }
