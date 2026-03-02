# Implementation Plan: Pipeline Proposal Engine

## 1. Objective

Build a pipeline proposal engine (`internal/pipeline/proposal/`) that consumes a codebase health analysis artifact and the available pipeline catalog to produce structured, prioritized pipeline execution proposals — including dependency ordering, parallel eligibility, and forge-type filtering.

## 2. Approach: Separate Engine (Not MetaPipelineExecutor Extension)

**Decision**: Build a separate `proposal` package rather than extending `MetaPipelineExecutor`.

**Rationale**:
- `MetaPipelineExecutor` is tightly coupled to the philosopher persona and LLM-based pipeline _generation_ (creating new YAML pipelines at runtime). The proposal engine performs deterministic _selection_ from an existing catalog — fundamentally different concerns.
- The meta executor generates pipeline steps dynamically via an adapter call. The proposal engine is a pure Go function: health artifact + catalog → proposal. No LLM invocation needed.
- Extending `MetaPipelineExecutor` would conflate two distinct responsibilities (pipeline generation vs. pipeline recommendation), making both harder to test and reason about.
- The proposal engine can be consumed as a pipeline step (producing a JSON artifact) or called programmatically by the TUI — the meta executor's interface doesn't support this dual usage pattern.
- #95 (wave meta stabilization) can proceed independently without being disrupted by proposal engine concerns.

**Coordination with `wave meta`**: The proposal engine's output (a `Proposal` struct) can be _consumed_ by the meta executor if desired — e.g., a meta pipeline could invoke the proposal engine as its first step, then generate child pipelines from the proposals. This keeps them composable without coupling.

## 3. File Mapping

### New Files (create)

| Path | Purpose |
|------|---------|
| `internal/pipeline/proposal/engine.go` | Core `Engine` type: health artifact parsing, catalog loading, proposal generation |
| `internal/pipeline/proposal/catalog.go` | `Catalog` type: pipeline discovery from `.wave/pipelines/` and `internal/defaults/pipelines/` |
| `internal/pipeline/proposal/filter.go` | Forge-type filtering logic (prefix matching `gh-*`, `gl-*`, `gt-*`, `bb-*`) |
| `internal/pipeline/proposal/types.go` | `Proposal`, `ProposalItem`, `HealthArtifact`, `ForgeType` types |
| `internal/pipeline/proposal/scoring.go` | Relevance scoring: maps health signals to pipeline recommendations |
| `internal/pipeline/proposal/engine_test.go` | Unit tests for Engine |
| `internal/pipeline/proposal/catalog_test.go` | Unit tests for Catalog |
| `internal/pipeline/proposal/filter_test.go` | Unit tests for forge filtering |
| `internal/pipeline/proposal/scoring_test.go` | Unit tests for scoring logic |
| `.wave/contracts/pipeline-proposal.schema.json` | JSON Schema for the proposal output artifact |

### Modified Files (modify)

| Path | Purpose |
|------|---------|
| `internal/pipeline/types.go` | Add `Category` to `PipelineMetadata` (already has the field, may need forge-prefix helper) |
| `internal/tui/pipelines.go` | Extend `DiscoverPipelines` or expose catalog for proposal engine reuse |

### No Changes Needed

| Path | Reason |
|------|--------|
| `internal/pipeline/meta.go` | Kept separate per design decision — proposal engine is composable, not coupled |
| `internal/pipeline/executor.go` | Proposal engine is pre-execution; executor unchanged |
| `cmd/wave/main.go` | CLI integration deferred to TUI issue |

## 4. Architecture Decisions

### AD-1: Package Structure
Place the proposal engine in `internal/pipeline/proposal/` as a sub-package of pipeline. It depends on pipeline types but doesn't modify the executor or meta executor.

### AD-2: Health Artifact as Interface
Define `HealthArtifact` as a Go struct matching the schema from #207. Since #207's schema is not yet finalized, define a provisional interface that can be adapted. Use `encoding/json` for deserialization with `json:"fieldname"` tags.

### AD-3: Pipeline Catalog Discovery
Reuse the existing `tui.DiscoverPipelines()` pattern (directory scan + YAML parse) but generalize it into `proposal.Catalog`. The catalog should scan both `.wave/pipelines/` and `internal/defaults/pipelines/` (embedded defaults).

### AD-4: Forge-Type Filtering
Use pipeline name prefix convention already in use: `gh-*` = GitHub, `gl-*` = GitLab, `gt-*` = Gitea, `bb-*` = Bitbucket. Pipelines without a forge prefix (e.g., `refactor`, `prototype`) are forge-agnostic and always eligible.

### AD-5: Proposal Output as JSON Artifact
The proposal is a structured JSON artifact conforming to `.wave/contracts/pipeline-proposal.schema.json`. This makes it consumable by downstream pipeline steps or the TUI.

### AD-6: Dependency Edge Construction
The engine infers pipeline ordering based on:
- Pipeline `requires.tools` overlap (e.g., if pipeline A produces an artifact pipeline B needs)
- Logical phase ordering (analyze → implement → review → deploy)
- Health signal severity (critical issues first)

### AD-7: Parallel Eligibility
Two proposals are parallelizable if they have no dependency edges between them and don't share workspace state. The engine marks each proposal item with a `parallel_group` identifier.

## 5. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Health artifact schema from #207 not finalized | High | Medium | Define provisional schema; use interface-based deserialization so adapter is swappable |
| Forge detection from #206 not available | High | Low | Use heuristic (git remote URL pattern matching) as fallback; accept `ForgeType` as input parameter |
| Scoring heuristics produce poor recommendations | Medium | Medium | Start with simple priority-tier mapping; make scoring pluggable via strategy pattern |
| Pipeline catalog grows; discovery becomes slow | Low | Low | Cache catalog at Engine construction; catalog is read-only at proposal time |
| Overlap with #95 wave meta stabilization | Medium | Low | Separate package means no merge conflicts; document composition pattern in ADR |

## 6. Testing Strategy

### Unit Tests
- **Engine**: Test proposal generation with mock health artifacts and mock catalogs. Cover empty catalog, empty health artifact, single pipeline, forge filtering.
- **Catalog**: Test discovery with temporary directories containing valid/invalid YAML files. Test deduplication when same pipeline exists in multiple directories.
- **Filter**: Test each forge type prefix. Test forge-agnostic pipelines. Test unknown forge types.
- **Scoring**: Test relevance scoring for various health signal combinations. Test edge cases (all scores zero, all scores maximum).

### Integration Tests
- **End-to-end**: Load real pipeline catalog from `.wave/pipelines/`, feed a synthetic health artifact, verify proposal structure matches schema.
- **Schema validation**: Validate proposal output against `.wave/contracts/pipeline-proposal.schema.json` using the existing contract validation infrastructure.

### Test Coverage Target
- Minimum 80% line coverage for the `proposal` package
- All public API functions must have at least one positive and one negative test case
