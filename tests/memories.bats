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
  export HOME="$BATS_TEST_TMPDIR/home" GELLEE_MACHINE=m1
  export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null
  export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.invalid
  export GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.invalid
  FERN="$GELLEE_PROJECTS_ROOT/alpha/fern"
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review"
  FORGE="$BATS_TEST_TMPDIR/forge.git"
  git init -q --bare -b main "$FORGE"
  export GELLEE_MEMORIES="$BATS_TEST_TMPDIR/memories"
  git clone -q "$FORGE" "$GELLEE_MEMORIES" 2>/dev/null
  git -C "$GELLEE_MEMORIES" symbolic-ref HEAD refs/heads/main
  MEM="$HOME/.claude/projects/$(readlink -f "$FERN" | tr '/.' '--')/memory"
  mkdir -p "$MEM"
  printf 'one\n' >"$MEM/one.md"
  gellee memories adopt --apply >/dev/null
}

# push_from_elsewhere <path> <content>: a commit another machine pushed
# to the store.
push_from_elsewhere() {
  local other="$BATS_TEST_TMPDIR/other"
  git clone -q "$FORGE" "$other"
  mkdir -p "$other/$(dirname "$1")"
  printf '%s\n' "$2" >"$other/$1"
  git -C "$other" add -- "$1"
  git -C "$other" commit -q -s -m elsewhere
  git -C "$other" push -q origin HEAD:main
}

@test "says when the sync imported memories for this project" {
  push_from_elsewhere alpha/fern/two.md two
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee memories: 1 memory files imported for alpha/fern, restart to use them"* ]]
  [ "$(cat "$MEM/two.md")" = two ]
}

@test "says nothing about imports when none was for this project" {
  push_from_elsewhere alpha/pond/two.md two
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee: alpha/fern is active."* ]]
  [[ "$output" != *"imported"* ]]
  [ "$(cat "$GELLEE_MEMORIES/alpha/pond/two.md")" = two ]
}

@test "says when versions set aside by a conflict are waiting" {
  push_from_elsewhere orphans/m1/conflicts/alpha/fern/one.md mine
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee memories: 1 versions set aside by a conflict, run /memories"* ]]
}

@test "says nothing about versions set aside on another machine" {
  push_from_elsewhere orphans/m2/conflicts/alpha/fern/one.md theirs
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee: alpha/fern is active."* ]]
  [[ "$output" != *"set aside"* ]]
  [ "$(cat "$GELLEE_MEMORIES/orphans/m2/conflicts/alpha/fern/one.md")" = theirs ]
}

@test "says when the store has not been pushed for a day" {
  git -C "$GELLEE_MEMORIES" remote set-url origin "$BATS_TEST_TMPDIR/nowhere.git"
  printf '2026-01-01T00:00:00Z' >"$GELLEE_MEMORIES/.git/gellee/pushed"
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee memories: no push to the store since 2026-01-01T00:00:00Z"* ]]
}

@test "says nothing about pushing after a push" {
  printf '2026-01-01T00:00:00Z' >"$GELLEE_MEMORIES/.git/gellee/pushed"
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee: alpha/fern is active."* ]]
  [[ "$output" != *"no push"* ]]
  # This run wrote a new time: a valid RFC 3339 value, not the old one.
  local pushed
  pushed=$(cat "$GELLEE_MEMORIES/.git/gellee/pushed")
  [ "$pushed" != 2026-01-01T00:00:00Z ]
  [[ "$pushed" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(Z|[+-][0-9]{2}:[0-9]{2})$ ]]
  date -d "$pushed" >/dev/null
}

@test "puts the notices in the session context too" {
  push_from_elsewhere orphans/m1/conflicts/alpha/fern/one.md mine
  run ctx "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee status for alpha/fern"* ]]
  [[ "$output" == *"gellee memories: 1 versions set aside by a conflict, run /memories"* ]]
}

@test "reports a store that is not a checkout in one line" {
  export GELLEE_MEMORIES="$BATS_TEST_TMPDIR/not-a-store"
  mkdir -p "$GELLEE_MEMORIES"
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee memories: $GELLEE_MEMORIES is not a checkout of the memory store"* ]]
}

@test "stays silent about memories where the store is not set up" {
  unset GELLEE_MEMORIES
  run ctx "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee status for alpha/fern"* ]]
  [[ "$output" != *"gellee memories"* ]]
}
