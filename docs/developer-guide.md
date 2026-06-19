# Developer Guide


| Command | Purpose |
|---------|---------|
| `make test-unit` | Run unit + golden tests, skip the network-dependent compile gate |
| `make build` | Compile source packages |
| `make lint` | Validate `.golangci.yml` and run the curated enterprise lint/static-analysis/security gate |
| `make vulncheck` | Scan reachable source packages for known vulnerabilities |
| `make test-generated` | Generate real services into temp dirs and run `go mod tidy`, `go build ./...`, and `go vet ./...` |
| `make golden` | Refresh generator golden snapshots |

## Adding OpenAPI extraction behavior

Extraction lives entirely in `internal/openapi/*`. Changes go here when you need to:

- Add support for a new OpenAPI construct (new schema types, parameter styles, response media types).
- Change how Go names or types are computed.
- Add or change diagnostic codes.

**Rules:**

- All Go names (handler names, model names, field names) must be computed during extraction and stored in `SpecModel`. Never compute names during rendering.
- Add tests in `internal/openapi/*_test.go`. Use `testdata/minimal.yaml` for targeted tests and `testdata/petstore.yaml` for integration-level coverage.
- New diagnostic codes follow the `SCHEMA_*` / `OPERATION_*` convention in `internal/openapi/schema.go`.

## Changing generated templates

Templates are defined as Go string constants in `internal/generator/templates.go`:

| Constant | Contains | Rendered by |
|----------|----------|-------------|
| `routerTmpl` | `{{define "router"}}` — full router file | `render("router", …)` |
| `handlerTmpl` | `{{define "handler"}}` — full handler file skeleton | `render("handler", …)` |
| `handlerFuncTmpl` | `{{define "handlerFunc"}}` — single handler function | `renderHandlerFunc(…)` |
| `modelTmpl` | `{{define "model"}}` — full model file | `render("model", …)` |
| `mainTmpl` | `{{define "main"}}` — full main.go | `render("main", …)` |
| `generateTmpl` | `{{define "generate"}}` — full generate.go | `render("generate", …)` |

**Critical rule:** `renderHandlerFunc` returns a raw Go function snippet — it does **not** call `hzutil.FormatGo`. Do not change `renderHandlerFunc` to format its output. `go/format.Source` requires a complete Go file (with `package` declaration), and formatting a bare function snippet will fail. The function snippet is later appended to a complete handler file, and only the final merged file is formatted by `writer.mergeGoFile`.

When you change a template, regenerate golden snapshots:

```
make golden
go test ./internal/generator/...
```

## Adding a generated file type

Before writing code, decide which of the three existing modes fits:

| If the file… | Use |
|--------------|-----|
| Is fully owned by the generator, can be replaced entirely | `WriteCover` |
| Should be written once and never touched again | `WriteCreate` |
| Should be extended incrementally (append missing pieces) | `WriteMerge` |

If no existing mode fits, add a new `WriteMode` constant in `internal/generator/generator.go`, add the corresponding branch in `internal/writer/writer.go:Write`, and write writer tests before generator tests. The writer is the safety boundary — test it first.

## Updating handler merge behavior

Handler merge is coordinated across three places:

1. `generator.GoMergePlan` — the merge plan structure in `generator.go`.
2. `writer.mergeGoFile` — the merge implementation in `writer.go`.
3. `writer/writer_test.go` — writer tests covering merge.

Duplicate detection uses `hzutil.HasGoFunction`, which matches `func Name(` (including pointer receivers `func (s *S) Name(`) with a regex. It does not parse the AST — this is deliberate to avoid dependency on the file being parseable during detection.

Import insertion uses `hzutil.AddImportForContent`, which parses the file with `go/parser`, inserts the import with `astutil.AddNamedImport`, and formats with `format.Node`. If the import path string is already present, insertion is skipped.

When changing merge behavior, update the writer test `TestWriterMerge` (or equivalent), then update the generator test to reflect the new plan shape.

## Golden files

Golden snapshots live under `testdata/golden/` and are generated from `testdata/minimal.yaml`.

**Regenerate:**

```
make golden
```

**Verify:**

```
go test ./internal/generator/...
```

Golden tests compare the rendered output byte-for-byte. Any template, name computation, or type-mapping change that affects output requires a golden update.

## Adding fixtures

- Place new OpenAPI specs in `testdata/*.yaml`.
- If the spec should produce golden snapshots, run `make golden` after adding it. A new golden subdirectory will be created under `testdata/golden/` based on the spec filename.
- If the spec is only for extraction or writer tests, no golden update is needed.
