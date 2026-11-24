# gellee

gellee keeps track of many projects developed mainly with Claude Code.
It provides a status file per project in a private state repository, a
`/handoff` skill that rewrites it, a session-start hook that shows it, a
terminal listing, a read-only dashboard, a memory store carried between
machines, and private files kept out of every repository.

## Install

gellee is installed on every machine you work on, from a clone of this
repository, together with a private repository that holds the state.
Requirements, the directory layout and the setup steps are in
[docs/install.md](docs/install.md).

## The server image

The dashboard server is published as `ghcr.io/aimd54/gellee:vX.Y.Z`.
Deploy it by digest. Each image is signed with cosign, keyless, from the
release workflow. To verify an image:

```bash
cosign verify ghcr.io/aimd54/gellee@sha256:<digest> \
  --certificate-identity-regexp '^https://github.com/aimd54/gellee/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

The digest of each release is in its release notes.

## Documentation

- [Architecture](docs/architecture.md)
- [Architecture decision records](docs/adr/)
- [Installation](docs/install.md)
- [The status convention](docs/status-convention.md)
- [The dashboard](docs/dashboard.md)
- [Memories](docs/memories.md)
- [Continuous integration](docs/ci.md)
- [Roadmap](docs/roadmap.md)

## Contributing, security, licence

- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
- Security reports: [SECURITY.md](SECURITY.md)
- Licence: [Apache-2.0](LICENSE)
