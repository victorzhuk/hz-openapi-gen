## Context

`hz-openapi-gen` currently renders three generated areas:

- router files: cover mode, DO-NOT-EDIT;
- model files: cover mode, DO-NOT-EDIT;
- handler files: merge mode, generated marker, default `501 Not Implemented` bodies.

Merge-mode handlers are useful for normal applications, where developers replace stubs with business logic. They are awkward for proxy/gateway consumers that want all operations to call one runtime shim and never hand-edit generated handler files.

Adventure Gateway needs this mode to move its generated public edge under `internal/api/gen/hertz` without carrying manual delegate edits.

The direction and the decisions below were aligned in `docs/brainstorms/add-delegate-handler-mode.md` (a design pressure-test of this change). The brief is the source for the operationId-constants, mode-switch, and validation decisions added here.

## Goals / Non-Goals

**Goals:**

- Add an opt-in delegate handler mode.
- Preserve existing default output and merge semantics.
- Make delegate handler files fully generated and reproducible.
- Pass the OpenAPI `operationId` to the delegate through a generated constant, so the dispatch key has a single source of truth and renames surface as compile errors in callers rather than silent runtime misses.
- Make switching a service between stub and delegate mode safe and predictable.
- Keep generated code compile-safe with explicit imports and clear flag validation.

**Non-Goals:**

- Generate a full ogen-style typed `Handler` interface.
- Remove model generation.
- Add custom per-operation business logic.
- Change OpenAPI extraction semantics.

## Decisions

### Add a handler mode option

Add a generator option and CLI flag `-handler-mode=stub|delegate`. `stub` remains the default. `delegate` switches handler rendering and write mode. All delegate configuration is explicit — mode is never inferred from the presence of another flag, and the delegate package name is never defaulted from the import basename. Explicit flags keep `--help` self-documenting and leave room for future modes.

Alternative considered: infer delegate mode from a delegate import flag. Rejected because explicit mode produces clearer validation and help output.

### Use a simple delegate function contract

Delegate-mode handlers call a configured function with this shape:

```go
func Serve(ctx context.Context, c *app.RequestContext, operationID string)
```

The generator renders:

```go
func AdventuresGetProductByID(ctx context.Context, c *app.RequestContext) {
    public.Serve(ctx, c, opid.AdventuresGetProductByID)
}
```

Flags configure the import path, package alias, and function name:

```sh
-handler-mode=delegate \
-delegate-import=gitlab.example/project/internal/public \
-delegate-package=public \
-delegate-func=Serve
```

A typed per-operation interface was rejected: for a single-shim proxy it is N ceremony methods all calling `Serve`, with no safety gain, and it breaks compilation on every new spec operation. The generic shim absorbs spec churn, which is the driver's requirement.

### Dispatch keys are generated operationId constants

Delegate handlers reference a generated constant, not a string literal. The generator emits a constants file (e.g. `biz/opid/opid.go`, cover mode, DO-NOT-EDIT) with one exported constant per operation: the name is `Exported(operationId)` (the same identifier as the handler func name), and the value is the verbatim `operationId`.

This makes the dispatch key a single source of truth. A spec rename changes the constant, so a gateway switching on the constant fails to compile instead of drifting into a silent runtime `404`. It also removes the smell of a Go-normalized func name sitting next to a raw spec string on adjacent lines. The value stays verbatim because the gateway keys its upstream routing on spec-level `operationId`.

Alternative considered: bare string literals in handler bodies. Rejected because renames drift silently with no compile or test signal — the exact gap oapi-codegen's strict-server was built to close.

### Delegate handlers use cover mode

In delegate mode, handler files are generated output and carry the DO-NOT-EDIT marker. Regeneration replaces them completely. This avoids stale manual bodies and prevents new operations from appearing as `501` stubs.

Alternative considered: keep merge mode and append delegate functions only for missing handlers. Rejected because it keeps the same stale-body problem the mode is meant to remove.

### Mode switches between stub and delegate are safe

Today the writer blocks a stub→delegate switch and silently mishandles delegate→stub:

- stub→delegate: cover mode requires `DoNotEditMarker`, but stub files carry the shorter `GeneratedMarker`, so the write is refused with `ErrUnsafeOverwrite`; the only escape, `-force`, bypasses marker checks for every file globally.
- delegate→stub: merge sees the functions already exist and silently keeps the delegate bodies, with no warning.

Decision: cover mode accepts any file carrying the tool's `GeneratedMarker` (which `DoNotEditMarker` contains) as overwritable. Stub→delegate then regenerates the tool-owned handler files cleanly without global `-force`. On delegate→stub, the writer surfaces a warning (in its result, reported by `main`) when an existing handler body carries the delegate-mode marker (i.e. it was generated in delegate mode), instead of silently leaving it.

Trade-off: a stub handler the developer hand-filled with real logic still carries `GeneratedMarker`, so switching that service to delegate mode overwrites it. This is acceptable and intended — delegate mode is opt-in and owns the handler files; a service with real business logic is not a candidate for delegate mode. Documented in the runbook.

### Validate delegate configuration before writing

The writer is all-or-nothing, so delegate configuration is validated in `main` before `Generate` runs, and operation identifiers are checked in `Generate` before any file is produced:

- `-delegate-import` and `-delegate-func` are required in delegate mode.
- `-delegate-package` and `-delegate-func` must be valid Go identifiers.
- `-delegate-package` must not collide with the import names the handler template already uses: `context`, `app`, `consts`, or the `opid` constants package.
- In delegate mode every selected operation must have a non-empty `operationId`, regardless of `-strict` (otherwise the body would be `Serve(ctx, c, opid.X)` with an empty-valued constant).

### Default stub mode stays unchanged

When `-handler-mode` is omitted or set to `stub`, generated output and write modes remain compatible with existing users.

## Risks / Trade-offs

- [Handler files become overwrite-owned in delegate mode] → This is opt-in and documented; default mode is unchanged.
- [Stub→delegate overwrites hand-edited stub bodies] → Intended and documented; delegate mode is not for services carrying real handler logic.
- [Too many delegate flags] → Keep the contract small: import path, package alias/name, function name.
- [Invalid generated import aliases] → Validate identifiers and reserved-name collisions before rendering; cover with CLI tests.
- [Users want other signatures] → Defer generalized templates until there is a second concrete consumer.

## Migration Plan

1. Add options and CLI flags with strict validation.
2. Add the delegate handler template, the operationId constants file, and write-mode selection.
3. Adjust the writer for safe mode switches (cover accepts the generated marker; delegate→stub warns).
4. Add golden fixtures for default and delegate modes, including the constants file.
5. Add generated-service compile tests for delegate mode using a tiny local delegate package.
6. Update docs and cut a release for downstream gateway consumption.

## Open Questions

- Whether to also emit a `ValidOperationIDs()` slice (or expose the full constant set) so the gateway can assert its `Serve` switch covers every generated operation. Deferred until the gateway asks for it.
