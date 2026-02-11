# Research: Release-Gated Pipeline Embedding

**Feature Branch**: `029-release-gated-embedding`
**Date**: 2026-02-11
**Spec**: `specs/029-release-gated-embedding/spec.md`

## Phase 0 — Unknowns & Research Findings

### RES-001: Struct Extension Strategy for `release` and `disabled` Fields

**Decision**: Extend `PipelineMetadata` struct with `Release bool` and `Disabled bool` fields using `yaml:"...,omitempty"` struct tags.

**Rationale**: Go's zero value for `bool` is `false`, which provides the correct default-to-false semantics without requiring pointer types (`*bool`). The `omitempty` tag ensures clean YAML roundtripping — fields set to `false` won't be emitted when marshalling. This is the idiomatic Go approach and aligns with the existing struct tag pattern for `Name` and `Description`.

**Alternatives Rejected**:
- **`*bool` pointer type**: Would require nil-checking everywhere. The spec explicitly says default is `false`, and Go's zero value provides this. No need to distinguish "not set" from "set to false".
- **Raw YAML string scanning**: Fragile (regex-based), doesn't benefit from type safety, and would be inconsistent with the existing structured YAML parsing approach.
- **Separate metadata struct**: Unnecessary complexity. The existing `PipelineMetadata` is the natural home.

### RES-002: Filtering Location (embed.go vs init.go)

**Decision**: Add filtering functions to `internal/defaults/embed.go` (new `GetReleasePipelines()` function) and apply filtering in `cmd/wave/commands/init.go` at extraction time.

**Rationale**: The defaults package already provides `GetPipelines()`, `GetContracts()`, etc. Adding `GetReleasePipelines()` follows the existing pattern. The init command currently calls these directly — it will switch to calling the filtered variant unless `--all` is passed. The transitive exclusion logic (determining which contracts/prompts to include) is init-specific behavior and belongs in init.go, not in the defaults package.

**Alternatives Rejected**:
- **Filtering only in init.go**: Would require unmarshalling pipeline YAML in init.go and duplicating YAML parsing knowledge. The defaults package is the natural place for pipeline-aware queries.
- **Filtering only in embed.go**: Transitive dependency resolution requires cross-asset knowledge (pipelines → contracts, pipelines → prompts) that couples too many concerns into the defaults package. The defaults package should provide primitive queries; init.go composes them.

### RES-003: Transitive Exclusion Algorithm

**Decision**: Reference-counting approach with prefix stripping for path normalization.

**Algorithm**:
1. Parse all pipelines, partition into release vs non-release sets.
2. For each release pipeline, extract `schema_path` and `source_path` references from its steps.
3. Normalize references by stripping known prefixes (`.wave/contracts/` → bare filename, `.wave/prompts/` → relative path).
4. Build a "referenced by release" set for contracts and prompts.
5. Include only contracts/prompts whose normalized key appears in the referenced set.
6. Personas are always included (FR-005).

**Rationale**: This is a simple set-inclusion check. Reference counting (checking if count > 0) is functionally equivalent to set membership for our boolean case (release or not). The prefix stripping follows the existing pattern in `readDir()` and `readDirNested()` where embedded FS paths are stripped to bare filenames or relative paths.

**Alternatives Rejected**:
- **Graph-based dependency resolution**: Over-engineered for this use case. There are no transitive chains (pipeline → contract → another contract). Dependencies are one level deep.
- **Embedding release status in filenames**: Would require renaming files and break existing references.

### RES-004: Pipeline YAML Parsing for Metadata Extraction

**Decision**: Unmarshal the full `Pipeline` struct (from `internal/pipeline/types.go`) for each embedded pipeline YAML when filtering.

**Rationale**: The `Pipeline` struct already models the complete pipeline YAML schema. By adding `Release` and `Disabled` to `PipelineMetadata`, the existing struct can parse the metadata. For extracting contract/prompt references, we need to walk `Steps[].Handover.Contract.SchemaPath` and `Steps[].Exec.SourcePath`, which are already modeled in the struct.

**Alternatives Rejected**:
- **Partial struct (metadata only)**: Would require a second pass to extract references for transitive exclusion. Using the full struct does both in one pass.
- **Raw YAML map parsing**: Loses type safety and is error-prone for nested field access.

### RES-005: `--all` Flag Implementation

**Decision**: Add `All bool` to `InitOptions` and register `--all` flag on the init command. When `All` is true, bypass release filtering entirely — use `GetPipelines()` (all) instead of `GetReleasePipelines()`.

**Rationale**: Simplest implementation. The `--all` flag is a binary switch that selects between filtered and unfiltered asset sets. No partial filtering or granular selection needed per the spec.

**Alternatives Rejected**:
- **`--include-experimental` flag**: Longer name, same behavior. `--all` is more intuitive and matches the spec exactly.
- **Per-pipeline include flags**: Over-engineered for the use case.

### RES-006: Warning When Zero Release Pipelines

**Decision**: When filtering produces zero pipelines, emit a warning via `fmt.Fprintf(cmd.ErrOrStderr(), ...)` and continue (FR-011). The `.wave/pipelines/` directory will be empty but the command succeeds.

**Rationale**: The spec is explicit that this should not fail. Warning on stderr keeps stdout clean for machine parsing.

### RES-007: Current Pipeline Release Status Audit

**Observation from codebase scan**:
- **Already marked `release: true`**: `doc-loop.yaml`, `github-issue-enhancer.yaml`, `issue-research.yaml`
- **Explicitly `release: false`**: `docs-to-impl.yaml`
- **No release field (defaults to false)**: `code-review.yaml`, `debug.yaml`, `docs.yaml`, `gh-poor-issues.yaml`, `hello-world.yaml`, `hotfix.yaml`, `migrate.yaml`, `plan.yaml`, `prototype.yaml`, `refactor.yaml`, `smoke-test.yaml`, `speckit-flow.yaml`, `test-gen.yaml`, `umami.yaml`

**Action needed**: As part of implementation, pipeline maintainers should review which pipelines should be `release: true`. This is a content decision outside the scope of this feature's code changes, but the feature must work correctly with whatever markings exist.

### RES-008: Impact on `printInitSuccess` Display

**Decision**: Modify `printInitSuccess` to accept counts/names of actually extracted assets rather than querying `defaults.Get*()` directly.

**Rationale**: CLR-005 in the spec explicitly requires this. The current code calls `defaults.GetPipelines()` and displays the total count. After filtering, it should display only the extracted set.

## Codebase Patterns Observed

| Pattern | Location | Relevance |
|---------|----------|-----------|
| `readDir()` strips directory prefix to bare filenames | `embed.go:46-67` | Must follow for contract key matching |
| `readDirNested()` preserves relative paths | `embed.go:69-93` | Must follow for prompt key matching |
| `PipelineMetadata` has `Name` and `Description` | `types.go:18-21` | Extend with `Release` and `Disabled` |
| init.go calls `defaults.Get*()` then writes to `.wave/` | `init.go:120-134` | Modify to use filtered variants |
| Merge mode uses `*IfMissing` variants | `init.go:184-201` | Must also apply release filtering |
| Pipeline YAMLs use `.wave/contracts/` prefix in `schema_path` | All pipelines | Must strip prefix for matching |
| Only `speckit-flow.yaml` uses `source_path` | `speckit-flow.yaml` | Must strip `.wave/prompts/` prefix |
