---
name: memories
description: Use when the user types /memories, or asks to review, merge, move or clean up Claude Code memories across projects. Works on the memory store that gellee carries between machines - conflicts set aside, orphans, rules repeated in several projects, stale entries.
---

# Memories

Review the memory store: a git checkout, named by `GELLEE_MEMORIES`,
holding every project's memory files and the global instructions file.
Changes are made in the checkout and reach every machine through
`gellee memories sync`.

## Arguments

| Invocation | Covers |
| --- | --- |
| `/memories` | Conflicts set aside, then orphans |
| `/memories review <project>` | One project, named by its key, such as `alpha/fern` |
| `/memories review all` | Every project, one after another, then the rules found in several |

## Rules

- Change only the checkout, by the batches of the steps below. Never
  write below `~/.claude/projects/` or to `~/.claude/CLAUDE.md`: the
  sync copies the checkout there.
- Apply nothing without the user's yes. Every proposal shows the path,
  the text before and the text after.
- A change to `global/CLAUDE.md` is shown as a diff and approved on its
  own, never inside a batch.
- Flag stale candidates; the user decides. Age alone never deletes a
  memory.
- Keep each project's `MEMORY.md` equal to its files: one line per
  memory, in the format the index already uses. Add the line of a memory
  moved in, remove the line of one removed.
- Memory text stays in the store. Never copy it into a project's
  repository, the state repository, or a request to an outside service.

## Steps

1. Run `gellee memories sync`.
   - If it says the checkout has uncommitted changes, an earlier review
     or sync was interrupted. Show `git -C "$GELLEE_MEMORIES" status -s`
     and the diff, and ask whether to commit or discard them. A new file
     under `orphans/` may be the only copy of a version: name each one
     and discard it only on an explicit yes. Commit with explicit paths,
     `git -C "$GELLEE_MEMORIES" add -- <paths>` then
     `git -C "$GELLEE_MEMORIES" commit -s -m "curate(<scope>): <what>"`;
     discard with `git -C "$GELLEE_MEMORIES" restore -- <paths>` for a
     changed file and by removing a new one. Then run
     `gellee memories sync` again.
   - If it says this machine has not joined the memory store, run
     `gellee memories adopt`, show the plan, and on a yes run
     `gellee memories adopt --apply`, then go on.
2. Run `gellee memories list --json` and read it. `memories` holds one
   entry per memory with `project`, `file`, `name`, `type`,
   `description`, `size` and `changed`; `conflicts` and `orphans` are
   paths relative to the checkout; `duplicates` maps a name to the
   projects that hold it. A version set aside may carry a numeric
   suffix, such as `one.md.2`, when an earlier version of the same file
   was still waiting.
3. Read `global/CLAUDE.md`, and every file a proposal concerns, before
   proposing.
4. Propose changes in batches of at most five.
5. Write the new text of every file the approved batch changes, with the
   `MEMORY.md` indexes it touches, into a temporary directory made with
   `mktemp -d`.
6. Apply and commit the batch in one command that holds the store's
   lock. It waits for a sync in progress, which would otherwise reset or
   commit a half-applied batch, and a sync that starts meanwhile steps
   aside:

   ```bash
   flock "$GELLEE_MEMORIES/.git/gellee/lock" sh -c '
     set -e
     cd "$GELLEE_MEMORIES"
     cp <temporary copies> <paths>
     git rm -q -- <paths removed>
     git add -- <paths>
     git commit -s -m "curate(<scope>): <what was done>"
   '
   ```

   Leave out `git rm` when the batch removes nothing. `<scope>` is the
   project key, `global` or `orphans`. Remove the temporary directory
   once the commit is made.
7. After the last batch, run `gellee memories sync`, report its summary,
   and say that sessions already open see the change after a restart.

## What a review proposes

### A conflict

`orphans/<machine>/conflicts/<root>/<name>/<file>` is a machine's
version of `<root>/<name>/<file>`, set aside because the store's copy
had changed the same lines. Propose one text merged from the two,
written to the project's file, and the removal of the version set
aside. For `orphans/<machine>/conflicts/global/CLAUDE.md` the file is
`global/CLAUDE.md`, and the change is shown as a diff and approved on
its own.

### An orphan

`orphans/<machine>/<dir>/` is a memory directory whose name matches no
project on that machine: a moved or deleted project, or a session
started above one. The directory name is the project's absolute path
with every `/` and `.` replaced by `-`. Find the project it belongs to
by the end of that name, then propose one of: merge into that project's
memories, move there, drop, or leave it to its machine.

`orphans/<machine>/CLAUDE.md` is a joining machine's global
instructions file, kept because it differed from the store's. Propose
one text merged from it and `global/CLAUDE.md`, written to
`global/CLAUDE.md`, shown as a diff and approved on its own, and then
the removal of the orphan.

### A rule present in several projects

From `duplicates`, and from `feedback` memories of different projects
that say the same thing in other words. Propose one wording in
`global/CLAUDE.md` first, as a diff, then the removal of every copy.

### A memory that restates the global instructions

Propose its deletion, quoting the line of `global/CLAUDE.md` it
repeats.

### Stale candidates

- `changed` more than three months ago.
- A memory naming a file, branch, command or flag that the project no
  longer has. Check in the project's working tree,
  `$GELLEE_PROJECTS_ROOT/<project>`, with `git ls-files`,
  `git branch -a` and a search.

List them with the reason for each, and delete only those the user
names.
