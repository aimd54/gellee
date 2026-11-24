# check=skip=InvalidDefaultArgInFrom
# Copyright The gellee Authors
# SPDX-License-Identifier: Apache-2.0
#
# The image the workflows run their gates in: the Playwright image the
# browser tests need, with the tools pinned in mise.toml and mise.lock.
# `make ci-image` builds it and names the base from PLAYWRIGHT_IMAGE in
# the Makefile, so the browsers and @playwright/test keep one version.

ARG PLAYWRIGHT_IMAGE
FROM ${PLAYWRIGHT_IMAGE}

# The race detector of `make test` needs a C compiler, the hooks read
# their input with jq, and gellee-status lays its table out with column,
# from bsdextrautils.
RUN apt-get update \
  && apt-get install --yes --no-install-recommends \
    bsdextrautils ca-certificates curl gcc git jq libc6-dev make \
  && rm -rf /var/lib/apt/lists/*

# The checkout is copied in with the owner it had in the job, which git
# would otherwise refuse to read.
RUN git config --system --add safe.directory /src

COPY ci/scripts/install-mise /usr/local/bin/install-mise
RUN install-mise

ENV PATH=/root/.local/share/mise/shims:/root/.local/bin:$PATH \
    MISE_TRUSTED_CONFIG_PATHS=/src

WORKDIR /src
COPY mise.toml mise.lock ./
COPY .mise/locks/ .mise/locks/
RUN mise install --locked
