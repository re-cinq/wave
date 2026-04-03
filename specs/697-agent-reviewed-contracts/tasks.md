# Tasks: Agent-Reviewed Contracts

**Feature**: #697 Agent-Reviewed Contracts
**Branch**: `697-agent-reviewed-contracts`
**Generated**: 2026-03-30
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Data Model**: [data-model.md](data-model.md)

---

## Phase 1 — Setup & Type Scaffolding

Foundational type definitions that all subsequent phases depend on.

- [X] T00X [P1] Add agent_review fields to `pipeline.ContractConfig` — add `Persona`, `CriteriaPath`, `Context` ([]ReviewContextSource), `TokenBudget`, `Timeout`, `ReworkStep` YAML fields to the struct at `internal/pipeline/types.go:424-440`
- [X] T00X [P1] Add `Contracts` plural field to `HandoverConfig` — add `Contracts []ContractConfig` to `HandoverConfig` at `internal/pipeline/types.go:417-422`
- [X] T00X [P1] Add `EffectiveContracts()` method to `HandoverConfig` — returns `Contracts` if non-empty, else wraps singular `Contract` in a slice, else nil. File: `internal/pipeline/types.go`
- [X] T00X [P1] Add agent_review fields to `contract.ContractConfig` — add `Persona`, `CriteriaPath`, `Context` ([]ReviewContextSource), `TokenBudget`, `Timeout`, `ReworkStep` JSON fields to the struct at `internal/contract/contract.go:10-35`
- [X] T00X [P1] Define `ReviewContextSource` type in contract package — struct with `Source` (string: "git_diff"|"artifact"), `Artifact` (string), `MaxSize` (int) at `internal/contract/agent_review.go`
- [X] T00X [P1] Define `ReviewFeedback` and `ReviewIssue` types in contract package — `ReviewFeedback` with Verdict/Issues/Suggestions/Confidence, `ReviewIssue` with Severity/Description, with JSON tags. File: `internal/contract/agent_review.go`
- [X] T00X [P1] [P] Unit tests for `EffectiveContracts()` — singular only, plural only, both set (plural wins), neither set (nil). File: `internal/pipeline/types_test.go`

---

## Phase 2 — Core Agent Review Validator (US1 + US5)

The `agent_review` contract type: validator, prompt building, feedback extraction, registry integration. Delivers P1 user stories 1 and 5.

- [X] T008 [P1] [US1] Create `agentReviewValidator` struct — implements `ContractValidator`, holds an `adapter.AdapterRunner` and manifest reference. Constructor: `newAgentReviewValidator(runner, manifest)`. File: `internal/contract/agent_review.go`
- [X] T009 [P1] [US1] Implement `buildReviewPrompt()` — assembles user prompt from: (1) criteria content loaded from `CriteriaPath`, (2) placeholder for assembled context (empty for now, wired in Phase 5), (3) `ReviewFeedback` JSON schema as required output format. File: `internal/contract/agent_review.go`
- [X] T010 [P1] [US1] Implement `parseReviewFeedback()` — extracts `ReviewFeedback` from agent stdout using existing `extractJSON()` helper, validates verdict enum (pass/fail/warn), validates confidence range [0.0, 1.0]. Returns parse error if extraction fails. File: `internal/contract/agent_review.go`
- [X] T011 [P1] [US1] Implement `Validate()` on `agentReviewValidator` — loads criteria from `CriteriaPath`, builds prompt, calls `runner.Run()` with `AdapterRunConfig` built from reviewer persona config (persona name, model override, workspace path, timeout), parses stdout for `ReviewFeedback`, returns error if verdict is "fail". File: `internal/contract/agent_review.go`
- [X] T012 [P1] [US1] Define `AgentContractValidator` interface — extends `ContractValidator` with `ValidateWithRunner(cfg ContractConfig, workspacePath string, runner adapter.AdapterRunner, manifest interface{}) (*ReviewFeedback, error)`. File: `internal/contract/contract.go`
- [X] T013 [P1] [US1] Add `ValidateWithRunner()` top-level function — dispatches to `agentReviewValidator` when type is `agent_review`, falls back to `Validate()` for all other types. File: `internal/contract/contract.go`
- [X] T014 [P1] [US1] Register `agent_review` in `NewValidator()` factory — add case to switch at `internal/contract/contract.go:75-94`. Note: `NewValidator` returns nil for agent_review since it needs a runner; the executor uses `ValidateWithRunner()` instead. Add a comment explaining this.
- [X] T015 [P1] [US5] [P] Unit tests for `parseReviewFeedback()` — valid JSON, JSON in markdown fences, missing fields, invalid verdict enum, confidence out of range, completely unparseable output. File: `internal/contract/agent_review_test.go`
- [X] T016 [P1] [US1] [P] Unit tests for `buildReviewPrompt()` — criteria content injection, schema format injection, empty criteria path error. File: `internal/contract/agent_review_test.go`
- [X] T017 [P1] [US1] Unit tests for `agentReviewValidator.Validate()` — mock adapter runner returning pass/fail/warn verdicts, runner error, stdout parse failure. File: `internal/contract/agent_review_test.go`

---

## Phase 3 — Pipeline Validation (US1)

Self-review prevention, criteria path existence, reviewer persona existence, token budget validation. All checked at pipeline load time.

- [X] T018 [P1] [US1] Add self-review prevention check — validate `contract.Persona != step.Persona` for each `agent_review` contract in `EffectiveContracts()`. Hard error at DAG validation time. File: `internal/pipeline/dag.go` (in `ValidateDAG` method, after existing rework validation)
- [X] T019 [P1] [US1] Add criteria path existence validation — for each `agent_review` contract, verify `CriteriaPath` file exists. Hard error at pipeline load. File: `internal/pipeline/validation.go` (new `ValidateAgentReviewContracts` function)
- [X] T020 [P1] [US1] Add reviewer persona existence validation — for each `agent_review` contract, verify `contract.Persona` references a persona defined in the manifest. Hard error at pipeline load. File: `internal/pipeline/validation.go`
- [X] T021 [P2] [US3] Add mixed singular/plural contract warning — if both `contract` and `contracts` are set on the same step, emit a validation warning (contracts takes precedence). File: `internal/pipeline/validation.go`
- [X] T022 [P2] [US4] Add token budget positive validation — if `TokenBudget` is set, must be > 0. Hard error at pipeline load. File: `internal/pipeline/validation.go`
- [X] T023 [P2] [US3] Validate rework targets from contract-level `ReworkStep` fields — for contracts with `on_failure: rework`, verify `ReworkStep` references a valid `rework_only` step. File: `internal/pipeline/dag.go`
- [X] T024 [P1] [US1] [P] Unit tests for self-review prevention — same persona rejected, different persona accepted, non-agent_review contracts skipped. File: `internal/pipeline/dag_test.go`
- [X] T025 [P1] [US1] [P] Unit tests for criteria path and persona validation — existing path passes, missing path fails, existing persona passes, unknown persona fails. File: `internal/pipeline/validation_test.go`

---

## Phase 4 — Executor Integration: Plural Contracts Loop (US3)

Replace the single contract validation block in `executor.go` with a loop over `EffectiveContracts()`. Each contract gets independent `on_failure` handling.

- [X] T026 [P2] [US3] Refactor executor contract validation to loop over `EffectiveContracts()` — replace the single-contract block at `internal/pipeline/executor.go:2664-2822` with a loop that iterates over `step.Handover.EffectiveContracts()`. Each iteration builds a `contract.ContractConfig`, resolves source/command, and calls validation. Early termination on failure unless `on_failure` allows continuation.
- [X] T027 [P2] [US3] Implement per-contract `on_failure` handling in the loop — `fail`: return error (existing behavior), `skip`: break loop, `continue`: proceed to next contract, `retry`: use existing retry logic, `rework`: handled in Phase 5 task. File: `internal/pipeline/executor.go`
- [X] T028 [P1] [US1] Wire adapter runner into contract validation — when contract type is `agent_review`, call `contract.ValidateWithRunner()` instead of `contract.Validate()`, passing `e.adapterRunner` (or equivalent from the executor's adapter registry) and the manifest. File: `internal/pipeline/executor.go`
- [X] T029 [P2] [US3] Maintain backward compatibility — when `EffectiveContracts()` returns a single contract from the singular `contract` field, behavior must be identical to current code (same events, same tracing, same logging). File: `internal/pipeline/executor.go`
- [X] T030 [P2] [US3] Add `agent_review` case to `buildContractPrompt()` — generate contract compliance section that describes the review criteria, expected output format, and reviewer persona. File: `internal/pipeline/executor.go` (in `buildContractPrompt` at lines 3913-3969)
- [X] T031 [P2] [US3] [P] Unit tests for plural contracts execution — ordered execution, early termination on fail, per-contract on_failure, backward compat with singular field. File: `internal/pipeline/executor_test.go`

---

## Phase 5 — Rework with Feedback (US2)

Contract-level `on_failure: rework` writes `ReviewFeedback` as artifact and triggers rework step. After rework, all contracts re-run from beginning, bounded by `max_retries`.

- [X] T032 [P1] [US2] Implement contract-level rework trigger — when an `agent_review` contract fails with `on_failure: rework`: (1) write `ReviewFeedback` JSON to `.wave/artifacts/review_feedback.json` in workspace, (2) build `AttemptContext` with review feedback path, (3) find and execute the rework step using the existing `runStepExecution` pattern from `executeReworkStep()`. File: `internal/pipeline/executor.go`
- [X] T033 [P1] [US2] Implement contract list re-run after rework — after rework step completes, restart the entire contract list from the beginning. Each full re-run counts as one retry attempt against the triggering contract's `max_retries`. If max retries exhausted and re-review still fails, apply the contract's final `on_failure` policy. File: `internal/pipeline/executor.go`
- [X] T034 [P1] [US2] Inject review feedback into rework step context — the rework step's prompt should include the path to `review_feedback.json` and a summary of issues found. Reuse `AttemptContext` pattern from existing `executeReworkStep()` with additional `ReviewFeedbackPath` field or similar. File: `internal/pipeline/executor.go`
- [X] T035 [P1] [US2] [P] Unit tests for rework-with-feedback loop — review fails → feedback written → rework triggered → all contracts re-run → pass on second attempt. Also: max retries exhausted → final failure. File: `internal/pipeline/executor_test.go`

---

## Phase 6 — Context Sources & Token Budget (US4)

Configurable context assembly: git_diff, artifact sources, truncation, token budget enforcement.

- [X] T036 [P2] [US4] Implement `git_diff` context source — run `git diff HEAD` via `exec.CommandContext` in workspace directory, capture output, truncate at `MaxSize` bytes (default 50KB) with `[... truncated at <limit> ...]` notice. File: `internal/contract/agent_review.go`
- [X] T037 [P2] [US4] Implement artifact context source — read artifact content by name from provided artifact paths map. When artifact not found, emit warning but continue with available context. File: `internal/contract/agent_review.go`
- [X] T038 [P2] [US4] Implement `assembleContext()` — iterate over `Context` sources in order, call appropriate handler for each source type, concatenate results with section headers. Pass assembled context into `buildReviewPrompt()`. File: `internal/contract/agent_review.go`
- [X] T039 [P2] [US4] Wire artifact paths into agent review validator — pass the execution's `ArtifactPaths` map to the validator so artifact context sources can resolve. Update `ValidateWithRunner()` signature or pass via config. File: `internal/contract/agent_review.go`, `internal/pipeline/executor.go`
- [X] T040 [P2] [US4] Implement token budget enforcement — after adapter execution, compare `AdapterResult.TokensUsed` against `TokenBudget`. If exceeded, treat review as failure with descriptive error. File: `internal/contract/agent_review.go`
- [X] T041 [P2] [US4] Handle empty git diff — when `git diff HEAD` returns empty output, include a notice in context: "No uncommitted changes detected in workspace." File: `internal/contract/agent_review.go`
- [X] T042 [P2] [US4] [P] Unit tests for context assembly — git_diff truncation at limit, git_diff empty, artifact found, artifact missing (warning), multiple sources concatenated. File: `internal/contract/agent_review_test.go`
- [X] T043 [P2] [US4] [P] Unit tests for token budget — within budget passes, over budget fails with descriptive error, no budget set (unlimited) passes. File: `internal/contract/agent_review_test.go`

---

## Phase 7 — Observability: Events, Dashboard, Retros (US6)

Review lifecycle events, dashboard display, retrospective friction tracking.

- [X] T044 [P3] [US6] Emit review lifecycle events — add `review_started` (before adapter call), `review_completed` (after successful parse, includes verdict/issue count/tokens), `review_failed` (on failure, includes error) events in the contract validation loop. Include reviewer persona and model in event fields. File: `internal/pipeline/executor.go`
- [X] T045 [P3] [US6] Add `FrictionReviewRework` constant to retro types — `FrictionReviewRework FrictionType = "review_rework"`. File: `internal/retro/types.go:19-25`
- [X] T046 [P3] [US6] Detect review rework in retro generator — scan events for `review_failed` events that triggered rework, create `FrictionPoint` with type `FrictionReviewRework`, step ID, and review feedback detail. File: `internal/retro/generator.go`
- [X] T047 [P3] [US6] Add review verdict fields to `StepDetail` — add `ReviewVerdict` (string), `ReviewIssueCount` (int), `ReviewerPersona` (string), `ReviewTokens` (int) to `StepDetail` struct. File: `internal/webui/types.go:52-80`
- [X] T048 [P3] [US6] Populate review fields in webui run handler — extract review verdict data from step events (`review_completed`/`review_failed`) and populate `StepDetail` fields. File: `internal/webui/handlers_runs.go`
- [X] T049 [P3] [US6] [P] Unit tests for review events and retro friction — verify event emission sequence (started → completed/failed), verify FrictionReviewRework detection in retro generator. File: `internal/pipeline/executor_test.go`, `internal/retro/generator_test.go`

---

## Phase 8 — Wave Pipeline Upgrades (US7)

Upgrade Wave's own pipelines with `agent_review` contracts. Depends on all prior phases.

- [X] T050 [P3] [US7] Create implementation review criteria file — write `.wave/contracts/impl-review-criteria.md` with criteria for reviewing implementation steps: correct approach (not no-op), requirement coverage, code quality, test coverage, no leaked files. File: `.wave/contracts/impl-review-criteria.md`
- [X] T051 [P3] [US7] Add `agent_review` contract to `impl-issue` pipeline — add `contracts` list with `test_suite` first (cheap), then `agent_review` with navigator persona, criteria path, haiku model, git_diff context, 8000 token budget. File: `.wave/pipelines/impl-issue.yaml`
- [X] T052 [P3] [US7] Add `agent_review` contract to `impl-speckit` pipeline — same pattern as T051 on the implementation step. File: `.wave/pipelines/impl-speckit.yaml`

---

## Phase 9 — Polish & Cross-Cutting

Integration tests, edge cases, backward compatibility verification.

- [X] T053 [P] Integration test — full pipeline with agent_review contract — end-to-end test: define a pipeline with an `agent_review` contract, mock adapter returns structured ReviewFeedback, verify validator runs, feedback extracted, verdict determines pass/fail. File: `internal/pipeline/executor_test.go`
- [X] T054 [P] Backward compatibility test — singular contract field — verify that all existing pipelines using the singular `contract` field produce identical behavior (same events, same tracing, same pass/fail) after the refactor. File: `internal/pipeline/executor_test.go`
- [X] T055 [P] Edge case tests — reviewer crash/timeout, missing context artifacts, unparseable reviewer output, empty git diff, zero token budget rejected at load, self-review rejected at load. File: `internal/contract/agent_review_test.go`, `internal/pipeline/validation_test.go`
- [X] T056 Run `go test ./...` and fix any failures — ensure all existing and new tests pass. File: project root
- [X] T057 Run `go test -race ./...` and fix any race conditions — required for PR readiness. File: project root

---

## Dependency Graph

```
Phase 1 (T001-T007) ──┬──→ Phase 2 (T008-T017) ──→ Phase 3 (T018-T025)
                       │                                    │
                       │                                    ▼
                       └──→ Phase 4 (T026-T031) ──→ Phase 5 (T032-T035)
                                    │                       │
                                    ▼                       ▼
                              Phase 6 (T036-T043) ──→ Phase 7 (T044-T049)
                                                           │
                                                           ▼
                                                     Phase 8 (T050-T052)
                                                           │
                                                           ▼
                                                     Phase 9 (T053-T057)
```

## Task Legend

- `[P1]` / `[P2]` / `[P3]` — Priority tier (maps to spec user story priorities)
- `[US1]` – `[US7]` — User story reference
- `[P]` — Parallelizable with other `[P]` tasks in the same phase

## Summary

| Phase | Description | Tasks | Parallelizable |
|-------|-------------|-------|----------------|
| 1 | Setup & Type Scaffolding | 7 | 1 |
| 2 | Core Agent Review Validator | 10 | 3 |
| 3 | Pipeline Validation | 8 | 2 |
| 4 | Executor: Plural Contracts Loop | 6 | 1 |
| 5 | Rework with Feedback | 4 | 1 |
| 6 | Context Sources & Token Budget | 8 | 2 |
| 7 | Observability | 6 | 1 |
| 8 | Wave Pipeline Upgrades | 3 | 0 |
| 9 | Polish & Cross-Cutting | 5 | 3 |
| **Total** | | **57** | **14** |
