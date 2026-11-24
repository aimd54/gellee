#!/usr/bin/env bats
# Copyright The gellee Authors
# SPDX-License-Identifier: Apache-2.0

setup() {
  REPO_ROOT=$(cd "$BATS_TEST_DIRNAME/.." && pwd)
  NOTES="$REPO_ROOT/ci/scripts/release-notes"
  repo="$BATS_TEST_TMPDIR/repo"
  git init -q -b main "$repo"
  cd "$repo"
  git config user.name Test
  git config user.email test@example.invalid
  git config commit.gpgsign false
}

c() { git commit -q --allow-empty -m "$1"; }

@test "the first release lists every commit by type" {
  c 'feat: add a page'
  c 'fix(server): keep the order'
  c 'docs: explain the page'
  c 'feat!: rename the flag'
  git tag v0.1.0
  run "$NOTES" v0.1.0
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf '%s\n' '## Breaking changes' '' '- rename the flag' '' \
    '## Features' '' '- add a page' '' '## Fixes' '' '- keep the order (server)' '' \
    '## Other changes' '' '- docs: explain the page')" ]
}

@test "a later release lists only the commits since the previous tag" {
  c 'feat: one'
  git tag v0.1.0
  c 'fix: two'
  git tag v0.1.1
  run "$NOTES" v0.1.1
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf '%s\n' '## Fixes' '' '- two')" ]
}

@test "a footer marks a breaking change" {
  git commit -q --allow-empty -m 'fix: three' -m 'BREAKING CHANGE: the flag is gone'
  git tag v1.0.0
  run "$NOTES" v1.0.0
  [[ "$output" == *"## Breaking changes"*"- three"* ]]
}

@test "an unknown tag fails" {
  c 'feat: one'
  run "$NOTES" v9.9.9
  [ "$status" -eq 1 ]
  [ "$output" = "no tag v9.9.9" ]
}

@test "an at-sign in a subject cannot mention anyone" {
  c 'chore(deps): update dependency @playwright/test'
  c 'fix: thank @someone for the report'
  git tag v0.1.0
  run "$NOTES" v0.1.0
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf '%s\n' '## Fixes' '' '- thank `@someone` for the report' '' \
    '## Other changes' '' '- chore(deps): update dependency `@playwright/test`')" ]
}
