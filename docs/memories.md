# Memories

Claude Code keeps what it learns about a project in memory files, on
the machine the session ran on. gellee carries them, and the global
instructions file `~/.claude/CLAUDE.md`, between machines through a
private git repository, the memory store, and adds a skill to review
them across projects. The decision is recorded in
[ADR 0006](adr/0006-sync-memories-through-a-git-checkout.md).

## How it works

Every machine has a checkout of the store, named by `GELLEE_MEMORIES`.
`gellee memories sync` copies the memory directories into it, lets git
merge what each machine changed, and copies the result back. The memory
directories stay plain directories, so Claude Code reads and writes
them as before. When `~/.claude/CLAUDE.md` is a symbolic link, the sync
reads and writes the file it points to and leaves the link in place.

```text
alpha/fern/               MEMORY.md and one file per memory
global/CLAUDE.md          the global instructions
orphans/<machine>/<dir>/  memories that match no project
orphans/<machine>/CLAUDE.md
                          a global file that differed when the
                          machine joined
orphans/<machine>/conflicts/
                          versions set aside by a conflict
```

A project's directory is its project key, so machines need neither the
same home directory nor the same user name. The store holds memory text
only. The collector reads two values from it, this machine's last push
time and the number of versions set aside, and the dashboard shows them
on the Workers page; neither ever reads memory text.

## Commands

| Command | What it does |
| --- | --- |
| `gellee memories adopt` | Prints what joining this machine would do, and changes nothing |
| `gellee memories adopt --apply` | Joins it: a union per project, nothing deleted; refused once the machine has joined |
| `gellee memories sync` | Exchanges this machine's memories with the store |
| `gellee memories list` | Every memory, then the versions set aside, the orphans, and the names found in several projects; `--json` for the skill |
| `gellee memories notices --project <key>` | The lines the session-start hook shows; `--sync` syncs first |

Every command takes `--store`, `--state`, `--projects`, `--claude`,
`--machine` and `--timeout`. They default to `GELLEE_MEMORIES`,
`GELLEE_STATE`, `GELLEE_PROJECTS_ROOT`, `~/.claude`, `GELLEE_MACHINE`
or the short host name, and a minute, five seconds for `notices`.

## When it runs

- A user timer every ten minutes, so a machine already holds what the
  other pushed when a session starts. Memory is read when a session
  starts and cannot be refreshed inside it.
- `/handoff`, after the collector.
- The session-start hook, which syncs with a five-second timeout and
  adds a line when memories came in for this project (restart to use
  them), when versions set aside are waiting (run `/memories`), and when
  nothing was pushed for more than a day.

## Conflicts

No conflict stops a sync, and none discards a memory.

| What happened since the last sync | Result |
| --- | --- |
| Different memories changed on each side | Both kept |
| The same memory, different lines | Merged by git |
| The same memory, the same lines | The store's version stays; this machine's goes to `orphans/<machine>/conflicts/` |
| Changed on one side, deleted on the other | The changed version stays; the summary counts an overridden deletion |
| A line added to `MEMORY.md` on each side | Both lines kept |

Until `/memories` merges them, sessions read the store's version.

## What a sync never does

- **Read a missing directory as a deletion.** A project cloned after the
  machine joined, or cloned again after it was removed, is joined by
  union. A joined project whose memory directory is missing or empty is
  restored from the store, and so is one left with only `MEMORY.md`
  while the store holds memories for it: deleting the last memory of a
  project in its memory directory brings it back. To remove every
  memory of a project, delete its files in the checkout and commit; the
  next sync removes them everywhere.
- **Undo what a session writes while it runs.** A memory created,
  changed or removed during a sync is left as the session made it, and
  the next sync carries it. When the same sync brought another
  machine's change to that file, it is settled as a conflict: a memory
  removed during the sync comes back with the other machine's change
  and its index line, a memory changed during it keeps the store's
  version and this machine's is set aside, and an index keeps the lines
  of both.
- **Lose work when the remote is unreachable.** The snapshot and the
  merge are committed in the checkout, and the next sync that reaches
  the remote delivers them.
- **Run twice at once.** A second sync finds the lock, says so and
  stops; the one running carries the changes.
- **Run over a review in progress.** It refuses a checkout with
  uncommitted changes and names them.

The checkout belongs to the tool. A sync aborts a merge it left half
done and removes the versions that merge had set aside, which the
repeated merge sets aside again, and it resets a detached checkout.
Edits to the store are therefore made on its branch, as `/memories`
makes them.

Two machines in use at the same time stay safe, not seamless: what one
writes reaches the other at its next session start.

## Reviewing

`/memories` works in the checkout and applies nothing without a yes.

| Invocation | Covers |
| --- | --- |
| `/memories` | Conflicts set aside, then orphans |
| `/memories review <project>` | One project |
| `/memories review all` | Every project, then the rules found in several |

It proposes merging a conflict into one text, placing an orphan in its
project, moving a rule repeated in several projects into
`global/CLAUDE.md`, deleting a memory that restates the global
instructions, and it flags stale candidates for you to decide on.

## Joining the machines

Set up the checkout and the settings on each machine as the
[installation guide](install.md#the-memory-store) describes, then:

1. On the first machine, `gellee memories adopt`, read the plan, then
   `gellee memories adopt --apply`. The store holds its memories and its
   orphans.
2. `/memories review all` there, before the second machine joins.
3. On the second machine, the same two `adopt` commands. It receives the
   reviewed set; its own orphans and any differing global file go under
   `orphans/`.
4. `/memories` there, for those.
5. Start the timer on both.

## Troubleshooting

- **`no push to the store yet` or `no push to the store since ...`.**
  Run `gellee memories sync` in a terminal and read its error. The
  timer runs without a terminal, so the remote has to accept a push
  without a prompt.
- **`the checkout of the memory store has uncommitted changes`.** A
  review or a sync stopped half way. Run `/memories`, which offers to
  commit or discard them.
- **`this machine has not joined the memory store`.** Run
  `gellee memories adopt`, read the plan, then
  `gellee memories adopt --apply`.
- **`this machine has already joined the memory store`.** Run
  `gellee memories sync`: a machine joins once, and the sync joins the
  projects cloned since.
- **`no git identity to commit to the memory store with`.** Set
  `user.name` and `user.email` for git, globally or in the checkout.
