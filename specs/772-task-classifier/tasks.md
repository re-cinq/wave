# Tasks: Wave Task Classifier

**Branch**: `772-task-classifier` | **Date**: 2026-04-12
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Phase 1: Setup

- [X] T001 [P1] Create `internal/classify/` package directory and `profile.go` with package declaration, imports, all exported type definitions: `Complexity`, `Domain`, `VerificationDepth` string enums with constants, `TaskProfile` struct (BlastRadius float64, Complexity, Domain, VerificationDepth, InputType suggest.InputType), and `PipelineConfig` struct (Name string, Reason string). File: `internal/classify/profile.go`. FR-001, FR-008.

## Phase 2: Foundational — Type Validation (Story 1 prerequisite)

- [X] T002 [P1] [Story1] Create `internal/classify/profile_test.go` with 10+ table-driven tests validating: all 4 Complexity constants have correct string values, all 7 Domain constants have correct string values, all 3 VerificationDepth constants have correct string values, TaskProfile zero-value behavior, PipelineConfig field assignment. File: `internal/classify/profile_test.go`. FR-007, SC-003.

## Phase 3: Story 3 — Pipeline Selection (no analyzer dependency)

- [X] T003 [P1] [Story3] Implement `SelectPipeline(profile TaskProfile) PipelineConfig` in `internal/classify/selector.go`. Sequential priority evaluation: (1) InputType==pr_url→ops-pr-review, (2) Domain==security→audit-security, (3) Domain==research→impl-research, (4) Domain==docs→doc-fix, (5) Domain==refactor AND Complexity∈{complex,architectural}→impl-speckit, (6) Complexity∈{simple,medium}→impl-issue, (7) Complexity∈{complex,architectural}→impl-speckit. Each case sets descriptive Reason string. File: `internal/classify/selector.go`. FR-005, FR-006.

- [X] T004 [P1] [Story3] Create `internal/classify/selector_test.go` with 10+ table-driven tests covering: all 7 routing rules from FR-006, PR URL short-circuit, security override at every complexity level, domain-specific routes (research, docs), architectural refactor→impl-speckit, simple bug→impl-issue, medium feature→impl-issue, complex feature→impl-speckit, architectural feature→impl-speckit, Reason field non-empty for all cases. File: `internal/classify/selector_test.go`. FR-007, SC-002, SC-003.

## Phase 4: Story 1 & 2 — Input Classification

- [X] T005 [P1] [Story1,Story2] Implement `Classify(input string, issueBody string) TaskProfile` in `internal/classify/analyzer.go`. Steps: (1) call suggest.ClassifyInput(input) for InputType, (2) combine input+issueBody lowercased for analysis, (3) keyword-match domain with priority ordering security>performance>bug>refactor>research>docs>feature, (4) keyword-match complexity (architectural/complex/simple/medium default), (5) derive blast_radius = base(complexity) + modifier(domain) clamped [0,1], (6) derive verification_depth from complexity. Handle edge cases: empty/whitespace→default profile (simple/feature/0.1/structural_only), no keywords→medium/feature/0.3/behavioral. File: `internal/classify/analyzer.go`. FR-002, FR-003, FR-004, FR-009, FR-010, FR-011, FR-012.

- [X] T006 [P1] [Story1,Story2] Create `internal/classify/analyzer_test.go` with 10+ table-driven tests covering: all 7 domain classifications via keyword matching, all 4 complexity levels, empty string input defaults, whitespace-only input defaults, mixed domain signals (security wins over bug), URL input with issue body combination, PR URL InputType detection, no-keyword fallback to medium/feature, blast_radius ranges (security>0.5, docs+simple<0.2, architectural>0.7), verification_depth derivation (simple→structural_only, medium→behavioral, complex→full_semantic). File: `internal/classify/analyzer_test.go`. FR-007, SC-001, SC-003, SC-004.

## Phase 5: Story 4 — End-to-End Integration

- [X] T007 [P2] [Story4] Add integration test cases to `internal/classify/analyzer_test.go` that exercise the full classify-then-select flow: call Classify() then SelectPipeline() on the result. Cases: "fix null pointer in logger"→impl-issue (simple/bug), "implement new GraphQL API layer with auth, rate limiting, and caching"→impl-speckit (complex+/feature), "fix typo in README"→doc-fix (simple/docs), PR URL→ops-pr-review, "fix SQL injection vulnerability"→audit-security. Verify both TaskProfile fields and PipelineConfig.Name. File: `internal/classify/analyzer_test.go`. SC-001, SC-002.

## Phase 6: Polish & Cross-Cutting

- [X] T008 [P] [P2] Run `go vet ./internal/classify/...` and `go test ./internal/classify/...` to verify all files compile, all tests pass, no vet warnings. Fix any issues. File: `internal/classify/`.

- [X] T009 [P2] Verify SC-005 (no new external dependencies) by checking go.mod is unchanged. Verify SC-006 (<1ms classification) by adding a benchmark test `BenchmarkClassify` in `internal/classify/analyzer_test.go` that asserts single classification completes in <1ms. File: `internal/classify/analyzer_test.go`. SC-005, SC-006.

## Dependency Graph

```
T001 (types)
 ├── T002 (type tests) [parallel with T003]
 ├── T003 (selector)
 │    └── T004 (selector tests) [parallel with T005]
 └── T005 (analyzer, depends on T001+T003 types)
      └── T006 (analyzer tests)
           └── T007 (integration tests)
                └── T008 (vet + test all)
                     └── T009 (benchmarks + dep check)
```

## Parallel Opportunities

- T002 and T003 can run in parallel after T001
- T004 and T005 can run in parallel after T003
