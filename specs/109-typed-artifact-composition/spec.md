# Feature Specification: Typed Artifact Composition

**Feature Branch**: `109-typed-artifact-composition`
**Created**: 2026-02-20
**Status**: Clarified
**Input**: GitHub Issue #109 - Capture step stdout as typed artifacts and enable pipeline composition

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Capture Step Stdout as Named Artifact (Priority: P1)

As a pipeline author, I want to declare that a step's stdout should be captured as a named, typed artifact so that downstream steps can consume structured output without relying on file-based workarounds.

**Why this priority**: This is the foundational capability that all other features depend on. Without stdout capture, the composition features cannot work. It addresses the most common use case: passing JSON output from one step to another.

**Independent Test**: Can be fully tested by running a single step that outputs JSON to stdout, then verifying the artifact is available with correct name, type, and content.

**Acceptance Scenarios**:

1. **Given** a step configuration with `output_artifacts: [{name: "report", source: stdout, type: json}]`, **When** the step completes and outputs valid JSON to stdout, **Then** an artifact named "report" is registered with the content from stdout and type "json".

2. **Given** a step with stdout artifact declaration and the step outputs text to stdout, **When** the step completes, **Then** the artifact content matches the exact stdout output (no JSON wrapper from adapter).

3. **Given** a step with stdout artifact declaration, **When** the step fails mid-execution, **Then** no partial stdout artifact is registered (atomicity guarantee).

---

### User Story 2 - Typed Artifact Consumption Declaration (Priority: P2)

As a pipeline author, I want to declare what artifact types a step consumes so that the orchestrator can validate dependencies exist before execution begins.

**Why this priority**: This enables fail-fast behavior and clear error messages. Without it, steps would fail at runtime when artifacts are missing, making debugging difficult.

**Independent Test**: Can be tested by declaring a step that consumes an artifact that does not exist and verifying the pipeline fails with a clear error before the step starts.

**Acceptance Scenarios**:

1. **Given** a step with `memory.inject_artifacts: [{step: prior, artifact: report, as: report, type: json}]` and the artifact "report" exists from step "prior", **When** the step begins execution, **Then** the artifact is injected and the step proceeds.

2. **Given** a step with `memory.inject_artifacts: [{step: prior, artifact: report, as: report, type: json}]` and no artifact named "report" exists, **When** the orchestrator evaluates the step, **Then** execution fails before the step starts with error: "required artifact 'report' not found".

3. **Given** a step with `memory.inject_artifacts: [{step: prior, artifact: report, as: report, type: json}]` and an artifact named "report" exists with type "text", **When** the orchestrator evaluates the step, **Then** execution fails before the step starts with error: "artifact 'report' type mismatch: expected json, got text".

---

### User Story 3 - Bidirectional Contract Validation (Priority: P3)

As a pipeline author, I want contracts to validate both inputs and outputs so that I can catch data quality issues at step boundaries rather than discovering them mid-execution.

**Why this priority**: This builds on P1 and P2 to provide schema-level validation. While P2 validates artifact existence and type, P3 validates the actual content structure.

**Independent Test**: Can be tested by declaring an input contract with a JSON schema and passing an artifact that violates the schema, verifying the step is rejected with schema validation errors.

**Acceptance Scenarios**:

1. **Given** a step with `memory.inject_artifacts: [{step: prior, artifact: data, as: data, schema_path: "./schemas/input.json"}]` and the injected artifact matches the schema, **When** the step begins, **Then** validation passes and execution proceeds.

2. **Given** a step with `memory.inject_artifacts: [{step: prior, artifact: data, as: data, schema_path: "./schemas/input.json"}]` and the injected artifact violates the schema, **When** the orchestrator validates input contracts, **Then** execution fails with detailed schema violation errors before the step runs.

3. **Given** a step with both input and output contracts, **When** the step completes, **Then** both contracts are validated in order: input contract before execution, output contract after execution.

---

### User Story 4 - Step-to-Step Artifact Piping (Priority: P3)

As a pipeline author, I want to reference a previous step's stdout artifact using template syntax so that I can build data transformation pipelines declaratively.

**Why this priority**: This is a convenience feature that builds on P1-P2. It provides syntactic sugar for common patterns but is not essential for basic functionality.

**Independent Test**: Can be tested by creating a two-step pipeline where step 2's prompt references `{{artifacts.step1-output}}` and verifying the content is substituted correctly.

**Acceptance Scenarios**:

1. **Given** step 1 produces stdout artifact "analysis" and step 2 prompt contains `{{artifacts.analysis}}`, **When** step 2 executes, **Then** the prompt contains the full content of the "analysis" artifact.

2. **Given** step 1 produces stdout artifact "data" with type "json" and step 2 injects it via `memory.inject_artifacts`, **When** step 2 executes, **Then** the artifact is available both as file injection and via prompt substitution.

---

### Edge Cases

- **Large stdout**: What happens when stdout exceeds configurable size limits (e.g., 10MB)?
  - Artifact creation fails with "stdout artifact too large" error; step continues but artifact is not registered.
- **Empty stdout**: What happens when stdout is empty?
  - Empty artifact is created (0 bytes); type validation still applies if declared.
- **Binary stdout**: What happens when stdout contains binary/non-UTF8 data?
  - For `type: application/octet-stream`, content is stored as-is; for text types (`application/json`, `text/plain`), invalid UTF8 sequences are replaced with replacement character.
- **Circular dependencies**: What happens when step A consumes from B and B consumes from A?
  - DAG validation already prevents this; error: "circular dependency detected".
- **Optional consumption**: What happens when an artifact is declared optional but missing?
  - Step proceeds; `{{artifacts.name}}` substitutes to empty string.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST capture step stdout as a named artifact when `source: stdout` is declared in the step's `output_artifacts` configuration.
- **FR-002**: System MUST write stdout artifacts to `.wave/artifacts/<step-id>/<artifact-name>` and register the path in `PipelineExecution.ArtifactPaths`.
- **FR-003**: System MUST make stdout artifacts available to downstream steps via existing `memory.inject_artifacts` mechanism.
- **FR-004**: System MUST allow steps to declare expected artifact types via the `type` field in `memory.inject_artifacts` entries.
- **FR-005**: System MUST validate that all injected artifacts exist before step execution begins.
- **FR-006**: System MUST validate that injected artifact types match declared types before step execution.
- **FR-007**: System MUST support input schema validation via an optional `schema_path` field in `memory.inject_artifacts` entries.
- **FR-008**: System MUST run input validation after artifact injection but before step execution; output contract validation runs after step completion.
- **FR-009**: System MUST enforce configurable stdout artifact size limits with sensible defaults (default: 10MB).
- **FR-010**: System MUST preserve stdout artifact content exactly without modification (no JSON wrapping).
- **FR-011**: System MUST support marking injected artifacts as `optional: true` to allow missing artifacts without failure.
- **FR-012**: System MUST resolve `{{artifacts.<name>}}` placeholders in step prompts to artifact content.

### Key Entities _(include if feature involves data)_

- **StdoutArtifact**: Runtime representation of captured stdout; attributes: `name`, `content`, `type`, `size`, `step_id`, `created_at`. Persisted to `.wave/artifacts/<step-id>/<name>`.
- **ArtifactRef (extended)**: Existing struct in `internal/pipeline/types.go:68-72`; extended with: `type` (string), `schema_path` (string), `optional` (bool).
- **ArtifactDef (extended)**: Existing struct in `internal/pipeline/types.go:97-102`; extended with: `source` (string: "stdout" | "file", default "file").

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A pipeline with stdout artifact capture and consumption runs end-to-end without manual file handling.
- **SC-002**: Pipeline fails with clear error message when a consumed artifact is missing (fail-fast, not mid-step).
- **SC-003**: Pipeline fails with clear error message when consumed artifact type does not match declaration.
- **SC-004**: Input contract validation errors include the same level of detail as existing output contract errors.
- **SC-005**: Stdout artifacts larger than configured limit (default 10MB) produce actionable error messages.
- **SC-006**: Documentation includes working examples of: (a) stdout capture, (b) typed consumption, (c) bidirectional contracts.

## Clarifications _(resolved)_

The following ambiguities were identified and resolved based on codebase analysis:

### Clarification 1: Stdout Artifact YAML Syntax Integration

**Ambiguity**: The spec proposed `artifacts: [{name: "report", source: stdout, type: application/json}]` but the existing codebase uses `output_artifacts` with a `path` field (see `internal/pipeline/types.go:97-102`).

**Resolution**: Extend the existing `output_artifacts` structure by adding an optional `source` field. When `source: stdout` is specified, the `path` field becomes optional (a generated path will be used internally). This maintains backward compatibility.

**Updated YAML syntax**:
```yaml
output_artifacts:
  - name: analysis-report
    source: stdout  # New field: "stdout" | "file" (default: "file")
    type: json      # Use short form to match existing schema
```

**Rationale**: The existing `ArtifactDef` structure in `types.go` and the JSON schema at `.wave/schemas/wave-pipeline.schema.json` already have the fields `name`, `path`, `type`, `required`. Adding `source` is additive and non-breaking. MIME types like `application/json` should be normalized to short forms (`json`, `text`, `markdown`) to match the existing schema enum.

### Clarification 2: Type Declaration Format

**Ambiguity**: The spec used MIME types (`application/json`) but the existing pipeline schema uses short forms (`json`, `text`, `markdown`).

**Resolution**: Use the existing short-form type names from the pipeline schema (`json`, `text`, `markdown`). Internally, these can map to MIME types for validation purposes, but the pipeline YAML should remain consistent.

**Type mapping**:
| Short form | MIME type |
|------------|-----------|
| `json` | `application/json` |
| `text` | `text/plain` |
| `markdown` | `text/markdown` |
| `binary` | `application/octet-stream` (new) |

**Rationale**: Consistency with existing `.wave/schemas/wave-pipeline.schema.json:294` which enumerates `["json", "text", "markdown"]`.

### Clarification 3: Consumption Declaration vs inject_artifacts

**Ambiguity**: The spec introduced a new `consumes` field but the existing system uses `memory.inject_artifacts` for artifact injection.

**Resolution**: Retain `memory.inject_artifacts` as the injection mechanism and add type validation to it. Do not introduce a parallel `consumes` field. Instead, extend `ArtifactRef` with an optional `type` and `schema_path` for input validation.

**Extended inject_artifacts syntax**:
```yaml
memory:
  inject_artifacts:
    - step: analyze
      artifact: report
      as: analysis_report
      type: json            # Optional: enables type checking
      schema_path: ./schemas/report.json  # Optional: enables schema validation
      optional: false       # Optional: default false
```

**Rationale**: The existing `inject_artifacts` mechanism is well-established in the codebase (see `internal/pipeline/types.go:64-72`). Adding fields to `ArtifactRef` is cleaner than introducing a parallel system. This also avoids confusion about which field to use.

### Clarification 4: Input Contract Validation Timing

**Ambiguity**: FR-008 specifies "input contract validation before step execution" but doesn't clarify the exact sequence relative to artifact injection.

**Resolution**: The execution sequence is:
1. Resolve artifact paths from prior steps
2. Inject artifacts into workspace (copy/symlink)
3. Validate input contracts against injected artifacts
4. Execute step
5. Validate output contracts

Input validation happens AFTER injection so the files are physically present for schema validation.

**Rationale**: Schema validators in `internal/contract/` operate on filesystem paths. Validating before injection would require holding artifact content in memory, which conflicts with the file-based design.

### Clarification 5: Stdout Artifact Storage

**Ambiguity**: The spec references "ArtifactPaths registry" but stdout artifacts have no natural filesystem path since they're captured from process output.

**Resolution**: Stdout artifacts are written to a generated path: `.wave/artifacts/<step-id>/<artifact-name>`. This path is then registered in `ArtifactPaths` just like file-based artifacts. The artifact is captured atomically—written only after step completion.

**Storage behavior**:
- Stdout is buffered during step execution (respecting size limits)
- On successful step completion, buffer is written to `.wave/artifacts/<step-id>/<artifact-name>`
- On step failure, stdout artifact is NOT written (atomicity guarantee per User Story 1, Scenario 3)
- The generated path is registered in `PipelineExecution.ArtifactPaths` using the same key format: `"<step-id>:<artifact-name>"`

**Rationale**: This reuses the existing `ArtifactPaths` map in `internal/pipeline/executor.go:127` without special-casing. File-based artifacts and stdout artifacts become indistinguishable to downstream consumers.
