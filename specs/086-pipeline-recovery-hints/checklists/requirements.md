# Quality Checklist: 086-pipeline-recovery-hints

## Structure & Completeness

- [x] Spec follows the required template structure (User Scenarios, Requirements, Success Criteria)
- [x] Feature branch name and metadata are filled in
- [x] Status is set to Draft
- [x] Input/source is documented (GitHub Issue #85)

## User Scenarios

- [x] At least 3 user stories with priorities assigned (P1, P1, P2, P3)
- [x] Each user story has a clear "Why this priority" justification
- [x] Each user story has an "Independent Test" description
- [x] Each user story has acceptance scenarios in Given/When/Then format
- [x] User stories are ordered by priority (P1 first)
- [x] Each user story is independently testable and deliverable
- [x] Edge cases section is populated with realistic scenarios (7 edge cases)

## Requirements Quality

- [x] All functional requirements use MUST/SHOULD/MAY language (RFC 2119)
- [x] Each requirement is independently testable
- [x] Requirements are unambiguous — no subjective terms without measurable criteria
- [x] No implementation details leaked into requirements (technology-agnostic)
- [x] Requirements cover the happy path (FR-001, FR-002)
- [x] Requirements cover error-specific variants (FR-003 for contract failures, FR-005 for ambiguous)
- [x] Requirements cover output constraints (FR-006 conciseness, FR-007 visual separation)
- [x] Requirements cover all output modes (FR-008 JSON mode)
- [x] Requirements cover shell safety (FR-009 escaping)
- [x] Requirements specify performance constraints (FR-010 no additional I/O)
- [x] No more than 3 `[NEEDS CLARIFICATION]` markers (current: 0)

## Key Entities

- [x] Key entities are defined with clear descriptions
- [x] Entity relationships are understandable
- [x] Entities describe WHAT, not HOW (no data structures or schemas)

## Success Criteria

- [x] All success criteria are measurable (SC-001: 100% coverage, SC-003: 0% false positive)
- [x] Success criteria are technology-agnostic
- [x] Success criteria cover correctness (SC-002 copy-paste works)
- [x] Success criteria cover conciseness (SC-004 max 8 lines)
- [x] Success criteria cover non-regression (SC-005 existing tests pass)
- [x] Success criteria cover test coverage (SC-006 unit tests for each classification)

## Alignment with Issue #85

- [x] `--from-step` recovery command is specified (FR-002, US-1)
- [x] `--force` for contract failures is specified (FR-003, US-2)
- [x] Workspace path inspection is specified (FR-004, US-3)
- [x] Debug suggestion is specified (FR-005, US-4)
- [x] Original input echo in commands is specified (FR-002, FR-009)
- [x] Conciseness constraint is specified (FR-006: 3-5 lines per issue, 8 lines max in spec)

## Traceability

- [x] Every acceptance criterion from issue #85 maps to at least one requirement
  - "Pipeline failure prints at least one recovery command" → FR-001
  - "`--from-step` command includes the original input" → FR-002, FR-009
  - "Contract failures suggest `--force`" → FR-003
  - "Workspace path is shown for artifact inspection" → FR-004
- [x] No gold-plating — spec stays within scope of issue #85
- [x] Spec focuses on WHAT and WHY, not HOW
