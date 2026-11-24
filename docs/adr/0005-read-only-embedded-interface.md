# 0005. Serve a read-only interface embedded in the binary

- Status: Accepted
- Date: 2026-11-23

## Context

The interface answers one question first: where was I, and what is
next. It also shows projects, usage and machines. It displays files that
Claude Code and the user edit elsewhere, so a second place to edit them
would create two sources of truth.

Running the dashboard should not need a web server, a Node runtime or a
separate deployment for the frontend, since it runs as one small
container next to a checkout.

## Decision

The interface is a single-page React application in `web/`, built by
Vite and embedded into the Go binary with `go:embed`. `gellee serve`
serves the static files and a read-only JSON API under `/api/v1/`.
Every endpoint answers `GET` only.

The API returns what is stored: projects with their sections, usage
rows, git facts, machines and ideas. Counts and totals that depend on
the view, such as active projects against the limit or tokens over a
chosen period, are computed in the browser from those rows. Unknown
paths under `/api/` return a JSON error, and any other path falls back
to the application's `index.html` so client-side routes load directly.

## Consequences

The binary is the whole product: the container image holds one static
executable and the interface it serves, and a developer needs Node only
to rebuild the frontend.

A new view of existing data needs no server change, because the browser
derives it. The cost is that the browser downloads rows rather than
answers, which is small at the size of one person's projects.

Capturing an idea or editing a status from the browser is not
possible. It would need a write path, which a later record would have
to justify.
