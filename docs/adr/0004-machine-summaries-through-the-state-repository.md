# 0004. Carry machine summaries through the state repository

- Status: Accepted
- Date: 2026-11-23

## Context

The dashboard shows usage, git activity and the state of each machine.
That information is produced on the machines where Claude Code runs:
session transcripts and working trees exist only there. The server that
renders the dashboard is a different machine and should not hold
transcripts, which contain prompts and file contents.

A design with an ingest endpoint would need authentication, a write
path on the server and a database to keep what arrives. The state
repository already exists, is private and already moves between
machines.

## Decision

One binary does both jobs. `gellee collect` runs on each working
machine, reads that machine's transcripts and working trees and writes
summaries into the state repository under the machine's name:

- `usage/<machine>/<YYYY-MM>.jsonl`, tokens per day, session, project
  and model;
- `git/<machine>.json`, branch, last commit and unpushed or
  uncommitted work per project;
- `machines/<machine>.json`, the machine's own facts.

It commits those files and nothing else, and pushes on request.
`gellee serve` reads a checkout of the state repository and serves what
it finds. The server has no ingest endpoint, no database and no write
path.

## Consequences

Transcripts never leave the machine that wrote them, and no prompt text
or account identifier is written into a summary. A session's title and
branch are recorded only for sessions inside the configured roots.

Because each machine writes only files under its own name, machines
cannot conflict with each other in the repository. Figures are as
fresh as the last collection and the server's last pull, so a timer
bounds the delay rather than a live feed.

Every collection changes a file and so makes a commit. The server can
be restarted, replaced or scaled out without losing anything, since its
only input is a checkout.
