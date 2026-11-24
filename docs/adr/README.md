# Architecture decision records

A decision that shapes gellee and would be costly to reverse gets a
record in this directory. Each record states the context, the decision
and its consequences, and is numbered in the order it was accepted. A
record is immutable once accepted: a decision that changes gets a new
record that supersedes the old one, and the old record's status then
points to the new one. [The template](template.md) is the starting
point for a new record. [The architecture overview](../architecture.md)
shows where each decision applies.

| Number | Title | Status |
| --- | --- | --- |
| [0001](0001-record-architecture-decisions.md) | Record architecture decisions | Accepted |
| [0002](0002-separate-private-state-repository.md) | Keep project data in a separate private state repository | Accepted |
| [0003](0003-one-markdown-status-file-per-project.md) | Describe each project in one small Markdown status file | Accepted |
| [0004](0004-machine-summaries-through-the-state-repository.md) | Carry machine summaries through the state repository | Accepted |
| [0005](0005-read-only-embedded-interface.md) | Serve a read-only interface embedded in the binary | Accepted |
| [0006](0006-sync-memories-through-a-git-checkout.md) | Sync Claude Code memories through a git checkout | Accepted |
| [0007](0007-report-workers-through-the-collector.md) | Report workers through the collector | Accepted |
| [0008](0008-private-files-in-the-state-repository.md) | Keep private files in the state repository, linked into checkouts | Accepted |
| [0009](0009-public-code-private-deployment.md) | Publish the code, keep each installation private | Accepted |
