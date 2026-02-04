# Tasks: Add Missing Implementer and Reviewer Personas

**Input**: Design documents from `/specs/021-add-missing-personas/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: No explicit test tasks requested. Verification via `go test ./...` and manual pipeline execution.

**Organization**: Tasks grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Exact file paths included in descriptions

## Path Conventions

This is a configuration-only feature. No source code changes required.

```
wave.yaml                              # Manifest with persona definitions
.wave/personas/                        # System prompt files
internal/defaults/personas/            # Embedded defaults for wave init
```

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify existing state and prepare for persona additions

- [ ] T001 Verify current branch is `021-add-missing-personas` with `git branch --show-current`
- [ ] T002 Run `go test ./...` to establish baseline - all tests must pass before changes
- [ ] T003 [P] Verify `.wave/personas/` directory exists and review existing persona structure

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: This feature has no blocking prerequisites beyond setup - all user stories are configuration file additions that can proceed immediately after setup.

**Checkpoint**: Setup complete - user story implementation can begin

---

## Phase 3: User Story 1 - Run Default Pipelines Successfully (Priority: P1) 🎯 MVP

**Goal**: Add `implementer` and `reviewer` persona definitions so pipelines can resolve them

**Independent Test**: `wave validate` should pass without "persona not found" errors

### Implementation for User Story 1

- [ ] T004 [P] [US1] Create implementer system prompt at `.wave/personas/implementer.md` following craftsman pattern
- [ ] T005 [P] [US1] Create reviewer system prompt at `.wave/personas/reviewer.md` following auditor pattern
- [ ] T006 [US1] Add implementer persona definition to `wave.yaml` in personas section with permissions: Read, Write, Edit, Bash, Glob, Grep and deny: Bash(rm -rf /*), Bash(sudo *)
- [ ] T007 [US1] Add reviewer persona definition to `wave.yaml` in personas section with permissions: Read, Glob, Grep, Write(artifact.json), Write(artifacts/*), Bash(go test*), Bash(npm test*) and deny: Write(*.go), Write(*.ts), Edit(*)

### Verification for User Story 1

- [ ] T008 [US1] Run `wave validate` and verify no "persona not found" errors for implementer or reviewer
- [ ] T009 [US1] Run `wave run gh-poor-issues --dry-run "test"` to verify pipeline can resolve personas

**Checkpoint**: User Story 1 complete - default pipelines can now resolve implementer and reviewer personas

---

## Phase 4: User Story 2 - Pipeline Steps Produce Artifacts (Priority: P1)

**Goal**: Ensure implementer and reviewer personas have correct Write permissions for artifact.json

**Independent Test**: Run a pipeline step using implementer/reviewer and verify artifact.json is created

### Implementation for User Story 2

- [ ] T010 [US2] Verify implementer in `wave.yaml` has Write permission (already added in T006 - confirm no wildcards blocking artifact.json)
- [ ] T011 [US2] Verify reviewer in `wave.yaml` has Write(artifact.json) and Write(artifacts/*) permissions (already added in T007 - confirm pattern correctness)
- [ ] T012 [US2] Update implementer.md Output Format section to emphasize artifact.json output for contract compatibility
- [ ] T013 [US2] Update reviewer.md Output Format section to emphasize artifact.json output for contract compatibility

### Verification for User Story 2

- [ ] T014 [US2] Review persona markdown files confirm JSON output guidance without embedded schema details

**Checkpoint**: User Story 2 complete - personas can write artifacts for pipeline handoff

---

## Phase 5: User Story 3 - Contract Validation Works End-to-End (Priority: P2)

**Goal**: Confirm personas work with json_schema contract validation

**Independent Test**: Run pipeline with json_schema contract and verify validation passes

### Implementation for User Story 3

- [ ] T015 [US3] Verify implementer.md does NOT contain embedded schema details (schemas injected at runtime by executor)
- [ ] T016 [US3] Verify reviewer.md does NOT contain embedded schema details (schemas injected at runtime by executor)
- [ ] T017 [US3] Confirm both personas mention "schema will be injected" in Output Format section

### Verification for User Story 3

- [ ] T018 [US3] Run `grep -i "schema" .wave/personas/implementer.md .wave/personas/reviewer.md` and verify only references to injected schemas, not embedded ones

**Checkpoint**: User Story 3 complete - personas compatible with contract validation

---

## Phase 6: User Story 4 - Wave Init Includes All Personas (Priority: P3)

**Goal**: Add personas to embedded defaults so `wave init` scaffolds them

**Independent Test**: Run `wave init` in empty directory and verify implementer.md and reviewer.md are created

### Implementation for User Story 4

- [ ] T019 [P] [US4] Copy `.wave/personas/implementer.md` to `internal/defaults/personas/implementer.md`
- [ ] T020 [P] [US4] Copy `.wave/personas/reviewer.md` to `internal/defaults/personas/reviewer.md`
- [ ] T021 [US4] Verify `internal/defaults/embed.go` uses `//go:embed personas/*` directive (should auto-include new files)

### Verification for User Story 4

- [ ] T022 [US4] Run `go build ./...` to verify embedded files compile correctly
- [ ] T023 [US4] Test `wave init` in temporary directory: `mkdir /tmp/test-wave-init && cd /tmp/test-wave-init && wave init && ls .wave/personas/`

**Checkpoint**: User Story 4 complete - wave init includes new personas

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and cleanup

- [ ] T024 Run full test suite: `go test ./...` and verify all tests pass
- [ ] T025 Run `wave validate` final verification
- [ ] T026 [P] Review all changes with `git diff` for consistency
- [ ] T027 Commit all changes with descriptive message

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - start immediately
- **Foundational (Phase 2)**: N/A - no foundational tasks for this feature
- **User Story 1 (Phase 3)**: Depends on Setup - core persona definitions
- **User Story 2 (Phase 4)**: Can run parallel with US1 or after - artifact permissions
- **User Story 3 (Phase 5)**: Depends on US1/US2 - contract compatibility verification
- **User Story 4 (Phase 6)**: Depends on US1 - embedded defaults
- **Polish (Phase 7)**: Depends on all stories complete

### User Story Dependencies

- **US1 (P1)**: No dependencies - core feature
- **US2 (P1)**: Overlaps with US1 (same files) - can be concurrent or sequential
- **US3 (P2)**: Verification only - depends on US1/US2 being complete
- **US4 (P3)**: Depends on US1 (needs persona files to exist first)

### Parallel Opportunities

Within Phase 3 (User Story 1):
- T004 and T005 can run in parallel (different files)

Within Phase 6 (User Story 4):
- T019 and T020 can run in parallel (different files)

---

## Parallel Example: User Story 1

```bash
# Launch persona file creation in parallel:
Task: "Create implementer system prompt at .wave/personas/implementer.md"
Task: "Create reviewer system prompt at .wave/personas/reviewer.md"

# Then sequentially update wave.yaml (same file):
Task: "Add implementer persona definition to wave.yaml"
Task: "Add reviewer persona definition to wave.yaml"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 3: User Story 1 (T004-T009)
3. **STOP and VALIDATE**: Run `wave validate` and `wave run gh-poor-issues --dry-run`
4. Feature is functional - pipelines can now resolve personas

### Incremental Delivery

1. Setup + US1 → Pipelines work (MVP!)
2. Add US2 → Artifacts work → Test independently
3. Add US3 → Contracts validated → Test independently
4. Add US4 → Wave init complete → Test independently
5. Polish → Final verification

### Single Developer Strategy

Execute in order: Setup → US1 → US2 → US3 → US4 → Polish

Estimated: 27 tasks, ~30 minutes for configuration-only feature

---

## Notes

- [P] tasks = different files, no dependencies
- This is a configuration-only feature - no Go code changes required
- Personas must NOT embed schema details - schemas injected at runtime
- Wave.yaml permission patterns support wildcards: `Write(artifact.json)`, `Bash(go test*)`
- Verify baseline tests pass before and after changes
- Commit after completing each user story phase
