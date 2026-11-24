#!/usr/bin/env bats
# Copyright The gellee Authors
# SPDX-License-Identifier: Apache-2.0

load helper

setup_file() {
  export GELLEE_TEST_BIN="$BATS_FILE_TMPDIR/bin"
  (cd "$BATS_TEST_DIRNAME/.." && go build -o "$GELLEE_TEST_BIN/gellee" ./cmd/gellee)
}

setup() {
  common_setup
  PATH="$GELLEE_TEST_BIN:$PATH"
  export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null
  FERN="$GELLEE_PROJECTS_ROOT/alpha/fern"
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review"
  printf '*.private.md\n' >"$FERN/.gitignore"
}

@test "the hook links the project's private files into the checkout" {
  mkdir -p "$GELLEE_STATE/private/alpha/fern/docs"
  printf 'plan\n' >"$GELLEE_STATE/private/alpha/fern/docs/a.private.md"
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee private: 1 linked"* ]]
  [ -L "$FERN/docs/a.private.md" ]
  [ "$(cat "$FERN/docs/a.private.md")" = plan ]
}

@test "the hook says nothing about private files already linked" {
  mkdir -p "$GELLEE_STATE/private/alpha/fern"
  printf 'plan\n' >"$GELLEE_STATE/private/alpha/fern/a.private.md"
  hook "$FERN" >/dev/null
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" != *"private"* ]]
}

@test "the hook says when private files could not be linked" {
  mkdir -p "$GELLEE_STATE/private" "$BATS_TEST_TMPDIR/stub"
  printf '#!/bin/sh\nexit 1\n' >"$BATS_TEST_TMPDIR/stub/gellee"
  chmod +x "$BATS_TEST_TMPDIR/stub/gellee"
  PATH="$BATS_TEST_TMPDIR/stub:$PATH"
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"private files could not be linked"* ]]
}

@test "the hook says nothing about private files outside a git checkout" {
  mkdir -p "$GELLEE_STATE/private" "$GELLEE_PROJECTS_ROOT/alpha/pond"
  run hook "$GELLEE_PROJECTS_ROOT/alpha/pond"
  [ "$status" -eq 0 ]
  [[ "$output" != *"could not be linked"* ]]
}
