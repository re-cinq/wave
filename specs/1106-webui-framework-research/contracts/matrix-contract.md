# Contract: Comparison Matrix

**Deliverable**: `specs/1106-webui-framework-research/matrix.md`  
**Type**: Structural (document completeness)  
**Validates**: FR-001, FR-002, FR-007, FR-008, FR-010, SC-001, SC-003

## Acceptance Criteria

### Structure Requirements

1. **Document contains a matrix table** with candidates as columns and criteria as rows
2. **Exactly 4 candidate columns**: Svelte/SvelteKit, Ripple, Astro, htmx (FR-001)
3. **Exactly 9 criteria rows**: embedding, migration, SSE, bundle size, devexp, build complexity, community, component reuse, auth integration (FR-002)
4. **36 cells populated** — zero empty cells (SC-001)

### Content Requirements

5. **Each cell contains**:
   - A rating: one of Strong / Good / Adequate / Weak / Fail
   - Evidence text: minimum 2 sentences of supporting analysis
   - Code snippet or technical reference where applicable

6. **Bundle size row** includes numeric KB measurements for each candidate compared to baseline (JS: ~127 KB, CSS: ~156 KB, total: ~283 KB) (FR-007, SC-003)

7. **Embedding row** describes how each candidate's build output maps to `go:embed` and what pipeline changes are needed (spec acceptance scenario US1-2)

8. **SSE row** addresses real-time streaming patterns with concrete examples (spec acceptance scenario US1-3)

9. **Handler impact section** assesses impact on Go handler layer per candidate (FR-008):
   - Category: No change / Template removal / Adapter needed
   - Specific files affected (if any)

10. **Component reuse section** evaluates extraction feasibility for all 6 partials: step_card, dag_svg, run_row, child_run_row, artifact_viewer, resume_dialog (FR-010)

### Elimination Documentation

11. If any candidate is eliminated, the matrix includes:
    - Which hard constraint failed
    - Specific technical evidence for the failure
    - At minimum: ecosystem maturity data (stars, contributors, release cadence)

## Validation Method

Manual review checklist — verify all 11 criteria above. Automated: count table cells to confirm 36 populated intersections.
