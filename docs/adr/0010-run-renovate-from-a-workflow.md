# 0010. Run Renovate from a workflow in the repository

- Status: Accepted
- Date: 2026-11-25

## Context

gellee pins every dependency: Go modules, npm packages, actions by
commit, images by digest, and tools in `mise.toml`, the `Makefile` and
the CI scripts. Renovate reads all of these, including the pins in the
`Makefile` through custom managers, which Dependabot cannot.

Renovate runs either as a hosted GitHub app, which the vendor operates
under its own account terms, or self-hosted from a workflow. The app
needs an account with the vendor, tied to the maintainer's GitHub
identity and contact address.

## Decision

Renovate runs from `.github/workflows/renovate.yml`, daily and on
demand, over this repository alone. The action is pinned by commit and
the Renovate image by tag and digest, and Renovate updates both through
its own GitHub Actions manager.

It acts with the `RENOVATE_TOKEN` secret: a fine-grained token limited
to this repository, with read and write access to contents, pull
requests, issues and workflows, and read access to Dependabot alerts.
The job's own `GITHUB_TOKEN` has read access to contents only. Forks
have no secret and skip the job.

Renovate commits as `Renovate Bot <renovate@whitesourcesoftware.com>`,
its self-hosted default, and signs off each commit with that identity,
so its commits pass the publication gate like any contributor's. A
maintainer merges its pull requests as any other.

`renovate.json` holds the repository configuration, unchanged by where
Renovate runs.

## Consequences

- No account with the vendor, and no app with access to the account's
  other repositories.
- Pull requests opened with the token run every workflow, the
  publication gate with its deny-list included, since their branches
  are branches of this repository.
- The token is a credential to rotate before it expires. When it is
  missing or expired, the job fails and no update arrives.
- A run takes a few minutes of Actions time each day.
