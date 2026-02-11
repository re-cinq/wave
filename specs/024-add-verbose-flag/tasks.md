# Tasks: Add --verbose Flag to Wave CLI

**Branch**: `024-add-verbose-flag` | **Date**: 2026-02-06
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Phase 1: Setup — Flag Registration & Core Infrastructure

- [ ] T001 [P1] [Setup] Register `--verbose/-v` persistent flag on root command in `cmd/wave/main.go` following the existing `--debug/-d` BoolP pattern at line 31
- [ ] T002 [P1] [Setup] [P] Add `verbose bool` field to `DefaultPipelineExecutor` struct in `internal/pipeline/executor.go` alongside existing `debug` field at line 50
- [ ] T003 [P1] [Setup] [P] Add `WithVerbose(verbose bool) ExecutorOption` function in `internal/pipeline/executor.go` following the `WithDebug` pattern at lines 74-76

## Phase 2: Foundational — Event System Extension

- [ ] T004 [P1] [Setup] Add verbose fields to `Event` struct in `internal/event/emitter.go`: `WorkspacePath string`, `InjectedArtifacts []string`, `ContractResult string`, `VerboseDetail string` — all with `omitempty` JSON tags
- [ ] T005 [P1] [US1] [P] Wire `DisplayConfig.VerboseOutput` field in `internal/display/types.go:101` to the verbose flag when constructing display config in `cmd/wave/commands/run.go`

## Phase 3: US1 — Pipeline Verbose Output (P1)

- [ ] T006 [P1] [US1] Read verbose flag in run command (`cmd/wave/commands/run.go:69`) and pass to `runRun()`, add `pipeline.WithVerbose(verbose)` to executor options at lines 219-231
- [ ] T007 [P1] [US1] Populate verbose Event fields in executor event emissions (`internal/pipeline/executor.go`) when `e.verbose == true`: set `WorkspacePath` at step start (lines 381-389), `InjectedArtifacts` during artifact injection (lines 438-467), persona name, and `ContractResult` during validation (lines 522-574)
- [ ] T008 [P1] [US1] Render verbose fields in human-readable emitter output (`internal/event/emitter.go:133-193`): display workspace path, injected artifacts, and contract results when Event fields are non-empty

## Phase 4: US2 — Non-Pipeline Command Verbose Output (P2)

- [ ] T009 [P2] [US2] [P] Add verbose output to `status` command (`cmd/wave/commands/status.go:122-172`): read global verbose flag, emit database path, last state transition timestamps, and workspace locations to stderr
- [ ] T010 [P2] [US2] [P] Add verbose output to `clean` command (`cmd/wave/commands/clean.go:254-260`): read global verbose flag, list each workspace being removed with its size before deletion to stderr
- [ ] T011 [P2] [US2] [P] Verify `validate` command (`cmd/wave/commands/validate.go:35`) composes correctly with global persistent flag — Cobra local flag shadows persistent flag, both activate verbose mode. No code change expected; add a test confirming both `wave validate -v` and `wave -v validate` work

## Phase 5: US3 — Verbose and Debug Interaction (P3)

- [ ] T012 [P3] [US3] Implement debug-supersedes-verbose precedence in `internal/pipeline/executor.go`: when both `debug` and `verbose` are true, use debug level; verbose fields only populated when `!debug && verbose`
- [ ] T013 [P3] [US3] [P] Update flag help text in `cmd/wave/main.go` to clarify verbose vs debug relationship: verbose = "Enable verbose output (operational context)" and debug = "Enable debug mode (supersedes --verbose)"

## Phase 6: Testing & Validation

- [ ] T014 [P1] [Test] [P] Add unit tests for flag registration and propagation in `cmd/wave/commands/run_test.go`: verify verbose flag reads correctly, passes through to executor via WithVerbose option
- [ ] T015 [P1] [Test] [P] Add unit tests for verbose Event field serialization in `internal/event/emitter_test.go`: verify verbose fields present in JSON when populated, omitted when empty, and rendered in human-readable format
- [ ] T016 [P2] [Test] [P] Add unit tests for status verbose output in `cmd/wave/commands/status_test.go`: verify verbose output includes db path/timestamps/workspace locations, and non-verbose output is unchanged
- [ ] T017 [P2] [Test] [P] Add unit tests for clean verbose output in `cmd/wave/commands/clean_test.go`: verify verbose output lists workspaces with sizes, and non-verbose output is unchanged
- [ ] T018 [P3] [Test] [P] Add unit tests for debug-verbose precedence in `cmd/wave/commands/run_test.go`: table-driven test with `--debug --verbose`, `--verbose` only, and neither flag
- [ ] T019 [P1] [Test] Run full test suite `go test ./... -race` to verify zero regression (SC-002, SC-004)

## Dependency Graph

```
T001 ──┬──► T005, T006, T009, T010, T011, T013
       │
T002 ──┤
T003 ──┘──► T006 ──► T007 ──► T012
                              ▲
T004 ──────► T007, T008 ──────┘

T006 + T007 + T008 ──► T012
T009 ──► T016
T010 ──► T017
T011 (verification only)
T012 ──► T018
T001-T013 ──► T019 (final gate)

Parallel groups:
  [T002, T003] (Phase 1)
  [T004, T005] (Phase 2)
  [T009, T010, T011] (Phase 4)
  [T014, T015, T016, T017, T018] (Phase 6 — after respective implementations)
```

## Task-to-Acceptance Mapping

| Acceptance Scenario | Tasks |
|---|---|
| US1-S1: `wave run --verbose` shows step details | T006, T007, T008 |
| US1-S2: Without `--verbose`, output unchanged | T019 (regression gate) |
| US1-S3: `wave --verbose run` works (persistent flag) | T001 |
| US2-S1: `wave validate --verbose` shows validator details | T011 (existing, verify) |
| US2-S2: `wave status --verbose` shows metadata | T009 |
| US2-S3: `wave clean --verbose` lists workspaces | T010 |
| US3-S1: `--debug --verbose` uses debug level | T012 |
| US3-S2: `--verbose` alone excludes debug traces | T012 |
| SC-003: `wave --help` documents verbose flag | T001, T013 |
| SC-005: Unit test coverage for verbose output | T014-T018 |
