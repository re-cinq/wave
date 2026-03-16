# Tasks: Restore and Stabilize `wave meta` Dynamic Pipeline Generation

**Branch**: `095-restore-meta-pipeline`
**Date**: 2026-03-16
**Spec**: `specs/095-restore-meta-pipeline/spec.md`
**Plan**: `specs/095-restore-meta-pipeline/plan.md`

---

## Phase 1: Setup

- [X] T001 P1 [Setup] Create feature branch and verify baseline tests pass with `go test -race ./internal/pipeline/... ./cmd/wave/commands/...` — `internal/pipeline/meta_test.go`, `cmd/wave/commands/meta_test.go`

---

## Phase 2: Foundational — Validation & Normalization Infrastructure

These tasks are blocking prerequisites for all user story phases.

- [X] T002 P1 [US1,US2] Add `ValidationOption` type and `WithManifest()` option to `ValidateGeneratedPipeline()` — extend signature to accept variadic `ValidationOption`, add `validationConfig` struct and `WithManifest(*manifest.Manifest)` constructor — `internal/pipeline/meta.go`
- [X] T003 P1 [US1,US2] Implement manifest-aware persona validation inside `ValidateGeneratedPipeline()` — when manifest is provided, check all step personas exist in `m.Personas` and their adapters exist in `m.Adapters`; append errors to the existing errors slice — `internal/pipeline/meta.go`
- [X] T004 P1 [US1,US2] Add `normalizeGeneratedPipeline()` function — for each step with `json_schema` contract and empty `OutputArtifacts`, auto-generate a default `OutputArtifact` with name `<step.ID>-output`, path `.wave/artifact.json`, type `json` — `internal/pipeline/meta.go`
- [X] T005 P1 [US1,US2] Wire `normalizeGeneratedPipeline()` into `GenerateOnly()` — call after `loader.Unmarshal()` and before `ValidateGeneratedPipeline()`, pass `WithManifest(m)` to validation — `internal/pipeline/meta.go`
- [X] T006 P1 [US1,US2] Wire `normalizeGeneratedPipeline()` into `Execute()` — call after `loader.Unmarshal()` and before `ValidateGeneratedPipeline()`, pass `WithManifest(m)` to validation — `internal/pipeline/meta.go`

---

## Phase 3: US1 — Dry-Run Pipeline Generation (P1)

- [X] T007 P1 [US1] Emit `meta_generate_failed` event on semantic validation failure in `GenerateOnly()` — add event emission before returning the validation error, include error details in event message — `internal/pipeline/meta.go`
- [X] T008 P1 [US1] [P] Add test for `normalizeGeneratedPipeline()` — verify steps with `json_schema` contract get `output_artifacts` auto-generated, steps with existing artifacts are untouched, steps with non-json_schema contracts are unmodified — `internal/pipeline/meta_test.go`
- [X] T009 P1 [US1] [P] Add test for `ValidateGeneratedPipeline()` with `WithManifest()` — verify unknown persona is detected and reported, valid personas pass, missing adapter for persona is detected — `internal/pipeline/meta_test.go`
- [X] T010 P1 [US1] [P] Add test for `meta_generate_failed` event on semantic validation failure — use `eventCapture` emitter, trigger validation failure, assert event state is `meta_generate_failed` — `internal/pipeline/meta_test.go`

---

## Phase 4: US2 — Full Pipeline Execution (P1)

- [X] T011 P1 [US2] Move timeout enforcement into `MetaPipelineExecutor.Execute()` — wrap context with `context.WithTimeout` using `getTimeout(m)` at the start of `Execute()`, before depth check — `internal/pipeline/meta.go`
- [X] T012 P1 [US2] Emit `meta_generate_failed` event on semantic validation failure in `Execute()` — mirror the event emission added in T007 for the Execute path — `internal/pipeline/meta.go`
- [X] T013 P1 [US2] [P] Add test for internal timeout enforcement in `Execute()` — create executor, set short timeout via manifest config, verify context deadline is applied (use a slow mock runner that respects context cancellation) — `internal/pipeline/meta_test.go`

---

## Phase 5: US3 — Save Generated Pipeline for Reuse (P2)

- [X] T014 P2 [US3] [P] Add test for `saveMetaPipeline()` path resolution — verify bare name gets `.wave/pipelines/` prefix and `.yaml` extension, name with `.yaml` doesn't get double extension, path with `/` is used as-is — `cmd/wave/commands/meta_test.go`

---

## Phase 6: US4 — Mock Adapter Testing (P2)

- [X] T015 P2 [US4] Add `buildMetaMockResponse()` function — returns a well-formed `--- PIPELINE --- / --- SCHEMAS ---` response with a minimal valid pipeline (navigator step → implementer step), each with fresh memory and json_schema contracts — `internal/pipeline/meta.go`
- [X] T016 P2 [US4] Wire mock response into `invokePhilosopherWithSchemas()` — when mock adapter is detected (check adapter binary or add a `mockMode` field), use `buildMetaMockResponse()` instead of parsing adapter output — `internal/pipeline/meta.go`
- [X] T017 P2 [US4] [P] Add test for `buildMetaMockResponse()` — verify returned string contains both delimiters, pipeline YAML parses to a valid pipeline with navigator-first step, schemas section contains valid JSON — `internal/pipeline/meta_test.go`

---

## Phase 7: US5 — Resource Limit Enforcement (P3)

- [X] T018 P3 [US5] [P] Add test for depth limit with call stack in error message — create executor with `WithMetaDepth(2)`, set `max_depth: 2`, verify error contains "depth limit reached" and call stack info — `internal/pipeline/meta_test.go`
- [X] T019 P3 [US5] [P] Add test for step limit enforcement — create executor, generate pipeline with more steps than `max_total_steps`, verify error contains "step limit exceeded" — `internal/pipeline/meta_test.go`
- [X] T020 P3 [US5] [P] Add test for token limit enforcement — create executor, set `max_total_tokens` low, mock runner returns high token count, verify error contains "token limit exceeded" — `internal/pipeline/meta_test.go`

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T021 P1 [All] Run `go test -race ./internal/pipeline/... ./cmd/wave/commands/...` and fix any failures — all tests must pass with zero failures, zero skips without linked issues — `internal/pipeline/meta_test.go`, `cmd/wave/commands/meta_test.go`
- [X] T022 P1 [All] Run `golangci-lint run ./internal/pipeline/... ./cmd/wave/commands/...` and fix any lint findings — `internal/pipeline/meta.go`, `cmd/wave/commands/meta.go`
- [X] T023 P1 [All] Run full test suite `go test -race ./...` to verify zero regressions across the entire codebase — all packages

---

## Dependency Graph

```
T001 (setup)
  ├── T002 → T003 (validation options → persona validation)
  ├── T004 (normalize function)
  │   ├── T005 (wire into GenerateOnly) → T007 (failed event in GenerateOnly)
  │   └── T006 (wire into Execute) → T011 (timeout) → T012 (failed event in Execute)
  ├── T008, T009, T010 [P] (validation tests — parallel after T003, T004, T007)
  ├── T013 [P] (timeout test — after T011)
  ├── T014 [P] (save path test — independent)
  ├── T015 → T016 (mock response → wire mock) → T017 [P] (mock test)
  ├── T018, T019, T020 [P] (resource limit tests — independent)
  └── T021 → T022 → T023 (final verification — sequential)
```

## Summary

| Metric | Value |
|--------|-------|
| Total tasks | 23 |
| P1 tasks | 17 |
| P2 tasks | 4 |
| P3 tasks | 3 (all tests for existing functionality) |
| Parallelizable tasks | 12 |
| Files modified | 4 (`internal/pipeline/meta.go`, `internal/pipeline/meta_test.go`, `cmd/wave/commands/meta.go`, `cmd/wave/commands/meta_test.go`) |
