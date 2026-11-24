# Architecture

gellee keeps track of projects developed with Claude Code, across
several machines. The tooling holds no data: everything it reads or
writes lives in a private state repository
([0002](adr/0002-separate-private-state-repository.md)). The decisions
behind the design are in [docs/adr](adr/README.md).

## Pieces

**The status convention.** One Markdown status file per project, kept
in the state repository
([0003](adr/0003-one-markdown-status-file-per-project.md)). It is made
of a Claude Code skill and hooks in `skills/` and `bin/`:

- the `/handoff` skill rewrites the file at the end of a session;
- `gellee-session-start` is the hook that shows it and pulls the state
  repository, links private files and syncs memories first;
- `gellee-init` creates a stub for every project under the configured
  roots, `gellee-status` lists every project with its state, and
  `gellee-path` prints the status file path for a directory.

**The `gellee` binary.** One Go program with four commands:

- `collect` summarises a machine's transcripts and working trees into
  the state repository
  ([0004](adr/0004-machine-summaries-through-the-state-repository.md),
  [0007](adr/0007-report-workers-through-the-collector.md));
- `serve` reads a checkout of the state repository and serves the
  interface ([0005](adr/0005-read-only-embedded-interface.md));
- `memories` carries Claude Code's memories between machines
  ([0006](adr/0006-sync-memories-through-a-git-checkout.md));
- `private` links each project's private files from the state
  repository into its checkouts
  ([0008](adr/0008-private-files-in-the-state-repository.md)).

**The interface.** A single-page React application in `web/`, built
into `web/dist` and embedded into the binary. It talks to a read-only
JSON API served by `gellee serve` and has views for the overview (which
includes the workbench), the project list, a single project, usage,
ideas and workers.

**The state repository.** A private git repository holding `config`, an
optional `prices` file, the status files, `usage/`, `git/`,
`machines/` and `private/`. Each machine has a checkout. Collectors
write only files under their own machine's name, so machines do not
conflict with each other.

**The memory store.** A second private git repository holding Claude
Code's memory directories by project key, the global instructions file
and the versions set aside by conflicts. It is read and written only by
`gellee memories`; the collector reads two figures from it.

## Data flow

```text
 working machine                state repository             server
+----------------------+       +------------------+     +----------------+
| transcripts          |       | <key>.md files   |     |                |
| working trees        |       | usage/           |     | gellee serve   |
|                      | push  | git/             | pull|  reads the     |
| gellee collect  -----+------>| machines/        |-----+->checkout,     |
| /handoff             |       | private/         |     |  serves the    |
| gellee private       |       | config           |     |  interface     |
+----------+-----------+       +------------------+     +----------------+
           |
           | sync
           v
   memory store (private git repository), one checkout per machine
```

Status files are written by `/handoff` and read by every other piece.
Summaries flow one way, from machines to the server. The server has no
write path and holds no data of its own.

## Repository layout

| Path | Content |
| --- | --- |
| `cmd/gellee/` | Entry point and the four commands |
| `internal/collect/` | Summaries of transcripts and working trees, and the machine file |
| `internal/gitfacts/` | State of working trees, read with the `git` command |
| `internal/memories/` | Memory sync: adopt, sync, list, notices and health |
| `internal/private/` | Private files: link, sweep and status |
| `internal/server/` | The read-only snapshot, the JSON API and static serving |
| `internal/state/` | Files of the state repository: configuration, status files, summaries |
| `internal/transcript/` | Reads session transcripts into usage rows |
| `web/` | The interface: `src/`, unit tests, end-to-end tests in `e2e/`, and `embed.go` |
| `skills/` | The `/handoff` and `/memories` Claude Code skills |
| `bin/` | The hook and listing scripts, and the built binary |
| `lib/` | Shell functions shared by the scripts in `bin/` |
| `tests/` | bats tests for the scripts, and the invented fixtures they use |
| `ci/` | Toolchain image, helper scripts, the publication gate and its hook |
| `docs/` | User guides, this overview and the decision records |

## Deployment

A server runs the released container image, pinned by digest, with a
checkout of the state repository mounted. A worker, a machine that
collects and syncs, builds a pinned release tag from source, after
checking that the tag resolves to the pinned commit. Only tagged
releases are deployed.

Each installation keeps its own values (remotes, machines, hosts, the
location of its state repository and the deny-list) in a private
deployment repository, so none of them reach this repository
([0009](adr/0009-public-code-private-deployment.md)). The
[installation guide](install.md) describes setting up a machine and the
[CI guide](ci.md) describes the workflows.
