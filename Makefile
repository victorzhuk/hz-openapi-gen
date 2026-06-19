DOTENV ?= .env
DOTENV_FILE := $(firstword $(wildcard $(DOTENV) .env.example))

ifneq (,$(DOTENV_FILE))
include $(DOTENV_FILE)
export
endif

BIN_DIR ?= bin
BINARY  := $(BIN_DIR)/hz-openapi-gen

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

SRC_PKGS := . ./internal/...

.DEFAULT_GOAL := help

.PHONY: help build install clean \
        fmt lint lint-new test test-unit test-race test-generated \
        verify-mod vulncheck golden

# ── Help ──────────────────────────────────────────────────────────────

## help: show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | column -t -s ':'

# ── Build ─────────────────────────────────────────────────────────────

## build: compile the binary into bin/ (static)
build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -trimpath -o $(BINARY) .

## install: install the binary to $GOPATH/bin
install:
	go install $(LDFLAGS) .

## clean: remove build artifacts
clean:
	@rm -rf $(BIN_DIR)

# ── Quality ───────────────────────────────────────────────────────────

## fmt: format Go source files (gofmt + goimports from .golangci.yml)
fmt:
	golangci-lint fmt $(SRC_PKGS)

## lint: verify config and run the linter (source packages only)
lint:
	golangci-lint config verify
	golangci-lint run $(SRC_PKGS)

## lint-new: run the linter on new/modified code only (since master)
lint-new:
	golangci-lint run --new-from-rev=origin/master $(SRC_PKGS)

## test: run all tests
test:
	go test ./...

## test-unit: run unit tests only (-short)
test-unit:
	go test -short ./...

## test-race: run all tests under the race detector
test-race:
	go test -race ./...

## test-generated: build and vet the generated golden service (needs network)
test-generated:
	go test . -run TestGeneratedServiceCompiles -count 1

## verify-mod: tidy modules and verify they are clean
verify-mod:
	go mod tidy
	go mod verify

## vulncheck: scan for known vulnerabilities (source packages only)
vulncheck:
	go tool govulncheck $(SRC_PKGS)

# ── Generate ──────────────────────────────────────────────────────────

## golden: regenerate golden output snapshots from testdata specs
golden:
	UPDATE_GOLDEN=1 go test ./internal/generator/...
