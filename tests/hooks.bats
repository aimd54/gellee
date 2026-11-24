#!/usr/bin/env bats
# Copyright The gellee Authors
# SPDX-License-Identifier: Apache-2.0

setup() {
  REPO_ROOT=$(cd "$BATS_TEST_DIRNAME/.." && pwd)
  HOOK="$REPO_ROOT/ci/hooks/pre-push"
  export TZ=UTC
  # A deny-list named in the user's git configuration must not leak in.
  export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null
  repo="$BATS_TEST_TMPDIR/repo"
  git init -q -b main "$repo"
  cd "$repo"
  git config user.name Test
  git config user.email test@example.invalid
  git config commit.gpgsign false
  printf 'hello\n' >README && git add README && git commit -q -s -m root
  export GELLEE_PUBLIC_ROOT
  GELLEE_PUBLIC_ROOT=$(git rev-parse HEAD)
  printf 'forbidden-word\n' >"$BATS_TEST_TMPDIR/denylist"
  export GELLEE_DENYLIST="$BATS_TEST_TMPDIR/denylist"
  export GELLEE_GITLEAKS_VERSION
  GELLEE_GITLEAKS_VERSION=$(gitleaks version)
  zero=0000000000000000000000000000000000000000
  public=git@github.com:aimd54/gellee.git
}

push_line() { printf '%s %s %s %s\n' "$1" "$2" "$3" "$4"; }

@test "a push to another remote is not checked" {
  unset GELLEE_DENYLIST
  run "$HOOK" origin git@git.internal:you/gellee.git <<<"$(push_line refs/heads/main "$(git rev-parse HEAD)" refs/heads/main "$zero")"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "a new branch to the public remote is checked and passes" {
  run "$HOOK" github "$public" <<<"$(push_line refs/heads/main "$(git rev-parse HEAD)" refs/heads/main "$zero")"
  [ "$status" -eq 0 ]
  [[ "$output" == *"public-gate: 1 commits, 1 patterns, 1 files"* ]]
}

@test "every public URL form is recognised" {
  for url in git@github.com:aimd54/gellee git@github.com:aimd54/gellee.git \
    git@github.com:aimd54/gellee/ git@github.com:AIMD54/gellee.git \
    git@github-work:aimd54/gellee.git \
    ssh://git@github.com/aimd54/gellee ssh://git@github.com/aimd54/gellee.git \
    ssh://github.com/aimd54/gellee.git ssh://git@github.com:22/aimd54/gellee.git \
    ssh://git@ssh.github.com:443/aimd54/gellee.git \
    https://github.com/aimd54/gellee https://github.com/aimd54/gellee.git \
    https://github.com/aimd54/gellee/ https://aimd54@github.com/aimd54/gellee.git; do
    unset GELLEE_DENYLIST
    run "$HOOK" github "$url" <<<"$(push_line refs/heads/main "$(git rev-parse HEAD)" refs/heads/main "$zero")"
    [ "$status" -eq 1 ] || { echo "not gated: $url"; false; }
    [[ "$output" == *"public-gate: deny-list"* ]] || { echo "not gated: $url"; false; }
  done
}

@test "look-alike URLs are not gated" {
  unset GELLEE_DENYLIST
  for url in git@git.internal:you/gellee.git \
    https://github.com/aimd54/gellee.git/extra \
    https://github.com/other-aimd54/gellee.git https://example.invalid/gellee; do
    run "$HOOK" origin "$url" <<<"$(push_line refs/heads/main "$(git rev-parse HEAD)" refs/heads/main "$zero")"
    [ "$status" -eq 0 ] || { echo "gated: $url"; false; }
    [ -z "$output" ]
  done
}

@test "a go.mod without a usable module line stops the hook" {
  fake="$BATS_TEST_TMPDIR/fake"
  mkdir -p "$fake/ci/hooks"
  command cp -f "$HOOK" "$fake/ci/hooks/pre-push"
  for body in '' $'module\n' $'go 1.22\n'; do
    printf '%s' "$body" >"$fake/go.mod"
    run "$fake/ci/hooks/pre-push" github "$public" </dev/null
    [ "$status" -ne 0 ]
    [[ "$output" == *"module"* ]]
  done
  rm -f "$fake/go.mod"
  run "$fake/ci/hooks/pre-push" github "$public" </dev/null
  [ "$status" -ne 0 ]
  [[ "$output" == *"module"* ]]
}

@test "a go.mod with CRLF endings and a comment is read" {
  fake="$BATS_TEST_TMPDIR/fake"
  mkdir -p "$fake/ci/hooks"
  command cp -f "$HOOK" "$fake/ci/hooks/pre-push"
  ln -sf "$REPO_ROOT/ci/public-gate" "$fake/ci/public-gate"
  printf 'module github.com/aimd54/gellee // the module\r\n\r\ngo 1.22\r\n' >"$fake/go.mod"
  unset GELLEE_DENYLIST
  run "$fake/ci/hooks/pre-push" github "$public" <<<"$(push_line refs/heads/main "$(git rev-parse HEAD)" refs/heads/main "$zero")"
  [ "$status" -eq 1 ]
  [[ "$output" == *"public-gate: deny-list"* ]]
}

@test "a hit blocks the push" {
  printf 'forbidden-word\n' >a.txt && git add a.txt && git commit -q -s -m a
  run "$HOOK" github "$public" <<<"$(push_line refs/heads/main "$(git rev-parse HEAD)" refs/heads/main "$zero")"
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list:"* ]]
}

@test "an update checks only the new commits, and the whole tree" {
  old=$(git rev-parse HEAD)
  printf 'fine\n' >a.txt && git add a.txt && git commit -q -s -m a
  run "$HOOK" github "$public" <<<"$(push_line refs/heads/main "$(git rev-parse HEAD)" refs/heads/main "$old")"
  [ "$status" -eq 0 ]
  [[ "$output" == *"public-gate: 1 commits, 1 patterns, 2 files"* ]]
}

@test "the pushed ref name is checked" {
  run "$HOOK" github "$public" <<<"$(push_line refs/heads/x "$(git rev-parse HEAD)" refs/heads/forbidden-word "$zero")"
  [ "$status" -eq 1 ]
  [[ "$output" == *"ref name"* ]]
}

@test "a deletion is not checked" {
  unset GELLEE_DENYLIST
  run "$HOOK" github "$public" <<<"$(push_line '(delete)' "$zero" refs/heads/old "$(git rev-parse HEAD)")"
  [ "$status" -eq 0 ]
}

@test "make hooks points git at ci/hooks" {
  run make -C "$REPO_ROOT" --no-print-directory -n hooks
  [ "$status" -eq 0 ]
  [[ "$output" == *"git config core.hooksPath ci/hooks"* ]]
}
