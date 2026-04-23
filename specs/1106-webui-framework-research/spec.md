# Feature Specification: WebUI Framework Research

**Feature Branch**: `1106-webui-framework-research`  
**Created**: 2026-04-14  
**Status**: Draft  
**Input**: User description: "https://github.com/re-cinq/wave/issues/1106"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Framework Comparison Matrix (Priority: P1)

As a Wave maintainer deciding on the frontend technology direction, I need a structured comparison of Svelte, Ripple, Astro, and htmx across all evaluation criteria so that I can make an informed decision with clear trade-offs documented.

**Why this priority**: The comparison matrix is the foundational deliverable — all other outputs (PoC, recommendation) depend on having rigorous, criteria-based analysis first. Without it, the team lacks a shared decision framework.

**Independent Test**: Can be verified by reviewing the matrix document and confirming every cell is populated with evidence-based findings for all 4 candidates across all 9 evaluation criteria.

**Acceptance Scenarios**:

1. **Given** all four candidate frameworks (Svelte/SvelteKit, Ripple, Astro, htmx), **When** a maintainer reads the comparison matrix, **Then** every evaluation criterion has a rating and supporting evidence for each candidate — no empty cells or unsupported claims.
2. **Given** the current webui stack baseline (Go templates, vanilla JS, single CSS, go:embed), **When** a maintainer reviews the "embedding story" criterion, **Then** each candidate's entry describes exactly how build output maps to go:embed and what pipeline changes are needed.
3. **Given** the existing SSE architecture (SSEBroker, sse.js with Last-Event-ID backfill, 512 lines), **When** a maintainer reviews the "SSE compatibility" criterion, **Then** each candidate's entry addresses real-time streaming patterns with concrete examples or code snippets.

---

### User Story 2 - Proof-of-Concept Implementation (Priority: P1)

As a Wave maintainer, I need a working proof-of-concept that reimplements the `run_detail.html` page (the most complex page: DAG visualization, log streaming, step cards, artifact viewing) in the top 1–2 candidate frameworks so that I can evaluate real-world developer experience and integration feasibility.

**Why this priority**: A matrix alone cannot capture developer experience, debugging pain, or integration gotchas. The PoC provides empirical evidence that validates or contradicts the theoretical analysis.

**Independent Test**: Can be verified by running each PoC, navigating to the reimplemented run detail page, and confirming that DAG visualization renders, log streaming displays real-time output, step cards reflect live status, and artifact viewing works.

**Acceptance Scenarios**:

1. **Given** a PoC implementation of run_detail in a candidate framework, **When** a run is actively executing, **Then** the DAG visualization updates node colors as steps transition through pending → running → completed/failed states.
2. **Given** a PoC implementation with SSE integration, **When** a step emits log output, **Then** the log viewer displays new lines in real time without page reload and supports scrolling to follow or pausing at a fixed position.
3. **Given** a PoC with the go:embed integration, **When** the Go binary is built with `go build`, **Then** the PoC assets are embedded and the page is served from the single binary without external file dependencies.

---

### User Story 3 - Migration Recommendation (Priority: P2)

As a Wave maintainer, I need a clear recommendation document that identifies the best candidate with a migration strategy (incremental vs. big-bang), timeline estimate, and risk assessment so that I can plan the rewrite with confidence.

**Why this priority**: The recommendation synthesizes findings from the matrix and PoC into an actionable decision. It is lower priority than the evidence-gathering stories because it depends on their outputs.

**Independent Test**: Can be verified by confirming the recommendation document names a winner, provides a migration strategy with phases, estimates effort per phase, and lists top 3 risks with mitigations.

**Acceptance Scenarios**:

1. **Given** a completed comparison matrix and PoC, **When** a maintainer reads the recommendation, **Then** it names one framework as the primary recommendation with a concise justification referencing specific evaluation criteria results.
2. **Given** the current architecture (23 page templates, 6 partials, 30 custom template functions, 4 auth modes), **When** a maintainer reviews the migration strategy, **Then** it specifies whether migration is incremental (page-by-page) or big-bang, and if incremental, names which pages to migrate first and why.
3. **Given** the single-binary distribution constraint, **When** a maintainer reviews the risk assessment, **Then** it addresses build pipeline complexity, CI impact, and any new runtime dependencies introduced.

---

### User Story 4 - Candidate Elimination with Justification (Priority: P3)

As a Wave maintainer, I need candidates that are clearly unsuitable to be eliminated early with documented reasoning so that research effort is focused on viable options.

**Why this priority**: Early elimination saves effort. If a candidate fails a hard constraint (e.g., cannot embed into Go binary, abandoned project), it should be flagged immediately rather than carried through full evaluation.

**Independent Test**: Can be verified by checking that any eliminated candidate has at least one documented hard-constraint failure, and that remaining candidates pass all hard constraints.

**Acceptance Scenarios**:

1. **Given** a candidate framework with fewer than 100 GitHub stars or no release in the past 12 months, **When** the maintainer reviews the elimination rationale, **Then** it cites ecosystem maturity data (stars, contributors, release cadence, adoption).
2. **Given** a candidate that cannot produce output compatible with go:embed, **When** the maintainer reviews the elimination rationale, **Then** it explains the specific technical incompatibility.

---

### Edge Cases

- What happens when a candidate framework has no native SSE support — does it fall back to polling, and how does that compare to the current sse.js polling fallback?
- How does each framework handle the 30 custom template functions currently defined in embed.go (statusIcon, formatDuration, friendlyModel, etc.) — are they migrated as utility functions, components, or filters?
- What happens if a candidate requires Node.js at build time but the CI environment or contributor machines do not have Node.js installed?
- How does each framework interact with the existing 4 authentication modes (none, bearer, JWT, mTLS) — especially CSRF token handling for mutations?
- What is the impact on the existing test suite (~1,900 lines of handler tests) — do handler tests need rewriting or do they continue to work against the same API?

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: Research MUST evaluate exactly four candidate frameworks: Svelte/SvelteKit, Ripple, Astro, and htmx — no additional frameworks in scope.
- **FR-002**: Research MUST produce a written comparison matrix covering all 9 evaluation criteria defined in the issue: embedding story, migration path, SSE compatibility, bundle size, developer experience, build complexity, community & longevity, component reuse, and auth integration.
- **FR-003**: Research MUST produce at least one proof-of-concept (target: two, for the top candidates) that reimplements the `run_detail.html` page — the most complex page with DAG visualization (dag.js, 338 lines), log streaming (log-viewer.js, 1,165 lines), step cards (step_card.html partial), and artifact viewing. A single PoC satisfies the minimum requirement; a second PoC is produced if time and candidate viability permit.
- **FR-004**: Each PoC MUST demonstrate go:embed compatibility — the framework's build output MUST be embeddable into the Go binary without runtime file dependencies.
- **FR-005**: Each PoC MUST demonstrate SSE integration that preserves current capabilities: real-time event streaming, Last-Event-ID reconnection backfill, and connection status indication.
- **FR-006**: Research MUST produce a recommendation document with a named winner, migration strategy (incremental or big-bang), and risk assessment.
- **FR-007**: The comparison matrix MUST include bundle size measurements comparing each candidate's output against the current baseline (~124 KB of JS across 5 files, ~152 KB CSS, ~276 KB total static assets).
- **FR-008**: Research MUST assess each candidate's impact on the existing Go handler layer (~6,700 lines across 23 handler files) — specifically whether handlers need modification to serve framework output.
- **FR-009**: Research MUST NOT propose changes to the Go backend API surface — all evaluation is frontend-only.
- **FR-010**: Research MUST evaluate component extraction feasibility for the 6 existing partials (step_card, dag_svg, run_row, child_run_row, artifact_viewer, resume_dialog).

### Key Entities

- **Candidate Framework**: A frontend technology being evaluated (Svelte, Ripple, Astro, htmx). Key attributes: name, version evaluated, license, build toolchain requirements, output format.
- **Evaluation Criterion**: A dimension along which candidates are compared (9 defined). Key attributes: name, weight/importance, measurement method.
- **Comparison Matrix**: The cross-product of candidates × criteria with evidence-based ratings using a qualitative scale (Strong / Good / Adequate / Weak / Fail) with supporting evidence text for each cell. Relationships: references all candidates and all criteria.
- **Proof-of-Concept**: A working implementation of run_detail in a candidate framework. Key attributes: candidate used, features demonstrated, build pipeline, embed compatibility status.
- **Recommendation**: The synthesis document. Relationships: references comparison matrix findings and PoC results, names one candidate, includes migration strategy.

### Deliverable Location

All research deliverables MUST be placed in `specs/1106-webui-framework-research/`:
- `matrix.md` — Comparison matrix document
- `recommendation.md` — Recommendation with migration strategy and risk assessment
- `poc/<candidate>/` — Proof-of-concept source code per candidate (each with its own README)

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Comparison matrix has zero empty cells — every candidate × criterion intersection contains an evidence-based finding (4 candidates × 9 criteria = 36 cells, all populated).
- **SC-002**: At least one PoC produces a working run_detail page where DAG renders, log stream updates in real time, and step cards reflect live status — all served from a single Go binary via go:embed.
- **SC-003**: Bundle size measurements are present for all candidates, quantified in KB, and compared against the current baseline (JS: ~124 KB, CSS: ~152 KB, total: ~276 KB).
- **SC-004**: The recommendation document names exactly one primary recommendation with explicit justification referencing at least 3 evaluation criteria by name.
- **SC-005**: The migration strategy identifies whether the approach is incremental or big-bang, and if incremental, specifies at least the first 3 pages to migrate in priority order with rationale.
- **SC-006**: Risk assessment identifies at least 3 risks and proposes a mitigation for each.
- **SC-007**: All deliverables (matrix, PoC(s), recommendation) are completed without modifying any existing Go backend code or API surface.

## Clarifications

The following ambiguities were identified and resolved during specification refinement:

### C1: PoC Quantity — Minimum vs Target
**Question**: User Story 2 references "top 1-2 candidate frameworks" while FR-003 requires "at least one." How many PoCs are expected?  
**Resolution**: FR-003 sets the binding minimum at 1. The target is 2 PoCs (for the top two candidates) if time and candidate viability permit. FR-003 updated to reflect this.  
**Rationale**: Aligns the requirement floor with the aspirational scope from User Story 2 without making a second PoC a hard blocker.

### C2: Deliverable Format and Location
**Question**: The spec did not specify where deliverables (matrix, PoCs, recommendation) should live in the repository or what format they should use.  
**Resolution**: All deliverables placed in `specs/1106-webui-framework-research/` — matrix and recommendation as markdown files, PoC source code in `poc/<candidate>/` subdirectories. A "Deliverable Location" section was added to Key Entities.  
**Rationale**: Consistent with existing `specs/` directory conventions. Markdown is reviewable in PRs. PoC code lives alongside the research for self-contained review.

### C3: Template Function Count Correction
**Question**: Spec referenced "37 custom template functions" but the actual count in `embed.go` is 30.  
**Resolution**: Corrected all references from 37 to 30 (Edge Case 2, User Story 3).  
**Rationale**: Verified by enumerating the `funcMap` entries in `internal/webui/embed.go:56-121`. Accurate baselines are critical for migration effort estimation.

### C4: JS Bundle Size Correction
**Question**: Spec stated "~280 KB of JS across 5 files" but actual JS totals ~124 KB; the ~280 KB figure is the total of JS + CSS combined.  
**Resolution**: Corrected FR-007 and SC-003 to: JS ~124 KB, CSS ~152 KB, total ~276 KB.  
**Rationale**: Measured from `internal/webui/static/` — 5 JS files sum to 126,830 bytes (~124 KB), `style.css` is 155,942 bytes (~152 KB). Accurate baseline prevents misleading bundle comparisons.

### C5: Rating Methodology for Comparison Matrix
**Question**: The matrix entity mentions "evidence-based ratings" but no scale or scoring system was defined.  
**Resolution**: Added a qualitative 5-point scale (Strong / Good / Adequate / Weak / Fail) with supporting evidence text per cell to the Comparison Matrix entity definition.  
**Rationale**: Qualitative scale with evidence text is the industry standard for framework evaluation research. Avoids false precision of numeric scores while enabling clear comparison.
