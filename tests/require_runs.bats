#!/usr/bin/env bats
# Copyright The gellee Authors
# SPDX-License-Identifier: Apache-2.0

setup() {
  REPO_ROOT=$(cd "$BATS_TEST_DIRNAME/.." && pwd)
  REQUIRE="$REPO_ROOT/ci/scripts/require-runs"
}

# run_json <workflow file> <run_number> <attempt> <status> <conclusion>
run_json() {
  printf '{"path":".github/workflows/%s","run_number":%s,"run_attempt":%s,"status":"%s","conclusion":%s}\n' \
    "$1" "$2" "$3" "$4" "$5"
}

@test "every named workflow has a successful run" {
  runs=$({ run_json ci.yml 3 1 completed '"success"'; run_json gate.yml 5 1 completed '"success"'; })
  run "$REQUIRE" ci.yml gate.yml <<<"$runs"
  [ "$status" -eq 0 ]
  [ "$output" = "2 workflows succeeded" ]
}

@test "a workflow with no run fails and is named" {
  runs=$(run_json ci.yml 3 1 completed '"success"')
  run "$REQUIRE" ci.yml docs.yml <<<"$runs"
  [ "$status" -eq 1 ]
  [ "$output" = "docs.yml has no run" ]
}

@test "a run still in progress fails" {
  runs=$(run_json gate.yml 5 1 in_progress null)
  run "$REQUIRE" gate.yml <<<"$runs"
  [ "$status" -eq 1 ]
  [ "$output" = "gate.yml: in_progress, not success" ]
}

@test "a failed run fails" {
  runs=$(run_json ci.yml 3 1 completed '"failure"')
  run "$REQUIRE" ci.yml <<<"$runs"
  [ "$status" -eq 1 ]
  [ "$output" = "ci.yml: completed failure, not success" ]
}

@test "the latest attempt decides" {
  runs=$({ run_json ci.yml 3 1 completed '"failure"'; run_json ci.yml 3 2 completed '"success"'; })
  run "$REQUIRE" ci.yml <<<"$runs"
  [ "$status" -eq 0 ]
  runs=$({ run_json ci.yml 3 1 completed '"success"'; run_json ci.yml 3 2 completed '"failure"'; })
  run "$REQUIRE" ci.yml <<<"$runs"
  [ "$status" -eq 1 ]
}

@test "the newer run number decides" {
  runs=$({ run_json ci.yml 4 1 completed '"success"'; run_json ci.yml 3 2 completed '"failure"'; })
  run "$REQUIRE" ci.yml <<<"$runs"
  [ "$status" -eq 0 ]
}

@test "a similarly named workflow file does not count" {
  runs=$(run_json myci.yml 3 1 completed '"success"')
  run "$REQUIRE" ci.yml <<<"$runs"
  [ "$status" -eq 1 ]
}
