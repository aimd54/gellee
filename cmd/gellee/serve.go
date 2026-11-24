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
	if _, err := server.Build(*stateDir, now()); err != nil {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	ui, err := web.FS()
	if err != nil {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	srv := &http.Server{
		Addr:              *listen,
		Handler:           server.New(*stateDir, ui, now, *refresh).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	fmt.Fprintf(out, "gellee: serving %s on http://%s\n", *stateDir, *listen)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(errw, "gellee: %v\n", err)
		return 1
	}
	return 0
}
