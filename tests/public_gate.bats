#!/usr/bin/env bats
# Copyright The gellee Authors
# SPDX-License-Identifier: Apache-2.0

setup() {
  REPO_ROOT=$(cd "$BATS_TEST_DIRNAME/.." && pwd)
  GATE="$REPO_ROOT/ci/public-gate"
  export TZ=UTC GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null
  repo="$BATS_TEST_TMPDIR/repo"
  git init -q -b main "$repo"
  cd "$repo"
  git config user.name Test
  git config user.email test@example.invalid
  git config commit.gpgsign false
  add README 'hello'
  export GELLEE_PUBLIC_ROOT
  GELLEE_PUBLIC_ROOT=$(git rev-parse HEAD)
  printf '# comment\n\nforbidden-word\n' >"$BATS_TEST_TMPDIR/denylist"
  export GELLEE_DENYLIST="$BATS_TEST_TMPDIR/denylist"
  export GELLEE_GITLEAKS_VERSION
  GELLEE_GITLEAKS_VERSION=$(gitleaks version)
}

# add <path> <content> [message]: commit one file, signed off.
add() {
  mkdir -p "$(dirname "$1")"
  printf '%s\n' "$2" >"$1"
  git add "$1"
  git commit -q -s -m "${3:-add $1}"
}

# A GitHub token shape, assembled at run time so no file holds one.
token() {
  printf 'gh%s_%s' p "$(head -c 96 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 36)"
}

refute_leak() {
  [[ "$output" != *forbidden* ]]
}

@test "a clean history passes and says what it covered" {
  add a.txt 'fine'
  run "$GATE" HEAD
  [ "$status" -eq 0 ]
  [ "$output" = "public-gate: 2 commits, 1 patterns, 2 files" ]
}

@test "an exclude limits the commits but not the tree" {
  add a.txt 'fine'
  run "$GATE" HEAD HEAD~1
  [ "$status" -eq 0 ]
  [ "$output" = "public-gate: 1 commits, 1 patterns, 2 files" ]
}

@test "a commit from another root fails the root guard" {
  git checkout -q --orphan other
  add b.txt 'other root'
  run "$GATE" other
  [ "$status" -eq 1 ]
  [[ "$output" == *"root guard:"* ]]
}

@test "no recorded root fails" {
  unset GELLEE_PUBLIC_ROOT
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"root guard: no public root"* ]]
}

@test "the root comes from ci/public-root at the tip" {
  root=$(git rev-parse HEAD)
  unset GELLEE_PUBLIC_ROOT
  add ci/public-root "$root"
  run "$GATE" HEAD
  [ "$status" -eq 0 ]
}

@test "a commit without a sign-off fails the metadata check" {
  printf 'x\n' >a.txt && git add a.txt && git commit -q -m 'no sign-off'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"metadata: commit $(git rev-parse HEAD) has no Signed-off-by matching its author"* ]]
}

@test "a sign-off by someone else fails" {
  printf 'x\n' >a.txt && git add a.txt
  git commit -q -m 'x' --trailer 'Signed-off-by: Other <other@example.invalid>'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"metadata:"* ]]
}

@test "a maintainer commit outside UTC fails" {
  git config user.email 1679489+aimd54@users.noreply.github.com
  printf 'x\n' >a.txt && git add a.txt
  GIT_AUTHOR_DATE='2026-01-02T12:00:00+0100' git commit -q -s -m 'x'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"metadata: commit $(git rev-parse HEAD) has an author date outside UTC"* ]]
}

@test "a maintainer committing outside UTC fails" {
  printf 'x\n' >a.txt && git add a.txt
  GIT_COMMITTER_NAME=Maintainer GIT_COMMITTER_EMAIL=1679489+aimd54@users.noreply.github.com \
    GIT_COMMITTER_DATE='2026-01-02T12:00:00+0100' git commit -q -s -m 'x'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"metadata: commit $(git rev-parse HEAD) has a committer date outside UTC"* ]]
}

@test "another contributor's offset is theirs" {
  printf 'x\n' >a.txt && git add a.txt
  GIT_AUTHOR_DATE='2026-01-02T12:00:00+0100' GIT_COMMITTER_DATE='2026-01-02T12:00:00+0100' \
    git commit -q -s -m 'x'
  run "$GATE" HEAD
  [ "$status" -eq 0 ]
}

@test "an unknown committer fails" {
  printf 'x\n' >a.txt && git add a.txt
  GIT_COMMITTER_NAME=Stranger GIT_COMMITTER_EMAIL=stranger@example.invalid git commit -q -s -m 'x'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"metadata: commit $(git rev-parse HEAD) has a committer who is neither its author, a maintainer nor GitHub"* ]]
}

@test "GitHub as committer is accepted" {
  printf 'x\n' >a.txt && git add a.txt
  GIT_COMMITTER_NAME=GitHub GIT_COMMITTER_EMAIL=noreply@github.com \
    git commit -q -m 'x' --trailer 'Signed-off-by: Test <test@example.invalid>'
  run "$GATE" HEAD
  [ "$status" -eq 0 ]
}

@test "a pattern in an added line fails without printing it" {
  add a.txt 'some Forbidden-Word here'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: added line in a.txt at commit $(git rev-parse HEAD) matches pattern on line 3"* ]]
  refute_leak
}

@test "a pattern in a commit message fails" {
  add a.txt 'fine' 'mention forbidden-word'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: message or identity of commit $(git rev-parse HEAD) matches pattern on line 3"* ]]
  refute_leak
}

@test "a pattern in an identity fails" {
  git config user.name 'forbidden-word'
  add a.txt 'fine'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: message or identity of commit"* ]]
  refute_leak
}

@test "a pattern in the tree fails even outside the range" {
  add a.txt 'forbidden-word'
  add b.txt 'fine'
  run "$GATE" HEAD HEAD~1
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: tree file a.txt at $(git rev-parse HEAD) matches pattern on line 3"* ]]
  refute_leak
}

@test "a path holding a pattern is withheld" {
  add 'docs/forbidden-word.md' 'fine'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"a withheld path"* ]]
  refute_leak
}

@test "a pattern in a symbolic link target fails" {
  ln -s /somewhere/forbidden-word link && git add link && git commit -q -s -m 'link'
  add b.txt 'fine'
  run "$GATE" HEAD HEAD~1
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: a path at $(git rev-parse HEAD) matches pattern on line 3"* ]]
  refute_leak
}

@test "a pattern in a ref name fails" {
  run "$GATE" --ref refs/heads/forbidden-word HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: ref name matches pattern on line 3"* ]]
  refute_leak
}

@test "a pattern in an annotated tag message fails" {
  git tag -a v0.0.1 -m 'forbidden-word'
  run "$GATE" refs/tags/v0.0.1
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: tag message matches pattern on line 3"* ]]
  refute_leak
}

@test "--all checks every ref" {
  git switch -q -c side
  add a.txt 'forbidden-word'
  git switch -q main
  run "$GATE" --all
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list:"* ]]
}

@test "--all works on a bare mirror through GIT_DIR" {
  git clone -q --mirror "$repo" "$BATS_TEST_TMPDIR/mirror.git"
  cd "$BATS_TEST_TMPDIR"
  export GIT_DIR="$BATS_TEST_TMPDIR/mirror.git"
  run "$GATE" --all
  [ "$status" -eq 0 ]
  [ "$output" = "public-gate: 1 commits, 1 patterns, 1 files" ]
}

@test "a missing deny-list fails" {
  export GELLEE_DENYLIST="$BATS_TEST_TMPDIR/absent"
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: cannot read the file"* ]]
}

@test "no deny-list named fails" {
  unset GELLEE_DENYLIST
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: no file named"* ]]
}

@test "a deny-list with only comments fails" {
  printf '# only\n\n' >"$GELLEE_DENYLIST"
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: no patterns"* ]]
}

@test "a pattern that matches the empty string fails" {
  printf 'x*\n' >"$GELLEE_DENYLIST"
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: line 1 matches the empty string"* ]]
}

@test "an invalid pattern fails" {
  printf 'a(\n' >"$GELLEE_DENYLIST"
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: line 1 is not a valid expression"* ]]
}

@test "gitleaks at another version fails" {
  export GELLEE_GITLEAKS_VERSION=0.0.1
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: version"* ]]
}

@test "missing gitleaks fails" {
  mkdir "$BATS_TEST_TMPDIR/bin"
  for t in bash env git grep sed awk tar jq mktemp rm mkdir find sort cut wc tr head cat dirname basename; do
    p=$(command -v "$t") && ln -s "$p" "$BATS_TEST_TMPDIR/bin/$t"
  done
  PATH="$BATS_TEST_TMPDIR/bin" run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: not installed"* ]]
}

@test "the gitleaks version defaults to the pin in mise.toml" {
  unset GELLEE_GITLEAKS_VERSION
  add mise.toml '"aqua:gitleaks/gitleaks" = "0.0.1"'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: version"*"want 0.0.1"* ]]
}

@test "a secret in history fails without printing it" {
  t=$(token)
  add config.txt "key = $t"
  git rm -q config.txt && git commit -q -s -m 'remove'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: rule "* ]]
  [[ "$output" != *"$t"* ]]
}

@test "a secret in a range limited by a remote-tracking exclude fails" {
  git update-ref refs/remotes/github/main HEAD
  t=$(token)
  add config.txt "key = $t"
  git rm -q config.txt && git commit -q -s -m 'remove'
  run "$GATE" HEAD --remotes=github
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: rule "* ]]
  [[ "$output" != *"$t"* ]]
}

@test "a secret is found through a bare mirror with --all" {
  t=$(token)
  add config.txt "key = $t"
  git rm -q config.txt && git commit -q -s -m 'remove'
  git clone -q --mirror "$repo" "$BATS_TEST_TMPDIR/mirror.git"
  cd "$BATS_TEST_TMPDIR"
  export GIT_DIR="$BATS_TEST_TMPDIR/mirror.git"
  run "$GATE" --all
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: rule "* ]]
  [[ "$output" != *"$t"* ]]
}

@test "a secret introduced inside a merge commit fails" {
  git switch -q -c side
  add side.txt 'side'
  git switch -q main
  add main.txt 'main'
  git merge -q --no-ff --no-commit side
  t=$(token)
  printf 'key = %s\n' "$t" >config.txt
  git add config.txt
  git commit -q -s -m 'merge'
  git rm -q config.txt && git commit -q -s -m 'remove'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: rule "* ]]
  [[ "$output" != *"$t"* ]]
}

@test "a gitleaksignore file in the working directory does not hide a finding" {
  t=$(token)
  add config.txt "key = $t"
  git rm -q config.txt && git commit -q -s -m 'remove'
  gitleaks git --no-banner --redact --exit-code 0 --report-format json \
    --report-path "$BATS_TEST_TMPDIR/r.json" . >/dev/null 2>&1
  jq -r '.[].Fingerprint' "$BATS_TEST_TMPDIR/r.json" >"$BATS_TEST_TMPDIR/ignore"
  [ -s "$BATS_TEST_TMPDIR/ignore" ]
  command cp -f "$BATS_TEST_TMPDIR/ignore" .gitleaksignore
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: rule "* ]]
}

@test "a gitleaks configuration in the environment does not hide a finding" {
  t=$(token)
  add config.txt "key = $t"
  git rm -q config.txt && git commit -q -s -m 'remove'
  printf '[extend]\nuseDefault = true\n[[allowlists]]\npaths = [".*"]\n' >"$BATS_TEST_TMPDIR/cfg.toml"
  export GITLEAKS_CONFIG="$BATS_TEST_TMPDIR/cfg.toml"
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: rule "* ]]
}

@test "inline gitleaks configuration in the environment does not hide a finding" {
  t=$(token)
  add config.txt "key = $t"
  git rm -q config.txt && git commit -q -s -m 'remove'
  export GITLEAKS_CONFIG_TOML=$'[extend]\nuseDefault = true\n[[allowlists]]\npaths = [".*"]\n'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: rule "* ]]
}

@test "a gitleaks configuration in the git directory does not hide a finding" {
  t=$(token)
  add config.txt "key = $t"
  git rm -q config.txt && git commit -q -s -m 'remove'
  printf '[extend]\nuseDefault = true\n[[allowlists]]\npaths = [".*"]\n' >.git/.gitleaks.toml
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: rule "* ]]
}

@test "a gitleaksignore file in the git directory fails" {
  t=$(token)
  add config.txt "key = $t"
  git rm -q config.txt && git commit -q -s -m 'remove'
  gitleaks git --no-banner --redact --exit-code 0 --report-format json \
    --report-path "$BATS_TEST_TMPDIR/r.json" . >/dev/null 2>&1
  jq -r '.[].Fingerprint' "$BATS_TEST_TMPDIR/r.json" >.git/.gitleaksignore
  [ -s .git/.gitleaksignore ]
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: the git directory holds .gitleaksignore"* ]]
}

@test "a gitleaks configuration in the tree fails" {
  add .gitleaks.toml '[allowlist]'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: the tree holds .gitleaks.toml"* ]]
}

@test "an inline gitleaks allow marker fails" {
  add a.txt "x # gitleaks$(printf ':')allow"
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: an allow marker in a.txt"* ]]
}

@test "an export-ignore attribute fails" {
  add .gitattributes 'secret.txt export-ignore'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"tree: .gitattributes changes what an export holds"* ]]
}

@test "a revision that does not resolve fails" {
  run "$GATE" no-such-branch
  [ "$status" -eq 1 ]
  [[ "$output" == *"revision no-such-branch does not resolve"* ]]
}

@test "an empty tree fails" {
  git checkout -q --orphan empty
  git rm -q -r --cached . && command rm -f README
  git commit -q -s --allow-empty -m empty
  GELLEE_PUBLIC_ROOT=$(git rev-parse HEAD)
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"tree: no files at"* ]]
}

@test "--without-denylist runs the other checks, then fails" {
  unset GELLEE_DENYLIST
  run "$GATE" --without-denylist HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"public-gate: 1 commits, 0 patterns, 1 files"* ]]
  [[ "$output" == *"deny-list unavailable"* ]]
}

@test "an added line starting with plus signs is deny-listed" {
  add a.txt '++ forbidden-word'
  add a.txt 'fine'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: added line in a.txt at commit $(git rev-parse HEAD~1) matches pattern on line 3"* ]]
  refute_leak
}

@test "an empty file whose name holds a pattern is caught after removal" {
  : >forbidden-word
  git add forbidden-word && git commit -q -s -m 'empty file'
  bad=$(git rev-parse HEAD)
  git rm -q forbidden-word && git commit -q -s -m 'remove it'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: path in commit $bad matches pattern on line 3"* ]]
  refute_leak
}

@test "a copied file's new name is caught whatever the diff configuration" {
  add a.txt 'some content long enough to be detected as a copy'
  git config diff.renames copies
  cp a.txt forbidden-word && printf 'changed\n' >>a.txt
  git add a.txt forbidden-word && git commit -q -s -m 'copy'
  git show --format= HEAD | grep -q '^copy to '
  bad=$(git rev-parse HEAD)
  git rm -q forbidden-word && git commit -q -s -m 'remove it'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: path in commit $bad matches pattern on line 3"* ]]
  refute_leak
}

@test "a deny-list with CRLF line endings still matches" {
  printf '# comment\r\n\r\nforbidden-word\r\n' >"$BATS_TEST_TMPDIR/denylist"
  add a.txt 'some forbidden-word here'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: added line in a.txt at commit $(git rev-parse HEAD) matches pattern on line 3"* ]]
  refute_leak
}

@test "a maintainer's annotated tag outside UTC fails" {
  GIT_COMMITTER_NAME=Maintainer GIT_COMMITTER_EMAIL=1679489+aimd54@users.noreply.github.com \
    GIT_COMMITTER_DATE='2026-01-02T12:00:00+0100' git tag -a v0.0.1 -m 'release'
  run "$GATE" refs/tags/v0.0.1
  [ "$status" -eq 1 ]
  [[ "$output" == *"metadata: tag $(git rev-parse v0.0.1) has a tagger date outside UTC"* ]]
}

@test "a maintainer's annotated tag in UTC passes" {
  GIT_COMMITTER_NAME=Maintainer GIT_COMMITTER_EMAIL=1679489+aimd54@users.noreply.github.com \
    GIT_COMMITTER_DATE='2026-01-02T12:00:00+0000' git tag -a v0.0.1 -m 'release'
  run "$GATE" refs/tags/v0.0.1
  [ "$status" -eq 0 ]
}

@test "another contributor's tag offset is theirs" {
  GIT_COMMITTER_DATE='2026-01-02T12:00:00+0100' git tag -a v0.0.1 -m 'release'
  run "$GATE" refs/tags/v0.0.1
  [ "$status" -eq 0 ]
}

@test "a byte order mark does not disable the first deny-list line" {
  printf '\xef\xbb\xbfforbidden-word\n' >"$BATS_TEST_TMPDIR/denylist"
  add a.txt 'some forbidden-word here'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: added line in a.txt at commit $(git rev-parse HEAD) matches pattern on line 1"* ]]
  refute_leak
}

@test "a deny-list line ending in several carriage returns still matches" {
  printf 'forbidden-word\r\r\n' >"$BATS_TEST_TMPDIR/denylist"
  add a.txt 'some forbidden-word here'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: added line in a.txt at commit $(git rev-parse HEAD) matches pattern on line 1"* ]]
  refute_leak
}

@test "a run from a subdirectory checks the whole tree" {
  add top.txt 'some forbidden-word here'
  add sub/a.txt 'fine'
  cd sub
  run "$GATE" HEAD HEAD~1
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: tree file top.txt at $(git rev-parse HEAD) matches pattern on line 3"* ]]
  [[ "$output" == *"1 commits, 1 patterns, 3 files"* ]]
  refute_leak
}

@test "diff.relative in a subdirectory does not hide an added line" {
  mkdir sub
  add top.txt 'some forbidden-word here'
  bad=$(git rev-parse HEAD)
  git rm -q top.txt && git commit -q -s -m 'remove it'
  add sub/a.txt 'fine'
  git config diff.relative true
  cd sub
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: added line in top.txt at commit $bad matches pattern on line 3"* ]]
  refute_leak
}

@test "diff.noprefix does not change the reported path" {
  git config diff.noprefix true
  add a.txt 'some forbidden-word here'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: added line in a.txt at commit $(git rev-parse HEAD) matches pattern on line 3"* ]]
  refute_leak
}

@test "a textconv filter does not hide an added line" {
  printf '*.txt diff=hide\n' >.git/info/attributes
  git config diff.hide.textconv 'true'
  add a.txt 'some forbidden-word here'
  bad=$(git rev-parse HEAD)
  git rm -q a.txt && git commit -q -s -m 'remove it'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: added line in a.txt at commit $bad matches pattern on line 3"* ]]
  refute_leak
}

@test "a maintainer address in another case is still a maintainer" {
  git config user.email 1679489+AIMD54@users.noreply.github.com
  printf 'x\n' >a.txt && git add a.txt
  GIT_AUTHOR_DATE='2026-01-02T12:00:00+0100' git commit -q -s -m 'x'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"metadata: commit $(git rev-parse HEAD) has an author date outside UTC"* ]]
}

@test "color.ui=always does not hide a secret from the history scan" {
  git config color.ui always
  t=$(token)
  add config.txt "key = $t"
  git rm -q config.txt && git commit -q -s -m 'remove'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: rule "*" in config.txt at commit "* ]]
  [[ "$output" != *"$t"* ]]
}

@test "a textconv filter does not hide a secret from the history scan" {
  printf '*.txt diff=hide\n' >.git/info/attributes
  git config diff.hide.textconv 'true'
  t=$(token)
  add config.txt "key = $t"
  git rm -q config.txt && git commit -q -s -m 'remove'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: rule "*" in config.txt at commit "* ]]
  [[ "$output" != *"$t"* ]]
}

@test "a replace ref does not hide a commit" {
  add a.txt 'some forbidden-word here'
  bad=$(git rev-parse HEAD)
  git checkout -q HEAD~1 && add a.txt 'fine' && clean=$(git rev-parse HEAD)
  git checkout -q main && git replace "$bad" "$clean"
  run "$GATE" main
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: added line in a.txt at commit $bad matches pattern on line 3"* ]]
  [[ "$output" == *"deny-list: tree file a.txt at $bad matches pattern on line 3"* ]]
  refute_leak
}

@test "export-ignore in info/attributes does not hide a tree file" {
  add a.txt 'some forbidden-word here'
  add b.txt 'fine'
  printf 'a.txt export-ignore\n' >.git/info/attributes
  run "$GATE" HEAD HEAD~1
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: tree file a.txt at $(git rev-parse HEAD) matches pattern on line 3"* ]]
  refute_leak
}

@test "a smudge filter does not blank a tree file" {
  add a.txt 'some forbidden-word here'
  add b.txt 'fine'
  printf '*.txt filter=blank\n' >.git/info/attributes
  git config filter.blank.smudge 'cat >/dev/null'
  run "$GATE" HEAD HEAD~1
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: tree file a.txt at $(git rev-parse HEAD) matches pattern on line 3"* ]]
  refute_leak
}

@test "an export-ignore in .gitattributes is found from a subdirectory" {
  add .gitattributes 'secret.txt export-ignore'
  add sub/a.txt 'fine'
  cd sub
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"tree: .gitattributes changes what an export holds"* ]]
}

@test "a relative gellee.denylist fails" {
  unset GELLEE_DENYLIST
  command cp -f "$BATS_TEST_TMPDIR/denylist" deny.list
  git config gellee.denylist deny.list
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: gellee.denylist is not an absolute path"* ]]
}

@test "a UTF-16 deny-list fails" {
  printf '\xff\xfef\x00o\x00o\x00\n\x00' >"$BATS_TEST_TMPDIR/denylist"
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: the file is UTF-16, not UTF-8"* ]]
}

@test "a deny-list holding a NUL byte fails" {
  printf 'forbidden\x00-word\n' >"$BATS_TEST_TMPDIR/denylist"
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: the file holds a NUL byte"* ]]
}

@test "a deny-list with carriage returns alone as line ends fails" {
  printf 'x\rforbidden-word\r' >"$BATS_TEST_TMPDIR/denylist"
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: line 1 holds a control character"* ]]
}

@test "diff.relative does not hide a secret from the history scan" {
  git config diff.relative true
  export GIT_DIR="$repo/.git" GIT_WORK_TREE="$repo"
  t=$(token)
  add config.txt "key = $t"
  git rm -q config.txt && git commit -q -s -m 'remove'
  run "$GATE" HEAD
  [ "$status" -eq 1 ]
  [[ "$output" == *"gitleaks: rule "*" in config.txt at commit "* ]]
  [[ "$output" != *"$t"* ]]
}

# crafted <tree>: commit <tree> twice on HEAD, signed off, and print the
# second commit; a range that excludes its parent adds no line.
crafted() {
  local msg first
  msg=$(printf 'crafted\n\nSigned-off-by: Test <test@example.invalid>')
  first=$(git commit-tree "$1" -p HEAD -m "$msg")
  git commit-tree "$1" -p "$first" -m "$msg"
}

@test "a tree entry named .. is not written outside the scan" {
  blob=$(printf 'forbidden-word\n' | git hash-object -w --stdin)
  sub=$(printf '100644 blob %s\tleak.txt\n' "$blob" | git mktree)
  top=$(printf '100644 blob %s\tREADME\n040000 tree %s\t..\n' "$(git rev-parse HEAD:README)" "$sub" | git mktree)
  c=$(crafted "$top")
  first=$(git rev-parse "$c~1")
  run "$GATE" "$c" "$first"
  [ "$status" -eq 1 ]
  [[ "$output" == *"tree: cannot export $c"* ]]
  refute_leak
}

@test "a symbolic link and a directory of the same name do not write outside the scan" {
  mkdir "$BATS_TEST_TMPDIR/victim"
  blob=$(printf 'forbidden-word\n' | git hash-object -w --stdin)
  sub=$(printf '100644 blob %s\tleak.txt\n' "$blob" | git mktree)
  target=$(printf '%s' "$BATS_TEST_TMPDIR/victim" | git hash-object -w --stdin)
  top=$(printf '100644 blob %s\tREADME\n120000 blob %s\td\n040000 tree %s\td\n' \
    "$(git rev-parse HEAD:README)" "$target" "$sub" | git mktree)
  c=$(crafted "$top")
  first=$(git rev-parse "$c~1")
  run "$GATE" "$c" "$first"
  [ "$status" -eq 1 ]
  # Depending on the engine, tar replaces the link or refuses the member.
  [[ "$output" == *"deny-list: tree file d/leak.txt at $c matches pattern on line 3"* ||
    "$output" == *"tree: cannot export $c"* ]]
  [ -z "$(ls -A "$BATS_TEST_TMPDIR/victim")" ]
  refute_leak
}

@test "a crafted path in a failed export is not printed" {
  blob=$(printf 'fine\n' | git hash-object -w --stdin)
  sub=$(printf '100644 blob %s\tforbidden-word.txt\n' "$blob" | git mktree)
  top=$(printf '100644 blob %s\tREADME\n040000 tree %s\t..\n' "$(git rev-parse HEAD:README)" "$sub" | git mktree)
  c=$(crafted "$top")
  first=$(git rev-parse "$c~1")
  run "$GATE" "$c" "$first"
  [ "$status" -eq 1 ]
  [[ "$output" == *"tree: cannot export $c"* ]]
  refute_leak
}

@test "TAR_OPTIONS does not drop a tree file" {
  add a.txt 'some forbidden-word here'
  add b.txt 'fine'
  export TAR_OPTIONS=--exclude=a.txt
  run "$GATE" HEAD HEAD~1
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: tree file a.txt at $(git rev-parse HEAD) matches pattern on line 3"* ]]
  refute_leak
}

@test "GIT_COMMON_DIR does not bring info/attributes into the export" {
  add a.txt 'some forbidden-word here'
  add b.txt 'fine'
  printf 'a.txt export-ignore\n' >.git/info/attributes
  export GIT_COMMON_DIR="$repo/.git"
  run "$GATE" HEAD HEAD~1
  [ "$status" -eq 1 ]
  [[ "$output" == *"deny-list: tree file a.txt at $(git rev-parse HEAD) matches pattern on line 3"* ]]
  refute_leak
}

@test "an eol attribute in the tree is caught by the export check" {
  printf 'forbidden-word$\n' >"$BATS_TEST_TMPDIR/denylist"
  add a.txt 'forbidden-word'
  add .gitattributes 'a.txt text eol=crlf'
  run "$GATE" HEAD HEAD~1
  [ "$status" -eq 1 ]
  [[ "$output" == *"tree: the export of $(git rev-parse HEAD) differs from its objects"* ]]
  refute_leak
}

@test "a working-tree-encoding attribute in the tree is caught by the export check" {
  add b.txt 'private.example'
  add .gitattributes 'b.txt working-tree-encoding=UTF-16'
  run "$GATE" HEAD HEAD~1
  [ "$status" -eq 1 ]
  [[ "$output" == *"tree: the export of $(git rev-parse HEAD) differs from its objects"* ]]
}

@test "a file and a symbolic link of the same name fail" {
  blob=$(printf 'forbidden-word\n' | git hash-object -w --stdin)
  printf 'forbidden-word\n' >"$BATS_TEST_TMPDIR/outside"
  target=$(printf '%s' "$BATS_TEST_TMPDIR/outside" | git hash-object -w --stdin)
  top=$(printf '100644 blob %s\tREADME\n100644 blob %s\ta\n120000 blob %s\ta\n' \
    "$(git rev-parse HEAD:README)" "$blob" "$target" | git mktree)
  c=$(crafted "$top")
  first=$(git rev-parse "$c~1")
  run "$GATE" "$c" "$first"
  [ "$status" -eq 1 ]
  [[ "$output" == *"tree: 1 paths appear more than once at $c"* ]]
  refute_leak
}

# rawtree <mode> <name> <oid> ...: write a tree object with the entries in
# the given order, which git mktree would sort.
rawtree() {
  while [ $# -gt 0 ]; do
    printf '%s %s\0' "$1" "$2"
    printf "$(printf '%s' "$3" | sed 's/../\\x&/g')"
    shift 3
  done | git hash-object -t tree --literally -w --stdin
}

@test "a path listed twice in an unsorted tree fails in a UTF-8 locale" {
  locale -a 2>/dev/null | grep -qix 'en_US.utf-\?8' || skip "en_US.UTF-8 is not installed"
  blob=$(printf 'forbidden-word\n' | git hash-object -w --stdin)
  printf 'forbidden-word\n' >"$BATS_TEST_TMPDIR/outside"
  target=$(printf '%s' "$BATS_TEST_TMPDIR/outside" | git hash-object -w --stdin)
  top=$(rawtree 100644 README "$(git rev-parse HEAD:README)" \
    100644 $'a\xff' "$blob" 100644 $'a\xfe' "$blob" 120000 $'a\xff' "$target")
  c=$(crafted "$top")
  first=$(git rev-parse "$c~1")
  LC_ALL=en_US.UTF-8 run "$GATE" "$c" "$first"
  [ "$status" -eq 1 ]
  [[ "$output" == *"tree: 1 paths appear more than once at $c"* ]]
  refute_leak
}
