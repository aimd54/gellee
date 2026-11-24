# Maintainers

| Name | GitHub |
| --- | --- |
| aimd54 | [@aimd54](https://github.com/aimd54) |

## Pull requests from forks

Never merge a pull request from a fork as it is. A fork controls its own
copy of the workflow, so a green Publication gate on such a pull request
does not mean the deny-list ran. After review, re-push the reviewed
commits as a branch of this repository, through the pre-push hook, and
merge that branch's pull request.

## Releasing

1. The commit to release must have been the tip of a push to `main`, and
   each of these workflows must have succeeded on that push: CI (its
   "CI gate" job), Docs ("Lint docs"), CodeQL ("Analyze Go" and
   "Analyze JavaScript") and Publication gate.
2. Tag that commit with `git tag -s vX.Y.Z <sha>`, or
   `git tag -a vX.Y.Z <sha>` without a signing key. The gate rejects
   an annotated tag with a tagger date outside UTC, so create it in UTC.
3. Push the tag through the pre-push hook.
4. `release.yml` refuses the tag unless the four workflows above
   succeeded on the tagged commit. It then builds and signs the image
   and writes the release.
5. Deploy the image by digest, through the GitOps repository.
