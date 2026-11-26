// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aimd54/gellee/internal/demo"
	"github.com/aimd54/gellee/internal/server"
	"github.com/aimd54/gellee/web"
)

func runServe(args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(errw)
	stateDir := fs.String("state", os.Getenv("GELLEE_STATE"), "state repository checkout")
	listen := fs.String("listen", "127.0.0.1:8080", "address to listen on")
	refresh := fs.Duration("refresh", time.Minute, "how long a snapshot is served before the files are read again")
	today := fs.String("today", "", "fix the current date (YYYY-MM-DD), for tests and demonstrations")
	demoMode := fs.Bool("demo", false, "serve invented data from a temporary directory, ignoring --state")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	now := time.Now
	if *today != "" {
		day, err := time.ParseInLocation("2006-01-02", *today, time.Local)
		if err != nil {
			fmt.Fprintf(errw, "gellee: --today: %v\n", err)
			return 2
		}
		fixed := day.Add(12 * time.Hour)
		now = func() time.Time { return fixed }
	}
	// Registered before the demo is written, so a signal during startup
	// only cancels ctx and the deferred removal still runs.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	dir, what := *stateDir, *stateDir
	if *demoMode {
		tmp, err := os.MkdirTemp("", "gellee-demo-")
		if err != nil {
			fmt.Fprintf(errw, "gellee: %v\n", err)
			return 1
		}
		defer func() { _ = os.RemoveAll(tmp) }()
		if err := demo.Write(tmp, now()); err != nil {
			fmt.Fprintf(errw, "gellee: demo: %v\n", err)
			return 1
		}
		dir, what = tmp, "the demo"
	}
	if _, err := server.Build(dir, now()); err != nil {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	ui, err := web.FS()
	if err != nil {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	api := server.New(dir, ui, now, *refresh)
	api.Demo = *demoMode
	srv := &http.Server{
		Addr:              *listen,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	fmt.Fprintf(out, "gellee: serving %s on http://%s\n", what, *listen)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	return 0
}
