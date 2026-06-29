## ADDED Requirements

### Requirement: Delegate handler mode

The generator SHALL provide an opt-in delegate handler mode. In delegate handler mode, every generated operation handler SHALL call a configured delegate function with the current `context.Context`, the Hertz `*app.RequestContext`, and the operation's OpenAPI `operationId` passed as a generated constant.

#### Scenario: Handler delegates with operationId

- **WHEN** the generator runs with delegate handler mode enabled for an operation whose `operationId` is `Adventures_GetProductById`
- **THEN** the generated handler calls the configured delegate function
- **AND** the call passes `ctx`, `c`, and the generated operationId constant whose value is `"Adventures_GetProductById"`

#### Scenario: Handler contains no stub logic

- **WHEN** a handler file generated in delegate mode is inspected
- **THEN** it contains no `StatusNotImplemented` response
- **AND** it contains no request model binding before the delegate call

### Requirement: Generated operationId constants

In delegate handler mode, the generator SHALL emit a constants file declaring one exported string constant per operation. Each constant's name SHALL be the exported form of the `operationId` and its value SHALL be the verbatim `operationId`. Delegate handlers SHALL reference these constants instead of string literals. The constants file SHALL be a cover-mode file carrying the DO-NOT-EDIT marker.

#### Scenario: Constant generated per operation

- **WHEN** the generator runs in delegate mode for an operation whose `operationId` is `Adventures_GetProductById`
- **THEN** the constants file declares an exported constant named `AdventuresGetProductByID` with value `"Adventures_GetProductById"`
- **AND** the generated handler passes that constant to the delegate function

#### Scenario: Constants are the single source of dispatch keys

- **WHEN** a handler file generated in delegate mode is inspected
- **THEN** no handler body contains an `operationId` string literal
- **AND** every handler passes a constant from the generated constants file

### Requirement: Delegate target is configurable and validated

The generator SHALL require a delegate import path and delegate function name when delegate handler mode is enabled. It SHALL validate that the delegate package name and function name are valid Go identifiers, and that the delegate package name does not collide with the import names used by generated handlers, before writing files.

#### Scenario: Missing delegate import fails

- **WHEN** delegate handler mode is enabled without a delegate import path
- **THEN** generation fails before writing files
- **AND** the error names the missing delegate import configuration

#### Scenario: Invalid delegate function name fails

- **WHEN** delegate handler mode is enabled with a delegate function name that is not a valid Go identifier
- **THEN** generation fails before writing files
- **AND** no generated file is changed

#### Scenario: Reserved delegate package name fails

- **WHEN** delegate handler mode is enabled with a delegate package name that collides with a reserved handler import name (`context`, `app`, `consts`, or `opid`)
- **THEN** generation fails before writing files
- **AND** no generated file is changed

### Requirement: Delegate mode requires operation identifiers

In delegate handler mode the generator SHALL fail when any selected operation has an empty `operationId`, regardless of the strict setting, before writing files.

#### Scenario: Empty operationId fails in delegate mode

- **WHEN** delegate handler mode is enabled and a selected operation has no `operationId`
- **THEN** generation fails before writing files
- **AND** the error names the operation missing an `operationId`

### Requirement: Delegate handlers are fully generated files

In delegate handler mode, handler files SHALL be generated cover-mode files with the DO-NOT-EDIT marker. Regeneration SHALL replace handler files from the OpenAPI document instead of merge-appending functions.

#### Scenario: Delegate handler regeneration replaces stale content

- **WHEN** a delegate-mode handler file already exists with stale generated content
- **AND** the generator runs again with delegate handler mode enabled
- **THEN** the handler file is replaced with fresh generated content
- **AND** no merge-mode stub is appended

#### Scenario: Switching a stub service to delegate mode replaces handlers

- **WHEN** handler files previously generated in stub mode exist and carry the generated marker
- **AND** the generator runs again with delegate handler mode enabled
- **THEN** the handler files are replaced with delegate content
- **AND** no force override is required

#### Scenario: Switching delegate handlers back to stub warns

- **WHEN** handler files previously generated in delegate mode exist
- **AND** the generator runs again in stub mode
- **THEN** the generator emits a diagnostic that an existing handler body was generated in delegate mode and was not replaced

### Requirement: Stub mode remains backward compatible

The existing stub handler mode SHALL remain the default. When delegate handler mode is not enabled, handler files SHALL keep their existing merge-mode behavior and default `501 Not Implemented` bodies.

#### Scenario: Default generation is unchanged

- **WHEN** the generator runs without delegate handler mode
- **THEN** generated handler files use merge mode
- **AND** missing handlers are generated with the existing `501 Not Implemented` stub behavior
