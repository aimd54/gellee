#!/usr/bin/env bats

load helper

setup() {
  common_setup
  mkdir -p "$GELLEE_PROJECTS_ROOT/alpha/fern/src/deep" \
    "$GELLEE_PROJECTS_ROOT/ignored/other"
}

@test "prints the status file path for a project directory" {
  run gellee-path "$GELLEE_PROJECTS_ROOT/alpha/fern"
  [ "$status" -eq 0 ]
  [ "$output" = "$GELLEE_STATE/alpha/fern.md" ]
}

@test "resolves a subdirectory to its project" {
  run gellee-path "$GELLEE_PROJECTS_ROOT/alpha/fern/src/deep"
  [ "$status" -eq 0 ]
  [ "$output" = "$GELLEE_STATE/alpha/fern.md" ]
}

@test "defaults to the current directory" {
  cd "$GELLEE_PROJECTS_ROOT/alpha/fern/src"
  run gellee-path
  [ "$status" -eq 0 ]
  [ "$output" = "$GELLEE_STATE/alpha/fern.md" ]
}

@test "refuses a directory under a root that is not configured" {
  run gellee-path "$GELLEE_PROJECTS_ROOT/ignored/other"
  [ "$status" -eq 1 ]
  [[ "$output" == "gellee: not inside a configured root:"* ]]
}

@test "refuses a scan root itself" {
  run gellee-path "$GELLEE_PROJECTS_ROOT/alpha"
  [ "$status" -eq 1 ]
  [[ "$output" == "gellee: not inside a configured root:"* ]]
}

@test "fails with a named reason when GELLEE_STATE is unset" {
  unset GELLEE_STATE
  run gellee-path "$GELLEE_PROJECTS_ROOT/alpha/fern"
  [ "$status" -eq 1 ]
  [ "$output" = "gellee: GELLEE_STATE is not set" ]
}

@test "reads the projects root from the config when the variable is unset" {
  printf 'projects_root=%s\nroots=alpha\n' "$GELLEE_PROJECTS_ROOT" \
    >"$GELLEE_STATE/config"
  local dir="$GELLEE_PROJECTS_ROOT/alpha/fern"
  unset GELLEE_PROJECTS_ROOT
  run gellee-path "$dir"
  [ "$status" -eq 0 ]
  [ "$output" = "$GELLEE_STATE/alpha/fern.md" ]
}

@test "the environment variable wins over the config" {
  printf 'projects_root=/nonexistent\nroots=alpha\n' >"$GELLEE_STATE/config"
  run gellee-path "$GELLEE_PROJECTS_ROOT/alpha/fern"
  [ "$status" -eq 0 ]
  [ "$output" = "$GELLEE_STATE/alpha/fern.md" ]
}

@test "fails when the config lists no roots" {
  printf 'limit=2\n' >"$GELLEE_STATE/config"
  run gellee-path "$GELLEE_PROJECTS_ROOT/alpha/fern"
  [ "$status" -eq 1 ]
  [ "$output" = "gellee: config lists no roots" ]
}
