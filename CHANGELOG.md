# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Changed

- **Breaking:** `main.go` moved from `cmd/hz-openapi-gen/` to repo root. Install with `go install github.com/victorzhuk/hz-openapi-gen@latest` (no `/cmd/hz-openapi-gen` suffix needed).

## [0.1.2] - 2026-06-19

### Changed

- **Breaking:** Module path migrated from `gitlab.wildberries.ru/travel/travel-backend/adv/core/hz-openapi-gen` to `github.com/victorzhuk/hz-openapi-gen`. Install with `go install github.com/victorzhuk/hz-openapi-gen/cmd/hz-openapi-gen@latest`.

### Fixed

- `gofmt` violation: missing blank line between `const` block and `var` declaration.

## [0.1.1] - 2026-06-19

### Fixed

- `gofmt` violation in `cmd/hz-openapi-gen/main.go`.

## [0.1.0] - 2026-06-19

### Added

- OpenAPI 3.x to CloudWeGo Hertz generator CLI (`main.go`) that emits routers, handler stubs, models, and optional `main.go` / `generate.go` files.
- Direct OpenAPI parsing with `pb33f/libopenapi`, without routing generation through `hz`, `thriftgo`, or `protoc`.
- Deterministic extraction for operation names, Hertz paths, schema models, parameters, responses, diagnostics, and unsupported schema warnings.
- Generated file safety modes: replace generated routers/models, create optional entrypoint files once, and merge missing handler functions while preserving custom handler code.
- Development and CI quality gates for build, short tests, generated-service compilation, linting, and `govulncheck`.
- Project documentation for architecture, developer workflow, and runbook procedures.

[Unreleased]: https://github.com/victorzhuk/hz-openapi-gen/compare/v0.1.2...main
[0.1.2]: https://github.com/victorzhuk/hz-openapi-gen/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/victorzhuk/hz-openapi-gen/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/victorzhuk/hz-openapi-gen/releases/tag/v0.1.0
