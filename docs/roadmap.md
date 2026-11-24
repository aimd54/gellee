# Roadmap

gellee keeps track of many projects developed mainly with Claude Code. It
is built in five pieces. The decisions behind them are recorded in
[docs/adr](adr/README.md), and each piece is useful without the ones
after it.

| Piece | Status |
| --- | --- |
| Status convention | Done |
| Dashboard | First slice and workers done |
| Continuity between machines | Memories and private files done; skills and settings not started |
| Publication | Preparation in progress |
| Idea notebook and articles | Not started |

## Status convention

One small status file per project in a private state repository, a
`/handoff` skill that rewrites it, a session-start hook that shows it and
a terminal listing. See [the user guide](status-convention.md) and
[the decision](adr/0003-one-markdown-status-file-per-project.md).

## Dashboard

A web view over what the status convention already records. It is built
in slices. The first is done: see
[the user guide](dashboard.md) and
the decisions on
[collection](adr/0004-machine-summaries-through-the-state-repository.md)
and [the interface](adr/0005-read-only-embedded-interface.md).

### Decided

- **Built here.** Existing dashboards for Claude Code projects were
  surveyed. They cover the display, which is the small part, and expect
  planning-file conventions these projects do not have. A tool that reads
  every transcript and memory file is also one to keep under one's own
  review.
- **Go, with an embedded frontend**, the layout the sibling Go projects
  use: same Makefile targets, mise pins, lint and end-to-end test setup.
- **Their visual structure**, reusing the design tokens with a palette
  of its own.
- **Read-only.** The dashboard displays files; Claude Code and the user
  edit them. It needs no database and no write path, and nothing has to
  keep a backend in step with the files.
- **Served from a server, fed by a collector.** Each working machine
  summarises its own transcripts and working trees into the state
  repository. The server reads a checkout of that repository and nothing
  else, so session transcripts stay where they were written.
- **The prototype is the interaction reference**, not the codebase. It is
  a static demo with invented data: overview with a focus project,
  project list with filters, tasks, skills library, memories library,
  idea notebook.
  It is kept outside this repository, because its sample data is not
  written for publication.

### First slice

- Every project with state, last handoff, next action and remaining work,
  read from the status files.
- Projects that share a `group` as one larger piece of work, and
  `superseded_by` links.
- The limit on active projects, with the active ones and how far over.
- Token and usage figures per project and per period, computed from the
  session transcripts Claude Code already writes to disk. Spend per
  project next to the active list is the point: it shows what each
  parallel thread costs.
- Git activity per project: last commit, unpushed work, stale status.
- The *Ideas* sections of all status files in one list.

### Workers

Every machine that reports, with its sessions, version and memory
sync, and on each project the session it was last worked in, with a
link that opens it in Claude. See [the user guide](dashboard.md) and
[the decision](adr/0007-report-workers-through-the-collector.md).

### Later slices

- Skills and memory files across projects, to find duplicates and what
  could be shared. This needs the collector to summarise them, and a
  decision on how much of their content may leave the working machine.
- Search across projects, ideas and, once they exist, the libraries.
- Default prices, if estimated cost proves more useful than token
  counts.
- History of a project's status, read from the state repository's log.

### Open

- Whether any of the prototype's editing returns. The current answer is
  none: editing happens in the files.
- Whether `/memories` turns run in the memory store's own directory are
  credited to a project or left counted as `other`.

## Continuity between machines

Claude Code memory files and skills stay on the machine that wrote them
unless something carries them to the next. The state repository already
travels through its remote, so "where does each project stand" is
solved. Memory files and the global instructions file travel through a
memory store: see [the user guide](memories.md) and
[the decision](adr/0006-sync-memories-through-a-git-checkout.md).
Private files travel through the state repository: see
[the installation guide](install.md) and
[the decision](adr/0008-private-files-in-the-state-repository.md).
What remains:

- User-level skills and settings.
- Handing a project between Claude Code and another coding agent, such
  as Codex: set aside.

Several machines are used, any project on any of them, kept in step
by synchronisation rather than through one central machine. The state
repository and the memory store travel through their remotes, and the
Workers page shows which machines report and whether each is in step.

## Publication

The code is published as a public repository with a demo mode, and each
installation is deployed from a separate private repository that takes
the published project as a dependency. Each part gets a design before
code.

- The public repository holds the binary, the web interface, the skills
  and the scripts, with their documentation and workflows. This
  repository becomes it; its history is written for publication.
- The deployment repository holds only what one installation needs.
- The demo mode serves invented data read-only, such as the test
  fixtures, so the dashboard can be shown without a state repository.
- Publication gate, small gaps left: the `--no-relative` options have no
  test of their own; committer and sign-off addresses are compared
  case-sensitively; repositories in the SHA-256 object format are
  refused.

## Idea notebook and articles

Articles about experiments and findings, written by the author. The tool
helps with ideas, structure and the site, and does not write the text.

- The feed is the *Ideas* section of each status file, already captured
  at every handoff.
- The site is an existing static site generator, chosen when this piece
  starts. No site engine is built here.
- Subjects: data, AI, genealogy, infrastructure and Kubernetes.
- The dashboard lists ideas. Drafting happens in an editor.

## Order

Publication comes next: the public repository and the deployment
repository first, then the demo mode. User-level skills and settings
follow. The notebook waits until ideas have accumulated in the status
files.

A standing constraint on all of it: the goal is less time spent keeping
track, so each piece has to earn its place against the time it takes to
build.
