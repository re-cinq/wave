# Quality Checklist: 261-tui-compose-ui

## Specification Completeness

- [x] **CL-001**: Feature branch name follows convention (`NNN-short-name`) — `261-tui-compose-ui`
- [x] **CL-002**: Issue reference links to correct GitHub issue (#261)
- [x] **CL-003**: Context section describes relationship to parent epic (#251) and prior issues (#252–#260)
- [x] **CL-004**: Dependency on #249 (cross-pipeline artifact handoff) is clearly documented
- [x] **CL-005**: All acceptance criteria from issue #261 are covered by user stories

## User Stories

- [x] **CL-006**: Each user story has a priority assignment (P1, P2, P3)
- [x] **CL-007**: Each user story is independently testable (each has "Independent Test" section)
- [x] **CL-008**: User stories use Given/When/Then acceptance scenario format
- [x] **CL-009**: P1 story covers the core compose mode interaction (enter, build, cancel)
- [x] **CL-010**: Artifact flow visualization is specified as a separate story (User Story 2)
- [x] **CL-011**: Artifact compatibility validation is specified with warning/error behavior (User Story 3)
- [x] **CL-012**: Sequence start and grouped running display are specified (User Story 4)
- [x] **CL-013**: CLI equivalent (`wave run p1 p2 p3` / `wave compose`) is specified (User Story 5)

## Edge Cases

- [x] **CL-014**: Duplicate pipeline in sequence behavior is specified (allow with notice)
- [x] **CL-015**: Single-pipeline sequence behavior is specified (behaves as normal launch)
- [x] **CL-016**: Narrow terminal graceful degradation is specified (text-only below 120 cols)
- [x] **CL-017**: No output artifacts pipeline behavior is specified (shows "No artifacts" + warning)
- [x] **CL-018**: Compose mode during running pipeline behavior is specified (compose still opens)
- [x] **CL-019**: Empty sequence (all removed) behavior is specified (Enter disabled)
- [x] **CL-020**: `s` key on non-available items behavior is specified (no effect)

## Requirements Quality

- [x] **CL-021**: All functional requirements use MUST/SHOULD/MAY language (all use MUST)
- [x] **CL-022**: Requirements are testable (each FR has a clear pass/fail condition)
- [x] **CL-023**: No implementation details in requirements (WHAT, not HOW)
- [x] **CL-024**: Key entities are defined with clear descriptions (Sequence, ArtifactFlow, CompatibilityResult)
- [x] **CL-025**: Maximum 3 `[NEEDS CLARIFICATION]` markers (0 markers present)

## Success Criteria

- [x] **CL-026**: Success criteria are measurable and technology-agnostic
- [x] **CL-027**: Performance criteria specified (single frame update responsiveness — SC-004)
- [x] **CL-028**: Accessibility criteria specified (terminal width 80+ cols — SC-005)
- [x] **CL-029**: Consistency criteria specified (follows existing Bubble Tea patterns — SC-008)

## Issue Acceptance Criteria Coverage

- [x] **CL-030**: `s` key on available pipeline opens compose/sequence mode — FR-001, User Story 1
- [x] **CL-031**: Compose mode shows sequence list with add (`a`), remove (`x`), reorder (`↑`/`↓`) controls — FR-003/004/005, User Story 1
- [x] **CL-032**: Right pane shows artifact flow visualization between chained pipelines — FR-006, User Story 2
- [x] **CL-033**: Artifact compatibility validated before starting — warns if inputs don't match outputs — FR-007, User Story 3
- [x] **CL-034**: `Enter` starts the full sequence — FR-008, User Story 4
- [x] **CL-035**: Running sequences show as grouped items in the Running section with per-pipeline progress — FR-010, User Story 4
- [x] **CL-036**: `Esc` cancels compose mode without starting — FR-009, User Story 1
- [x] **CL-037**: CLI equivalent: `wave run pipeline1 pipeline2 pipeline3` or `wave compose --sequence p1,p2,p3` — FR-012, User Story 5
