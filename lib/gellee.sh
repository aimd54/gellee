# shellcheck shell=bash disable=SC2034
# Copyright The gellee Authors
# SPDX-License-Identifier: Apache-2.0
#
# Shared rules: configuration, path mapping, frontmatter. Sourced by every
# script in bin/.

gellee_die() {
  printf 'gellee: %s\n' "$*" >&2
  exit 1
}

# Sets GELLEE_PROJECTS_ROOT, GELLEE_ROOTS, GELLEE_LIMIT, GELLEE_ACTIVE_DAYS.
gellee_load_config() {
  [ -n "${GELLEE_STATE:-}" ] || gellee_die "GELLEE_STATE is not set"
  [ -d "$GELLEE_STATE" ] ||
    gellee_die "state repository not found: $GELLEE_STATE"
  local cfg="$GELLEE_STATE/config" key value cfg_root=""
  [ -f "$cfg" ] || gellee_die "config not found: $cfg"
  GELLEE_ROOTS=""
  GELLEE_LIMIT=3
  GELLEE_ACTIVE_DAYS=10
  while IFS='=' read -r key value; do
    case "$key" in
    projects_root) cfg_root=$value ;;
    roots) GELLEE_ROOTS=$value ;;
    limit) GELLEE_LIMIT=$value ;;
    active_days) GELLEE_ACTIVE_DAYS=$value ;;
    esac
  done <"$cfg"
  [ -n "$GELLEE_ROOTS" ] || gellee_die "config lists no roots"
  GELLEE_PROJECTS_ROOT="${GELLEE_PROJECTS_ROOT:-$cfg_root}"
  [ -n "$GELLEE_PROJECTS_ROOT" ] || gellee_die "projects root is not set"
  [ -d "$GELLEE_PROJECTS_ROOT" ] ||
    gellee_die "projects root not found: $GELLEE_PROJECTS_ROOT"
  GELLEE_PROJECTS_ROOT=$(cd "$GELLEE_PROJECTS_ROOT" && pwd -P)
}

# gellee_project_key <dir>: prints <root>/<name>; returns 1 outside the
# configured roots.
gellee_project_key() {
  local dir rel root name r
  dir=$(cd "$1" 2>/dev/null && pwd -P) || return 1
  case "$dir" in
  "$GELLEE_PROJECTS_ROOT"/*/*) ;;
  *) return 1 ;;
  esac
  rel=${dir#"$GELLEE_PROJECTS_ROOT"/}
  root=${rel%%/*}
  rel=${rel#*/}
  name=${rel%%/*}
  for r in $GELLEE_ROOTS; do
    if [ "$r" = "$root" ]; then
      printf '%s/%s\n' "$root" "$name"
      return 0
    fi
  done
  return 1
}

# gellee_last_commit <dir>: committer date of the last commit, or nothing.
gellee_last_commit() {
  if [ -e "$1/.git" ]; then
    git -C "$1" log -1 --format=%cs 2>/dev/null || true
  fi
}

# gellee_fm <file> <field>: value of a frontmatter field, or nothing.
gellee_fm() {
  awk -v f="$2" '
    NR == 1 { if ($0 != "---") exit; next }
    $0 == "---" { exit }
    index($0, f ": ") == 1 { print substr($0, length(f) + 3); exit }
  ' "$1"
}

# gellee_next_action <file>: first paragraph under "## Next action", its
# wrapped lines joined into one.
gellee_next_action() {
  awk '
    /^## / { if (insec) exit; insec = ($0 == "## Next action"); next }
    !insec { next }
    !NF { if (out != "") exit; next }
    { $1 = $1; out = (out == "" ? $0 : out " " $0) }
    END { if (out != "") print out }
  ' "$1"
}

# gellee_valid <file>: 0 when the required frontmatter is present.
gellee_valid() {
  local state
  state=$(gellee_fm "$1" state)
  [ -n "$(gellee_fm "$1" project)" ] || return 1
  [ -n "$(gellee_fm "$1" updated)" ] || return 1
  case "$state" in
  active | parked | done) return 0 ;;
  *) return 1 ;;
  esac
}

# gellee_status_files: every status file, sorted.
gellee_status_files() {
  find "$GELLEE_STATE" -mindepth 2 -name '*.md' \
    -not -path "$GELLEE_STATE/archive/*" \
    -not -path "$GELLEE_STATE/private/*" \
    -not -path "$GELLEE_STATE/.git/*" | LC_ALL=C sort
}
