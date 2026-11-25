# Contributing

Thank you for helping with gellee. Participation is governed by the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Sign-off

Every commit carries a `Signed-off-by` line, added with
`git commit -s`. The line certifies the Developer Certificate of Origin:
that you wrote the change or have the right to submit it under the
project's licence, and that the contribution may be kept in the public
history. The full text is at <https://developercertificate.org/>.

## Commits

Commit subjects follow [Conventional Commits](https://www.conventionalcommits.org/),
for example `fix(collect): skip empty transcripts` or
`docs: describe the memory store`. History is linear: a maintainer
rebases a pull request onto `main`, signing each commit as its
committer, and moves `main` forward to it, so each commit lands as
written and stands on its own: it builds, its
tests pass, and its message says what changed and why.

## Checks

Run `make check` before pushing. It runs formatting, vet, lint, tests,
the interface checks and the documentation and workflow linters. It
leaves out two CI jobs: the browser tests, which `make e2e-web` runs
(run it for changes to the interface; it needs Docker), and the
vulnerability scan, which `make vuln` runs.

## Decisions

An accepted architecture decision stays as written. A change to one gets
a new record that supersedes it, started from
[docs/adr/template.md](docs/adr/template.md).

## For maintainers

- Run `make hooks` in every clone. It installs the pre-push hook that
  runs the publication gate.
- Set the deny-list the gate reads:
  `git config gellee.denylist <path to the deny-list>`.
- Never push to this repository with `--no-verify`.

Public CI cannot check a pull request from a fork against the deny-list,
because a contributor could probe the list through the verdict. After
review, a maintainer pushes the reviewed commits as a branch of this
repository, through the hook, and merges that pull request. A pull
request from a fork is never merged itself, and a green Publication
gate on one does not mean the deny-list ran.
