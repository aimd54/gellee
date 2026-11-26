<!-- markdownlint-disable MD033 MD041 -->
<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/mark-dark.svg">
  <img src="docs/assets/mark-light.svg" alt="" width="64" height="64">
</picture>

# gellee

**Every Claude Code session starts where the last one stopped, on any
machine, across all your projects.**

[Try it](#try-it) | [Install](docs/install.md) |
[Documentation](#documentation) | [Contributing](CONTRIBUTING.md)

</div>
<!-- markdownlint-enable MD033 MD041 -->

![On a laptop, a session runs /handoff and rewrites the status file of a
project. The file goes through a private state repository in git. On a
desktop, the next session opens with the project's state and its next
action.](docs/assets/readme-hero.svg)

## What you get

- A status file per project: where work stopped, the next action, what
  remains. The `/handoff` skill rewrites it at the end of a session.
- A session-start hook that shows it to the next session, on whichever
  machine that runs.
- A terminal listing and a read-only dashboard across every project,
  with usage and git facts.
- Memories and private files carried between machines and kept out of
  every project repository.

## Try it

Run the dashboard over invented projects, with no state repository:

```bash
docker run --rm -p 8080:8080 ghcr.io/aimd54/gellee:v0.2.0 --demo
```

and open <http://127.0.0.1:8080>:

![The dashboard overview over the demo projects: four active projects,
each with where work stopped, the next action and the cost of the last
seven days.](docs/assets/dashboard-light.png#gh-light-mode-only)
![The dashboard overview over the demo projects: four active projects,
each with where work stopped, the next action and the cost of the last
seven days.](docs/assets/dashboard-dark.png#gh-dark-mode-only)

From a built checkout, `gellee serve --demo` does the same; see
[the dashboard guide](docs/dashboard.md#demo).

## Install

gellee is installed from a clone of this repository on every machine you
work on, together with a private repository that holds the state. See
[docs/install.md](docs/install.md), which also covers verifying the
signed server image.

## Documentation

[Architecture](docs/architecture.md) |
[Decision records](docs/adr/) |
[Status convention](docs/status-convention.md) |
[Dashboard](docs/dashboard.md) |
[Memories](docs/memories.md) |
[Continuous integration](docs/ci.md) |
[Roadmap](docs/roadmap.md)

Contributing: [CONTRIBUTING.md](CONTRIBUTING.md) | Security reports:
[SECURITY.md](SECURITY.md) | Licence: [Apache-2.0](LICENSE)
