# Research: Restore and Stabilize `wave meta` Dynamic Pipeline Generation

**Date**: 2026-03-16
**Spec**: `specs/095-restore-meta-pipeline/spec.md`

## Current State Assessment

The `wave meta` subsystem is **already substantially implemented**. The core infrastructure exists and all existing tests pass. This is a **restore and stabilize** task, not a greenfield build.

### Existing Implementation

| Component | File | Status |
|-----------|------|--------|
| `MetaPipelineExecutor` | `internal/pipeline/meta.go` | Complete — generate, execute, recurse |
| CLI command | `cmd/wave/commands/meta.go` | Complete — `--dry-run`, `--save`, `--mock`, `--model` |
| Philosopher prompt | `internal/pipeline/meta.go:416` | Complete — `buildPhilosopherPrompt()` |
| Output parsing | `internal/pipeline/meta.go:565` | Complete — `extractPipelineAndSchemas()` |
| Semantic validation | `internal/pipeline/meta.go:692` | Complete — `ValidateGeneratedPipeline()` |
| JSON auto-repair | `internal/pipeline/meta.go:379` | Complete — `attemptJSONFix()` |
| Schema file saving | `internal/pipeline/meta.go:309` | Complete — `saveSchemaFiles()` |
| Resource limits | `internal/pipeline/meta.go:803-836` | Complete — depth, steps, tokens |
| Child executor | `internal/pipeline/meta.go:862` | Complete — `CreateChildMetaExecutor()` |
| Unit tests | `internal/pipeline/meta_test.go` | Passing |
| Command tests | `cmd/wave/commands/meta_test.go` | Passing |

### What's Missing or Needs Stabilization

1. **Timeout enforcement** — the `timeout_minutes` config exists but `context.WithTimeout` is only applied in the CLI command layer (`meta.go:172`), not within `MetaPipelineExecutor.Execute()` itself. If the executor is called directly (not via CLI), timeout is not enforced.

2. **Auto-generated output artifacts** (FR-011) — `ValidateGeneratedPipeline()` checks contracts but does not verify that steps with `json_schema` contracts have corresponding `output_artifacts`. The philosopher prompt mentions this requirement but there's no enforcement.

3. **Persona existence validation** — `ValidateGeneratedPipeline()` validates structure but does not check whether generated step personas actually exist in the manifest. A generated pipeline referencing `"reviewer"` would pass validation but fail at execution time.

4. **Progress events** — Most events are emitted (`meta_generate_started`, `meta_generate_completed`, `philosopher_invoking`, `schema_saved`) but `meta_generate_failed` is only emitted for YAML parse errors, not for semantic validation failures.

5. **Mock adapter response for meta** — `NewMockAdapter()` returns a generic response. For `--mock --dry-run` to produce a "structurally valid pipeline" (US4-AS1), the mock needs to return properly formatted `--- PIPELINE ---` / `--- SCHEMAS ---` output.

6. **Test coverage gaps** — Tests cover validation and parsing but don't test the full `Execute()` flow with a mock child executor. No integration-level test exercises the CLI → MetaPipelineExecutor → PipelineExecutor path.

## Decisions

### D-001: Stabilize, Don't Rewrite

**Decision**: Fix identified gaps within the existing architecture. No structural changes needed.
**Rationale**: The implementation is sound. All 13 FRs have existing code; gaps are edge cases and validation tightening.
**Alternatives rejected**: Full rewrite (unnecessary — code works), new package (over-engineering for stabilization).

### D-002: Add manifest-aware validation to `ValidateGeneratedPipeline`

**Decision**: Extend `ValidateGeneratedPipeline()` to accept an optional `*manifest.Manifest` parameter for persona existence checking.
**Rationale**: The spec (edge case 5) requires checking "references a persona not defined in the manifest." Current validation is structure-only.
**Alternatives rejected**: Validate at execution time only (fails late, poor UX), separate validation function (duplicates DAG/contract checks).

### D-003: Meta-specific mock adapter response

**Decision**: Add a `MetaMockResponse()` function that returns a properly delimited pipeline+schemas response for mock adapter testing.
**Rationale**: The mock adapter's generic response doesn't contain `--- PIPELINE ---` markers, so `--mock --dry-run` fails at `extractPipelineAndSchemas()`.
**Alternatives rejected**: Hardcode in mock adapter (too specific), separate mock (over-engineering).

### D-004: Enforce timeout inside MetaPipelineExecutor

**Decision**: Move `context.WithTimeout` wrapping into `MetaPipelineExecutor.Execute()` using the manifest config, removing the CLI-only enforcement.
**Rationale**: Callers of the executor (not just CLI) need timeout enforcement. Constitutional principle 11 requires it.
**Alternatives rejected**: Keep CLI-only timeout (violates principle), add timeout as executor option (unnecessary indirection — manifest config is already available).

### D-005: Auto-generate output_artifacts for json_schema steps

**Decision**: Add a `normalizeGeneratedPipeline()` function that ensures steps with `json_schema` contracts have `output_artifacts` configured, auto-generating them if missing.
**Rationale**: FR-011 requires auto-generation. The philosopher may omit `output_artifacts` even when prompted to include them.
**Alternatives rejected**: Fail validation (bad UX — auto-fix is better), always require in prompt (non-deterministic, can't guarantee).
