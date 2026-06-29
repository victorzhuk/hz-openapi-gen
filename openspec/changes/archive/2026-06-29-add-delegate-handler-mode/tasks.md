## 1. CLI and Options

- [x] 1.1 Add a handler mode option with supported values `stub` and `delegate`, defaulting to `stub`.
- [x] 1.2 Add explicit delegate flags for import path, package alias/name, and function name (no inference, no basename default).
- [x] 1.3 Validate delegate configuration before generation: require import and function; require package and function to be valid Go identifiers; reject a package name that collides with `context`, `app`, `consts`, or `opid`.
- [x] 1.4 Add CLI tests for default mode, valid delegate mode, missing delegate import, invalid identifier, and reserved package name.

## 2. Generator

- [x] 2.1 Extend generator options with handler mode and delegate target settings.
- [x] 2.2 Add a delegate handler template that calls the configured function with `ctx`, `c`, and the operationId constant.
- [x] 2.3 Generate an operationId constants file (cover mode, DO-NOT-EDIT) with one exported constant per operation: exported-operationId name, verbatim operationId value.
- [x] 2.4 Fail generation in delegate mode when any selected operation has an empty operationId, regardless of strict.
- [x] 2.5 Switch handler files to cover mode with the DO-NOT-EDIT marker when delegate mode is enabled.
- [x] 2.6 Preserve existing merge-mode handler generation when handler mode is `stub`.

## 3. Writer

- [x] 3.1 Accept the generated marker for cover-mode overwrite so a stub→delegate switch proceeds without global `-force`.
- [x] 3.2 Surface a diagnostic on stub regeneration when an existing handler body was generated in delegate mode (delegate→stub), instead of silently keeping it.

## 4. Tests and Fixtures

- [x] 4.1 Add golden output for delegate mode, including the operationId constants file.
- [x] 4.2 Add generator tests proving delegate handlers contain no `StatusNotImplemented` or `BindAndValidate` and reference the generated constants rather than string literals.
- [x] 4.3 Add writer tests proving delegate-mode handler files are replaced rather than merged, that stub→delegate proceeds without `-force`, and that delegate→stub emits the diagnostic.
- [x] 4.4 Extend generated-service compile tests with a small delegate package and the constants file.

## 5. Documentation

- [x] 5.1 Update `README.md` flags and examples.
- [x] 5.2 Update `docs/architecture.md` with the new handler mode, the constants file, and write-mode behavior.
- [x] 5.3 Update `docs/developer-guide.md` and `docs/runbook.md` with delegate-mode regeneration, mode-switch behavior, and troubleshooting.
- [x] 5.4 Add a changelog entry.

## 6. Validation

- [x] 6.1 Run `make test-unit`.
- [x] 6.2 Run `make test-generated`.
- [x] 6.3 Run `make build`.
- [x] 6.4 Run `make lint`.
- [x] 6.5 Run `openspec validate add-delegate-handler-mode --strict`.
