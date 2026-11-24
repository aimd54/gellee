# Status convention

One small file per project records where work stopped, the next action
and what remains. The files live in a separate private repository, so
they follow you between machines and never enter a project's own history.

## The pieces

| Piece | What it does |
| --- | --- |
| `gellee-init` | Creates a stub for every project under the configured roots |
| `gellee-status` | Prints every project with its state and next action |
| `gellee-path` | Prints the status file path for a directory |
| `gellee-session-start` | Claude Code hook: shows the status at session start |
| `/handoff` | Claude Code skill: rewrites the status at the end of a session |

The [dashboard](dashboard.md) shows the same files in a browser, with
git activity and token usage.

## Status file

```markdown
---
project: oss/webapp
state: active
group: webapp
updated: 2026-11-09
---

## Where I stopped

Milestone 2, task 17 paused before its review.

## Next action

Run the task 17 review.

## Remaining

- Milestone 2, tasks 18 to 24

## Ideas

- Compare two seed mixes on the same field plan
```

| Field | Required | Values |
| --- | --- | --- |
| `project` | yes | Path relative to the projects root |
| `state` | yes | `active`, `parked`, `done` |
| `updated` | yes | Date of the last handoff |
| `group` | no | Name shared by related projects |
| `parked_reason` | when parked | One line |
| `superseded_by` | no | `project` value of the replacement |
| `formerly` | no | Previous `project` values, separated by spaces |

`formerly` keeps the usage figures of a renamed project attached to it:
the dashboard counts what was recorded under an old name as belonging to
the current one.

The file is rewritten at each handoff. Its history is the git log of the
state repository.

## Installation

[The installation guide](install.md) covers the state repository, the
layout of the projects, and what each machine needs.

## Daily use

- Starting a session in a project first pulls the state repository, so
  the status read is the one the last handoff on any machine wrote, and
  links the project's private files into the checkout. It then prints
  one line with the project's state and next action, followed by any
  warning: a status older than the last commit, a parked project, or
  more active projects than the limit. The whole status file goes into
  the session context, so asking "where was I?" is answered from it.
- Type `/handoff` before stopping. It sweeps the project's private files
  into the state repository, commits them with the status file and,
  when that repository has a remote, pushes it. It ends with one line
  saying whether the project can be picked up on another machine:
  nothing uncommitted, nothing unpushed, every private file swept and
  linked, and the memory sync done. Otherwise the line lists what is
  missing.
- A status file stays under sixty lines. An item that does not fit
  moves to the project's roadmap or backlog rather than being shortened,
  and the roadmap is updated when a piece of work starts or ends.
- `/handoff park <reason>` sets a project aside with a resume note.
  `/handoff activate` brings it back.
- `gellee-status` answers "what is next, everywhere" from a terminal.

## The limit

The limit is a warning, never a refusal. `gellee-init` marks every
recently touched project as active, which usually exceeds the limit. The
first use of the tool is choosing which of them to park.

## Development

The tools are pinned in `mise.toml`. Run the commands below in a shell
where mise is activated, or prefix each with `mise exec --`.

```bash
mise install
make test
make lint
make lint-docs
```
