# Feature Specification: Restore and Stabilize `wave meta` Dynamic Pipeline Generation

**Feature Branch**: `095-restore-meta-pipeline`
**Created**: 2026-03-16
**Status**: Draft
**Input**: User description: "https://github.com/re-cinq/wave/issues/95"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Dry-Run Pipeline Generation (Priority: P1)

A developer wants to preview what pipeline `wave meta` would generate for a given task without actually executing it, so they can verify the pipeline design before committing resources.

**Why this priority**: Dry-run is the fastest way to validate the meta pipeline end-to-end — philosopher invocation, output parsing, semantic validation — without needing a full execution cycle. It is the foundational feedback loop for all other modes.

**Independent Test**: Can be fully tested by running `wave meta "<task>" --dry-run` and verifying that valid pipeline YAML is displayed with step names, personas, and contracts.

**Acceptance Scenarios**:

1. **Given** a project with a valid `wave.yaml` manifest containing a philosopher persona, **When** the user runs `wave meta "implement a logging feature" --dry-run`, **Then** the system displays a generated pipeline with at least one step, each step having a persona, contract, and fresh memory strategy.
2. **Given** a project with a valid manifest, **When** the user runs `wave meta "build a REST API" --dry-run`, **Then** the generated pipeline passes semantic validation (navigator-first step, all steps have contracts, all steps use fresh memory).
3. **Given** a project without a philosopher persona in the manifest, **When** the user runs `wave meta "any task" --dry-run`, **Then** the system exits with a clear error message indicating the philosopher persona is missing and how to add it.

---

### User Story 2 - Full Pipeline Execution (Priority: P1)

A developer wants to describe a task in natural language and have Wave design and execute a complete pipeline automatically, so they can get work done without manually authoring pipeline YAML.

**Why this priority**: This is the core value proposition of `wave meta` — dynamic pipeline generation and execution. Without this working, the feature has no practical utility.

**Independent Test**: Can be tested by running `wave meta "<task>"` and verifying that the generated pipeline executes to completion, with each step producing valid output artifacts.

**Acceptance Scenarios**:

1. **Given** a valid manifest with philosopher persona, **When** the user runs `wave meta "implement feature X"`, **Then** the philosopher generates a pipeline, the system validates it semantically, and executes all steps in dependency order.
2. **Given** a generated pipeline with multiple steps, **When** execution completes, **Then** each step's contract is validated and the final output is reported to the user.
3. **Given** a generated pipeline where a step fails contract validation, **When** the failure occurs, **Then** the system reports which step failed, what contract was violated, and does not mark the pipeline as successful.

---

### User Story 3 - Save Generated Pipeline for Reuse (Priority: P2)

A developer wants to save a dynamically generated pipeline as a reusable YAML file, so they can refine it, version-control it, or re-run it without invoking the philosopher again.

**Why this priority**: Reuse reduces cost (no redundant philosopher invocations) and enables iterative refinement of generated pipelines.

**Independent Test**: Can be tested by running `wave meta "<task>" --save <name>` and verifying a valid YAML file is written to the expected location with associated schema files.

**Acceptance Scenarios**:

1. **Given** a valid manifest, **When** the user runs `wave meta "build a CLI tool" --save my-cli-pipeline`, **Then** a pipeline YAML file is saved to `.wave/pipelines/my-cli-pipeline.yaml` and any associated contract schemas are saved to `.wave/contracts/`.
2. **Given** a saved pipeline, **When** the user runs `wave run my-cli-pipeline`, **Then** the saved pipeline executes identically to how it would have executed inline.

---

### User Story 4 - Mock Adapter Testing (Priority: P2)

A developer or CI system wants to test the meta pipeline flow without making live LLM API calls, so they can validate the pipeline generation and execution machinery in isolation.

**Why this priority**: Essential for testing, CI pipelines, and development without API costs.

**Independent Test**: Can be tested by running `wave meta "<task>" --mock --dry-run` and verifying a structurally valid pipeline is generated from the mock adapter's deterministic output.

**Acceptance Scenarios**:

1. **Given** a valid manifest, **When** the user runs `wave meta "any task" --mock --dry-run`, **Then** the mock adapter returns a well-formed philosopher response and the system generates a valid pipeline without any network calls.
2. **Given** a valid manifest, **When** the user runs `wave meta "any task" --mock`, **Then** the generated pipeline executes end-to-end using mock adapter responses for all steps.

---

### User Story 5 - Resource Limit Enforcement (Priority: P3)

An operator wants meta pipelines to respect configurable resource limits (depth, steps, tokens, timeout), so that runaway generation or recursive meta calls cannot exhaust system resources.

**Why this priority**: Safety net for production usage. Without limits, a poorly-designed philosopher prompt could generate unbounded pipelines or recursive meta calls.

**Independent Test**: Can be tested by configuring low limits in the manifest and verifying that meta pipeline execution terminates with a clear limit-exceeded error.

**Acceptance Scenarios**:

1. **Given** `runtime.meta_pipeline.max_total_steps: 3` in the manifest, **When** the philosopher generates a pipeline with 5 steps, **Then** execution is rejected with a clear error indicating the step limit was exceeded.
2. **Given** `runtime.meta_pipeline.max_depth: 1`, **When** a meta-generated pipeline attempts to invoke another meta pipeline, **Then** the nested call is rejected with a depth limit error including the call stack.
3. **Given** `runtime.meta_pipeline.timeout_minutes: 1`, **When** a meta pipeline runs longer than 1 minute, **Then** execution is terminated with a timeout error.

---

### Edge Cases

- What happens when the philosopher returns malformed YAML? The system MUST report a parsing error with the raw output for debugging.
- What happens when the philosopher returns valid YAML but with circular step dependencies? The DAG validator MUST reject it with a clear cycle description.
- What happens when the philosopher generates schemas with invalid JSON? The JSON auto-repair logic MUST attempt fixes; if unfixable, the system MUST report the schema error.
- What happens when the manifest has no `runtime.meta_pipeline` configuration? The system MUST use sensible defaults (max_depth: 3, max_total_steps: 20, max_total_tokens: 500000, timeout: 30 min).
- What happens when a generated pipeline references a persona not defined in the manifest? Validation MUST reject the pipeline with a clear error listing the missing persona.
- What happens when the philosopher persona has insufficient tool permissions? The system MUST report which permissions are needed before invocation fails.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST invoke the philosopher persona to generate pipeline YAML from a natural language task description.
- **FR-002**: System MUST parse the philosopher's delimited output format (`--- PIPELINE ---` / `--- SCHEMAS ---`) to extract pipeline YAML and JSON schemas. Both delimiters are required; a missing `--- SCHEMAS ---` section is a parse error (the section may be empty but the delimiter must be present).
- **FR-003**: System MUST validate generated pipelines semantically: the topologically-first step (after DAG sort, not positional order) uses navigator persona, all steps have contracts, all steps use fresh memory strategy, no circular dependencies. Additionally, JSON schema files referenced by contracts MUST exist and contain valid JSON with a `type` field.
- **FR-004**: System MUST execute validated generated pipelines through the standard pipeline executor with full contract validation at each step.
- **FR-005**: System MUST support `--dry-run` mode that generates and displays the pipeline without executing it.
- **FR-006**: System MUST support `--save <name>` mode that persists the generated pipeline YAML and associated schemas to disk. When `<name>` is a bare name (no path separators), the pipeline is saved to `.wave/pipelines/<name>.yaml`; when `<name>` contains a path separator, it is used as-is. The `.yaml` extension is appended automatically if missing for bare names.
- **FR-007**: System MUST support `--mock` flag to use the mock adapter for testing without live API calls.
- **FR-008**: System MUST enforce configurable resource limits: max recursion depth, max total steps, max total tokens, and execution timeout.
- **FR-009**: System MUST emit structured progress events for monitoring (meta_generate_started, meta_generate_completed, philosopher_invoking, schema_saved).
- **FR-010**: System MUST provide clear, actionable error messages when prerequisites are missing (no philosopher persona, invalid manifest, malformed philosopher output).
- **FR-011**: System MUST auto-generate output artifact definitions for each contract in the generated pipeline.
- **FR-012**: System MUST auto-repair common JSON schema errors (missing braces, trailing commas) in philosopher output before validation.
- **FR-013**: All existing tests in `cmd/wave/commands/meta_test.go` and `internal/pipeline/meta_test.go` MUST pass, including with `-race` flag.

### Key Entities

- **MetaPipelineExecutor**: Core orchestrator that manages philosopher invocation, output parsing, validation, and child pipeline execution. Tracks depth, step count, and token usage across recursive invocations. Delegates child pipeline execution to a `PipelineExecutor` interface (typically `DefaultPipelineExecutor`) injected via `WithChildExecutor()`. Nested meta calls create child executors via `CreateChildMetaExecutor()` with incremented depth and shared counters.
- **Philosopher Persona**: AI agent specialized in architecture design that generates pipeline YAML and contract schemas from natural language task descriptions. Uses the opus model for high reasoning capability.
- **Generated Pipeline**: A dynamically-created pipeline definition (YAML) with steps, personas, contracts, and schemas — produced at runtime rather than authored by hand. Subject to semantic validation before execution.
- **Resource Limits**: Configurable boundaries (depth, steps, tokens, timeout) defined in the manifest under `runtime.meta_pipeline` that prevent runaway meta pipeline execution.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: `wave meta "<task>" --dry-run` generates a semantically valid pipeline for any well-formed task description, completing within 120 seconds.
- **SC-002**: `wave meta "<task>"` executes the generated pipeline end-to-end with all step contracts validated and passing.
- **SC-003**: `wave meta "<task>" --save <name>` produces a pipeline YAML file that can be re-executed via `wave run <name>` without modification.
- **SC-004**: `wave meta "<task>" --mock` completes without network calls and produces structurally valid output.
- **SC-005**: All tests in `cmd/wave/commands/meta_test.go` and `internal/pipeline/meta_test.go` pass with `go test -race ./...` — zero failures, zero skips without linked issues.
- **SC-006**: Resource limit violations (depth, steps, tokens, timeout) are caught and reported with actionable error messages within 1 second of the limit being reached.
- **SC-007**: Error messages for missing prerequisites (philosopher persona, manifest configuration) include specific guidance on how to resolve the issue.

## Clarifications

### CL-001: Default resource limits align with codebase constants
**Ambiguity**: Edge case section originally stated defaults of `max_depth: 2` and `timeout: 60 min`, but the existing code in `internal/pipeline/meta.go` defines `DefaultMaxDepth = 3` and `DefaultMetaTimeout = 30 * time.Minute`.
**Resolution**: Updated spec to match codebase constants: `max_depth: 3`, `max_total_steps: 20`, `max_total_tokens: 500000`, `timeout: 30 min`. These values are well-established in tests and production usage.

### CL-002: Navigator-first validation uses topological sort order
**Ambiguity**: FR-003 said "first step uses navigator persona" without specifying whether "first" means positionally first in the YAML or topologically first after DAG sort.
**Resolution**: Clarified that validation uses `TopologicalSort()` to determine the first step, matching the existing `ValidateGeneratedPipeline()` implementation. This ensures correctness regardless of step ordering in the YAML.

### CL-003: `--- SCHEMAS ---` delimiter is always required
**Ambiguity**: FR-002 mentioned the delimited output format but didn't specify whether both sections are mandatory.
**Resolution**: Both `--- PIPELINE ---` and `--- SCHEMAS ---` delimiters are required in philosopher output. The schemas section may contain zero schema definitions, but the delimiter must be present. This matches `extractPipelineAndSchemas()` which returns an error on missing delimiter.

### CL-004: `--save` path resolution semantics
**Ambiguity**: FR-006 said `--save <name>` saves to `.wave/pipelines/` but didn't specify behavior for full paths or extension handling.
**Resolution**: Bare names (no `/`) get `.wave/pipelines/` prefix and automatic `.yaml` extension. Paths with `/` are used as-is. Matches existing `saveMetaPipeline()` logic.

### CL-005: Schema file validation scope in generated pipelines
**Ambiguity**: FR-003 mentioned contract validation but didn't specify that schema files must also be validated for structural correctness after being written.
**Resolution**: `ValidateGeneratedPipeline()` checks that JSON schema files referenced by `json_schema` contracts exist on disk and contain valid JSON with at least a `type` field. This catches philosopher errors where schema paths are declared but files are malformed or missing.
