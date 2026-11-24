#!/usr/bin/env bats
# Copyright The gellee Authors
# SPDX-License-Identifier: Apache-2.0

# ci/scripts/install-gitleaks and mise.toml pin gitleaks separately; these
# tests keep the version and the linux digests of the two equal.

setup() {
  REPO_ROOT=$(cd "$BATS_TEST_DIRNAME/.." && pwd)
  script="$REPO_ROOT/ci/scripts/install-gitleaks"
}

# lock_sum <platform>: the digest mise.lock records for gitleaks on it.
lock_sum() {
  awk -v want="[tools.\"aqua:gitleaks/gitleaks\".\"platforms.$1\"]" '
    $0 == want { on = 1; next }
    /^\[/ { on = 0 }
    on && /^checksum = / { gsub(/^checksum = "sha256:|"$/, ""); print; exit }
  ' "$REPO_ROOT/mise.lock"
}

# script_sum <arch>: the digest install-gitleaks pins for that arch.
script_sum() {
  sed -n "s/^  [a-z0-9_]*) arch=$1 sum=\([0-9a-f]*\) ;;\$/\1/p" "$script"
}

@test "install-gitleaks installs the version mise.toml pins" {
  want=$(sed -n 's/^"aqua:gitleaks\/gitleaks" = "\(.*\)"$/\1/p' "$REPO_ROOT/mise.toml")
  have=$(sed -n 's/^version=//p' "$script")
  [ -n "$want" ]
  [ "$want" = "$have" ]
}

@test "install-gitleaks pins the amd64 digest mise.lock records" {
  want=$(lock_sum linux-x64)
  [[ $want =~ ^[0-9a-f]{64}$ ]]
  [ "$want" = "$(script_sum x64)" ]
}

@test "install-gitleaks pins the arm64 digest mise.lock records" {
  want=$(lock_sum linux-arm64)
  [[ $want =~ ^[0-9a-f]{64}$ ]]
  [ "$want" = "$(script_sum arm64)" ]
}
