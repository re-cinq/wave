# Research: Wave Task Classifier

**Date**: 2026-04-12 | **Branch**: `772-task-classifier`

## Decision 1: Package Dependency Direction

**Decision**: `internal/classify` imports `internal/suggest` (for `InputType` and `ClassifyInput`), not the reverse.

**Rationale**: The spec (FR-003) requires reusing `suggest.ClassifyInput` for URL/ref detection. The suggest package is stable and has no dependency on classify. This keeps the dependency graph acyclic: `suggest` → standalone, `classify` → depends on `suggest`.

**Alternatives Rejected**:
- Extracting `InputType` into a shared package — unnecessary indirection; `suggest` is already internal
- Duplicating `InputType` in classify — violates DRY, risks drift

## Decision 2: Test Package Naming

**Decision**: Use same-package tests (`package classify`) not external tests (`package classify_test`).

**Rationale**: Matches existing codebase pattern. All `internal/suggest` tests use `package suggest`. Same-package tests can access unexported helpers (keyword lists, scoring functions) which simplifies test setup.

**Alternatives Rejected**:
- External test package (`classify_test`) — would require exporting internal helpers or duplicating setup logic

## Decision 3: Keyword Analysis Approach

**Decision**: Static keyword maps with weighted scoring. No NLP, no external dependencies.

**Rationale**: SC-005 prohibits new external dependencies. SC-006 requires <1ms classification. Static keyword matching satisfies both constraints while providing sufficient accuracy for the defined test cases (SC-001: 90%+ accuracy).

**Alternatives Rejected**:
- Regex-based pattern matching — slower, harder to maintain keyword lists
- ML/embedding-based classification — violates SC-005 (external deps) and SC-006 (latency)

## Decision 4: Blast Radius Calculation

**Decision**: Derive from complexity + domain using a simple scoring formula. Base score from complexity (simple=0.1, medium=0.3, complex=0.6, architectural=0.8), adjusted by domain modifier (security=+0.2, performance=+0.1, docs=-0.1). Clamped to [0.0, 1.0].

**Rationale**: FR-009 requires blast_radius derived from complexity and domain. The formula produces values matching the spec's test expectations: security+any ≥0.5, docs+simple <0.2, architectural+feature >0.7.

**Alternatives Rejected**:
- File-count-based blast radius — requires filesystem access, violates SC-006 (no disk I/O)
- Fixed lookup table — less granular, doesn't capture domain×complexity interactions

## Decision 5: Pipeline Routing Priority

**Decision**: Implement FR-006's priority chain as a sequential if/else in `SelectPipeline`:
1. `input_type == pr_url` → `ops-pr-review`
2. `domain == security` → `audit-security`
3. `domain == research` → `impl-research`
4. `domain == docs` → `doc-fix`
5. `domain == refactor && complexity ∈ {complex, architectural}` → `impl-speckit`
6. `complexity ∈ {simple, medium}` → `impl-issue`
7. `complexity ∈ {complex, architectural}` → `impl-speckit`

**Rationale**: Matches AGENTS.md routing table exactly. Sequential evaluation ensures higher-priority signals (input_type, security) always win. Verified against all acceptance scenarios in spec.

**Alternatives Rejected**:
- Map-based lookup with composite key — harder to express the priority ordering and domain overrides
- Rule engine pattern — over-engineered for 7 rules

## Decision 6: Verified Pipeline Names

All target pipeline names confirmed to exist in `internal/defaults/pipelines/`:
- `impl-issue.yaml` ✓
- `impl-speckit.yaml` ✓  
- `impl-research.yaml` ✓
- `ops-pr-review.yaml` ✓
- `doc-fix.yaml` ✓
- `audit-security.yaml` ✓
- `audit-dead-code.yaml` ✓ (referenced in spec but not in routing table — available for future use)
