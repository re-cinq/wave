# Tasks: Wave CLI Implementation (Hardening)

**Input**: Design documents from `/specs/015-wave-cli-implementation/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md (symlink to 014)

**Focus**: Harden existing codebase - add tests, improve error handling, fix bugs, add missing features

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Exact file paths included in all task descriptions

---

## Phase 1: Setup (Test Infrastructure)

**Purpose**: Establish testing infrastructure for hardening work

- [X] T001 Create test fixtures directory structure at cmd/wave/commands/testdata/
- [X] T002 [P] Create valid test manifest at cmd/wave/commands/testdata/valid/wave.yaml
- [X] T003 [P] Create invalid adapter reference manifest at cmd/wave/commands/testdata/invalid-adapter/wave.yaml
- [X] T004 [P] Create missing file manifest at cmd/wave/commands/testdata/missing-file/wave.yaml
- [X] T005 [P] Create test pipeline fixtures at cmd/wave/commands/testdata/pipelines/
- [X] T006 Add testify/assert import to go.mod if not present

---

## Phase 2: Foundational (State Store Tests)

**Purpose**: State persistence is foundational - must be tested before user story work

**⚠️ CRITICAL**: State store underlies resume, pipeline execution, and cleanup

- [X] T007 Create internal/state/store_test.go with test setup and teardown
- [X] T008 [P] Add test for SavePipelineState in internal/state/store_test.go
- [X] T009 [P] Add test for GetPipelineState in internal/state/store_test.go
- [X] T010 [P] Add test for SaveStepState in internal/state/store_test.go
- [X] T011 [P] Add test for GetStepStates in internal/state/store_test.go
- [X] T012 Add test for concurrent access from matrix workers in internal/state/store_test.go
- [X] T013 Add test for state transitions (pending→running→completed) in internal/state/store_test.go
- [X] T014 Verify all state store tests pass with `go test -race ./internal/state/...`

**Checkpoint**: State persistence is fully tested - user story hardening can begin

---

## Phase 3: User Story 1 - Project Initialization (Priority: P1) 🎯 MVP

**Goal**: Harden `wave init` command with tests and improved error handling

**Independent Test**: Run `wave init` in empty dir, verify files created, run `wave validate`

### Tests for User Story 1

- [X] T015 [P] [US1] Create cmd/wave/commands/init_test.go with test helpers
- [X] T016 [P] [US1] Add test for init in empty directory in cmd/wave/commands/init_test.go
- [X] T017 [P] [US1] Add test for init with existing wave.yaml in cmd/wave/commands/init_test.go
- [X] T018 [P] [US1] Add test for init --merge flag in cmd/wave/commands/init_test.go
- [X] T019 [US1] Add test for init creates all persona prompt files in cmd/wave/commands/init_test.go

### Implementation for User Story 1

- [X] T020 [US1] Add --merge flag handling in cmd/wave/commands/init.go
- [X] T021 [US1] Improve error messages with file paths in cmd/wave/commands/init.go
- [X] T022 [US1] Add confirmation prompt for overwriting existing config in cmd/wave/commands/init.go
- [X] T023 [US1] Verify init output validates with wave validate

**Checkpoint**: `wave init` fully tested and handles edge cases

---

## Phase 4: User Story 2 - Manifest Validation (Priority: P1)

**Goal**: Harden `wave validate` with verbose mode and improved error messages

**Independent Test**: Create invalid manifests, verify errors include file paths and line numbers

### Tests for User Story 2

- [X] T024 [P] [US2] Create cmd/wave/commands/validate_test.go with test helpers
- [X] T025 [P] [US2] Add test for validate with valid manifest in cmd/wave/commands/validate_test.go
- [X] T026 [P] [US2] Add test for validate with invalid adapter reference in cmd/wave/commands/validate_test.go
- [X] T027 [P] [US2] Add test for validate with missing system prompt file in cmd/wave/commands/validate_test.go
- [X] T028 [P] [US2] Add test for validate --verbose output in cmd/wave/commands/validate_test.go
- [X] T029 [US2] Add test for validate with malformed YAML in cmd/wave/commands/validate_test.go

### Implementation for User Story 2

- [X] T030 [US2] Add --verbose flag with summary output in cmd/wave/commands/validate.go
- [X] T031 [US2] Improve error messages with line numbers in internal/manifest/parser.go
- [X] T032 [US2] Add suggestions to error messages (e.g., "run wave init") in cmd/wave/commands/validate.go
- [X] T033 [US2] Add adapter binary availability check to verbose output in cmd/wave/commands/validate.go

**Checkpoint**: `wave validate` provides actionable error messages

---

## Phase 5: User Story 3 - Ad-Hoc Task Execution (Priority: P1)

**Goal**: Harden `wave do` with --save flag, --dry-run, and tests

**Independent Test**: Run `wave do "test task" --dry-run`, verify pipeline YAML output

### Tests for User Story 3

- [X] T034 [P] [US3] Create cmd/wave/commands/do_test.go with test helpers
- [X] T035 [P] [US3] Add test for do generates two-step pipeline in cmd/wave/commands/do_test.go
- [X] T036 [P] [US3] Add test for do --persona override in cmd/wave/commands/do_test.go
- [X] T037 [P] [US3] Add test for do --dry-run output in cmd/wave/commands/do_test.go
- [X] T038 [P] [US3] Add test for do --save writes pipeline file in cmd/wave/commands/do_test.go
- [X] T039 [US3] Add test for adhoc pipeline generation in internal/pipeline/adhoc_test.go

### Implementation for User Story 3

- [X] T040 [US3] Add --dry-run flag to print pipeline YAML in cmd/wave/commands/do.go
- [X] T041 [US3] Add --save flag to write pipeline to file in cmd/wave/commands/do.go
- [X] T042 [US3] Improve error handling for missing manifest in cmd/wave/commands/do.go

**Checkpoint**: `wave do` supports all spec flags

---

## Phase 6: User Story 4 - Pipeline Execution (Priority: P1)

**Goal**: Harden `wave run` with --dry-run and executor tests

**Independent Test**: Run `wave run --pipeline hotfix --dry-run`, verify execution plan output

### Tests for User Story 4

- [X] T043 [P] [US4] Create cmd/wave/commands/run_test.go with test helpers
- [X] T044 [P] [US4] Add test for run --dry-run output in cmd/wave/commands/run_test.go
- [X] T045 [P] [US4] Add test for run with non-existent pipeline in cmd/wave/commands/run_test.go
- [X] T046 [P] [US4] Add test for run --from-step in cmd/wave/commands/run_test.go
- [X] T047 [US4] Add executor test for step ordering in internal/pipeline/executor_test.go
- [X] T048 [US4] Add executor test for parallel step execution in internal/pipeline/executor_test.go
- [X] T049 [US4] Add executor test for contract failure retry in internal/pipeline/executor_test.go

### Implementation for User Story 4

- [X] T050 [US4] Add --dry-run flag to print execution plan in cmd/wave/commands/run.go
- [X] T051 [US4] Improve --from-step validation in cmd/wave/commands/run.go
- [X] T052 [US4] Add progress event emission to executor in internal/pipeline/executor.go

**Checkpoint**: `wave run` supports dry-run and has executor tests

---

## Phase 7: User Story 5 - Permission Enforcement (Priority: P1)

**Goal**: Verify and test permission enforcement in adapter layer

**Independent Test**: Configure deny pattern, attempt denied operation, verify blocked

### Tests for User Story 5

- [X] T053 [P] [US5] Add test for deny pattern blocks Write in internal/adapter/adapter_test.go
- [X] T054 [P] [US5] Add test for allow pattern permits operation in internal/adapter/adapter_test.go
- [X] T055 [P] [US5] Add test for deny takes precedence over allow in internal/adapter/adapter_test.go
- [X] T056 [US5] Add test for permission error message format in internal/adapter/adapter_test.go

### Implementation for User Story 5

- [X] T057 [US5] Verify permission check order (deny first) in internal/adapter/adapter.go
- [X] T058 [US5] Improve permission denied error message with persona name in internal/adapter/adapter.go
- [X] T059 [US5] Add glob pattern matching tests in internal/adapter/adapter_test.go

**Checkpoint**: Permission enforcement is verified and has clear error messages

---

## Phase 8: User Story 6 - State Persistence & Resume (Priority: P2)

**Goal**: Harden `wave resume` with listing mode and tests

**Independent Test**: Start pipeline, kill process, run `wave resume`, verify continuation

### Tests for User Story 6

- [X] T060 [P] [US6] Create cmd/wave/commands/resume_test.go with test helpers
- [X] T061 [P] [US6] Add test for resume lists recent pipelines in cmd/wave/commands/resume_test.go
- [X] T062 [P] [US6] Add test for resume continues from last step in cmd/wave/commands/resume_test.go
- [X] T063 [US6] Add test for resume with retrying state in cmd/wave/commands/resume_test.go

### Implementation for User Story 6

- [X] T064 [US6] Add pipeline listing when no ID provided in cmd/wave/commands/resume.go
- [X] T065 [US6] Improve resume progress messages in cmd/wave/commands/resume.go

**Checkpoint**: `wave resume` supports listing and continuation

---

## Phase 9: User Story 7 - Context Relay (Priority: P2)

**Goal**: Test relay mechanism and checkpoint handling

**Independent Test**: Set low token threshold, trigger relay, verify checkpoint created

### Tests for User Story 7

- [X] T066 [P] [US7] Add test for threshold detection in internal/relay/relay_test.go
- [X] T067 [P] [US7] Add test for checkpoint parsing in internal/relay/checkpoint_test.go
- [X] T068 [P] [US7] Add test for checkpoint injection in internal/relay/relay_test.go
- [X] T069 [US7] Add test for relay with summarizer failure in internal/relay/relay_test.go

### Implementation for User Story 7

- [X] T070 [US7] Add checkpoint format validation in internal/relay/checkpoint.go
- [X] T071 [US7] Improve relay error handling in internal/relay/relay.go

**Checkpoint**: Relay mechanism is tested and handles edge cases

---

## Phase 10: User Story 8 - Contract Validation (Priority: P2)

**Goal**: Test all contract types and graceful degradation

**Independent Test**: Define JSON schema contract, produce invalid output, verify retry

### Tests for User Story 8

- [X] T072 [P] [US8] Add test for JSON schema validation failure in internal/contract/contract_test.go
- [X] T073 [P] [US8] Add test for TypeScript validation without tsc in internal/contract/typescript_test.go
- [X] T074 [P] [US8] Add test for test suite validation in internal/contract/testsuite_test.go
- [X] T075 [US8] Add test for max_retries exhaustion in internal/contract/contract_test.go

### Implementation for User Story 8

- [X] T076 [US8] Add tsc availability check with graceful degradation in internal/contract/typescript.go
- [X] T077 [US8] Improve contract error messages with validation details in internal/contract/contract.go

**Checkpoint**: Contract validation handles all types and missing tools

---

## Phase 11: User Story 9 - Matrix Execution (Priority: P2)

**Goal**: Test matrix strategy including partial failures

**Independent Test**: Run matrix with 5 tasks, fail 1, verify others complete

### Tests for User Story 9

- [X] T078 [P] [US9] Add test for matrix spawns correct worker count in internal/pipeline/matrix_test.go
- [X] T079 [P] [US9] Add test for max_concurrency limit in internal/pipeline/matrix_test.go
- [X] T080 [P] [US9] Add test for partial failure handling in internal/pipeline/matrix_test.go
- [X] T081 [US9] Add test for zero tasks in matrix in internal/pipeline/matrix_test.go

### Implementation for User Story 9

- [X] T082 [US9] Improve partial failure reporting in internal/pipeline/matrix.go
- [X] T083 [US9] Handle zero tasks gracefully in internal/pipeline/matrix.go

**Checkpoint**: Matrix execution handles concurrency and failures

---

## Phase 12: User Story 10 - List Commands (Priority: P2)

**Goal**: Test `wave list` subcommands

**Independent Test**: Run `wave list pipelines`, verify all pipelines shown

### Tests for User Story 10

- [X] T084 [P] [US10] Create cmd/wave/commands/list_test.go with test helpers
- [X] T085 [P] [US10] Add test for list pipelines output in cmd/wave/commands/list_test.go
- [X] T086 [P] [US10] Add test for list personas output in cmd/wave/commands/list_test.go
- [X] T087 [US10] Add test for list adapters with binary check in cmd/wave/commands/list_test.go

### Implementation for User Story 10

- [X] T088 [US10] Add step count to pipeline listing in cmd/wave/commands/list.go
- [X] T089 [US10] Add permission summary to persona listing in cmd/wave/commands/list.go

**Checkpoint**: `wave list` shows complete information

---

## Phase 13: User Story 11 - Workspace Cleanup (Priority: P3)

**Goal**: Harden `wave clean` with --keep-last and --dry-run flags

**Independent Test**: Run pipelines, run `wave clean --dry-run`, verify listing

### Tests for User Story 11

- [X] T090 [P] [US11] Create cmd/wave/commands/clean_test.go with test helpers
- [X] T091 [P] [US11] Add test for clean removes all workspaces in cmd/wave/commands/clean_test.go
- [X] T092 [P] [US11] Add test for clean --keep-last N in cmd/wave/commands/clean_test.go
- [X] T093 [US11] Add test for clean --dry-run output in cmd/wave/commands/clean_test.go

### Implementation for User Story 11

- [X] T094 [US11] Add --keep-last flag in cmd/wave/commands/clean.go
- [X] T095 [US11] Add --dry-run flag in cmd/wave/commands/clean.go
- [X] T096 [US11] Sort workspaces by creation time for keep-last in internal/workspace/workspace.go

**Checkpoint**: `wave clean` supports selective cleanup

---

## Phase 14: User Story 12 - Meta-Pipeline (Priority: P3)

**Goal**: Test meta-pipeline recursion limits

**Independent Test**: Trigger meta-pipeline, verify depth limit enforced

### Tests for User Story 12

- [X] T097 [P] [US12] Add test for meta-pipeline depth limit in internal/pipeline/meta_test.go
- [X] T098 [P] [US12] Add test for meta-pipeline validation in internal/pipeline/meta_test.go
- [X] T099 [US12] Add test for meta-pipeline failure trace preservation in internal/pipeline/meta_test.go

### Implementation for User Story 12

- [X] T100 [US12] Improve depth limit error message in internal/pipeline/meta.go
- [X] T101 [US12] Add generated pipeline preservation on failure in internal/pipeline/meta.go

**Checkpoint**: Meta-pipeline has safety limits

---

## Phase 15: Polish & Cross-Cutting Concerns

**Purpose**: Security hardening, documentation, final validation

- [X] T102 [P] Add credential scrubbing patterns test in internal/audit/logger_test.go
- [X] T103 [P] Add concurrent event emission test in internal/event/emitter_test.go
- [X] T104 [P] Add workspace isolation test in internal/workspace/workspace_test.go
- [X] T105 Add subprocess timeout test with hanging mock in internal/adapter/adapter_test.go
- [X] T106 Run full test suite with race detector: `go test -race ./...`
- [X] T107 Run quickstart.md validation manually
- [X] T108 Update CLAUDE.md with any new commands or flags

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Stories (Phase 3-14)**: All depend on Foundational phase completion
  - P1 stories (US1-US5): Can proceed in parallel after Phase 2
  - P2 stories (US6-US10): Can proceed in parallel after Phase 2
  - P3 stories (US11-US12): Can proceed in parallel after Phase 2
- **Polish (Phase 15)**: Depends on user story phases complete

### User Story Dependencies

All user stories are independent - they harden different parts of the codebase:

- **US1 (init)**: cmd/wave/commands/init.go
- **US2 (validate)**: cmd/wave/commands/validate.go, internal/manifest/
- **US3 (do)**: cmd/wave/commands/do.go, internal/pipeline/adhoc.go
- **US4 (run)**: cmd/wave/commands/run.go, internal/pipeline/executor.go
- **US5 (permissions)**: internal/adapter/
- **US6 (resume)**: cmd/wave/commands/resume.go, internal/state/
- **US7 (relay)**: internal/relay/
- **US8 (contracts)**: internal/contract/
- **US9 (matrix)**: internal/pipeline/matrix.go
- **US10 (list)**: cmd/wave/commands/list.go
- **US11 (clean)**: cmd/wave/commands/clean.go, internal/workspace/
- **US12 (meta)**: internal/pipeline/meta.go

### Parallel Opportunities

Within each user story phase, all [P] tasks can run in parallel.
Different user stories can be worked on in parallel by different developers.

---

## Parallel Example: Phase 2 (Foundational)

```bash
# All store tests can be written in parallel:
T008: Test for SavePipelineState
T009: Test for GetPipelineState
T010: Test for SaveStepState
T011: Test for GetStepStates
```

## Parallel Example: User Story 4

```bash
# All US4 tests can be written in parallel:
T043: Test for run command helpers
T044: Test for run --dry-run
T045: Test for run with non-existent pipeline
T046: Test for run --from-step
```

---

## Implementation Strategy

### MVP First (P1 Stories Only)

1. Complete Phase 1: Setup (test fixtures)
2. Complete Phase 2: Foundational (state store tests)
3. Complete Phases 3-7: User Stories 1-5 (all P1)
4. **STOP and VALIDATE**: Run `go test ./...` - all tests pass
5. Core CLI is hardened and tested

### Incremental Delivery

1. Setup + Foundational → Test infrastructure ready
2. Add US1-US5 (P1) → Core functionality hardened
3. Add US6-US10 (P2) → Extended functionality hardened
4. Add US11-US12 (P3) → Edge cases handled
5. Polish → Security and documentation complete

### Test Count Summary

| Phase | Tasks | Tests | Implementation |
|-------|-------|-------|----------------|
| Setup | 6 | 0 | 6 |
| Foundational | 8 | 7 | 1 |
| US1 (init) | 9 | 5 | 4 |
| US2 (validate) | 10 | 6 | 4 |
| US3 (do) | 9 | 6 | 3 |
| US4 (run) | 10 | 7 | 3 |
| US5 (permissions) | 7 | 4 | 3 |
| US6 (resume) | 6 | 4 | 2 |
| US7 (relay) | 6 | 4 | 2 |
| US8 (contracts) | 6 | 4 | 2 |
| US9 (matrix) | 6 | 4 | 2 |
| US10 (list) | 6 | 4 | 2 |
| US11 (clean) | 7 | 4 | 3 |
| US12 (meta) | 5 | 3 | 2 |
| Polish | 7 | 4 | 3 |
| **Total** | **108** | **66** | **42** |

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Each user story hardens a specific part of the codebase
- Verify tests pass before moving to next phase
- Use `go test -race ./...` to detect race conditions
- Commit after each task or logical group
