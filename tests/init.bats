#!/usr/bin/env bats

load helper

setup() {
  common_setup
  TODAY=$(date +%F)
}

@test "a recently committed project gets an active stub" {
  make_repo alpha/fern "$(days_ago 2)"
  run gellee-init
  [ "$status" -eq 0 ]
  expected="---
project: alpha/fern
state: active
updated: $TODAY
---

## Where I stopped

## Next action

## Remaining

## Ideas"
  [ "$(cat "$GELLEE_STATE/alpha/fern.md")" = "$expected" ]
  [[ "$output" == *"created alpha/fern (active)"* ]]
}

@test "an old project gets a parked stub with a reason" {
  make_repo beta/cluster "$(days_ago 40)"
  run gellee-init
  [ "$status" -eq 0 ]
  run cat "$GELLEE_STATE/beta/cluster.md"
  [ "${lines[1]}" = "project: beta/cluster" ]
  [ "${lines[2]}" = "state: parked" ]
  [ "${lines[3]}" = "parked_reason: stub, not yet reviewed" ]
  [ "${lines[4]}" = "updated: $TODAY" ]
}

@test "a directory that is not a git repository gets a parked stub" {
  mkdir -p "$GELLEE_PROJECTS_ROOT/alpha/notes"
  run gellee-init
  [ "$status" -eq 0 ]
  run cat "$GELLEE_STATE/alpha/notes.md"
  [ "${lines[2]}" = "state: parked" ]
}

@test "an existing status file keeps its bytes" {
  make_repo alpha/fern "$(days_ago 2)"
  mkdir -p "$GELLEE_STATE/alpha"
  printf 'hand written\n' >"$GELLEE_STATE/alpha/fern.md"
  run gellee-init
  [ "$status" -eq 0 ]
  [ "$(cat "$GELLEE_STATE/alpha/fern.md")" = "hand written" ]
  [ "${lines[-1]}" = "0 stub(s) created" ]
}

@test "only the configured roots are scanned" {
  make_repo alpha/fern "$(days_ago 2)"
  make_repo beta/cluster "$(days_ago 2)"
  make_repo ignored/other "$(days_ago 2)"
  run gellee-init
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "2 stub(s) created" ]
  run bash -c "cd '$GELLEE_STATE' && find . -name '*.md' | LC_ALL=C sort"
  [ "$output" = "./alpha/fern.md
./beta/cluster.md" ]
}

@test "a missing scan root is skipped" {
  printf 'roots=alpha absent\n' >"$GELLEE_STATE/config"
  make_repo alpha/fern "$(days_ago 2)"
  run gellee-init
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "1 stub(s) created" ]
}
