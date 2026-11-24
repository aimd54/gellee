# 0006. Sync Claude Code memories through a git checkout

- Status: Accepted
- Date: 2026-11-23

## Context

Claude Code keeps what it learns about a project in memory files on the
machine where the session ran, and reads them when a session starts. A
person who works from several machines gets a different memory on each.
The global instructions file `~/.claude/CLAUDE.md` has the same
problem.

Memories are edited by Claude Code during sessions, by hand, and on
several machines between syncs. A scheme that loses an edit silently,
or deletes a memory because a directory looked empty, would be worse
than none.

## Decision

Memory directories stay plain directories that Claude Code reads and
writes as it always does. A private git repository, the memory store,
is checked out on every machine and named by `GELLEE_MEMORIES`.
`gellee memories sync` copies the memory directories into the checkout
and commits them. Git merges the changes each machine made since the
previous sync, and the command copies the merged result back and
pushes it.

A project's directory in the store is its project key, so machines need
neither the same home directory nor the same user name. Edits made
on two machines are reconciled by git's three-way merge. Different
memories are both kept and different lines of one memory are merged.
When both sides changed the same lines, the store's version stays and
the machine's version is set aside under `orphans/<machine>/conflicts/`.
No conflict stops a sync and none discards a memory. A missing or empty
memory directory is never read as a deletion.

A new machine joins with `gellee memories adopt`, which prints what it
would do, and with `--apply` takes the union per project. A skill,
`/memories`, reviews the set-aside versions and merges them.

## Consequences

Claude Code needs no change and no plugin, and the store is readable
and diffable as ordinary text. Merging is done by a tool that is
already trusted with the same job for code.

A conflicting edit is not applied until someone reviews it, so for a
while a session reads the store's version. Memory is read when a
session starts, so a sync during a session takes effect at the next
one.

The store holds memory text only. The collector reads two values from
it, the machine's last push time and the number of set-aside versions,
and never the memory text.
