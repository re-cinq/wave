# Research: WebUI Framework Evaluation

**Feature**: 1106-webui-framework-research  
**Date**: 2026-04-14  
**Phase**: 0 — Outline & Research

## Unknowns Extracted from Spec

### U1: Ripple Framework Maturity

**Status**: Requires early investigation  
The spec lists "Ripple" as a candidate but this is a less established framework compared to Svelte, Astro, and htmx. If Ripple fails hard constraints (go:embed compatibility, community viability), it should be eliminated per User Story 4 before full evaluation effort is spent.

**Decision**: Evaluate Ripple against hard constraints first (go:embed output format, active maintenance, community size). If any hard constraint fails, document elimination rationale and focus PoC effort on remaining candidates.  
**Rationale**: FR-001 requires all four be evaluated in the matrix, but FR-003 only requires PoC for top 1–2. Early elimination saves PoC effort without violating spec.  
**Alternatives**: (a) Full evaluation of all four equally — rejected because spec allows early elimination (User Story 4). (b) Skip Ripple entirely — rejected because FR-001 requires it in the matrix.

### U2: PoC Scope Boundaries for run_detail

**Status**: Resolved  
The spec requires reimplementing `run_detail.html` which is the most complex page. The current page involves:
- DAG visualization (`dag.js`, 338 lines + `dag_svg.html` partial + `dag.go` server-side generation)
- Log streaming (`log-viewer.js`, 1,165 lines with search, filtering, batch updates)
- Step cards (`step_card.html` partial with collapse/expand, status badges, artifact viewing)
- SSE integration (`sse.js`, 512 lines with EventSource, Last-Event-ID backfill, polling fallback)
- Artifact viewing (`artifact_viewer.html` partial)

**Decision**: PoC reimplements the core interactive behaviors (DAG renders, logs stream, step cards update) but does NOT need to replicate every edge case (log search/Ctrl+G, diff viewer, export). The PoC proves integration feasibility, not feature parity.  
**Rationale**: Feature parity would turn a research spike into a full rewrite. The PoC must demonstrate: (1) go:embed works, (2) SSE streams, (3) DAG renders, (4) step cards reflect live status. These are the four acceptance criteria from User Story 2.  
**Alternatives**: Full feature parity PoC — rejected due to scope explosion for a research deliverable.

### U3: Bundle Size Measurement Methodology

**Status**: Resolved  
The spec requires bundle size comparisons (FR-007, SC-003) against the current baseline: JS ~127 KB (5 files), CSS ~156 KB, total ~283 KB. Need consistent measurement.

**Decision**: Measure production build output sizes (minified + gzipped where applicable). Report both raw and gzipped sizes. Use `wc -c` on build output for raw, `gzip -c | wc -c` for compressed. Include any framework runtime overhead.  
**Rationale**: Raw size matters for go:embed binary size impact. Gzipped size matters for transfer. Both needed for fair comparison since some frameworks ship larger runtimes.  
**Alternatives**: (a) Only raw sizes — rejected, doesn't capture compression advantages of different approaches. (b) Include tree-shaking analysis — rejected as excessive for research scope.

### U4: SSE Compatibility Testing Approach

**Status**: Resolved  
Current SSE architecture: `SSEBroker` (Go) → `EventSource` (browser) with 7 event types, Last-Event-ID backfill, 3s polling fallback, 15s keepalive. Need to evaluate how each framework handles this.

**Decision**: For each candidate, document: (a) native SSE/reactive primitives available, (b) whether EventSource can be used directly or needs a wrapper, (c) how reactive DOM updates work when SSE events arrive, (d) whether the existing Go SSE endpoint needs modification (it should NOT per FR-009).  
**Rationale**: The SSE endpoint is backend — frameworks only need to consume it client-side. The key question is how naturally each framework handles reactive updates from an external event stream.  
**Alternatives**: Build custom SSE abstraction per framework — rejected, the current vanilla JS approach already works; the question is whether a framework improves or complicates it.

### U5: go:embed Integration Strategy

**Status**: Resolved  
Current approach: `//go:embed static/*` and `//go:embed templates/*` in `embed.go`. Static files served directly, templates parsed at startup. Any framework must produce output compatible with this pattern.

**Decision**: Evaluate two embedding patterns per candidate: (a) framework builds to static HTML/JS/CSS → embed as static assets (replacing current `static/` dir), (b) framework uses SSR that requires a Node.js runtime — this would violate Constitution Principle 1 (single binary, no runtime deps). Pattern (a) is the only acceptable path.  
**Rationale**: Constitution Principle 1 explicitly prohibits Node.js runtime dependency. Any framework requiring server-side Node.js at runtime is automatically disqualified from primary recommendation, though it may still be documented in the matrix.  
**Alternatives**: Ship Node.js alongside Go binary — explicitly rejected by constitution.

### U6: Handler Layer Impact Assessment

**Status**: Resolved  
FR-008 requires assessing impact on the Go handler layer (~21 handler files). FR-009 prohibits backend API changes.

**Decision**: For each candidate, categorize handler impact as: (a) No change — framework consumes existing HTML template output or API endpoints as-is, (b) Template removal — framework replaces Go templates, handlers return JSON only, existing `/api/` endpoints suffice, (c) Adapter needed — handlers need new endpoints or response formats. Category (b) is the expected path for SPA-style frameworks; (a) for progressive enhancement frameworks like htmx.  
**Rationale**: The current architecture serves both HTML (template-rendered pages) and JSON (`/api/` endpoints for SSE and AJAX). SPA frameworks would only use JSON endpoints; htmx would likely use HTML endpoints with modified fragments.  
**Alternatives**: Require zero handler changes as hard constraint — rejected because even htmx may need fragment-specific endpoints, which is new code not modification of existing code.

## Technology Assessment Summary

### Evaluation Criteria Weights (from issue context)

| # | Criterion | Weight | Measurement |
|---|-----------|--------|-------------|
| 1 | Embedding story (go:embed) | **Critical** | Binary builds, serves from single binary |
| 2 | Migration path | High | Incremental adoption possible, coexistence with Go templates |
| 3 | SSE compatibility | **Critical** | EventSource consumption, reactive DOM updates |
| 4 | Bundle size | Medium | Raw + gzipped vs. 283 KB baseline |
| 5 | Developer experience | Medium | Learning curve, tooling, debugging |
| 6 | Build complexity | High | Build steps added, CI impact, Node.js requirement |
| 7 | Community & longevity | Medium | Stars, releases, corporate backing, adoption |
| 8 | Component reuse | Medium | Partial extraction feasibility (6 partials) |
| 9 | Auth integration | High | CSRF token flow, no auth mode regression |

**Hard constraints** (automatic fail):
- Cannot produce static output compatible with go:embed
- Requires Node.js at runtime (violates Constitution P1)
- Requires Go backend API changes (violates FR-009)

### Framework Quick Profiles

**Svelte/SvelteKit**  
- Compiler-based: compiles to vanilla JS at build time, zero runtime overhead
- SvelteKit provides SSR/SSG but can output static site (adapter-static)
- Requires Node.js at BUILD time only (acceptable per constitution — build tools are optional)
- Reactive by default: `$:` statements, stores, fine-grained DOM updates
- Strong ecosystem: 80k+ GitHub stars, active development, corporate backing (Vercel)

**Ripple**  
- Less established framework — needs immediate viability assessment
- Key question: does it produce static build output?
- Key question: ecosystem maturity (stars, releases, contributors)
- Risk: may be eliminated early per User Story 4

**Astro**  
- Content-focused framework with "islands" architecture
- Ships zero JS by default, adds interactivity via "islands" (Svelte, React, Vue components)
- Static site generation (SSG) is primary mode — natural go:embed fit
- Can use any UI framework for interactive islands
- Build-time Node.js only
- 50k+ GitHub stars, backed by venture capital

**htmx**  
- Progressive enhancement library (14 KB minified)
- No build step required — works with Go templates directly
- Server returns HTML fragments, htmx swaps DOM regions
- SSE support via `hx-ext="sse"` extension
- Minimal migration: add attributes to existing templates
- 40k+ GitHub stars, philosophically aligned with Go server-rendered approach

### Approach Strategy

**Phase order for implementation**:
1. **Candidate screening** — Verify hard constraints for all 4 (go:embed, no runtime Node.js, no API changes)
2. **Matrix population** — Evaluate each criterion for each candidate, fill all 36 cells
3. **PoC candidate selection** — Pick top 2 from matrix (or top 1 + second if viable)
4. **PoC implementation** — Reimplement run_detail core features
5. **Recommendation synthesis** — Combine matrix + PoC findings into recommendation

**Expected outcomes based on initial analysis**:
- htmx has the lowest migration risk (progressive enhancement, no build step)
- Svelte likely wins on developer experience and component model
- Astro offers a middle ground (static-first with interactive islands)
- Ripple is the highest-risk candidate for elimination
