// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// Package demo writes a state repository of invented projects, for
// trying the dashboard and for its browser tests. Every date is placed
// relative to an anchor time, and the same anchor gives the same files.
package demo

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

//go:embed state
var files embed.FS

// Write fills dir, which must exist, with the demo state repository as
// it stands at anchor.
func Write(dir string, anchor time.Time) error {
	if err := writeFiles(dir, anchor); err != nil {
		return err
	}
	return writeData(dir, anchor)
}

// writeFiles copies the embedded files into dir. A .tmpl file is
// rendered, where {{day N}} is the anchor day moved by N days, and
// written without its suffix.
func writeFiles(dir string, anchor time.Time) error {
	funcs := template.FuncMap{
		"day": func(n int) string { return anchor.AddDate(0, 0, n).Format("2006-01-02") },
	}
	return fs.WalkDir(files, "state", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := files.ReadFile(name)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(name, "state/")
		if strings.HasSuffix(rel, ".tmpl") {
			t, err := template.New(rel).Funcs(funcs).Option("missingkey=error").Parse(string(data))
			if err != nil {
				return err
			}
			var buf bytes.Buffer
			if err := t.Execute(&buf, nil); err != nil {
				return err
			}
			data, rel = buf.Bytes(), strings.TrimSuffix(rel, ".tmpl")
		}
		out := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o644) //nolint:gosec // invented data, not a secret
	})
}
