# Requirements Checklist: TUI Finished Pipeline Interactions

**Purpose**: Validate that the feature specification for #258 meets quality standards and completeness requirements.
**Created**: 2026-03-06
**Feature**: [spec.md](../spec.md)

## Specification Completeness

- [x] CHK001 All acceptance criteria from issue #258 are covered by user stories or functional requirements
- [x] CHK002 User stories are prioritized (P1/P2/P3) with clear rationale
- [x] CHK003 Each user story has acceptance scenarios in Given/When/Then format
- [x] CHK004 Edge cases are identified and documented
- [x] CHK005 All requirements use testable MUST/MUST NOT language
- [x] CHK006 Key entities are defined with clear descriptions
- [x] CHK007 Success criteria are measurable and tied to test strategies

## Issue Coverage

- [x] CHK008 Enter on finished pipeline opens chat session (US-1, FR-001/002/003)
- [x] CHK009 Chat session spawns Claude Code in workspace directory (US-1, FR-001/002)
- [x] CHK010 TUI suspends cleanly during chat (US-1, FR-003, C3)
- [x] CHK011 TUI restores after chat exits (US-1, FR-003/004)
- [x] CHK012 Chat inherits workspace directory and artifacts (US-1, FR-002/006, C2)
- [x] CHK013 `b` key checks out pipeline branch (US-2, FR-007/008)
- [x] CHK014 `d` key shows diff of changes (US-3, FR-010/011/012)
- [x] CHK015 Header branch updates on finished pipeline selection (US-4, FR-016, C6)
- [x] CHK016 Header reverts when no finished pipeline selected (US-4, FR-016)
- [x] CHK017 Error feedback for failed actions (US-2/3 scenarios, FR-014, C8)

## Clarification Quality

- [x] CHK018 All ambiguities have explicit Ambiguity → Resolution pairs
- [x] CHK019 No more than 3 [NEEDS CLARIFICATION] markers (0 present)
- [x] CHK020 Clarifications reference specific code/types where relevant
- [x] CHK021 Clarifications are consistent with existing TUI patterns (#253-#257)

## Architectural Consistency

- [x] CHK022 Follows Bubble Tea message-driven architecture (no direct method calls between components)
- [x] CHK023 Uses focus-gating pattern consistent with #256 (form) and #257 (live output)
- [x] CHK024 Uses `tea.Exec()` for subprocess management (standard Bubble Tea pattern)
- [x] CHK025 Status bar messaging follows `FormActiveMsg`/`LiveOutputActiveMsg` pattern
- [x] CHK026 Error display follows existing `stateError` pattern from detail model
- [x] CHK027 Data provider extension (`WorkspacePath`) follows existing pattern

## Scope Boundaries

- [x] CHK028 Spec focuses on WHAT and WHY, not HOW (no implementation details)
- [x] CHK029 Chat content/behavior is explicitly out of scope (handled by Claude Code)
- [x] CHK030 Pipeline launching is out of scope (handled by #256)
- [x] CHK031 Live streaming is out of scope (handled by #257)
- [x] CHK032 Header bar base implementation is out of scope (handled by #253)

## Notes

- All checklist items pass initial validation
- Zero [NEEDS CLARIFICATION] markers — all ambiguities were resolved via codebase analysis
- C6 confirms that header branch display is already implemented in #253 and requires no new work
- C9 clarifies that chat exit uses Claude Code's native exit, not Esc/q key bindings
