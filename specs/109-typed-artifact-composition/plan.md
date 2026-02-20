# Implementation Plan: Typed Artifact Composition

**Branch**: `109-typed-artifact-composition` | **Date**: 2026-02-20 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/109-typed-artifact-composition/spec.md`

## Summary

Extend Wave's artifact system to support capturing step stdout as typed artifacts and enable bidirectional contract validation at step boundaries. This involves:
1. Adding `source: stdout` option to artifact definitions
2. Extending artifact references with `type`, `schema_path`, and `optional` fields
3. Implementing input contract validation before step execution
4. Supporting `{{artifacts.<name>}}` template substitution in prompts

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `gopkg.in/yaml.v3` (YAML parsing), existing `internal/contract` package
**Storage**: Filesystem (`.wave/artifacts/<step-id>/<name>`)
**Testing**: `go test ./...` with table-driven tests
**Target Platform**: Linux/macOS (single binary)
**Project Type**: Single (Wave CLI)
**Performance Goals**: Artifact validation <100ms, stdout capture unbounded until size limit
**Constraints**: Max stdout artifact size configurable (default 10MB), fail-fast on validation errors
**Scale/Scope**: Per-pipeline, typically <100 artifacts per run

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✓ Pass | No new dependencies |
| P2: Manifest as Truth | ✓ Pass | Schema extends `wave-pipeline.schema.json` |
| P3: Persona Boundaries | ✓ Pass | No persona changes |
| P4: Fresh Memory | ✓ Pass | Artifacts remain the sole inter-step communication |
| P5: Navigator-First | ✓ Pass | No navigator changes |
| P6: Contracts at Handover | ✓ Pass | Extends contracts bidirectionally |
| P7: Relay via Summarizer | ✓ Pass | No relay changes |
| P8: Ephemeral Workspaces | ✓ Pass | Artifacts stored in workspace |
| P9: No Credentials on Disk | ✓ Pass | No credential handling |
| P10: Observable Progress | ✓ Pass | Emit events for validation phases |
| P11: Bounded Recursion | ✓ Pass | No recursion changes |
| P12: Step State Machine | ✓ Pass | No state changes |
| P13: Test Ownership | ✓ Required | Full test suite for new code |

## Project Structure

### Documentation (this feature)

```
specs/109-typed-artifact-composition/
├── plan.md              # This file
├── research.md          # Phase 0 output - codebase research
├── data-model.md        # Phase 1 output - entity definitions
├── contracts/           # Phase 1 output - JSON schemas
│   ├── artifact-def.schema.json
│   ├── artifact-ref.schema.json
│   ├── runtime-artifacts-config.schema.json
│   └── input-validation-result.schema.json
└── tasks.md             # Phase 2 output - implementation tasks
```

### Source Code (repository root)

```
internal/
├── pipeline/
│   ├── types.go         # Extend ArtifactDef, ArtifactRef
│   ├── executor.go      # Stdout capture, input validation
│   └── context.go       # Artifact template resolution
├── manifest/
│   └── types.go         # Add RuntimeArtifactsConfig
└── contract/
    └── input_validator.go  # New: input artifact validation

.wave/schemas/
└── wave-pipeline.schema.json  # Schema updates

tests/
└── pipeline/
    ├── stdout_capture_test.go
    ├── input_validation_test.go
    └── artifact_templates_test.go
```

**Structure Decision**: Single project structure. Changes are additive to existing packages without new top-level directories.

## Implementation Phases

### Phase 1: Type Extensions (FR-001, FR-004)

**Goal**: Extend data types without breaking existing functionality.

**Changes**:
1. `internal/pipeline/types.go`:
   - Add `Source string` to `ArtifactDef` (default: "file")
   - Add `Type`, `SchemaPath`, `Optional` to `ArtifactRef`

2. `internal/manifest/types.go`:
   - Add `Artifacts RuntimeArtifactsConfig` to `RuntimeConfig`

3. `.wave/schemas/wave-pipeline.schema.json`:
   - Update `ArtifactDef` definition
   - Update `ArtifactRef` definition

**Tests**: Ensure existing pipeline YAML still parses correctly.

---

### Phase 2: Stdout Capture (FR-001, FR-002, FR-010)

**Goal**: Capture step stdout as artifact when `source: stdout` is declared.

**Changes**:
1. `internal/pipeline/executor.go`:
   - In `runStepExecution()`, check for stdout artifacts
   - Buffer stdout during adapter execution
   - On success: write to `.wave/artifacts/<step-id>/<name>`
   - Register in `execution.ArtifactPaths`

2. Implement size limit check (FR-009):
   - Read `runtime.artifacts.max_stdout_size`
   - Fail if exceeded

**Tests**:
- Stdout captured and available to downstream
- Size limit enforced
- Step failure = no partial artifact

---

### Phase 3: Input Validation (FR-005, FR-006, FR-007, FR-008)

**Goal**: Validate injected artifacts before step execution.

**Changes**:
1. `internal/contract/input_validator.go` (new file):
   - `ValidateInputArtifacts(refs []ArtifactRef, artifactPaths map[string]string, workspacePath string) []InputValidationResult`
   - Type checking: compare declared type vs artifact metadata
   - Schema validation: reuse `contract.Validate()` for JSON schemas

2. `internal/pipeline/executor.go`:
   - After `injectArtifacts()`, call `ValidateInputArtifacts()`
   - Fail step if validation fails (unless `optional: true`)

**Tests**:
- Missing artifact fails (unless optional)
- Type mismatch fails with clear error
- Schema violation fails with detailed errors
- Optional missing artifact proceeds

---

### Phase 4: Artifact Templates (FR-012)

**Goal**: Resolve `{{artifacts.<name>}}` placeholders in prompts.

**Changes**:
1. `internal/pipeline/context.go`:
   - Extend `ResolvePlaceholders()` to handle `{{artifacts.<name>}}`
   - Read artifact content from `execution.ArtifactPaths`
   - For optional missing artifacts, substitute empty string

**Tests**:
- Artifact content substituted into prompt
- Missing required artifact fails before substitution
- Optional missing artifact = empty string

---

### Phase 5: Documentation & Examples

**Goal**: Provide working examples for users.

**Changes**:
1. Update `docs/pipelines.md` with:
   - Stdout artifact capture example
   - Typed consumption example
   - Bidirectional contract example

2. Add example pipeline in `.wave/pipelines/examples/`:
   - `typed-artifact-pipeline.yaml`

---

## Complexity Tracking

_No constitution violations. Table empty._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| (none)    | -          | -                                    |

## Risk Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Large stdout OOM | Low | High | Enforce size limit before buffering completes |
| Breaking existing pipelines | Low | High | All new fields are optional with backward-compatible defaults |
| Slow schema validation | Medium | Low | Cache compiled schemas; timeout at 5s |
| Circular artifact dependencies | Low | Medium | Existing DAG validator prevents this |

## Success Metrics

| Criterion | Measurement | Target |
|-----------|-------------|--------|
| SC-001: End-to-end stdout pipeline | Integration test | Pass |
| SC-002: Missing artifact fail-fast | Unit test | Error before step runs |
| SC-003: Type mismatch error | Unit test | Clear error message |
| SC-004: Input validation errors | Unit test | Same detail as output errors |
| SC-005: Size limit error | Unit test | Actionable message |
| SC-006: Documentation | Manual review | Examples work |

## Next Steps

After plan approval:
1. Run `/speckit.tasks` to generate `tasks.md` with ordered implementation tasks
2. Begin Phase 1 type extensions
3. Run `go test ./...` after each phase
