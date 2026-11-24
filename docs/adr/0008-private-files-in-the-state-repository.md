# 0008. Keep private files in the state repository, linked into checkouts

- Status: Accepted
- Date: 2026-11-23

## Context

Plans, reviews and notes that must not enter a project's public history
are kept in files named `*.private.*` and ignored by git. They exist
only on the machine that wrote them, yet a status file or a plan may
cite one, so the next task cannot run on another machine. A session
spawned in a fresh worktree holds no ignored file at all.

A private file must follow its project to every machine, at the path
the plans cite, without being committed to the project.

## Decision

A file counts as private when its name matches `*.private.*` and git
ignores it in its checkout. The state repository stores the real file
at `private/<project key>/<path in the checkout>`; the checkout holds a
symbolic link at the usual path pointing to it. Each machine creates
its own links, and they are never committed.

`gellee private` offers three commands:

- `link` creates any missing links and leaves a regular file alone;
- `sweep` moves each private file that is still a regular file into the
  state repository, puts a link where it was, then links;
- `status` reports files not yet swept and links that are missing or
  broken.

A tracked file is left where it is. An untracked file that git does not
ignore is reported, not moved, because the project's ignore rule is
missing. The session-start hook runs `link`, and `/handoff` runs
`sweep` before it commits.

## Consequences

A project's private notes appear at the same path on every machine, and
a worker's new worktree gets them at session start. They are versioned
in the state repository, which is private.

Neither command commits by itself, so a file edited through its link is
committed at the next handoff. Removing a private file means removing
it from the state repository, since deleting only the link brings it
back at the next session start.

Other ignored files a checkout may need, such as local environment
files, are not carried.
