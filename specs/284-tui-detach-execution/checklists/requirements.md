# Requirements Checklist: Detach Pipeline Execution from TUI Process Lifecycle

**Purpose**: Validate spec completeness, testability, and alignment with Wave architecture
**Created**: 2026-03-08
**Feature**: [spec.md](../spec.md)

## Specification Completeness

- [x] CHK001 All user stories have acceptance scenarios in Given/When/Then format
- [x] CHK002 User stories are prioritized (P1-P3) and independently testable
- [x] CHK003 Edge cases are identified with expected behavior described (6 edge cases)
- [x] CHK004 All functional requirements use MUST/SHOULD/MAY language consistently
- [x] CHK005 Key entities are defined with clear descriptions (3 entities)
- [x] CHK006 No more than 3 `[NEEDS CLARIFICATION]` markers remain (1 marker: FR-011)

## Testability

- [x] CHK007 Every functional requirement maps to at least one acceptance scenario
- [x] CHK008 Success criteria are measurable with specific thresholds or conditions (7 criteria)
- [x] CHK009 Edge case behaviors are specific enough to write test assertions against
- [x] CHK010 Race conditions and concurrency scenarios are addressed (concurrent cancel, spawn-during-exit)

## Architecture Alignment

- [x] CHK011 Spec respects Wave's fresh-memory-at-step-boundaries principle (detached subprocess = fresh process)
- [x] CHK012 Spec leverages existing infrastructure: SQLite state store, event logging, cancellation table
- [x] CHK013 Spec does not introduce new runtime dependencies beyond the single static binary constraint
- [x] CHK014 Spec addresses security considerations (FR-012: env_passthrough for credentials, process isolation)
- [x] CHK015 Spec is compatible with the existing WebUI execution model (both use subprocess + SQLite pattern)

## Scope Boundaries

- [x] CHK016 Spec focuses on WHAT and WHY, not HOW (no implementation details like specific syscall usage)
- [x] CHK017 Spec does not prescribe specific data structures or function signatures
- [x] CHK018 Spec requirements are technology-agnostic where possible
- [x] CHK019 Spec does not duplicate or contradict existing CLAUDE.md guidelines

## Completeness Cross-Check

- [x] CHK020 All issue #284 requirements are covered: survive TUI exit, reconnect on reopen, `wave logs` streaming, cancellation
- [x] CHK021 Prerequisites from issue #284 are acknowledged (dbLoggingEmitter, FetchRunEvents, DismissRun, status bar support)
- [x] CHK022 TUI-CLI parity requirement from the issue is addressed as a user story (US3)

## Notes

- Self-validation completed: 22/22 items pass
- 1 NEEDS CLARIFICATION marker remains (FR-011 re: PID storage location)
- FR-012 added during validation to address CHK014 (credential security across process boundary)
