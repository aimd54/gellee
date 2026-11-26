# Copyright The gellee Authors
# SPDX-License-Identifier: Apache-2.0

SHELL := bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := help

BINARY := gellee

.PHONY: help
help: ## List the targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "%-16s %s\n", $$1, $$2}'

.PHONY: test
test: ## Run the bats and Go tests
	bats --print-output-on-failure tests
	go test -race -timeout 10m ./...

.PHONY: lint
lint: ## Run shellcheck and golangci-lint
	shellcheck -x -P SCRIPTDIR bin/gellee-* lib/gellee.sh tests/helper.bash ci/public-gate ci/scripts/* ci/hooks/*
	golangci-lint run

.PHONY: fmt-check
fmt-check: ## Fail if any Go file needs gofmt
	@out=$$(gofmt -l cmd internal); if [ -n "$$out" ]; then echo "gofmt needed on:"; echo "$$out"; exit 1; fi

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: tidy-check
tidy-check: ## Fail if go.mod is not tidy
	go mod tidy -diff

web/node_modules/.package-lock.json: web/package-lock.json
	cd web && npm ci

.PHONY: web
web: web/node_modules/.package-lock.json ## Build the interface into web/dist/app
	cd web && npm run build

.PHONY: web-check
web-check: web/node_modules/.package-lock.json ## Lint, type-check and test the interface
	cd web && npm run lint
	cd web && npm run typecheck
	cd web && npm test

.PHONY: build
build: web ## Build the binary into bin/
	CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o bin/$(BINARY) ./cmd/gellee

PLAYWRIGHT_IMAGE := mcr.microsoft.com/playwright:v1.64.1-noble

.PHONY: e2e-web
e2e-web: build ## Run the Playwright suite in its pinned image
	docker run --rm --ipc=host --init -v "$(CURDIR)":/src -w /src/web -u "$$(id -u):$$(id -g)" -e HOME=/tmp \
	  $(PLAYWRIGHT_IMAGE) npx playwright test $(E2E_ARGS)

.PHONY: readme-shots
readme-shots: build ## Capture the demo overview for the README into docs/assets
	docker run --rm --ipc=host --init -v "$(CURDIR)":/src -w /src/web -u "$$(id -u):$$(id -g)" -e HOME=/tmp \
	  $(PLAYWRIGHT_IMAGE) node scripts/readme-shots.mjs

CI_IMAGE ?= gellee-ci

.PHONY: ci-image
ci-image: ## Build the image the workflows run their gates in
	docker build --build-arg PLAYWRIGHT_IMAGE=$(PLAYWRIGHT_IMAGE) \
	  -f ci/toolchain.Dockerfile -t $(CI_IMAGE) .

# The name the image is built and pushed under, registry included:
# make image-push IMAGE=registry.internal/tools/gellee
IMAGE ?= gellee
IMAGE_INPUTS := Dockerfile go.mod cmd internal web
IMAGE_TAG = $(shell git rev-parse --short HEAD)

.PHONY: image
image: ## Build the container image of the current commit, tagged with its short hash
	@dirty=$$(git status --porcelain -- $(IMAGE_INPUTS)); if [ -n "$$dirty" ]; then \
	  echo "the image is named after a commit; commit or remove these first:"; echo "$$dirty"; exit 1; fi
	docker build --label org.opencontainers.image.revision=$$(git rev-parse HEAD) -t $(IMAGE):$(IMAGE_TAG) .

.PHONY: image-push
image-push: image ## Push the image and print the digest to deploy
	docker push $(IMAGE):$(IMAGE_TAG)
	@docker image inspect --format '{{range .RepoDigests}}{{println .}}{{end}}' $(IMAGE):$(IMAGE_TAG) | grep -F '$(IMAGE)@'

.PHONY: check
check: fmt-check vet lint tidy-check test web-check web lint-docs lint-ci ## Every local gate

.PHONY: hooks
hooks: ## Run the publication gate before every push to the public repository
	git config core.hooksPath ci/hooks

.PHONY: playwright-image
playwright-image: ## Print the Playwright image the browser tests run in
	@echo $(PLAYWRIGHT_IMAGE)

.PHONY: lint-docs
lint-docs: ## Lint the tracked Markdown files
	markdownlint-cli2 $$(git ls-files '*.md' ':!tests/fixtures')

.PHONY: lint-ci
lint-ci: ## Lint the workflows
	@if [ -d .gitea/workflows ]; then \
	  echo actionlint .gitea/workflows; \
	  actionlint -config-file .gitea/actionlint.yaml .gitea/workflows/*.yml; fi
	@if [ -d .github/workflows ]; then \
	  echo actionlint .github/workflows; actionlint .github/workflows/*.yml; fi
	@if [ -d .github ]; then echo zizmor .github; zizmor --offline .github; fi

GOVULNCHECK := golang.org/x/vuln/cmd/govulncheck@v1.9.0

.PHONY: vuln
vuln: web/node_modules/.package-lock.json ## Look for known vulnerabilities in the Go and npm dependencies
	go run $(GOVULNCHECK) ./...
	cd web && npm audit --audit-level=high
