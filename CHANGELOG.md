# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Changed

- **Breaking:** Module path migrated from `gitlab.wildberries.ru/travel/travel-backend/adv/core/hz-openapi-gen` to `github.com/victorzhuk/hz-openapi-gen` so the tool is installable via `go install github.com/victorzhuk/hz-openapi-gen/cmd/hz-openapi-gen@latest`.

### Fixed

- `gofmt` violation: missing blank line between `const` block and `var` declaration in `cmd/hz-openapi-gen/main.go`.

## [0.1.0] - 2026-06-19

### Added

- OpenAPI 3.x to CloudWeGo Hertz generator CLI (`cmd/hz-openapi-gen`) that emits routers, handler stubs, models, and optional `main.go` / `generate.go` files.
- Direct OpenAPI parsing with `pb33f/libopenapi`, without routing generation through `hz`, `thriftgo`, or `protoc`.
- Deterministic extraction for operation names, Hertz paths, schema models, parameters, responses, diagnostics, and unsupported schema warnings.
- Generated file safety modes: replace generated routers/models, create optional entrypoint files once, and merge missing handler functions while preserving custom handler code.
- Development and CI quality gates for build, short tests, generated-service compilation, linting, and `govulncheck`.
- Project documentation for architecture, developer workflow, and runbook procedures.

[Unreleased]: https://github.com/victorzhuk/hz-openapi-gen/compare/v0.1.0...main
[0.1.0]: https://github.com/victorzhuk/hz-openapi-gen/releases/tag/v0.1.0
