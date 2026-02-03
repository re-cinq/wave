---
description: "Task list for Enhanced Pipeline Progress Visualization implementation"
---

# Tasks: Enhanced Pipeline Progress Visualization

**Input**: Design documents from `/specs/018-enhanced-progress/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Not explicitly requested in specification, focusing on implementation tasks

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on Wave's Go project structure:
- **Core**: `internal/` packages at repository root
- **Commands**: `cmd/wave/commands/` for CLI integration
- **Tests**: `tests/` with unit/ and integration/ subdirectories

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure for enhanced progress visualization

- [ ] T001 Create internal/display package structure per implementation plan
- [ ] T002 [P] Create internal/display/terminal.go for TTY detection and terminal capability queries
- [ ] T003 [P] Create internal/display/capability.go for ANSI color and Unicode support detection
- [ ] T004 [P] Create internal/display/types.go for shared progress visualization types and constants

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T005 Extend Event struct in internal/event/types.go with optional progress fields (Progress, SubTaskCurrent, SubTaskTotal, EstimatedETA, FilesProcessed, ArtifactCount, WorkspacePath, ContractProgress)
- [ ] T006 Add new event state constants in internal/event/types.go (step_progress, eta_updated, contract_validating, compaction_progress)
- [ ] T007 Enhance NDJSONEmitter in internal/event/emitter.go to support dual-stream output (stderr for enhanced display, stdout for NDJSON)
- [ ] T008 [P] Create internal/display/formatter.go for ANSI escape sequence management and color formatting
- [ ] T009 [P] Add terminal size detection and window resize handling in internal/display/terminal.go
- [ ] T010 [P] Extend SQLite schema in internal/state/store.go to add progress tracking columns to event_log table
- [ ] T011 [P] Create new SQLite tables in internal/state/store.go (progress_snapshots, step_performance, display_settings)

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Real-time Step Progress Display (Priority: P1) 🎯 MVP

**Goal**: Users can see which pipeline step is currently executing with animated indicators confirming system activity

**Independent Test**: Run any pipeline and observe clear visual indicators of current step execution and system activity

### Implementation for User Story 1

- [ ] T012 [P] [US1] Create ProgressBar struct and rendering logic in internal/display/progress.go
- [ ] T013 [P] [US1] Create StepStatus display components in internal/display/progress.go for current step visualization
- [ ] T014 [P] [US1] Implement spinner animations in internal/display/animation.go for active processing indicators
- [ ] T015 [US1] Add real-time step progress event emission in internal/pipeline/executor.go during step execution
- [ ] T016 [US1] Integrate enhanced progress display in cmd/wave/commands/run.go for pipeline execution
- [ ] T017 [US1] Add step state transition detection and visual updates in internal/display/progress.go
- [ ] T018 [US1] Implement graceful fallback to basic text progress for non-TTY environments in internal/event/emitter.go
- [ ] T019 [US1] Add elapsed time tracking and display for currently executing step in internal/display/progress.go

**Checkpoint**: At this point, User Story 1 should be fully functional - users can see current step execution with animated progress indicators

---

## Phase 4: User Story 2 - Overall Pipeline Progress Tracking (Priority: P2)

**Goal**: Users can understand their position within overall pipeline execution, seeing completed, current, and pending steps

**Independent Test**: Run a multi-step pipeline and verify overall progress is clearly communicated throughout execution

### Implementation for User Story 2

- [ ] T020 [P] [US2] Create PipelineContext struct in internal/display/types.go for overall progress tracking
- [ ] T021 [P] [US2] Implement dashboard panel system in internal/display/dashboard.go with Wave ASCII logo integration
- [ ] T022 [P] [US2] Create ProgressContext calculation logic in internal/display/progress.go for overall pipeline completion percentage
- [ ] T023 [US2] Add pipeline overview display in internal/display/dashboard.go showing "Step X of Y" with progress bar
- [ ] T024 [US2] Implement ETA calculation based on completed vs remaining steps in internal/display/progress.go
- [ ] T025 [US2] Add project information panel in internal/display/dashboard.go (manifest path, pipeline name, workspace)
- [ ] T026 [US2] Integrate dashboard display with cmd/wave/commands/run.go for pipeline execution
- [ ] T027 [US2] Add step completion status indicators (✓ ✗ ⏳) in internal/display/dashboard.go
- [ ] T028 [US2] Implement responsive layout for different terminal sizes in internal/display/dashboard.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently - users see both current step progress and overall pipeline context

---

## Phase 5: User Story 3 - Step Duration and Performance Insights (Priority: P3)

**Goal**: Users can see step timing information and identify performance patterns for optimization

**Independent Test**: Run pipelines multiple times and verify timing information is displayed and historically useful

### Implementation for User Story 3

- [ ] T029 [P] [US3] Create PerformanceMetric collection in internal/state/store.go for historical step duration tracking
- [ ] T030 [P] [US3] Implement step duration display in internal/display/progress.go showing elapsed and completed times
- [ ] T031 [P] [US3] Add animated counter components in internal/display/animation.go for "numbers going up" effect on tokens/files/artifacts
- [ ] T032 [US3] Add historical performance queries in internal/state/store.go for step duration averages and trends
- [ ] T033 [US3] Implement performance metrics panel in internal/display/dashboard.go (tokens used, files modified, artifacts generated)
- [ ] T034 [US3] Add performance comparison indicators in internal/display/dashboard.go to highlight unusually slow steps
- [ ] T035 [US3] Integrate performance insights with cmd/wave/commands/logs.go for enhanced log analysis
- [ ] T036 [US3] Add token burn rate calculation and display in internal/display/dashboard.go
- [ ] T037 [US3] Implement performance history cleanup and retention in internal/state/store.go

**Checkpoint**: All user stories should now be independently functional - complete progress visualization system with performance insights

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories and overall system integration

- [ ] T038 [P] Add enhanced progress display integration to cmd/wave/commands/status.go for pipeline status queries
- [ ] T039 [P] Add enhanced progress display integration to cmd/wave/commands/logs.go for real-time log streaming
- [ ] T040 [P] Implement display configuration options in internal/display/types.go (animation enable/disable, refresh rate, color themes)
- [ ] T041 [P] Add progress display tests in tests/unit/display/progress_test.go for progress bar rendering and animation logic
- [ ] T042 [P] Add dashboard tests in tests/unit/display/dashboard_test.go for panel layout and responsive design
- [ ] T043 [P] Add integration tests in tests/integration/progress_test.go for end-to-end progress display functionality
- [ ] T044 [P] Add performance monitoring for progress display overhead (target <5%) in internal/display/metrics.go
- [ ] T045 Code cleanup and optimization across all display components
- [ ] T046 Documentation updates for new progress visualization features in relevant docs/
- [ ] T047 Run quickstart.md validation to ensure implementation meets specification requirements

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Independent but builds on US1 visual patterns
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Independent but uses dashboard components from US2

### Within Each User Story

- Core display components before CLI integration
- Visual rendering before animation features
- Basic functionality before enhancement features
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- Models within a story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch display components for User Story 1 together:
Task: "Create ProgressBar struct and rendering logic in internal/display/progress.go"
Task: "Create StepStatus display components in internal/display/progress.go"
Task: "Implement spinner animations in internal/display/animation.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Verify real-time step progress display works with any pipeline

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Real-time step progress (MVP!)
3. Add User Story 2 → Test independently → Add overall pipeline tracking and dashboard
4. Add User Story 3 → Test independently → Add performance insights and optimization features
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (step-level progress)
   - Developer B: User Story 2 (pipeline-level tracking)
   - Developer C: User Story 3 (performance insights)
3. Stories complete and integrate independently

---

## Technical Notes

### Key Integration Points
- **Event System**: Extend existing events without breaking NDJSON compatibility
- **CLI Commands**: Integrate enhanced display with run, logs, status commands
- **State Persistence**: Add progress tracking to existing SQLite schema
- **Terminal Compatibility**: Graceful degradation for non-TTY environments

### Performance Requirements
- Progress display updates within 1 second of state changes
- Visualization overhead must remain under 5% of total execution time
- Smooth animations at 60ms refresh rate for enhanced user experience
- Efficient rendering using double-buffering to prevent screen flicker

### Backward Compatibility
- All new Event fields are optional with omitempty JSON tags
- NDJSON output to stdout remains unchanged
- Enhanced display uses stderr to avoid breaking existing tool integrations
- Non-TTY environments automatically fall back to existing text progress

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Focus on Go stdlib + existing dependencies (no new runtime dependencies per Wave constitution)
- Verify enhanced progress works across Linux/macOS/Windows terminals
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Target: Beautiful, engaging progress visualization matching modern CLI tools like Claude Code