#!/usr/bin/env bats

# The Go tests read the same table, so both parsers agree.

setup() {
  REPO_ROOT=$(cd "$BATS_TEST_DIRNAME/.." && pwd)
  # shellcheck source=../lib/gellee.sh
  . "$REPO_ROOT/lib/gellee.sh"
  FIX="$REPO_ROOT/tests/fixtures"
}

@test "the shared fixtures parse to the expected values" {
  local n=0 file valid state updated next got
  while IFS=$'\t' read -r file valid state updated next; do
    [ "$file" = file ] && continue
    n=$((n + 1))
    if gellee_valid "$FIX/state/$file"; then got=yes; else got=no; fi
    [ "$got" = "$valid" ] || { echo "$file: valid is $got"; return 1; }
    [ "$(gellee_fm "$FIX/state/$file" state)" = "$state" ] ||
      { echo "$file: state"; return 1; }
    [ "$(gellee_fm "$FIX/state/$file" updated)" = "$updated" ] ||
      { echo "$file: updated"; return 1; }
    [ "$(gellee_next_action "$FIX/state/$file")" = "$next" ] ||
      { echo "$file: next action"; return 1; }
  done <"$FIX/expected.tsv"
  [ "$n" -eq 6 ]
}
