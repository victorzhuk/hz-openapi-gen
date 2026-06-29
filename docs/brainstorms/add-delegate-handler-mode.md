# Brainstorm: delegate handler mode

Design pressure-test of the existing OpenSpec change `add-delegate-handler-mode`.
The core design was confirmed; four deltas were agreed. This brief is the handoff
into spec authoring — it records the decisions so the spec update does not
re-investigate them.

## Framing

`hz-openapi-gen` generates Hertz handler scaffolds. Today handler bodies are
`501 Not Implemented` stubs written in merge mode: developers fill them in by
hand, and every new operation regenerates as a fresh stub to patch again.

Adventure Gateway uses the tool as a proxy/edge generator. It wants every handler
to forward to one runtime shim, with handler files fully owned by the generator
and reproducible — no manual patching when operations are added or removed.

The change adds an opt-in delegate mode. Default stub behaviour is untouched.

## Approach

The written design stands on its load-bearing decisions:

- **Generic dispatch.** Delegate handlers call one configured function
  `Serve(ctx, c, operationID)` rather than a typed per-operation interface. For a
  proxy that forwards everything to a single shim, a typed interface is N methods
  of ceremony with no safety gain, and it breaks compilation on every spec
  addition. The generic shim absorbs spec churn — which is the point.
- **Cover mode + DO-NOT-EDIT for delegate handler files.** Same ownership model as
  router and model files. Merge mode would leave stale handlers when operations
  are removed, the exact problem this mode exists to kill.
- **Default stub mode unchanged**, opt-in via flag.

Four deltas to fold into the change:

1. **operationId constants.** Generate `biz/opid/opid.go` (cover mode,
   DO-NOT-EDIT): one constant per operation, named with `Exported(operationId)`,
   valued with the verbatim spec `operationId`. Delegate handler bodies reference
   the constant:

   ```go
   func AdventuresGetProductByID(ctx context.Context, c *app.RequestContext) {
       public.Serve(ctx, c, opid.AdventuresGetProductByID)
   }
   ```

   This makes the dispatch key a single source of truth. A spec rename changes the
   constant, so a gateway switching on the constant fails to compile instead of
   drifting into a silent runtime 404. It also removes the smell of a Go-normalized
   func name sitting next to a raw spec string on adjacent lines. The value stays
   verbatim because the gateway keys its upstream routing on spec-level
   `operationId`. Touches `internal/generator/generator.go` and `templates.go`
   (new template, new generated file, new render call).

2. **All four flags explicit.** `-handler-mode=stub|delegate`, `-delegate-import`,
   `-delegate-package`, `-delegate-func`, exactly as design.md already has them.
   No inference of mode from import presence, no defaulting of the package from the
   import basename. Explicit keeps `--help` self-documenting and leaves room for
   future modes.

3. **Mode-switch handling** in `internal/writer/writer.go`. The current writer has
   two sharp edges on a mode switch:
   - stub → delegate: cover mode checks for `DoNotEditMarker`, but stub files carry
     the shorter `GeneratedMarker`, so the switch is blocked by `ErrUnsafeOverwrite`
     and `-force` is too blunt (it bypasses marker checks for all files globally).
   - delegate → stub: merge sees the functions already exist and silently keeps the
     delegate bodies, with no warning.

   Fix: cover mode accepts the tool's own `GeneratedMarker` as overwritable, so
   stub → delegate regenerates cleanly without global `-force`. On delegate → stub,
   emit a diagnostic when an existing body is a delegate call rather than silently
   leaving it.

4. **Validation hardening** in `main.go`, before `Generate` and before any write
   plan is applied (the writer is all-or-nothing):
   - `-delegate-package` and `-delegate-func` must be valid Go identifiers.
   - `-delegate-package` must not collide with the reserved import names already in
     the handler template: `context`, `app`, `consts`. A collision compiles past
     `go/format` but fails `go build`.
   - In delegate mode, every operation must have a non-empty `operationId`,
     regardless of `-strict`. Otherwise the body is `Serve(ctx, c, "")`, which
     compiles and dispatches to nothing.

Layer separation holds: `OpDef` already carries both `OperationID` and `FuncName`,
so extraction is untouched. All new behaviour is render-time config on
`generator.Options` plus the writer and validation changes above.

## Rejected

- **Typed per-operation `Handler` interface (ogen-style).** Forces the proxy to
  implement one method per operation, every body identical, and breaks compilation
  on spec growth. No safety gain for a single-shim proxy. Already a stated non-goal.
- **Bare string literals as the dispatch key.** Simplest, but spec renames drift
  silently into runtime 404s with no compile or test signal. Superseded by the
  opid constants.
- **Normalized `FuncName` as the dispatch key.** Internally consistent with the Go
  func name but diverges from the spec-level `operationId` the gateway keys on.
- **Inferred or minimal flag set.** Inferring delegate mode from `-delegate-import`
  presence and defaulting the package from the import basename was considered and
  rejected in favour of explicit flags.

## Open questions

- Optionally emit a `ValidOperationIDs()` slice (or the `opid` package's full
  constant set) so the gateway can write a coverage test asserting its `Serve`
  switch handles every generated operation. Recommend deferring until the gateway
  asks for it.
