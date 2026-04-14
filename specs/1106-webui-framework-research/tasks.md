# Tasks: WebUI Framework Research

**Branch**: `1106-webui-framework-research` | **Date**: 2026-04-14  
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)  
**Total Tasks**: 42 | **Parallelizable**: 20

## Task Format

```
- [ ] [TaskID] [P?] [Story?] Description with file path
```

- `[P]` = parallelizable with sibling tasks in the same phase
- Story tags: `[US1]`=Comparison Matrix, `[US2]`=PoC, `[US3]`=Recommendation, `[US4]`=Elimination

---

## Phase 1: Setup

Initialize deliverable structure. No content yet — just scaffolding.

- [ ] T001 Create deliverable directory stubs: empty `matrix.md`, `recommendation.md`, and `poc/` directory → `specs/1106-webui-framework-research/`
- [ ] T002 Write matrix.md header section: document baseline metrics (JS: ~124 KB / 5 files, CSS: ~152 KB, total: ~276 KB), rating scale (Strong/Good/Adequate/Weak/Fail), and column headers for all 4 candidates → `specs/1106-webui-framework-research/matrix.md`

---

## Phase 2: Foundational — Candidate Screening (US4, P3)

Screen all candidates against hard constraints before investing in full evaluation. Parallelizable per candidate.

- [ ] T003 [P] [US4] Research Ripple framework viability: check GitHub stars (threshold: ≥100), last release date (threshold: within 12 months), static build output compatibility with go:embed, and document elimination rationale if any hard constraint fails → `specs/1106-webui-framework-research/matrix.md` (elimination section)
- [ ] T004 [P] [US4] Verify Svelte/SvelteKit hard constraints: confirm `@sveltejs/adapter-static` produces static HTML/JS/CSS embeddable via `//go:embed`, confirm Node.js required only at build time (not runtime) → `specs/1106-webui-framework-research/matrix.md` (candidate profiles section)
- [ ] T005 [P] [US4] Verify Astro hard constraints: confirm `output: 'static'` mode produces embeddable static files, confirm Node.js build-only, document any SSR mode that would require Node.js runtime → `specs/1106-webui-framework-research/matrix.md` (candidate profiles section)
- [ ] T006 [P] [US4] Verify htmx hard constraints: confirm it ships as a single JS file embeddable directly, no build step, no Node.js at build or runtime, confirm `hx-ext="sse"` extension availability → `specs/1106-webui-framework-research/matrix.md` (candidate profiles section)
- [ ] T007 [US4] Write elimination section in matrix.md for any candidate(s) that failed hard constraints: cite specific failing constraint, provide ecosystem maturity data (stars, contributors, release cadence), cross-reference to candidate profile → `specs/1106-webui-framework-research/matrix.md`

---

## Phase 3: User Story 1 — Framework Comparison Matrix (P1)

Populate all 36 cells (4 candidates × 9 criteria). Criterion rows are parallelizable.

- [ ] T008 [P] [US1] Research and write **embedding story** criterion row: for each candidate, describe exactly how build output maps to `go:embed` directive, what pipeline changes are needed, and whether the binary serves the page without external files — rate each Strong/Good/Adequate/Weak/Fail with ≥2 sentences evidence → `specs/1106-webui-framework-research/matrix.md`
- [ ] T009 [P] [US1] Research and write **SSE compatibility** criterion row: for each candidate, document native SSE/reactive primitives, whether `EventSource` is used directly or needs a wrapper, how reactive DOM updates work when SSE events arrive, include code snippet or reference — rate each with evidence → `specs/1106-webui-framework-research/matrix.md`
- [ ] T010 [P] [US1] Research and write **migration path** criterion row: for each candidate, assess incremental adoption feasibility (can it coexist with Go templates page-by-page?), whether existing URL structure is preserved, and impact on 24 existing page templates — rate each with evidence → `specs/1106-webui-framework-research/matrix.md`
- [ ] T011 [P] [US1] Research and write **bundle size** criterion row: measure or estimate production build output in KB (raw and gzipped) for each candidate for a run_detail equivalent page, compare against baseline (JS: ~124 KB, CSS: ~152 KB, total: ~276 KB), document framework runtime overhead — all measurements in KB → `specs/1106-webui-framework-research/matrix.md`
- [ ] T012 [P] [US1] Research and write **developer experience** criterion row: for each candidate, assess learning curve for Go developers, quality of TypeScript support, debugging tooling, IDE support, and hot-reload/dev server experience — rate each with evidence → `specs/1106-webui-framework-research/matrix.md`
- [ ] T013 [P] [US1] Research and write **build complexity** criterion row: for each candidate, document number of build steps added to CI pipeline, Node.js version requirements, build tool configuration files added, and estimated CI time impact — rate each with evidence → `specs/1106-webui-framework-research/matrix.md`
- [ ] T014 [P] [US1] Research and write **community & longevity** criterion row: for each candidate, document GitHub stars, release cadence (releases per year), corporate/VC backing, number of contributors, and Stack Overflow/ecosystem activity — rate each with evidence → `specs/1106-webui-framework-research/matrix.md`
- [ ] T015 [P] [US1] Research and write **component reuse** criterion row: for each candidate, evaluate extraction feasibility for all 6 existing partials (step_card, dag_svg, run_row, child_run_row, artifact_viewer, resume_dialog) — assess how each partial maps to framework's component model, migration complexity, and whether shared logic (statusIcon, formatDuration, friendlyModel) migrates as utility functions or components → `specs/1106-webui-framework-research/matrix.md`
- [ ] T016 [P] [US1] Research and write **auth integration** criterion row: for each candidate, assess CSRF token handling compatibility with Wave's 4 auth modes (none, bearer, JWT, mTLS), whether middleware.go CSRF patterns need changes, and how mutations (form submissions, API calls) include CSRF tokens in the framework's paradigm — rate each with evidence → `specs/1106-webui-framework-research/matrix.md`
- [ ] T017 [US1] Write handler impact assessment section in matrix.md: categorize each candidate's impact on the Go handler layer (~21 handler files, ~6,700+ lines) as "No change / Template removal / Adapter needed", list specific files that would be affected if any, verify FR-009 compliance (no backend API surface changes) → `specs/1106-webui-framework-research/matrix.md`
- [ ] T018 [US1] Validate matrix completeness: verify exactly 36 cells are populated (4 columns × 9 rows), every cell has a rating from the approved scale and ≥2 sentences of evidence, bundle size row has numeric KB values, no unsupported claims → `specs/1106-webui-framework-research/matrix.md`

**Phase 3 independent test**: Matrix document complete with all 36 cells, each with rating + evidence. Review checklist from `contracts/matrix-contract.md` fully satisfied.

---

## Phase 4: User Story 2 — Proof-of-Concept Implementation (P1)

Build working run_detail reimplementation in top 1–2 candidates. Depends on Phase 3 results.

- [ ] T019 [US2] Select top 1–2 PoC candidates based on matrix results: document selection rationale in research.md (which candidates scored highest on Critical criteria: embedding + SSE), name candidate-1 and optionally candidate-2 → `specs/1106-webui-framework-research/research.md`
- [ ] T020 [US2] Initialize PoC 1 project structure: create `poc/<candidate-1>/` directory, scaffold build toolchain config (package.json/vite.config.js or equivalent), configure build output to `dist/` directory, verify `go build` with `//go:embed poc/<candidate-1>/dist/*` works → `specs/1106-webui-framework-research/poc/<candidate-1>/`
- [ ] T021 [US2] Implement go:embed integration for PoC 1: write `embed_server.go` in poc directory that embeds `dist/*`, registers a route to serve the run_detail page, and can be launched standalone with `go run embed_server.go` against a running Wave instance → `specs/1106-webui-framework-research/poc/<candidate-1>/embed_server.go`
- [ ] T022 [P] [US2] Implement SSE client for PoC 1: connect to existing `/api/runs/{id}/events` endpoint using `EventSource`, handle all 7 event types (started, running, completed, failed, step_progress, stream_activity, eta_updated), implement Last-Event-ID header for reconnection backfill, add connection status indicator → `specs/1106-webui-framework-research/poc/<candidate-1>/src/`
- [ ] T023 [P] [US2] Implement DAG visualization component for PoC 1: render pipeline steps as a directed graph with status-colored nodes (pending/running/completed/failed), update node colors in real time via SSE step_progress events, match the visual layout of the existing `dag_svg.html` partial → `specs/1106-webui-framework-research/poc/<candidate-1>/src/`
- [ ] T024 [P] [US2] Implement log streaming component for PoC 1: display log lines in real time from SSE stream_activity events, support follow/pause scroll behavior (auto-scroll to bottom when following, freeze when user scrolls up), display log lines without page reload → `specs/1106-webui-framework-research/poc/<candidate-1>/src/`
- [ ] T025 [P] [US2] Implement step cards component for PoC 1: render one card per pipeline step with status badge (pending/running/completed/failed), update card status live via SSE step_progress events, collapse/expand support for logs section → `specs/1106-webui-framework-research/poc/<candidate-1>/src/`
- [ ] T026 [US2] Build PoC 1 and verify go:embed: run production build, confirm output is static files in `dist/`, run `go build` with embed directive, start binary and navigate to run detail page, confirm all 4 core features work (DAG renders, logs stream, step cards update, no external file deps) → `specs/1106-webui-framework-research/poc/<candidate-1>/`
- [ ] T027 [US2] Write PoC 1 README: document build prerequisites, step-by-step build instructions, what features are demonstrated, known limitations, empirical findings (build time, output size, developer experience notes) → `specs/1106-webui-framework-research/poc/<candidate-1>/README.md`
- [ ] T028 [P] [US2] (If 2nd candidate viable per T019) Initialize PoC 2 project structure and scaffold build toolchain → `specs/1106-webui-framework-research/poc/<candidate-2>/`
- [ ] T029 [P] [US2] (If 2nd candidate viable) Implement PoC 2 core features: SSE client (T022 equivalent), DAG visualization (T023 equivalent), log streaming (T024 equivalent), step cards (T025 equivalent) → `specs/1106-webui-framework-research/poc/<candidate-2>/src/`
- [ ] T030 [US2] (If 2nd candidate viable) Build PoC 2, verify go:embed, write README with empirical findings → `specs/1106-webui-framework-research/poc/<candidate-2>/`

**Phase 4 independent test**: Run each PoC binary, navigate to run_detail page during active run, verify: DAG renders with colored nodes, logs stream without page reload, step cards update live, single binary serves page.

---

## Phase 5: User Story 3 — Migration Recommendation (P2)

Synthesize matrix + PoC findings into actionable recommendation. Depends on Phases 3 and 4.

- [ ] T031 [US3] Write winner selection section: name exactly one framework as primary recommendation, write justification referencing ≥3 named evaluation criteria with specific evidence from matrix findings and PoC empirical results, optionally name runner-up with rationale for non-selection → `specs/1106-webui-framework-research/recommendation.md`
- [ ] T032 [US3] Write migration strategy section: declare incremental or big-bang approach, if incremental specify ≥3 pages to migrate first in priority order with rationale for each, explain coexistence strategy (how Go templates and new framework coexist per-page during migration), estimate effort per phase → `specs/1106-webui-framework-research/recommendation.md`
- [ ] T033 [US3] Write risk assessment section: identify ≥3 risks each with title, severity (High/Medium/Low), likelihood (High/Medium/Low), description, and mitigation — must cover: build pipeline complexity + CI impact, Node.js build toolchain requirement for contributors, developer learning curve, and any risks specific to the recommended framework → `specs/1106-webui-framework-research/recommendation.md`
- [ ] T034 [P] [US3] Write template function migration strategy subsection: for each of the 30 custom template functions in `embed.go` funcMap (statusIcon, formatDuration, friendlyModel, etc.), propose migration as utility function, component helper, or framework filter; group similar functions; estimate migration effort → `specs/1106-webui-framework-research/recommendation.md`
- [ ] T035 [P] [US3] Write handler test suite impact section: assess whether ~1,900 lines of handler tests in `internal/webui/` need changes, categorize impact for each candidate test (no change / add new tests / modify existing), verify SC-007 compliance (no existing handler tests broken) → `specs/1106-webui-framework-research/recommendation.md`
- [ ] T036 [P] [US3] Write authentication mode compatibility section: document how recommended framework handles each of 4 auth modes (none, bearer, JWT, mTLS), CSRF token injection pattern for mutations, whether `middleware.go` patterns need complementary client-side changes → `specs/1106-webui-framework-research/recommendation.md`
- [ ] T037 [US3] Add cross-references: link/cite specific matrix cells by criterion name (≥3 citations), cite specific PoC empirical findings (build output size, SSE performance, developer experience), address all 5 edge cases from spec (SSE fallback, template functions, Node.js CI, auth modes, handler test suite) → `specs/1106-webui-framework-research/recommendation.md`

**Phase 5 independent test**: Recommendation names one winner, ≥3 criteria cited, migration phases specified, ≥3 risks with mitigations. Validate against `contracts/recommendation-contract.md` checklist.

---

## Phase 6: Polish & Cross-Cutting Concerns

Final validation against all contracts and success criteria.

- [ ] T038 [P] Validate matrix against contract: run through all 11 checklist items in `contracts/matrix-contract.md` — confirm 4 columns, 9 rows, 36 non-empty cells, rating scale used, evidence text present, bundle sizes in KB, handler impact section included, component reuse covers all 6 partials → `specs/1106-webui-framework-research/matrix.md`
- [ ] T039 [P] Validate recommendation against contract: run through all 14 checklist items in `contracts/recommendation-contract.md` — winner named, ≥3 criteria cited, migration phases ≥3 (if incremental), ≥3 risks, edge cases addressed, no backend API changes proposed → `specs/1106-webui-framework-research/recommendation.md`
- [ ] T040 [P] Validate PoC(s) against contract: run through all 16 checklist items in `contracts/poc-contract.md` — go:embed works, SSE connects, DAG renders, step cards live, no `internal/` files modified → `specs/1106-webui-framework-research/poc/`
- [ ] T041 Verify SC-007 compliance: run `git diff --name-only internal/ | wc -l` and confirm output is `0` — no existing Go code was modified; all deliverables are additive files in `specs/` only → repository root
- [ ] T042 Run handler test suite: execute `go test ./...` and confirm all tests pass — verifies no regressions were introduced by PoC scaffolding or any accidental changes → repository root

**Phase 6 independent test**: All 7 success criteria met (SC-001 through SC-007), `go test ./...` passes, `git diff internal/` is empty.
