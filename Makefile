SHELL:=/usr/bin/env bash

.DEFAULT_GOAL := build

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

.PHONY: build
build: ## Build stern binary
	go build -o dist/stern .

TOOLS_BIN_DIR := $(CURDIR)/hack/tools/bin
GORELEASER_VERSION ?= v2.12.0
GORELEASER := $(TOOLS_BIN_DIR)/goreleaser
GOLANGCI_LINT_VERSION ?= v2.4.0
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
VALIDATE_KREW_MAIFEST_VERSION ?= v0.4.5
VALIDATE_KREW_MAIFEST := $(TOOLS_BIN_DIR)/validate-krew-manifest
GORELEASER_FILTER_VERSION ?= v0.3.0
GORELEASER_FILTER := $(TOOLS_BIN_DIR)/goreleaser-filter

$(GORELEASER):
	GOBIN=$(TOOLS_BIN_DIR) go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

$(GOLANGCI_LINT):
	GOBIN=$(TOOLS_BIN_DIR) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(VALIDATE_KREW_MAIFEST):
	GOBIN=$(TOOLS_BIN_DIR) go install sigs.k8s.io/krew/cmd/validate-krew-manifest@$(VALIDATE_KREW_MAIFEST_VERSION)

$(GORELEASER_FILTER):
	GOBIN=$(TOOLS_BIN_DIR) go install github.com/t0yv0/goreleaser-filter@$(GORELEASER_FILTER_VERSION)

.PHONY: build-cross
build-cross: $(GORELEASER) ## Build cross-platform binaries
	$(GORELEASER) build --snapshot --clean

.PHONY: test
test: lint ## Run tests with linting
	go test -v ./...

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: $(GOLANGCI_LINT) ## Run linter with auto-fix
	$(GOLANGCI_LINT) run --fix

README_FILE ?= ./README.md

.PHONY: update-readme
update-readme: ## Update README with generated content
	go run hack/update-readme/update-readme.go $(README_FILE)

.PHONY: verify-readme
verify-readme: ## Verify README is up to date
	./hack/verify-readme.sh

.PHONY: validate-krew-manifest
validate-krew-manifest: $(VALIDATE_KREW_MAIFEST) ## Validate Krew manifest
	$(VALIDATE_KREW_MAIFEST) -manifest dist/krew/stern.yaml -skip-install

.PHONY: dist
dist: $(GORELEASER) $(GORELEASER_FILTER) ## Build distribution for current platform
	cat .goreleaser.yaml | $(GORELEASER_FILTER) -goos $(shell go env GOOS) -goarch $(shell go env GOARCH) | $(GORELEASER) release -f- --clean --skip=publish --snapshot

.PHONY: dist-all
dist-all: $(GORELEASER) ## Build distribution for all platforms
	$(GORELEASER) release --clean --skip=publish --snapshot

.PHONY: release
release: $(GORELEASER) ## Create a new release
	$(GORELEASER) release --clean --skip=validate

.PHONY: clean
clean: clean-tools clean-dist ## Clean all build artifacts

.PHONY: clean-tools
clean-tools: ## Clean downloaded tools
	rm -rf $(TOOLS_BIN_DIR)

.PHONY: clean-dist
clean-dist: ## Clean distribution files
	rm -rf ./dist
