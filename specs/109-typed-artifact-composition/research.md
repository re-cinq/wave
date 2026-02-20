# Phase 0 Research: Typed Artifact Composition

**Feature Branch**: `109-typed-artifact-composition`
**Created**: 2026-02-20
**Status**: Complete

## Unknowns Identified from Spec

The following areas required codebase research to resolve:

### 1. Stdout Capture Integration Point

**Question**: Where in the pipeline execution flow should stdout be captured for artifact registration?

**Research Findings**:
- `internal/pipeline/executor.go:622-625` captures stdout from adapter result: `io.ReadAll(result.Stdout)`
- Artifact registration happens at line 1091-1119 in `writeOutputArtifacts()`
- Artifacts are keyed by `"<step-id>:<artifact-name>"` and stored in `execution.ArtifactPaths` map

**Decision**: Capture stdout in `runStepExecution()` after adapter completes. If `ArtifactDef.Source == "stdout"`, buffer content and write to `.wave/artifacts/<step-id>/<name>` before registering in `ArtifactPaths`.

**Rationale**: This reuses existing artifact registration infrastructure. The current flow already handles file-based artifacts; stdout artifacts simply have a different source.

**Alternatives Rejected**:
- In-memory only artifacts: Would break existing file-based injection mechanism (`injectArtifacts()` at line 1035-1089 reads from filesystem)
- Adapter-level capture: Would require changes to adapter interface; current design keeps orchestrator in control

---

### 2. Type Validation Mechanism

**Question**: How should artifact type validation be implemented?

**Research Findings**:
- Current `ArtifactRef` in `types.go:68-72` has no type field
- Current `ArtifactDef` in `types.go:97-102` has `Type string` field but it's informational only
- Contract validation in `internal/contract/contract.go` provides `ContractValidator` interface
- Existing contract types: `json_schema`, `typescript_interface`, `test_suite`, `markdown_spec`, `template`, `format`
- JSON schema validation exists at `internal/contract/jsonschema.go`

**Decision**: Add `type`, `schema_path`, and `optional` fields to `ArtifactRef`. Type validation happens in `injectArtifacts()` before copying artifact. Schema validation reuses existing `contract.Validate()`.

**Rationale**:
- Leverages existing contract validation infrastructure
- Type checking is simple string comparison (lightweight)
- Schema validation can be optional for performance

**Alternatives Rejected**:
- New validation package: Unnecessary duplication; `internal/contract/` already has validators
- Validation in adapter: Violates orchestrator-owns-validation principle from constitution

---

### 3. Stdout Size Limit Implementation

**Question**: How should stdout size limits be enforced?

**Research Findings**:
- No existing size limits for artifacts
- Runtime config in `internal/manifest/types.go` has `Runtime` struct
- Workspace root configurable via `Runtime.WorkspaceRoot`
- Relay has token limits via `Runtime.Relay.TokenThresholdPercent`

**Decision**: Add `Runtime.Artifacts.MaxStdoutSize` config (default 10MB). Buffer stdout during capture; fail if limit exceeded BEFORE writing artifact.

**Rationale**:
- Early failure prevents partial artifacts
- Config-driven allows per-project tuning
- Consistent with relay threshold pattern

**Alternatives Rejected**:
- Truncation instead of failure: Data loss risk; better to fail explicitly
- Per-step limits: Overcomplicates config; runtime-level is simpler

---

### 4. Input Contract Validation Timing

**Question**: When exactly should input contract validation run relative to artifact injection?

**Research Findings**:
- Current execution sequence in `runStepExecution()` (lines 434-736):
  1. Create workspace (line 448)
  2. Inject artifacts (line 467)
  3. Build prompt (line 471)
  4. Run adapter (line 580)
  5. Write output artifacts (line 637)
  6. Validate output contracts (lines 659-722)
- Artifact injection copies files to `<workspace>/.wave/artifacts/<name>` (line 1043)
- Contract validation in `Validate()` takes `workspacePath` and reads files from there

**Decision**: Insert input validation AFTER artifact injection (line 467) but BEFORE prompt building (line 471). Sequence:
1. Resolve artifact paths
2. Inject artifacts into workspace (copy files)
3. **NEW: Validate input contracts against injected artifacts**
4. Build prompt
5. Execute step
6. Validate output contracts

**Rationale**:
- Files must exist on disk for schema validators to read
- Fail-fast before expensive adapter execution
- Symmetric with output validation

---

### 5. Artifact Template Resolution

**Question**: How should `{{artifacts.<name>}}` placeholders work in prompts?

**Research Findings**:
- Template resolution in `internal/pipeline/context.go` handles `{{ input }}`, `{{ project.* }}`
- `ResolvePlaceholders()` method processes prompt templates
- Current artifact injection writes to `.wave/artifacts/<name>` in workspace

**Decision**: Extend `PipelineContext.ResolvePlaceholders()` to handle `{{artifacts.<name>}}`. Read artifact content from `execution.ArtifactPaths` and inline into prompt.

**Rationale**:
- Consistent with existing template mechanism
- Single point of template resolution
- Artifacts already registered by path

**Alternatives Rejected**:
- Separate template processor: Fragments template logic
- Lazy loading in adapter: Adapter should receive fully-resolved prompt

---

## Technology Decisions Summary

| Area | Decision | Rationale |
|------|----------|-----------|
| Stdout capture | In `runStepExecution()` after adapter | Reuses existing artifact flow |
| Artifact storage | `.wave/artifacts/<step-id>/<name>` | Matches current pattern |
| Type validation | Extend `ArtifactRef` with `type` field | Minimal struct change |
| Schema validation | Reuse `contract.Validate()` | DRY, existing infrastructure |
| Size limits | `Runtime.Artifacts.MaxStdoutSize` | Config-driven, fail-fast |
| Input validation | After injection, before prompt build | Fail-fast, files must exist |
| Template syntax | `{{artifacts.<name>}}` | Consistent with existing |

## Files to Modify

### Core Changes
1. `internal/pipeline/types.go`
   - Extend `ArtifactRef` with `type`, `schema_path`, `optional` fields
   - Extend `ArtifactDef` with `source` field ("stdout" | "file")

2. `internal/pipeline/executor.go`
   - Add stdout buffering in `runStepExecution()`
   - Add input validation before prompt building
   - Modify `writeOutputArtifacts()` to handle stdout source

3. `internal/manifest/types.go`
   - Add `Artifacts` config to `RuntimeConfig`
   - Add `MaxStdoutSize` field

4. `internal/pipeline/context.go`
   - Extend `ResolvePlaceholders()` for artifact templates

### Schema Updates
5. `.wave/schemas/wave-pipeline.schema.json`
   - Add `source` to `ArtifactDef`
   - Add `type`, `schema_path`, `optional` to `ArtifactRef`

### Tests
6. `internal/pipeline/executor_test.go`
   - Tests for stdout capture
   - Tests for type validation
   - Tests for input contracts

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing pipelines | High | `source: file` is default; backward compatible |
| Performance (large stdout) | Medium | Size limits prevent memory exhaustion |
| Circular dependency detection | Low | Existing DAG validator handles this |
| Schema validation latency | Low | Optional via `schema_path` presence |

## Constitution Compliance

- **Principle 4 (Fresh Memory)**: ✓ Artifacts remain the only inter-step communication
- **Principle 6 (Contracts)**: ✓ Extends contracts bidirectionally (input + output)
- **Principle 8 (Ephemeral Workspaces)**: ✓ Artifacts stored in workspace, not main repo
- **Principle 13 (Test Ownership)**: ✓ Must add comprehensive tests
