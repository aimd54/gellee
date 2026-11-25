# Continuous integration

The workflows live in `.github/workflows/` and run on GitHub Actions.
Each job of `ci.yml` starts one container of the toolchain image and
calls Makefile targets in it, so a job and the same target on a working
machine run the same thing. Every third-party action is pinned to a
commit.

## Workflows

| Workflow | Runs on | Jobs |
| --- | --- | --- |
| `ci.yml` | every push to `main`, every pull request, every Monday, by hand | Lint, Test, Interface, Build and browser tests, Known vulnerabilities, CI gate |
| `docs.yml` | every push to `main`, every pull request | Lint docs |
| `codeql.yml` | every push to `main`, every pull request, every Wednesday | Analyze Go, Analyze JavaScript |
| `scorecard.yml` | every push to `main`, every Tuesday, a change to the branch protection rules | Scorecard analysis |
| `gate.yml` | every push to `main`, every pull request, every tag `v*` | Publication gate |
| `release.yml` | every tag `v*` | Required checks, Image, GitHub release |

| Job | Runs |
| --- | --- |
| Lint | `make fmt-check vet tidy-check lint lint-ci lint-docs` |
| Test | `make test` |
| Interface | `make web-check` |
| Build and browser tests | `make build`, then the Playwright suite |
| Known vulnerabilities | `make vuln` |
| Lint docs | `make lint-docs`, with mise installing Node and markdownlint |
| Analyze Go, Analyze JavaScript | CodeQL, with a manual `go build ./...` for Go |
| Scorecard analysis | the OpenSSF Scorecard action, publishing its results and uploading them as code scanning results |
| Publication gate | `ci/public-gate`, described below |

`make check` runs the gates of the Lint, Test and Interface jobs in one
command, and builds the interface. It does not run `make vuln`, the
Go binary build or the browser tests. `make e2e-web` runs the browser
tests locally, in the Playwright image.

## Required checks

Two sets of checks guard `main`.

Rulesets on pull requests and on `main` require five checks by name:
`CI gate`, `Lint docs`, `Analyze Go`, `Analyze JavaScript` and
`Publication gate`. `CI gate` is the last job of `ci.yml`. It waits for
Lint, Test, Interface, Build and browser tests, and Known
vulnerabilities, and fails unless it sees exactly those five results and
each one succeeded or was skipped. It is the one name to require for the
whole of `ci.yml`, so adding a job there means adding it to the `needs`
of `CI gate` and raising the count of results that its script expects.

A release has a second requirement. For the commit a tag points at,
`release.yml` needs the tag to be an ancestor of `main` and the latest
push-to-`main` run of `ci.yml`, `docs.yml`, `codeql.yml` and `gate.yml`
to have succeeded; `ci/scripts/require-runs` reads those runs. A commit
that was never the tip of a push to `main`, such as one in the middle of
a merged branch, has no such runs and cannot be released. Scorecard is
not required.

## The toolchain image

`ci/toolchain.Dockerfile` starts from `PLAYWRIGHT_IMAGE` in the
`Makefile`, so the browsers match `@playwright/test`. It adds the system
packages the gates need (a C compiler for the race detector, `jq` for
the hooks, `column` for the status table), installs one fixed release of
mise with `ci/scripts/install-mise`, which checks its digest, and
installs the tools of `mise.toml` and `mise.lock`.

The composite action `.github/actions/toolchain` builds it with buildx
and the Actions cache, loads it into the runner's engine under the name
`gellee-ci`, and runs `toolchain start`. The image is never pushed to a
registry. `make ci-image` builds it locally.

`ci/scripts/toolchain` drives it from a job:

- `toolchain start` builds the image unless the job built it already,
  starts the job's container and copies the checkout into its `/src`.
- `toolchain run [-C dir] cmd` runs a command in `/src`, or in a
  directory under it. Every step of a job shares the container, so
  `node_modules` and the Go caches carry over.
- `toolchain stop` removes the container. The container removes itself
  after an hour if a job ends without this step.

The same commands work on a working machine with Docker, from a clean
clone.

## The publication gate

`ci/public-gate` checks commits, trees and ref names before they reach
the public repository, and rejects anything it cannot check.
[ADR 0009](adr/0009-public-code-private-deployment.md) explains why. It
runs four checks:

- Root guard: every tip descends from the one public root commit that
  the tree records, and from no other root.
- Metadata: every commit has a `Signed-off-by` trailer that matches its
  author, a committer that is the author, a maintainer or GitHub, and a
  UTC date when a maintainer wrote or committed it. An annotated tag
  that a maintainer created needs a UTC tagger date too. The merge
  button on GitHub writes the committer date in the merging account's
  time zone, so pull requests are merged by a fast-forward push from a
  clone, as `MAINTAINERS.md` describes.
- Deny-list: a file of patterns that no commit message, identity, added
  line, path touched by a commit, file in the tree, ref name or tag
  message may match.
- gitleaks: a scan of every tree and of the history, with its default
  rules and no ignore files, and no allow marker anywhere.

Output names the check, the file, the commit and the line number of the
pattern, and never a pattern or a matched value. A path that matches a
pattern is shown as "a withheld path".

The deny-list is a UTF-8 file of one extended regular expression per
line, matched without regard to case. Blank lines and lines whose first
non-blank character is `#` are skipped. A byte order mark at the start
of the file and carriage returns at line ends are ignored. A UTF-16
file, a NUL byte, or a control character in a pattern (a tab included)
stops the gate; write `[[:blank:]]` to match a tab. The gate reads the
file from `GELLEE_DENYLIST`, or from the absolute path that the git
setting `gellee.denylist` names. In CI it is the repository secret
`GATE_DENYLIST`.

The gate checks the objects themselves: it runs from the top level of
the checkout, ignores replace refs, exports trees through an empty
repository that borrows the objects, and overrides the diff settings and
attributes that would change what it reads. A file in an exported tree
that no longer hashes to its blob, as after an `eol`, `ident` or
`working-tree-encoding` attribute, fails the gate. In a partial clone
the export stops with "cannot export" while blobs are missing locally:
run the gate from a full clone.

Run the gate before every push to the public repository with:

```bash
make hooks
```

This points `core.hooksPath` at `ci/hooks`. The `pre-push` hook there
runs the gate on each branch or tag pushed to a remote whose URL ends in
the owner and repository of the module path in `go.mod`, whatever the
host, and lets pushes to other remotes pass.

`gate.yml` runs the gate for pull requests, pushes to `main` and tags.
For a pull request it checks the commits that the head adds to the base.
A pull request from a fork gets no secrets, so the deny-list is not
available: the job runs the other checks, then fails on purpose, and a
maintainer re-pushes the commits as a branch of this repository. The
workflow never uses `pull_request_target`.

A fork controls its own copy of the workflow, so a green Publication
gate on a pull request from a fork says nothing about the deny-list:
it may not have run. Such a pull request is never merged as it is.

## Releases

A release is a tag `vX.Y.Z` on a commit of `main`, following
[Semantic Versioning](https://semver.org). Pushing the tag starts
`release.yml`, which runs three jobs in order:

1. Required checks: the tag is on `main` and the four workflows named
   above succeeded for it.
2. Image: builds the `Dockerfile` for `linux/amd64` and `linux/arm64`
   with QEMU and buildx, from the tagged tree alone and without a build
   cache, and pushes `ghcr.io/aimd54/gellee:vX.Y.Z`. The build attaches
   an SBOM and maximum-detail provenance, and sets the OCI labels for
   source, revision, version and license. cosign then signs the digest
   without a key, with the workflow's identity.
3. GitHub release: writes the notes with `ci/scripts/release-notes`,
   which groups the Conventional Commit subjects since the previous
   `v*` tag into breaking changes, features, fixes and other changes,
   adds the image reference with its digest, and creates the release.

There is no `latest` tag: a deployment names a version and its digest.
[The installation guide](install.md#the-server) shows how to verify the
signature.

## Dependency updates

Renovate proposes updates through its GitHub app, with the
`config:best-practices` preset and the schedule in `renovate.json`.
Two custom managers read what the presets do not: the
`PLAYWRIGHT_IMAGE` line of the `Makefile`, and the pinned `govulncheck`
version in it. Package rules group the updates that must move together:
Playwright's image and library, the Go toolchain, Node, and the minor
and patch updates of the interface's development tools. mise itself
updates once a month.

## Changing a pin by hand

- A tool: edit `mise.toml`, then run `mise lock` and commit `mise.lock`
  with any file that changed under `.mise/locks/`.
- mise itself: the version and both digests in
  `ci/scripts/install-mise`, from the release's `SHASUMS256.txt`.
- gitleaks: the version in `mise.toml`, then `mise lock`, then the
  version and both digests in `ci/scripts/install-gitleaks`. A test in
  `tests/gitleaks_pin.bats` fails when the script's version or either
  linux digest differs from `mise.toml` and `mise.lock`.
- Playwright: the version of `@playwright/test` and `PLAYWRIGHT_IMAGE`
  in the `Makefile` are one version.

## Workflow changes

A token needs the `workflow` scope to push a change to
`.github/workflows`, and the GitHub command-line token does not have it.
A maintainer pushes such a branch over SSH, and a pull request that
changes a workflow cannot be updated or merged through that token.
