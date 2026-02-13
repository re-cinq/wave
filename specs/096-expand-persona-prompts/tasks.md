# Tasks: Expand Persona Definitions with Detailed System Prompts

**Feature**: 096-expand-persona-prompts
**Generated**: 2026-02-13
**Source**: spec.md, plan.md, research.md, data-model.md

> **Context**: Commit `6fdb3e9` already expanded all 13 persona files in `.wave/personas/`.
> The remaining scope is: fix 4 FR-008 violations (language-specific references),
> sync all 13 files to `internal/defaults/personas/` for parity, and validate all requirements.

---

## Phase 1: Setup

- [X] T001 [P1] Verify current state of all 13 persona files in `.wave/personas/` — confirm line counts and structural template conformance match research.md findings
  - File: `.wave/personas/*.md`
  - Verify: All 13 files exist, each has 30–200 lines, all 7 structural concepts present
  - Verify: Identify exactly which files fail FR-008 (expected: craftsman, reviewer, auditor, debugger)

---

## Phase 2: FR-008 Fixes — Language-Agnostic Persona Content (US2)

> User Story 2: Language-agnostic persona definitions (P1)
> These 4 files contain hardcoded language-specific references that must be generalized.
> Depends on: T001 (verification of current state)

- [X] T002 [P] [P1] [US2] Fix FR-008 violations in `craftsman.md` — remove Go-specific references
  - File: `.wave/personas/craftsman.md`
  - Edit line 12: `Go conventions including effective Go practices, formatting, and idiomatic patterns` → `Language conventions and idiomatic patterns for the target codebase`
  - Edit line 46: `go test, go build, go vet, etc.` → `build, test, and static analysis commands for the project's toolchain`

- [X] T003 [P] [P1] [US2] Fix FR-008 violations in `reviewer.md` — remove language-specific test runner references
  - File: `.wave/personas/reviewer.md`
  - Edit line 35: `` Run available tests (`go test`, `npm test`) to verify passing state `` → `Run the project's test suite to verify passing state`
  - Edit line 46: `Bash(go test*)` → `Bash(...)`: Run the project's test suite to validate implementation behavior
  - Edit line 47: `Bash(npm test*)` → remove (merged into generic description above)

- [X] T004 [P] [P1] [US2] Fix FR-008 violations in `auditor.md` — remove Go-specific identity, expertise, and tools
  - File: `.wave/personas/auditor.md`
  - Edit line 3: `specializing in Go systems` → `specializing in software systems`
  - Edit line 14: `Go-specific security concerns: unsafe pointer usage, race conditions, path traversal` → `Language-specific security concerns: memory safety, race conditions, path traversal, type confusion`
  - Edit line 33: `` Run static analysis tools (`go vet`) `` → `Run static analysis tools available in the project's toolchain`
  - Edit line 42: `Bash(go vet*)` → `Bash(...)`: Run static analysis tools for the project's toolchain
  - Edit line 43: `Bash(npm audit*)` → `Bash(...)`: Check dependency vulnerabilities when applicable

- [X] T005 [P] [P1] [US2] Fix FR-008 violations in `debugger.md` — remove Go-specific identity, expertise, and tools
  - File: `.wave/personas/debugger.md`
  - Edit line 2: `specializing in Go systems` → `specializing in software systems`
  - Edit line 9: `concurrent Go programs` → `concurrent programs`
  - Edit line 13: `Go-specific debugging: goroutine leaks, race conditions, deadlocks, channel misuse` → `Concurrency debugging: race conditions, deadlocks, resource leaks, and synchronization issues`
  - Edit line 51: `Bash(go test*)` → `Bash(...)`: Run the project's test suite to reproduce failures and validate hypotheses

> **Parallelism**: T002, T003, T004, T005 are independent file edits and can execute in parallel.

---

## Phase 3: Parity Sync — .wave/ to internal/defaults/ (US3)

> User Story 3: Byte-identical parity between `.wave/personas/` and `internal/defaults/personas/` (P1)
> Depends on: Phase 2 completion (FR-008 fixes must land in `.wave/personas/` before syncing)

- [X] T006 [P] [P1] [US3] Copy `navigator.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/navigator.md`
  - Target: `internal/defaults/personas/navigator.md`

- [X] T007 [P] [P1] [US3] Copy `philosopher.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/philosopher.md`
  - Target: `internal/defaults/personas/philosopher.md`

- [X] T008 [P] [P1] [US3] Copy `planner.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/planner.md`
  - Target: `internal/defaults/personas/planner.md`

- [X] T009 [P] [P1] [US3] Copy `craftsman.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/craftsman.md`
  - Target: `internal/defaults/personas/craftsman.md`

- [X] T010 [P] [P1] [US3] Copy `implementer.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/implementer.md`
  - Target: `internal/defaults/personas/implementer.md`

- [X] T011 [P] [P1] [US3] Copy `reviewer.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/reviewer.md`
  - Target: `internal/defaults/personas/reviewer.md`

- [X] T012 [P] [P1] [US3] Copy `auditor.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/auditor.md`
  - Target: `internal/defaults/personas/auditor.md`

- [X] T013 [P] [P1] [US3] Copy `debugger.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/debugger.md`
  - Target: `internal/defaults/personas/debugger.md`

- [X] T014 [P] [P1] [US3] Copy `researcher.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/researcher.md`
  - Target: `internal/defaults/personas/researcher.md`

- [X] T015 [P] [P1] [US3] Copy `summarizer.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/summarizer.md`
  - Target: `internal/defaults/personas/summarizer.md`

- [X] T016 [P] [P1] [US3] Copy `github-analyst.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/github-analyst.md`
  - Target: `internal/defaults/personas/github-analyst.md`

- [X] T017 [P] [P1] [US3] Copy `github-commenter.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/github-commenter.md`
  - Target: `internal/defaults/personas/github-commenter.md`

- [X] T018 [P] [P1] [US3] Copy `github-enhancer.md` from `.wave/personas/` to `internal/defaults/personas/`
  - Source: `.wave/personas/github-enhancer.md`
  - Target: `internal/defaults/personas/github-enhancer.md`

> **Parallelism**: T006–T018 are all independent file copies and can execute in parallel.

---

## Phase 4: Validation — Consistent Behavior & No Disruption (US1, US4, US5)

> User Stories 1, 4, 5: Consistent agent behavior, self-contained identity, no disruption
> Depends on: Phases 2 and 3 completion

- [X] T019 [P] [P1] [US4] Validate structural template conformance — verify all 7 required concepts present in each of the 13 persona files in `.wave/personas/`
  - Check: Identity statement ("You are...") within first 3 lines of body (FR-001)
  - Check: Domain Expertise section present (FR-002)
  - Check: Responsibilities section present (FR-003)
  - Check: Process/Methodology section present (FR-004)
  - Check: Tools and Permissions section present (FR-005)
  - Check: Output Format section present (FR-006)
  - Check: Constraints section present (FR-007)
  - Files: `.wave/personas/*.md` (all 13)

- [X] T020 [P] [P1] [US2] Validate FR-008 compliance — grep for language-specific toolchain references across all 26 persona files
  - Pattern: `go test`, `go vet`, `go build`, `npm test`, `npm audit`, `cargo build`, `pytest` (and similar hardcoded tool invocations)
  - Files: `.wave/personas/*.md` and `internal/defaults/personas/*.md`
  - Expected: Zero matches (SC-004)

- [X] T021 [P] [P1] [US4] Validate line count bounds — each persona file must be ≥30 and ≤200 lines (FR-009, FR-013)
  - Files: `.wave/personas/*.md` and `internal/defaults/personas/*.md` (26 files total)
  - Expected: All files in range [30, 200] (SC-002)

- [X] T022 [P1] [US3] Validate parity — run `diff -r .wave/personas/ internal/defaults/personas/` and confirm zero differences
  - Files: Both directories (26 files total)
  - Expected: Zero differences (SC-005)
  - Depends on: T006–T018

- [X] T023 [P1] [US5] Validate no source code changes — confirm no `.go` files, `wave.yaml`, or JSON schema files are in the change set
  - Check: Only `.md` files in `.wave/personas/` and `internal/defaults/personas/` are modified
  - FR: FR-011, SC-007

- [X] T024 [P1] [US5] Run full test suite — `go test ./...` must pass with zero failures
  - Command: `go test ./...`
  - Expected: All tests pass (SC-006, FR-012)

> **Parallelism**: T019, T020, T021 can run in parallel. T022 depends on Phase 3. T023, T024 depend on all prior phases.

---

## Phase 5: Polish & Cross-Cutting Concerns

- [X] T025 [P2] [US1] Spot-check persona prompt quality — review 3 representative personas (navigator, craftsman, debugger) for completeness, clarity, and self-containedness
  - Verify: Each prompt provides enough context for an LLM to perform its role without external documentation (US4)
  - Verify: Contract schema precedence note present in Output Format section (edge case from spec)
  - Verify: Pipeline orchestrator enforcement note present in Tools section (FR-005)
  - Verify: No language-specific tool references remain after FR-008 fixes

---

## Dependency Graph

```
T001 (verify current state)
  │
  ├──→ T002, T003, T004, T005 (FR-008 fixes — parallel)
  │       │
  │       └──→ T006–T018 (parity sync — parallel)
  │              │
  │              ├──→ T019, T020, T021 (validation — parallel)
  │              ├──→ T022 (parity validation)
  │              ├──→ T023 (no source changes check)
  │              └──→ T024 (test suite)
  │                     │
  │                     └──→ T025 (quality spot-check)
```

## Summary

| Phase | Tasks | Parallel Opportunities | Description |
|-------|-------|------------------------|-------------|
| 1: Setup | 1 | 0 | Verify current state |
| 2: FR-008 | 4 | 4 | Fix language-specific references |
| 3: Parity | 13 | 13 | Sync to internal/defaults/ |
| 4: Validation | 6 | 3 | Verify all requirements |
| 5: Polish | 1 | 0 | Quality spot-check |
| **Total** | **25** | **20** | |
