# 0001. Record architecture decisions

- Status: Accepted
- Date: 2026-11-23

## Context

gellee is a small tool with a few decisions that everything else
depends on: where data lives, which process may write it, and what
reaches a public history. The reasons behind them are easy to lose
once the code works, and a contributor reading the code sees what was
built but not what was ruled out.

Design notes written while building a feature describe that feature at
one moment. They are not kept in step with the code and say little
about what still holds.

## Decision

Decisions of lasting effect are recorded as short documents in
`docs/adr/`, one per decision, numbered in order. Each has a status,
a date, a context, the decision in the present tense and its
consequences.

A record is immutable once it is accepted into `main`. When a decision
changes, a new record states the new decision and the old record's
status becomes `Superseded by` a link to it. While a record is still
in an unmerged branch it can be corrected freely.

The index in `docs/adr/README.md` lists every record with its status,
and `docs/adr/template.md` is the starting point for a new one.

## Consequences

A reader can learn why the system has its shape from a handful of
pages, and can see which decisions have been replaced.

A mistake in an accepted record stays in it. It is reported in a new
record or in the superseding one rather than edited away, so the
history of the reasoning stays intact.

Writing a record costs a pull request's worth of attention, so only
decisions that are expensive to reverse get one.
