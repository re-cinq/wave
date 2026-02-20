# Implementation Tasks: Typed Artifact Composition

**Feature Branch**: `109-typed-artifact-composition`
**Generated**: 2026-02-20
**Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

## Task Legend

- `[P]` = Parallelizable with other `[P]` tasks in same phase
- `[T001]` = Unique task ID
- `[US1]` = Maps to User Story 1, etc.

---

## Phase 1: Setup & Foundations

_Project initialization and foundational type changes that all other phases depend on._

- [X] [T001] [P] [US1] Extend `ArtifactDef` struct with `Source` field (`internal/pipeline/types.go:97-102`)
- [X] [T002] [P] [US2] Extend `ArtifactRef` struct with `Type`, `SchemaPath`, `Optional` fields (`internal/pipeline/types.go:68-72`)
- [X] [T003] [P] [US1] Add `RuntimeArtifactsConfig` struct with `MaxStdoutSize` to manifest types (`internal/manifest/types.go`)
- [X] [T004] Update JSON schema for `ArtifactDef` with `source` field (`.wave/schemas/wave-pipeline.schema.json`)
- [X] [T005] Update JSON schema for `ArtifactRef` with `type`, `schema_path`, `optional` fields (`.wave/schemas/wave-pipeline.schema.json`)
- [X] [T006] Add unit tests for type extensions ensuring YAML parsing works (`internal/pipeline/types_test.go`)

**Phase 1 Gate**: All type extensions compile, existing pipeline YAML parses correctly, tests pass.

---

## Phase 2: US1 - Stdout Artifact Capture (P1)

_Core capability: capture step stdout as named artifact._

### 2.1 Stdout Buffering

- [X] [T007] [US1] Add `StdoutArtifact` runtime struct to hold buffered stdout (`internal/pipeline/executor.go`)
- [X] [T008] [US1] Modify `runStepExecution()` to detect stdout artifacts in step config (`internal/pipeline/executor.go:434-736`)
- [X] [T009] [US1] Implement stdout buffering during adapter execution with size limit check (`internal/pipeline/executor.go`)

### 2.2 Stdout Artifact Writing

- [X] [T010] [US1] Implement `writeStdoutArtifact()` function to persist to `.wave/artifacts/<step-id>/<name>` (`internal/pipeline/executor.go`)
- [X] [T011] [US1] Modify `writeOutputArtifacts()` to handle `Source: stdout` artifacts (`internal/pipeline/executor.go:1091-1119`)
- [X] [T012] [US1] Register stdout artifact paths in `execution.ArtifactPaths` map (`internal/pipeline/executor.go`)

### 2.3 Atomicity & Error Handling

- [X] [T013] [US1] Ensure stdout artifacts are NOT written on step failure (atomicity guarantee) (`internal/pipeline/executor.go`)
- [X] [T014] [US1] Implement size limit exceeded error with actionable message (`internal/pipeline/executor.go`)
- [X] [T015] [US1] Handle empty stdout case (create 0-byte artifact) (`internal/pipeline/executor.go`)

### 2.4 US1 Tests

- [X] [T016] [P] [US1] Test: stdout captured and available to downstream steps (`internal/pipeline/executor_test.go`)
- [X] [T017] [P] [US1] Test: size limit enforced with clear error (`internal/pipeline/executor_test.go`)
- [X] [T018] [P] [US1] Test: step failure produces no partial stdout artifact (`internal/pipeline/executor_test.go`)
- [X] [T019] [P] [US1] Test: empty stdout creates valid artifact (`internal/pipeline/executor_test.go`)

**Phase 2 Gate**: A step with `source: stdout` produces artifact accessible by name. Size limit works.

---

## Phase 3: US2 - Typed Artifact Consumption (P2)

_Validate artifact existence and type before step execution._

### 3.1 Existence Validation

- [X] [T020] [US2] Implement artifact existence check in `injectArtifacts()` (`internal/pipeline/executor.go:1035-1089`)
- [X] [T021] [US2] Add error path for missing required artifact: "required artifact 'X' not found" (`internal/pipeline/executor.go`)
- [X] [T022] [US2] Implement `optional: true` handling - skip missing optional artifacts (`internal/pipeline/executor.go`)

### 3.2 Type Validation

- [X] [T023] [US2] Implement type comparison between declared `ArtifactRef.Type` and `ArtifactDef.Type` (`internal/pipeline/executor.go`)
- [X] [T024] [US2] Add error path for type mismatch: "artifact 'X' type mismatch: expected Y, got Z" (`internal/pipeline/executor.go`)
- [X] [T025] [US2] Handle case where type is not declared (skip type check) (`internal/pipeline/executor.go`)

### 3.3 US2 Tests

- [X] [T026] [P] [US2] Test: missing required artifact fails before step starts (`internal/pipeline/executor_test.go`)
- [X] [T027] [P] [US2] Test: type mismatch fails with clear error (`internal/pipeline/executor_test.go`)
- [X] [T028] [P] [US2] Test: optional missing artifact proceeds (`internal/pipeline/executor_test.go`)
- [X] [T029] [P] [US2] Test: type not declared skips validation (`internal/pipeline/executor_test.go`)

**Phase 3 Gate**: Pipeline fails fast with clear errors when artifacts missing or type mismatched.

---

## Phase 4: US3 - Bidirectional Contract Validation (P3)

_Schema-level validation of artifact content at step boundaries._

### 4.1 Input Validator

- [X] [T030] [US3] Create `InputValidationResult` struct (`internal/contract/input_validator.go`)
- [X] [T031] [US3] Implement `ValidateInputArtifacts()` function (`internal/contract/input_validator.go`)
- [X] [T032] [US3] Integrate JSON schema validation via existing `contract.Validate()` (`internal/contract/input_validator.go`)

### 4.2 Executor Integration

- [X] [T033] [US3] Insert input validation call after `injectArtifacts()` in `runStepExecution()` (`internal/pipeline/executor.go:467-471`)
- [X] [T034] [US3] Ensure input validation runs BEFORE prompt building (`internal/pipeline/executor.go`)
- [X] [T035] [US3] Handle schema validation errors with detailed messages matching output contract format (`internal/pipeline/executor.go`)

### 4.3 US3 Tests

- [X] [T036] [P] [US3] Test: input artifact matching schema proceeds (`internal/contract/input_validator_test.go`)
- [X] [T037] [P] [US3] Test: input artifact violating schema fails with detailed errors (`internal/contract/input_validator_test.go`)
- [X] [T038] [P] [US3] Test: both input and output contracts validated in order (`internal/pipeline/executor_test.go`)
- [X] [T039] [P] [US3] Test: schema_path not specified skips validation (`internal/contract/input_validator_test.go`)

**Phase 4 Gate**: Input contracts validated before step execution. Schema errors are detailed.

---

## Phase 5: US4 - Artifact Template Resolution (P3)

_Resolve `{{artifacts.<name>}}` placeholders in step prompts._

### 5.1 Template Extension

- [X] [T040] [US4] Extend `ResolvePlaceholders()` to recognize `{{artifacts.<name>}}` pattern (`internal/pipeline/context.go`)
- [X] [T041] [US4] Implement artifact content lookup from `execution.ArtifactPaths` (`internal/pipeline/context.go`)
- [X] [T042] [US4] Handle missing required artifact in template (fail before substitution) (`internal/pipeline/context.go`)
- [X] [T043] [US4] Handle optional missing artifact in template (substitute empty string) (`internal/pipeline/context.go`)

### 5.2 US4 Tests

- [X] [T044] [P] [US4] Test: artifact content substituted correctly into prompt (`internal/pipeline/context_test.go`)
- [X] [T045] [P] [US4] Test: missing required artifact fails before substitution (`internal/pipeline/context_test.go`)
- [X] [T046] [P] [US4] Test: optional missing artifact substitutes to empty string (`internal/pipeline/context_test.go`)

**Phase 5 Gate**: `{{artifacts.<name>}}` placeholders resolve to artifact content in prompts.

---

## Phase 6: Polish & Cross-Cutting Concerns

_Documentation, integration tests, and final quality checks._

### 6.1 Documentation

- [ ] [T047] [P] Update `docs/pipelines.md` with stdout artifact capture example
- [ ] [T048] [P] Update `docs/pipelines.md` with typed consumption example
- [ ] [T049] [P] Update `docs/pipelines.md` with bidirectional contract example

### 6.2 Integration Testing

- [ ] [T050] Create end-to-end integration test: stdout capture pipeline (`tests/pipeline/stdout_capture_integration_test.go`)
- [ ] [T051] Create end-to-end integration test: typed artifact consumption pipeline (`tests/pipeline/typed_consumption_integration_test.go`)
- [ ] [T052] Create end-to-end integration test: bidirectional contracts pipeline (`tests/pipeline/bidirectional_contracts_integration_test.go`)

### 6.3 Example Pipeline

- [ ] [T053] Create example pipeline demonstrating all features (`.wave/pipelines/examples/typed-artifact-pipeline.yaml`)

### 6.4 Final Verification

- [ ] [T054] Run `go test ./...` to verify all tests pass
- [ ] [T055] Run `go test -race ./...` to check for race conditions
- [ ] [T056] Verify existing pipelines still work (backward compatibility)

**Phase 6 Gate**: All tests pass, documentation complete, example pipeline works.

---

## Summary

| Phase | Tasks | Parallelizable | Primary User Story |
|-------|-------|----------------|-------------------|
| 1. Setup | 6 | 3 | Foundation |
| 2. US1 Stdout | 13 | 4 | P1 |
| 3. US2 Types | 10 | 4 | P2 |
| 4. US3 Contracts | 10 | 4 | P3 |
| 5. US4 Templates | 7 | 3 | P3 |
| 6. Polish | 10 | 3 | Cross-cutting |
| **Total** | **56** | **21** | - |

## Dependencies

```
Phase 1 (Setup)
    │
    ├───► Phase 2 (US1 - Stdout)
    │           │
    │           └───► Phase 3 (US2 - Types) ────► Phase 5 (US4 - Templates)
    │                       │
    │                       └───► Phase 4 (US3 - Contracts)
    │
    └───────────────────────────────────────────► Phase 6 (Polish)
```

All phases depend on Phase 1 completion. Phase 2 (US1) must complete before Phase 3 (US2) because typed consumption needs stdout artifacts to exist. Phase 4 and 5 can proceed in parallel after Phase 3.
