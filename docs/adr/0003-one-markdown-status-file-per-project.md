# 0003. Describe each project in one small Markdown status file

- Status: Accepted
- Date: 2026-11-23

## Context

Work is spread over many projects, several of them in progress at the
same time, and sessions with Claude Code start and stop at any
moment. Picking a project up again means finding where work stopped and
what comes next. Project-local planning files differ from project to
project and are not visible together.

A person has to be able to read and edit the record by hand, and a
program has to be able to read it without a parser for a bespoke
format.

## Decision

Each project has one Markdown file in the state repository, at a path
derived from its project key (`alpha/fern`). The file has YAML
frontmatter (`project`, `state`, `updated`, and optionally `group`,
`parked_reason`, `superseded_by`, `formerly`) and four sections:

- Where I stopped
- Next action
- Remaining
- Ideas

`state` is `active`, `parked` or `done`. The `/handoff` skill rewrites
the file at the end of a session, and a session-start hook shows it at
the beginning of the next. A soft limit on the number of active
projects, 3 by default and set in `config`, produces a warning at
session start and on activation. It never refuses anything.

## Consequences

The status of every project is readable with `cat`, editable in any
editor and diffable in git. The dashboard and the terminal listing read
the same files, so they cannot disagree.

The format is small on purpose: anything that needs more structure,
such as usage or git activity, is collected separately and not written
into the status file.

The limit nudges toward finishing or parking work instead of
enforcing it. Renaming a project needs a `formerly` entry so that usage
recorded under the old key stays attached.
