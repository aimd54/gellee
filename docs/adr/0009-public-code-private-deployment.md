# 0009. Publish the code, keep each installation private

- Status: Accepted
- Date: 2026-11-23

## Context

gellee is developed for one person's use and meant to be public. Once a
commit is pushed to a public host it can be cached, forked and indexed
even if it is deleted later, so nothing specific to one installation
may reach the public repository, in files or in history, on this push
or any later one. An installation's specifics are its remotes, machine
names, hosts, project names and the location of its state repository.

An earlier private history contained such details. Rewriting it would
leave doubt about what remained.

## Decision

The public repository on GitHub starts from a fresh root commit of a
scrubbed tree. The earlier history stays private and archived.

Each installation keeps its own values in a private deployment
repository: the deny-list, its installation notes and a daily scan.
Only tagged releases are deployed. A release is a SemVer tag, built by
CI into a multi-platform image with an SBOM and provenance, pushed as
`ghcr.io/<owner>/gellee:vX.Y.Z` with no `latest` tag, and signed with
cosign. A server pins the image by digest, and a worker builds the
tagged source after checking that the tag resolves to the pinned
commit.

A publication gate, `ci/public-gate`, runs four checks over a range of
commits and fails closed:

1. every commit descends from the public root commit;
2. every commit carries a `Signed-off-by` matching its author, its
   committer is the author, a maintainer or GitHub, and commits by a
   maintainer carry UTC dates;
3. a deny-list of regular expressions matches nothing in added lines,
   messages, identities or the tip's tree;
4. gitleaks finds no secret.

The gate runs in a `pre-push` hook for pushes to the public remote, in
CI on pull requests, pushes to `main` and tags, and in a daily scan
from the deployment repository. Its output names the check, file,
commit and pattern line number, never a matched value or a pattern, and
prints what it covered. Pull requests from forks receive no secrets. The
job runs the checks that need none and then fails with a message that
the deny-list is unavailable; a maintainer re-pushes the commits as a
branch of the repository. The deny-list is never evaluated against
untrusted content in public CI, since a contributor could read the
results and recover the patterns.

## Consequences

The tool can be read, built and forked by anyone, while an
installation's details live in a repository nobody else can see.

Fork contributors cannot get a green gate directly, so their changes
take one extra step by a maintainer. A pattern added to the deny-list
today also catches older content, because the whole tree at the tip is
scanned.

Deployment waits for a release: a fix reaches an installation as a
patch release tagged after its commit is green, and no image exists for
an untagged commit.
