# 0002. Keep project data in a separate private state repository

- Status: Accepted
- Date: 2026-11-23

## Context

gellee records where work stopped on each project, the configuration
for the machines that report, token usage, git activity and the files
that describe each machine. This data names projects and describes
what is being worked on, so it is private to its owner. It also has to
follow the owner between machines.

Keeping it inside gellee's own repository would put it in a public
history once the tool is published. Keeping it inside each project's
repository would mix notes with project commits and split one view of
all projects across many places.

## Decision

The tooling holds no data. All of it lives in one separate private git
repository, the state repository, named by `GELLEE_STATE`:

- status files, one per project;
- the `config` file with the project roots, the soft limit and the
  age that decides whether a new stub starts active;
- an optional `prices` file with the token price table;
- usage rows, git facts and machine files written by the collector;
- private files carried for each project.

The tool's code, tests and documentation use only invented data. Every
command that reads or writes the state repository takes its location
from `GELLEE_STATE` or `--state`, and the server reads a checkout of
it and nothing else.

## Consequences

The tool can be public without any note, status or usage figure
reaching its history. A new installation starts from an empty state
repository and a configuration file.

Each machine needs a checkout of the state repository, and a push
after each write keeps machines in step. Collector commits make the
repository grow with every collection, which is acceptable for text
files of this size.

Anything stored there is only as private as the repository's hosting.
The state repository must not be public.
