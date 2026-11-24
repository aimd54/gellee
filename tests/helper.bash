# shellcheck shell=bash
# Fixture builders shared by every test file.

common_setup() {
  REPO_ROOT=$(cd "$BATS_TEST_DIRNAME/.." && pwd)
  PATH="$REPO_ROOT/bin:$PATH"
  export GELLEE_STATE="$BATS_TEST_TMPDIR/state"
  export GELLEE_PROJECTS_ROOT="$BATS_TEST_TMPDIR/projects"
  # A test never reaches a real memory store.
  unset GELLEE_MEMORIES
  mkdir -p "$GELLEE_STATE" \
    "$GELLEE_PROJECTS_ROOT/alpha" \
    "$GELLEE_PROJECTS_ROOT/beta" \
    "$GELLEE_PROJECTS_ROOT/ignored"
  printf 'roots=alpha beta\nlimit=2\nactive_days=10\n' >"$GELLEE_STATE/config"
}

# make_repo <root/name> <YYYY-MM-DD>: a git repository with one commit on
# that date.
make_repo() {
  local dir="$GELLEE_PROJECTS_ROOT/$1"
  mkdir -p "$dir"
  git -C "$dir" init -q
  GIT_AUTHOR_DATE="$2T12:00:00" GIT_COMMITTER_DATE="$2T12:00:00" \
    git -C "$dir" -c user.name=test -c user.email=test@example.invalid \
    -c commit.gpgsign=false commit -q --allow-empty -m init
}

days_ago() {
  date -d "$1 days ago" +%F
}

# write_status <key> <state> <updated> [next-action] [group] [parked-reason]
write_status() {
  local file="$GELLEE_STATE/$1.md"
  mkdir -p "$(dirname "$file")"
  {
    printf -- '---\nproject: %s\nstate: %s\n' "$1" "$2"
    if [ -n "${5:-}" ]; then
      printf 'group: %s\n' "$5"
    fi
    if [ -n "${6:-}" ]; then
      printf 'parked_reason: %s\n' "$6"
    fi
    printf 'updated: %s\n---\n\n' "$3"
    printf '## Where I stopped\n\nSomewhere.\n\n'
    printf '## Next action\n\n%s\n\n' "${4:-}"
    printf '## Remaining\n\n## Ideas\n'
  } >"$file"
}

# hook <cwd>: run the hook the way Claude Code does, JSON on stdin.
hook() {
  printf '{"cwd":"%s","hook_event_name":"SessionStart"}' "$1" |
    gellee-session-start
}

# field <jq-path> <json>: one string field of the hook's output; fails
# when the field is absent.
field() {
  jq -er "$1" <<<"$2"
}

# ctx <cwd>: the text the hook adds to the session context.
ctx() {
  local json
  json=$(hook "$1") || return
  field .hookSpecificOutput.additionalContext "$json"
}

# notice <cwd>: the text the hook shows to the user.
notice() {
  local json
  json=$(hook "$1") || return
  field .systemMessage "$json"
}

# state_remote: commits the state directory and makes it a clone of a
# bare remote, state.git, holding that commit.
state_remote() {
  local g=(git -C "$GELLEE_STATE" -c user.name=test
    -c user.email=test@example.invalid -c commit.gpgsign=false)
  "${g[@]}" init -q -b main
  "${g[@]}" add .
  "${g[@]}" commit -q -m init
  git clone -q --bare "$GELLEE_STATE" "$BATS_TEST_TMPDIR/state.git"
  "${g[@]}" remote add origin "$BATS_TEST_TMPDIR/state.git"
  "${g[@]}" fetch -q origin
  "${g[@]}" branch -q -u origin/main
}
