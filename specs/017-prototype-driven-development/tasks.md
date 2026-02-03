# Tasks: Prototype-Driven Development Pipelines

**Input**: Design documents from `/specs/017-prototype-driven-development/`
**Prerequisites**: spec.md (required), wave.yaml (manifest), existing pipeline patterns
**Feature Branch**: `017-prototype-driven-development`

**Tests**: Tests are included as this is a pipeline feature requiring contract validation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: gopkg.in/yaml.v3, github.com/spf13/cobra
**Storage**: SQLite for pipeline state, filesystem for artifacts
**Testing**: go test ./...
**Target Platform**: Linux/macOS CLI
**Project Type**: Single project (existing Wave codebase)

## Path Conventions

- **Pipeline YAML**: `.wave/pipelines/prototype.yaml`
- **Personas**: `.wave/personas/` (existing directory)
- **Contracts**: `.wave/contracts/` (existing directory)
- **Source**: `internal/` (existing Go packages)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the prototype pipeline definition and supporting configuration

- [x] T001 Create prototype pipeline YAML skeleton in .wave/pipelines/prototype.yaml
- [ ] T002 [P] Add specifier persona configuration to wave.yaml for spec phase
- [ ] T003 [P] Add documenter persona configuration to wave.yaml for docs phase
- [ ] T004 [P] Add prototyper persona configuration to wave.yaml for dummy phase
- [ ] T005 [P] Create specifier persona system prompt in .wave/personas/specifier.md
- [ ] T006 [P] Create documenter persona system prompt in .wave/personas/documenter.md
- [ ] T007 [P] Create prototyper persona system prompt in .wave/personas/prototyper.md

**Checkpoint**: Pipeline skeleton and persona configurations ready for phase implementation

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Define contracts and validation schemas that all phases depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T008 Create spec phase output contract schema in .wave/contracts/spec-phase.schema.json
- [x] T009 [P] Create docs phase output contract schema in .wave/contracts/docs-phase.schema.json
- [x] T010 [P] Create dummy phase output contract schema in .wave/contracts/dummy-phase.schema.json
- [x] T011 [P] Create implementation phase contract schema in .wave/contracts/implement-phase.schema.json
- [x] T012 Define artifact paths and naming convention for prototype pipeline artifacts

**Checkpoint**: Foundation ready - all contracts defined, user story implementation can begin

---

## Phase 3: User Story 1 - Initialize New Greenfield Project with Spec Phase (Priority: P1)

**Goal**: Enable developers to initialize a prototype-driven project and complete the specification phase using speckit integration.

**Independent Test**: Initialize a new project, run the spec phase, verify a valid specification artifact is produced.

### Tests for User Story 1

- [ ] T013 [P] [US1] Create integration test for spec phase initialization in internal/pipeline/prototype_spec_test.go
- [ ] T014 [P] [US1] Create contract validation test for spec phase output in internal/contract/spec_phase_test.go

### Implementation for User Story 1

- [ ] T015 [US1] Implement spec phase step in .wave/pipelines/prototype.yaml with speckit integration
- [ ] T016 [US1] Define spec phase input handling (project description) in pipeline input section
- [ ] T017 [US1] Configure spec phase workspace isolation in .wave/pipelines/prototype.yaml
- [ ] T018 [US1] Add spec phase output artifacts definition (spec.md, requirements.md) in pipeline
- [ ] T019 [US1] Implement handover contract validation for spec phase in .wave/pipelines/prototype.yaml
- [ ] T020 [US1] Add iterative refinement support (allow re-running spec phase) in pipeline definition
- [ ] T021 [US1] Test spec phase with mock adapter: wave run --pipeline prototype --mock --input "test project"

**Checkpoint**: Spec phase independently functional - can initialize project and produce valid specification

---

## Phase 4: User Story 2 - Generate Documentation from Specification (Priority: P2)

**Goal**: Automatically generate human-readable documentation from the completed specification that describes the planned feature.

**Independent Test**: Provide a completed spec artifact, run docs phase, verify documentation is generated matching the spec.

### Tests for User Story 2

- [ ] T022 [P] [US2] Create integration test for docs phase in internal/pipeline/prototype_docs_test.go
- [ ] T023 [P] [US2] Create contract validation test for docs phase prerequisites in internal/contract/docs_phase_test.go

### Implementation for User Story 2

- [ ] T024 [US2] Implement docs phase step in .wave/pipelines/prototype.yaml with dependency on spec phase
- [ ] T025 [US2] Configure artifact injection from spec phase (inject spec.md as context)
- [ ] T026 [US2] Define docs phase prompt for generating feature documentation in pipeline
- [ ] T027 [US2] Add docs phase output artifacts (feature-docs.md, stakeholder-summary.md)
- [ ] T028 [US2] Implement prerequisite validation - block if spec phase artifacts missing
- [ ] T029 [US2] Add handover contract for docs phase output validation
- [ ] T030 [US2] Test docs phase with completed spec artifact input

**Checkpoint**: Docs phase independently functional - can generate documentation from valid spec

---

## Phase 5: User Story 3 - Build Dummy Implementation (Priority: P3)

**Goal**: Create a working prototype/dummy that demonstrates interfaces and user flows with stub implementations.

**Independent Test**: Provide completed docs, run dummy phase, verify a runnable prototype with placeholder responses is produced.

### Tests for User Story 3

- [ ] T031 [P] [US3] Create integration test for dummy phase in internal/pipeline/prototype_dummy_test.go
- [ ] T032 [P] [US3] Create contract validation test for dummy output in internal/contract/dummy_phase_test.go

### Implementation for User Story 3

- [ ] T033 [US3] Implement dummy phase step in .wave/pipelines/prototype.yaml with dependency on docs phase
- [ ] T034 [US3] Configure artifact injection from docs phase (inject feature-docs.md)
- [ ] T035 [US3] Define dummy phase prompt for generating stub implementation
- [ ] T036 [US3] Configure workspace mount with readwrite access for prototype code generation
- [ ] T037 [US3] Add dummy phase output artifacts (prototype code, interface definitions)
- [ ] T038 [US3] Implement prerequisite validation - block if docs artifacts missing
- [ ] T039 [US3] Add handover contract for dummy phase (validate runnable prototype)
- [ ] T040 [US3] Test dummy phase with completed docs artifact input

**Checkpoint**: Dummy phase independently functional - can generate working prototype from docs

---

## Phase 6: User Story 4 - Transition to Full Implementation (Priority: P4)

**Goal**: Carry forward all artifacts and provide a clear starting point for real implementation work.

**Independent Test**: Complete all prior phases, run implement phase, verify all artifacts are accessible and guidance is provided.

### Tests for User Story 4

- [ ] T041 [P] [US4] Create integration test for implement phase in internal/pipeline/prototype_implement_test.go
- [ ] T042 [P] [US4] Create end-to-end pipeline test in internal/pipeline/prototype_e2e_test.go

### Implementation for User Story 4

- [ ] T043 [US4] Implement implement phase step in .wave/pipelines/prototype.yaml with dependency on dummy phase
- [ ] T044 [US4] Configure artifact injection from all prior phases (spec, docs, dummy)
- [ ] T045 [US4] Define implement phase prompt providing implementation guidance
- [ ] T046 [US4] Configure workspace with full readwrite access for implementation
- [ ] T047 [US4] Add implement phase output artifacts (implementation progress, checklist)
- [ ] T048 [US4] Implement stale artifact detection - warn if upstream phases re-run
- [ ] T049 [US4] Add handover contract with test suite validation (go test ./...)
- [ ] T050 [US4] Test full pipeline end-to-end with mock adapter

**Checkpoint**: Full pipeline functional - all four phases work in sequence

---

## Phase 7: Edge Cases & Error Handling

**Purpose**: Handle error conditions and edge cases identified in spec

- [ ] T051 Add phase skip validation - error when attempting to skip phases (e.g., spec to dummy)
- [ ] T052 [P] Add stale artifact detection when upstream phases are re-run
- [ ] T053 [P] Add clear error messages for phase failures with retry guidance
- [ ] T054 Implement phase resume support (--from-step flag integration)
- [ ] T055 Add concurrent run protection using workspace isolation

**Checkpoint**: Error handling complete - all edge cases covered

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, cleanup, and integration validation

- [ ] T056 [P] Add prototype pipeline to wave list pipelines output
- [ ] T057 [P] Update CLAUDE.md with prototype pipeline usage examples
- [ ] T058 Run all tests with race detector: go test -race ./...
- [ ] T059 Run quickstart validation for prototype pipeline
- [ ] T060 Final end-to-end test without mock adapter

---

## Phase 9: User Story 5 - Automated PR Cycle (Priority: P5)

**Goal**: Automate the pull request lifecycle from creation through review, response, fixes, and merge.

**Independent Test**: Complete implementation phase, run PR cycle phases, verify PR is created, reviewed, comments addressed, and merged.

### Tests for User Story 5

- [ ] T061 [P] [US5] Create integration test for PR creation step in internal/pipeline/prototype_pr_test.go
- [ ] T062 [P] [US5] Create contract validation test for PR cycle phases in internal/contract/pr_cycle_test.go

### Implementation for User Story 5

- [ ] T063 [US5] Implement pr-create step in .wave/pipelines/prototype.yaml with dependency on implement phase
- [ ] T064 [US5] Implement pr-review step (add Copilot reviewer, poll for completion) in prototype.yaml
- [ ] T065 [US5] Implement pr-respond step (Claude analyzes and responds to comments) in prototype.yaml
- [ ] T066 [US5] Implement pr-fix step (craftsman implements small fixes based on review) in prototype.yaml
- [ ] T067 [US5] Implement follow-up issue creation for large changes that exceed pr-fix scope
- [ ] T068 [US5] Implement pr-merge step with auto-merge support in prototype.yaml
- [ ] T069 [US5] Add timeout and retry handling for review polling with configurable intervals
- [ ] T070 [US5] End-to-end test for full PR cycle with mock adapter

**Checkpoint**: PR cycle independently functional - can create, review, respond, fix, and merge PRs automatically

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - User stories are sequential (US1 -> US2 -> US3 -> US4) due to artifact dependencies
- **Edge Cases (Phase 7)**: Depends on all user stories complete
- **Polish (Phase 8)**: Depends on Edge Cases complete
- **User Story 5 (Phase 9)**: Depends on Phase 8 completion - extends pipeline with PR automation

### User Story Dependencies

- **User Story 1 (P1)**: Spec Phase - No dependencies on other stories, can be tested independently
- **User Story 2 (P2)**: Docs Phase - Depends on US1 artifacts but can be tested with mock spec artifacts
- **User Story 3 (P3)**: Dummy Phase - Depends on US2 artifacts but can be tested with mock docs artifacts
- **User Story 4 (P4)**: Implement Phase - Depends on US3 artifacts but can be tested with mock inputs
- **User Story 5 (P5)**: PR Cycle - Depends on US4 artifacts but can be tested with mock implementation artifacts

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Contract schemas must exist before pipeline step implementation
- Artifact injection configured before output artifact definition
- Story complete and tested before moving to next priority

### Parallel Opportunities

- All Setup persona tasks (T002-T007) can run in parallel
- All Foundational contract tasks (T009-T011) can run in parallel
- Test tasks within each user story can run in parallel
- Edge case tasks T052-T053 can run in parallel
- PR cycle test tasks T061-T062 can run in parallel

---

## Parallel Example: Setup Phase

```bash
# Launch all persona configurations in parallel:
Task: "Add specifier persona configuration to wave.yaml for spec phase"
Task: "Add documenter persona configuration to wave.yaml for docs phase"
Task: "Add prototyper persona configuration to wave.yaml for dummy phase"

# Launch all persona prompts in parallel:
Task: "Create specifier persona system prompt in .wave/personas/specifier.md"
Task: "Create documenter persona system prompt in .wave/personas/documenter.md"
Task: "Create prototyper persona system prompt in .wave/personas/prototyper.md"
```

## Parallel Example: Foundational Phase

```bash
# Launch all contract schemas in parallel (after T008):
Task: "Create docs phase output contract schema in .wave/contracts/docs-phase.schema.json"
Task: "Create dummy phase output contract schema in .wave/contracts/dummy-phase.schema.json"
Task: "Create implementation phase contract schema in .wave/contracts/implement-phase.schema.json"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (pipeline skeleton, personas)
2. Complete Phase 2: Foundational (contracts)
3. Complete Phase 3: User Story 1 (spec phase)
4. **STOP and VALIDATE**: Test spec phase independently
5. Deploy/demo if ready - developers can at least generate specifications

### Incremental Delivery

1. Complete Setup + Foundational -> Foundation ready
2. Add User Story 1 -> Test independently -> Demo (MVP: spec generation)
3. Add User Story 2 -> Test independently -> Demo (spec + docs)
4. Add User Story 3 -> Test independently -> Demo (spec + docs + dummy)
5. Add User Story 4 -> Test independently -> Demo (full pipeline)
6. Add User Story 5 -> Test independently -> Demo (full pipeline + automated PR cycle)
7. Each story adds value without breaking previous stories

### Suggested MVP Scope

Focus on User Story 1 (Spec Phase) first:
- Provides immediate value for requirement gathering
- Validates pipeline infrastructure
- Allows iteration on persona prompts
- Foundation for subsequent phases

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Pipeline phases are sequential but can be tested independently with mock inputs
- Use existing Wave personas (craftsman, navigator) where possible before creating new ones
- Contract schemas follow existing patterns in .wave/contracts/
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
