#!/usr/bin/env bats

load helper

setup() {
  common_setup
  FERN="$GELLEE_PROJECTS_ROOT/alpha/fern"
}

@test "prints the status file of the project named on stdin" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review"
  cd "$BATS_TEST_TMPDIR"
  run ctx "$FERN"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "gellee status for alpha/fern ($GELLEE_STATE/alpha/fern.md):" ]
  [[ "$output" == *"state: active"* ]]
  [[ "$output" == *"Run the review"* ]]
  [[ "$output" != *"gellee warning"* ]]
}

@test "falls back to the current directory without usable stdin" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review"
  cd "$FERN"
  run gellee-session-start </dev/null
  [ "$status" -eq 0 ]
  run field .hookSpecificOutput.additionalContext "$output"
  [ "${lines[0]}" = "gellee status for alpha/fern ($GELLEE_STATE/alpha/fern.md):" ]
}

@test "prints nothing outside the configured roots" {
  mkdir -p "$GELLEE_PROJECTS_ROOT/ignored/other"
  run hook "$GELLEE_PROJECTS_ROOT/ignored/other"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "suggests a handoff when no status file exists" {
  make_repo alpha/fern 2026-01-01
  run ctx "$FERN"
  [ "$status" -eq 0 ]
  [ "$output" = "gellee: no status file for alpha/fern. Run /handoff at the end of this session to create it." ]
}

@test "warns when the status is older than the last commit" {
  make_repo alpha/fern 2026-01-05
  write_status alpha/fern active 2026-01-02 "Run the review"
  run ctx "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee warning: status dated 2026-01-02 is older than the last commit (2026-01-05)."* ]]
}

@test "warns when the project is parked, with its reason" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern parked 2026-01-02 "Later" "" "waiting on a release"
  run ctx "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee warning: this project is parked: waiting on a release"* ]]
}

@test "warns when active projects exceed the limit, naming them" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "l"
  write_status alpha/apps active 2026-01-02 "a"
  write_status beta/zeta active 2026-01-02 "z"
  run ctx "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"gellee warning: 3 active projects, limit is 2: alpha/apps, alpha/fern, beta/zeta"* ]]
}

@test "does not warn about the limit when it is met exactly" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "l"
  write_status alpha/apps active 2026-01-02 "a"
  run ctx "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"state: active"* ]]
  [[ "$output" != *"active projects"* ]]
}

@test "prints a malformed file as is, with a warning" {
  make_repo alpha/fern 2026-01-01
  mkdir -p "$GELLEE_STATE/alpha"
  printf 'no frontmatter here\n' >"$GELLEE_STATE/alpha/fern.md"
  run ctx "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"no frontmatter here"* ]]
  [[ "$output" == *"gellee warning: malformed frontmatter in $GELLEE_STATE/alpha/fern.md"* ]]
}

@test "reports one line and exits 0 when GELLEE_STATE is unset" {
  unset GELLEE_STATE
  run ctx "$FERN"
  [ "$status" -eq 0 ]
  [ "$output" = "gellee: session start skipped: GELLEE_STATE is not set" ]
}

@test "reports one line and exits 0 when the config is missing" {
  rm "$GELLEE_STATE/config"
  run ctx "$FERN"
  [ "$status" -eq 0 ]
  [ "$output" = "gellee: session start skipped: config not found: $GELLEE_STATE/config" ]
}

@test "emits one JSON object for the SessionStart event" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review"
  run hook "$FERN"
  [ "$status" -eq 0 ]
  run field .hookSpecificOutput.hookEventName "$output"
  [ "$output" = "SessionStart" ]
}

@test "shows the user the state and the whole next action" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review
on the second cluster."
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [ "$output" = \
    "gellee: alpha/fern is active. Next: Run the review on the second cluster." ]
}

@test "shows the user a stub as having no handoff, with its warnings" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern parked 2026-01-02 "" "" "waiting on a release"
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [ "$output" = "gellee: alpha/fern is parked. No handoff yet.
warning: this project is parked: waiting on a release" ]
}

@test "shows the user the suggestion when no status file exists" {
  make_repo alpha/fern 2026-01-01
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [ "$output" = "gellee: no status file for alpha/fern. Run /handoff at the end of this session to create it." ]
}

@test "shows the user why the session start was skipped" {
  unset GELLEE_STATE
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [ "$output" = "gellee: session start skipped: GELLEE_STATE is not set" ]
}

@test "prints plain text when jq cannot build the JSON" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review"
  mkdir -p "$BATS_TEST_TMPDIR/nojq"
  printf '#!/bin/sh\nexit 1\n' >"$BATS_TEST_TMPDIR/nojq/jq"
  chmod +x "$BATS_TEST_TMPDIR/nojq/jq"
  cd "$FERN"
  PATH="$BATS_TEST_TMPDIR/nojq:$PATH" run gellee-session-start </dev/null
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "gellee status for alpha/fern ($GELLEE_STATE/alpha/fern.md):" ]
  [[ "$output" == *"Run the review"* ]]
}

@test "starts from the status file the state remote holds" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Old action"
  state_remote
  other="$BATS_TEST_TMPDIR/other"
  git clone -q "$BATS_TEST_TMPDIR/state.git" "$other"
  sed -i 's/Old action/New action/' "$other/alpha/fern.md"
  git -C "$other" -c user.name=test -c user.email=test@example.invalid \
    -c commit.gpgsign=false commit -q -am next
  git -C "$other" push -q
  run ctx "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"New action"* ]]
}

@test "says the status may be out of date when the state remote fails" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review"
  state_remote
  git -C "$GELLEE_STATE" remote set-url origin "$BATS_TEST_TMPDIR/missing.git"
  run notice "$FERN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"the status may be out of date"* ]]
}
