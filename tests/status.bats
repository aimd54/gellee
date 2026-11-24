#!/usr/bin/env bats

load helper

setup() {
  common_setup
}

# row <project>: the table row for one project, columns joined by "|".
row() {
  gellee-status | awk -v p="$1" '$2 == p' | sed -E 's/  +/|/g'
}

@test "prints a header and one row per project" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review" fieldwork
  run gellee-status
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "${lines[0]}" | sed -E 's/  +/|/g')" = \
    "GROUP|PROJECT|STATE|UPDATED|NEXT ACTION" ]
  run row alpha/fern
  [ "$output" = "fieldwork|alpha/fern|active|2026-01-02|Run the review" ]
}

@test "a project without a group shows a dash" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review"
  run row alpha/fern
  [ "$output" = "-|alpha/fern|active|2026-01-02|Run the review" ]
}

@test "a stub is marked as having no handoff" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern parked 2026-01-02
  run row alpha/fern
  [ "$output" = "-|alpha/fern|parked|2026-01-02|(no handoff yet)" ]
}

@test "a status older than the last commit is marked stale" {
  make_repo alpha/fern 2026-01-05
  write_status alpha/fern active 2026-01-02 "Run the review"
  run row alpha/fern
  [ "$output" = "-|alpha/fern|active|2026-01-02 (stale)|Run the review" ]
}

@test "a status updated on the day of the last commit is not stale" {
  make_repo alpha/fern 2026-01-02
  write_status alpha/fern active 2026-01-02 "Run the review"
  run row alpha/fern
  [ "$output" = "-|alpha/fern|active|2026-01-02|Run the review" ]
}

@test "malformed frontmatter shows the state invalid" {
  make_repo alpha/fern 2026-01-01
  mkdir -p "$GELLEE_STATE/alpha"
  printf 'no frontmatter here\n' >"$GELLEE_STATE/alpha/fern.md"
  run row alpha/fern
  [ "$output" = "-|alpha/fern|invalid|-|(no handoff yet)" ]
}

@test "a removed project directory shows the state missing" {
  write_status alpha/gone active 2026-01-02 "Something"
  run row alpha/gone
  [ "$output" = "-|alpha/gone|missing|2026-01-02|Something" ]
}

@test "rows are sorted by group, then project" {
  write_status beta/zeta active 2026-01-02 "z" fieldwork
  write_status alpha/fern active 2026-01-02 "l" fieldwork
  write_status alpha/apps parked 2026-01-02 "a" cluster
  run bash -c "gellee-status | awk 'NR > 1 && \$2 ~ \"/\" { print \$2 }'"
  [ "$output" = "alpha/apps
alpha/fern
beta/zeta" ]
}

@test "the footer counts active projects against the limit" {
  write_status alpha/fern active 2026-01-02 "l"
  write_status alpha/apps active 2026-01-02 "a"
  write_status beta/zeta active 2026-01-02 "z"
  write_status beta/old parked 2026-01-02 "o"
  run gellee-status
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "active: 3 (limit 2)" ]
}

@test "archives and top-level files are not listed" {
  write_status alpha/fern active 2026-01-02 "l"
  mkdir -p "$GELLEE_STATE/archive/alpha"
  printf 'old history\n' >"$GELLEE_STATE/archive/alpha/fern-history.md"
  printf 'readme\n' >"$GELLEE_STATE/README.md"
  run bash -c "gellee-status | awk 'NR > 1 && \$2 ~ \"/\" { print \$2 }'"
  [ "$output" = "alpha/fern" ]
}

@test "a next action wrapped over several lines is printed whole" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review on
the second cluster,  then
report."
  run row alpha/fern
  [ "$output" = \
    "-|alpha/fern|active|2026-01-02|Run the review on the second cluster, then report." ]
}

@test "only the first paragraph of the next action is printed" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review.

A note that follows."
  run row alpha/fern
  [ "$output" = "-|alpha/fern|active|2026-01-02|Run the review." ]
}

@test "files under private/ are not status files" {
  make_repo alpha/fern 2026-01-01
  write_status alpha/fern active 2026-01-02 "Run the review"
  mkdir -p "$GELLEE_STATE/private/alpha/fern"
  printf 'notes\n' >"$GELLEE_STATE/private/alpha/fern/notes.private.md"
  run gellee-status
  [ "$status" -eq 0 ]
  [[ "$output" != *"private"* ]]
}
