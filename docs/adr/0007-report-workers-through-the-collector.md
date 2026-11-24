# 0007. Report workers through the collector

- Status: Accepted
- Date: 2026-11-23

## Context

Work is done on several machines, some of them always-on virtual
machines that sessions are started on and steered from a distance. Any
project may be worked on from any of them. A machine that stops
collecting, runs an old Claude Code version or stops pushing its
memories is not noticed until a session starts there.

The dashboard needs one page that shows every machine and whether it is
in step, and a way from a project to the session last worked in. It
already has a one-way channel, the state repository (see
[0004](0004-machine-summaries-through-the-state-repository.md)).

## Decision

Each machine writes one machine file, `machines/<machine>.json`,
through `gellee collect`. It holds the time of the collection, the
Claude Code version, the memory store's health and the machine's
sessions from the last seven days, with their ids, times and links that
open them. The memory store's health is the last push time and the
count of versions set aside, read from the store without reading memory
text.

A machine becomes a worker from its first collection, with no
registration step. The server reads the machine files with the rest of
the state repository and adds no write path. The Workers page lists
every machine, and the overview and project pages show the machine and
session a project was last worked in.

## Consequences

A machine that stops reporting is visible as one whose last collection
is old. Figures are as fresh as the last collection, which a timer
bounds.

A session's title and branch leave the machine only for sessions inside
the configured roots, and no prompt text leaves it.

The server cannot start or steer a session or show live status. Usage
rows carry no machine, so usage is not broken down per worker.
