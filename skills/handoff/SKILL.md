---
name: handoff
description: Use at the end of a work session, or when the user types /handoff, to record where the project stands (where work stopped, the next action, what remains) in its status file, and to park, activate or close a project.
---

# Handoff

Rewrite the current project's status file so the next session, on any
machine, can resume without rereading anything.

## Arguments

| Invocation | Meaning |
| --- | --- |
| `/handoff` | Rewrite the status file, keep the state |
| `/handoff park <reason>` | Rewrite it, then set `state: parked` |
| `/handoff activate` | Set `state: active` |
| `/handoff done` | Rewrite it, then set `state: done` |

## Steps

1. Find the file: run `gellee-path`. If it exits non-zero, tell the user
   this directory is outside the configured roots and stop.
2. Read the existing file if there is one, with the file reading tool
   and in this session, even when its content is already in the context
   from the session start: a file that was not read is refused when it
   is written. Keep its `group`, `superseded_by` and `formerly` values
   unless the user asks to change them.
3. Collect the facts from the project itself: `git status -sb`, the
   current branch, `git log --oneline -5`, and what this session did.
   State unpushed and uncommitted work explicitly.
4. Write the whole file again in the format below. Do not append to the
   old content. Wait for the write to succeed before starting step 8:
   a commit sent together with a write that fails pushes the collector's
   files without the handoff.
5. For `activate`: run `gellee-status` and read its last line. If the
   count of active projects would exceed the limit, name the active
   projects to the user and ask which one to park, or whether to go over
   the limit. Going over is allowed.
6. For `park`: a reason is required. Ask for one if it is missing. Write
   it to `parked_reason`. Remove `parked_reason` on `activate`.
7. Show the user the file as written.
8. Collect the usage and git summaries, when the collector is installed:

   ```bash
   if command -v gellee >/dev/null; then gellee collect; fi
   ```

   It commits its own files in the state repository and nothing else.
   Report its one-line summary. If it exits non-zero, show its output
   and carry on with the handoff.
9. Exchange memories with the memory store, when it is set up on this
   machine:

   ```bash
   if command -v gellee >/dev/null && [ -n "${GELLEE_MEMORIES:-}" ]; then
     gellee memories sync
   fi
   ```

   This carries what the session taught to the other machines now
   rather than at the next timer run. Report its summary line. If it
   exits non-zero, show its output and carry on with the handoff: the
   next sync delivers what this one could not. The memory store is a
   repository of its own; never stage its files in the state
   repository.
10. Move this project's private files into the state repository, when
    the collector is installed:

    ```bash
    if command -v gellee >/dev/null; then
      gellee private sweep
      gellee private status --json
    fi
    ```

    Report the sweep's summary line. Keep the JSON for the last step.
    If either exits non-zero, show its output and carry on.
11. Commit in the state repository, staging only this file and the
    project's private directory:

    ```bash
    file=$(gellee-path)
    repo=$(git -C "$(dirname "$file")" rev-parse --show-toplevel)
    git -C "$repo" add -- "$file"
    if [ -d "$repo/private/<project>" ]; then
      git -C "$repo" add -f -- "private/<project>"
    fi
    git -C "$repo" commit -s -m "chore(<project>): handoff"
    ```

    Replace `<project>` with the `project` value.
12. Push the state repository, and only it. The push carries the handoff
    commit and the collector's commit. Skip this step when
    `git -C "$repo" remote` prints nothing, and report the repository as
    local only.

    ```bash
    git -C "$repo" push
    git -C "$repo" status -sb
    ```

    Run the push on its own line and read its exit status. The last
    command must show no `ahead` count. If the push is refused because
    the remote has commits from another machine, run
    `git -C "$repo" pull --rebase --autostash` and push again; never force. If it
    still fails, say so and leave the commit unpushed.
13. Report what went to the remote, and list any other uncommitted file
    in the state repository without touching it. Never push the project's
    own repository as part of a handoff. End with one line: "Ready to
    pick up elsewhere." when the checkout has nothing uncommitted, its
    branch has no commit missing from its upstream, `ready` was true and
    the memory sync succeeded; otherwise "Not ready to pick up
    elsewhere:" followed by what is missing (uncommitted work, unpushed
    commits, the private files listed under `unswept`, `in_the_way`,
    `missing`, `broken` or `not_ignored`, a failed memory sync). This
    line reports; it never stops the handoff.

## Format

```markdown
---
project: <root>/<name>
state: active
group: <optional, shared by related projects>
updated: <today, YYYY-MM-DD>
---

## Where I stopped

<The point reached. Branch names. Anything uncommitted or unpushed.>

## Next action

<Exactly one action, concrete enough to start cold.>

## Remaining

- <Short list. Link to the project's backlog or plan for detail.>

## Ideas

- <Things that came up and are deliberately not being done now.>
```

Frontmatter fields, in this order when present: `project`, `state`,
`group`, `parked_reason`, `superseded_by`, `formerly`, `updated`.

## Rules

- Sixty lines at most. Detail belongs in the project's own backlog or
  plan; link to it.
- Never shorten or merge an item to make the file fit. Move whole items
  out of *Remaining* instead, into the project's roadmap or backlog when
  it has one (`docs/roadmap.md` or similar), and name in the report what
  moved where. Cut *Ideas* last.
- When the session finished, started or reordered a piece of work the
  project's roadmap lists, update the roadmap in the project's
  repository as well. Leave that edit uncommitted and list it in the
  report: a handoff commits only in the state repository.
- *Next action* holds one action. A second one goes under *Remaining*.
- Keep entries under *Ideas* from the previous file unless the user drops
  them. Add any idea raised in this session that was set aside.
- `project` always equals the file's path under the state repository,
  without `.md`.
- Write facts read from the repository, not recollection. Check the branch
  and the commit before naming them.

## Moving history out of the memory index

Do this once per project, the first time `/handoff` runs there, and only
if the project's `MEMORY.md` index holds a running history: an entry that
lists completed tasks, dates and commit hashes and keeps growing.

1. Propose it to the user and show the exact text that would move.
2. On a yes, write that text unchanged to
   `archive/<root>/<name>-history-<today>.md` in the state repository.
3. Replace it in `MEMORY.md` with one line: the project's one-sentence
   description and a pointer to the status file path.
4. Show the diff of `MEMORY.md`.
5. Commit the archive file in the state repository together with the
   status file.

Nothing is deleted: the text exists in the archive before it leaves the
index. Never do this without the user's yes.
