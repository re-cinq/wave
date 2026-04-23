# Data Model: WebUI Framework Research

**Feature**: 1106-webui-framework-research  
**Date**: 2026-04-14  
**Phase**: 1 — Design & Contracts

> **Note**: This feature is research-only (FR-009). No Go code changes, no database
> changes, no new API endpoints. The "data model" here describes the structure of
> research deliverables, not runtime entities.

## Entity Definitions

### CandidateFramework

Represents a frontend technology being evaluated.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| name | string | required, enum: Svelte, Ripple, Astro, htmx | Framework identifier |
| version | string | required | Specific version evaluated |
| license | string | required | SPDX license identifier |
| build_toolchain | string[] | required | Tools needed at build time (e.g., Node.js, npm, esbuild) |
| runtime_deps | string[] | required, should be empty | Runtime dependencies (hard constraint: must be none) |
| output_format | string | required | Build output type (static HTML/JS/CSS, SSR, hybrid) |
| eliminated | boolean | default: false | Whether eliminated via hard constraint failure |
| elimination_reason | string | nullable | Documented reason if eliminated |

**Invariants**:
- `runtime_deps` must be empty for a candidate to pass hard constraints
- If `eliminated == true`, `elimination_reason` must be non-empty
- Exactly 4 candidates defined (FR-001)

---

### EvaluationCriterion

A dimension along which candidates are compared.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | string | required, unique | Short identifier (e.g., `embedding`, `sse`, `bundle`) |
| name | string | required | Human-readable name |
| weight | enum | Critical / High / Medium | Importance level |
| measurement_method | string | required | How this criterion is evaluated |
| is_hard_constraint | boolean | default: false | Whether failure means automatic elimination |

**Fixed set** (9 criteria from issue):

| ID | Name | Weight | Hard Constraint |
|----|------|--------|-----------------|
| embedding | Embedding story (go:embed) | Critical | Yes |
| migration | Migration path | High | No |
| sse | SSE compatibility | Critical | No |
| bundle | Bundle size | Medium | No |
| devexp | Developer experience | Medium | No |
| build | Build complexity | High | No |
| community | Community & longevity | Medium | No |
| components | Component reuse | Medium | No |
| auth | Auth integration | High | No |

---

### MatrixCell

One intersection of candidate × criterion in the comparison matrix.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| candidate | ref → CandidateFramework | required | Which framework |
| criterion | ref → EvaluationCriterion | required | Which evaluation dimension |
| rating | enum | Strong / Good / Adequate / Weak / Fail | Qualitative assessment |
| evidence | string | required, min 50 chars | Supporting evidence for the rating |
| code_snippet | string | nullable | Optional code example |

**Invariants**:
- All 36 cells (4 candidates × 9 criteria) must be populated (SC-001)
- `rating == Fail` on a hard-constraint criterion triggers candidate elimination
- Every rating must have non-empty evidence (no unsupported claims)

---

### ComparisonMatrix

The complete cross-product evaluation document.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| candidates | CandidateFramework[] | length == 4 | All evaluated frameworks |
| criteria | EvaluationCriterion[] | length == 9 | All evaluation dimensions |
| cells | MatrixCell[] | length == 36 | All intersections |
| baseline | BaselineMetrics | required | Current webui measurements |

**Relationships**: Contains all CandidateFrameworks and EvaluationCriteria. Each cell references one of each.

---

### BaselineMetrics

Current webui measurements for comparison.

| Field | Type | Value | Source |
|-------|------|-------|--------|
| js_size_bytes | int | 126,830 | `wc -c internal/webui/static/*.js` |
| css_size_bytes | int | 155,942 | `wc -c internal/webui/static/style.css` |
| total_static_bytes | int | 282,772 | Sum of all static assets |
| js_file_count | int | 5 | app.js, dag.js, diff-viewer.js, log-viewer.js, sse.js |
| template_count | int | 24 | Page templates (excluding partials) |
| partial_count | int | 6 | step_card, run_row, dag_svg, artifact_viewer, resume_dialog, child_run_row |
| template_func_count | int | 30 | Custom functions in embed.go funcMap |
| handler_file_count | int | 21 | handlers_*.go files |
| sse_event_types | int | 7 | started, running, completed, failed, step_progress, stream_activity, eta_updated |
| auth_modes | int | 4 | none, bearer, jwt, mtls |

---

### ProofOfConcept

A working implementation of run_detail in a candidate framework.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| candidate | ref → CandidateFramework | required | Which framework |
| location | string | required, pattern: `poc/<name>/` | Directory path |
| features_demonstrated | string[] | required | List of demonstrated capabilities |
| embed_compatible | boolean | required | go:embed works (SC-002) |
| sse_working | boolean | required | Real-time streaming functional |
| dag_renders | boolean | required | DAG visualization renders |
| step_cards_live | boolean | required | Step cards reflect live status |
| build_command | string | required | How to build the PoC |
| build_output_size | int | required | Total build output in bytes |

**Invariants**:
- At least 1 PoC required (FR-003), target 2
- `embed_compatible` must be true for the PoC to satisfy SC-002
- All four demo features (embed, SSE, DAG, step cards) must work per User Story 2

---

### Recommendation

The synthesis document with decision and migration plan.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| winner | ref → CandidateFramework | required | Primary recommendation |
| justification | string | required, references ≥3 criteria | Why this candidate won (SC-004) |
| strategy | enum | incremental / big-bang | Migration approach |
| migration_phases | MigrationPhase[] | if incremental: length ≥ 3 | Ordered migration phases (SC-005) |
| risks | Risk[] | length ≥ 3 | Risk assessment (SC-006) |
| runner_up | ref → CandidateFramework | nullable | Second-best option |

---

### MigrationPhase

One phase of an incremental migration.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| order | int | required, sequential | Phase number |
| pages | string[] | required | Which pages to migrate |
| rationale | string | required | Why these pages in this order |
| effort_estimate | string | required | Relative effort (e.g., "1-2 days", "1 week") |

---

### Risk

An identified risk with mitigation.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| title | string | required | Short risk description |
| severity | enum | High / Medium / Low | Impact level |
| likelihood | enum | High / Medium / Low | Probability |
| description | string | required | Detailed risk explanation |
| mitigation | string | required | Proposed mitigation strategy |

## Entity Relationship Diagram

```
ComparisonMatrix
├── has 4 → CandidateFramework
├── has 9 → EvaluationCriterion
├── has 36 → MatrixCell (candidate × criterion)
└── has 1 → BaselineMetrics

ProofOfConcept
└── implements 1 → CandidateFramework

Recommendation
├── selects 1 → CandidateFramework (winner)
├── has N → MigrationPhase (if incremental)
└── has ≥3 → Risk
```

## Deliverable-to-Entity Mapping

| Deliverable | File | Primary Entity |
|-------------|------|---------------|
| Comparison matrix | `matrix.md` | ComparisonMatrix (contains MatrixCells) |
| Proof-of-concept(s) | `poc/<candidate>/` | ProofOfConcept |
| Recommendation | `recommendation.md` | Recommendation (contains Risks, MigrationPhases) |
