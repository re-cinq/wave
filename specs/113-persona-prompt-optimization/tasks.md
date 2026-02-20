# Tasks: Persona Prompt Optimization

**Feature Branch**: `113-persona-prompt-optimization`
**Date**: 2026-02-20
**Spec**: `specs/113-persona-prompt-optimization/spec.md`
**Plan**: `specs/113-persona-prompt-optimization/plan.md`

## Phase 1: Setup & Foundation

- [X] T001 P1 US1 Create `internal/defaults/personas/base-protocol.md` with Wave-universal operational context (~80–120 tokens) covering fresh context, artifact I/O, workspace isolation, contract compliance, and permission enforcement — `internal/defaults/personas/base-protocol.md`
- [X] T002 P1 US1 Copy `base-protocol.md` to `.wave/personas/base-protocol.md` (byte-identical) — `.wave/personas/base-protocol.md`

## Phase 2: Runtime Injection (US1 — Shared Base Protocol Extraction)

- [X] T003 P1 US1 Modify `prepareWorkspace` in `internal/adapter/claude.go` (lines 260–284) to read `.wave/personas/base-protocol.md` and prepend its content before the persona prompt, separated by `\n\n---\n\n`. If the file is missing, return an error (fail-secure). ~10 LOC addition — `internal/adapter/claude.go`
- [X] T004 P1 US1 Add unit test in `internal/adapter/claude_test.go` verifying that `prepareWorkspace` prepends base protocol content to the generated CLAUDE.md (assert "Wave Agent Protocol" heading present before persona content, separated by `---`) — `internal/adapter/claude_test.go`
- [X] T005 P1 US1 Add unit test in `internal/adapter/claude_test.go` verifying that `prepareWorkspace` returns an error when `base-protocol.md` is missing from `.wave/personas/` — `internal/adapter/claude_test.go`
- [X] T006 P1 US1 Add unit test in `internal/adapter/claude_test.go` verifying that base protocol is prepended even when `cfg.SystemPrompt` is set directly (inline prompt case) — `internal/adapter/claude_test.go`

## Phase 3: Persona Prompt Compaction — Core Personas (US2, P1)

All tasks in this phase are parallelizable. Each persona file must be optimized to contain only: (1) identity statement (H1), (2) role-specific responsibilities, (3) output contract section. Remove generic process descriptions, "Communication Style" sections, "Domain Expertise" restating responsibilities, shared contract boilerplate ("When a contract schema is provided..."), and any content now covered by the base protocol. Retain role-specific behavioral constraints as defense-in-depth. Target: 100–400 tokens.

- [X] T007 [P] P1 US2 Optimize `navigator.md` — remove generic content, retain identity + responsibilities + output contract. Target: 100–150 tokens — `internal/defaults/personas/navigator.md`
- [X] T008 [P] P1 US2 Optimize `implementer.md` — remove shared contract boilerplate, retain identity + responsibilities + output contract + behavioral constraints. Target: 100–180 tokens — `internal/defaults/personas/implementer.md`
- [X] T009 [P] P1 US2 Optimize `reviewer.md` — remove shared contract boilerplate and generic process steps, retain identity + responsibilities + output contract + behavioral constraints. Target: 100–180 tokens — `internal/defaults/personas/reviewer.md`
- [X] T010 [P] P1 US2 Optimize `planner.md` — remove generic process descriptions, retain identity + responsibilities + output contract. Target: 100–180 tokens — `internal/defaults/personas/planner.md`
- [X] T011 [P] P1 US2 Optimize `researcher.md` — remove "Research Process" generic workflow section, compact "Source Evaluation Criteria", retain identity + responsibilities + output contract. Target: 150–300 tokens — `internal/defaults/personas/researcher.md`
- [X] T012 [P] P1 US2 Optimize `debugger.md` — compact generic "Debugging Process" steps, retain identity + responsibilities + output contract. Target: 100–200 tokens — `internal/defaults/personas/debugger.md`
- [X] T013 [P] P1 US2 Optimize `auditor.md` — remove generic content, retain identity + responsibilities + output contract + behavioral constraints. Target: 100–180 tokens — `internal/defaults/personas/auditor.md`
- [X] T014 [P] P1 US2 Optimize `craftsman.md` — remove generic content, retain identity + responsibilities + output contract. Target: 100–180 tokens — `internal/defaults/personas/craftsman.md`
- [X] T015 [P] P1 US2 Optimize `summarizer.md` — remove generic content, retain identity + responsibilities + output contract. Target: 100–180 tokens — `internal/defaults/personas/summarizer.md`

## Phase 4: Persona Prompt Compaction — GitHub & Specialized Personas (US2, P1)

All tasks in this phase are parallelizable.

- [X] T016 [P] P1 US2 Optimize `github-analyst.md` — remove shared contract boilerplate, retain identity + responsibilities + output contract. Target: 100–250 tokens — `internal/defaults/personas/github-analyst.md`
- [X] T017 [P] P1 US2 Optimize `github-commenter.md` — remove shared contract boilerplate, retain identity + responsibilities + output contract. Target: 100–250 tokens — `internal/defaults/personas/github-commenter.md`
- [X] T018 [P] P1 US2 Optimize `github-enhancer.md` — remove shared contract boilerplate, retain identity + responsibilities + output contract. Target: 100–200 tokens — `internal/defaults/personas/github-enhancer.md`
- [X] T019 [P] P1 US2 Optimize `philosopher.md` — remove generic content, retain identity + responsibilities + output contract. Target: 100–150 tokens — `internal/defaults/personas/philosopher.md`
- [X] T020 [P] P1 US2 Optimize `provocateur.md` — remove shared contract boilerplate, retain "Thinking Style" and "Evidence Gathering" (role-differentiating), retain identity + responsibilities + output contract + behavioral constraints. Target: 200–400 tokens — `internal/defaults/personas/provocateur.md`
- [X] T021 [P] P1 US2 Optimize `validator.md` — remove generic content, retain identity + responsibilities + output contract. Target: 100–250 tokens — `internal/defaults/personas/validator.md`
- [X] T022 [P] P1 US2 Optimize `synthesizer.md` — remove generic content, retain identity + responsibilities + output contract. Target: 100–200 tokens — `internal/defaults/personas/synthesizer.md`
- [X] T023 [P] P1 US2 Optimize `supervisor.md` — retain "Evidence Gathering" and "Evaluation Criteria" (role-differentiating), remove shared contract boilerplate, retain identity + responsibilities + output contract + behavioral constraints. Target: 200–400 tokens — `internal/defaults/personas/supervisor.md`

## Phase 5: Parity Maintenance (US3, P2)

- [X] T024 P2 US3 Copy all 17 optimized persona files from `internal/defaults/personas/` to `.wave/personas/` (byte-identical) — `.wave/personas/*.md`
- [X] T025 P2 US3 Create a Go test that asserts byte-identical parity between all files in `internal/defaults/personas/` and `.wave/personas/` (including `base-protocol.md`). Place in `internal/defaults/parity_test.go` or `tests/parity_test.go`. Test should fail with a clear message identifying which files diverge — `internal/defaults/parity_test.go`

## Phase 6: Language-Agnostic Verification (US4, P2)

- [X] T026 [P] P2 US4 Verify no programming language references exist in any persona file or `base-protocol.md` by grepping for language keywords (Go, Golang, Python, TypeScript, JavaScript, Java, Rust, Ruby, Swift, Kotlin, C++, C#). Fix any found — `internal/defaults/personas/*.md`
- [X] T027 [P] P2 US4 Add a Go test asserting zero language-specific keyword matches across all persona files and `base-protocol.md`. Use regex patterns from `contracts/persona-validation.md` — `internal/defaults/personas_test.go`

## Phase 7: Validation & Polish

- [X] T028 P1 US2 Validate all 17 persona files are within 100–400 token range using word count heuristic (words × 100/75). Log results. Fix any outliers — `internal/defaults/personas/*.md`
- [X] T029 P1 US2 Validate all 17 persona files contain the three mandatory structural elements: H1 identity heading, responsibilities section, output contract section — `internal/defaults/personas/*.md`
- [X] T030 Run `go test ./...` to verify all existing tests pass with the changes. Fix any failures — all test files
- [X] T031 Run `go test -race ./...` to verify no race conditions. Fix any failures — all test files

## Dependency Graph

```
T001 → T002 → T003 → T004, T005, T006
                  ↓
            T007–T023 (all parallel, depend on T001 base protocol existing)
                  ↓
            T024 (depends on T007–T023 completion)
                  ↓
            T025 (depends on T024)
            T026, T027 (parallel, depend on T007–T023)
                  ↓
            T028, T029 (depend on T007–T023)
                  ↓
            T030 → T031
```

## Notes

- **17 personas**: navigator, implementer, reviewer, planner, researcher, debugger, auditor, craftsman, summarizer, github-analyst, github-commenter, github-enhancer, philosopher, provocateur, validator, synthesizer, supervisor
- **Token estimation heuristic**: ~0.75 words per token → `word_count × (100/75)` ≈ token count
- **No new dependencies**: base-protocol.md leverages existing `//go:embed personas/*.md` directive
- **No manifest changes**: base protocol is not a persona — no `wave.yaml` entry needed
- **Init compatibility**: `GetPersonas()` in `embed.go` naturally includes `base-protocol.md`; init test (`init_test.go:821`) asserts `len(GetPersonas()) == len(entries)` so both counts increase by 1 — test passes without changes
