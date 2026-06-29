## Why

Consumers that use `hz-openapi-gen` as an edge/proxy generator need reproducible handler bodies that delegate to their own runtime, not merge-mode `501` stubs that must be filled by hand after every new operation. Adventure Gateway currently carries that manual patching burden. The approach and the operationId-constants, mode-switch, and validation decisions come from the design pressure-test in `docs/brainstorms/add-delegate-handler-mode.md`.

## What Changes

- Add an opt-in delegate handler mode to the generator, configured with explicit flags.
- In delegate mode, generated handler functions call a configured delegate function with `(ctx, c, opid.<Operation>)`.
- Generate an operationId constants file as the single source of dispatch keys, so spec renames surface as compile errors in callers instead of silent runtime misses.
- Make stub↔delegate mode switches safe: cover mode regenerates tool-owned handler files without global `-force`, and stub regeneration warns when an existing delegate body would be kept.
- Validate delegate-mode flags (required import/function, valid Go identifiers, reserved package-name collisions) and require a non-empty `operationId` per operation, all before any file is written.
- Keep the current stub/merge behavior as the default for existing users.
- Add golden tests and generated-service compile tests for delegate mode.

## Capabilities

### New Capabilities

- `delegate-handler-generation`: generation mode for proxy-style handlers that delegate to a caller-provided function via generated operationId constants, instead of emitting `501` stubs.

### Modified Capabilities

- None.

## Impact

- CLI flags, delegate-mode validation, and option parsing in `main.go`.
- Generator options, delegate handler rendering, the operationId constants file, and write-mode selection in `internal/generator`.
- Cover-mode marker acceptance and the delegate-body diagnostic for safe mode switches in `internal/writer`.
- README, architecture, developer guide, and runbook docs.
- Testdata/golden coverage for the new mode, including the constants file.
