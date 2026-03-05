# Requirements Checklist: TUI Pipeline List Left Pane

**Purpose**: Validate that the specification for issue #254 is complete, testable, and aligned with the TUI epic (#251) architecture.  
**Created**: 2026-03-05  
**Feature**: [spec.md](../spec.md)  
**Validation**: Pass (35/35) — 2026-03-05

## Completeness

- [x] **CHK001**: All three pipeline sections (Running, Finished, Available) are specified with data sources and sort order — FR-001, FR-003, FR-004, FR-005, FR-013/014/015
- [x] **CHK002**: Section header format is defined (name + count) — FR-002
- [x] **CHK003**: Running pipeline display includes elapsed time indicator — FR-003, US1-AS2
- [x] **CHK004**: Finished pipeline display includes status and duration — FR-004, US1-AS4
- [x] **CHK005**: Available pipeline display includes pipeline names from manifest discovery — FR-005, FR-015

## Navigation

- [x] **CHK006**: Arrow key (↑/↓) navigation is specified with cross-section boundary behavior — FR-006, US2-AS1
- [x] **CHK007**: Selection clamping at list boundaries is specified (no wrap) — FR-007, US2-AS2/AS3
- [x] **CHK008**: Visual selection indicator (▶) is defined — FR-006, US2-AS4
- [x] **CHK009**: Selection-changed message emission is specified for inter-component communication — FR-008, US2-AS5
- [x] **CHK010**: Default focus on left pane at TUI launch is specified — FR-012

## Search/Filter

- [x] **CHK011**: `/` key activation for filter input is specified — FR-009, US3-AS1
- [x] **CHK012**: Case-insensitive substring matching behavior is defined — FR-009, US3-AS2
- [x] **CHK013**: Escape key to dismiss filter is specified — FR-010, US3-AS3
- [x] **CHK014**: Empty results state ("No matching pipelines") is specified — US3-AS4
- [x] **CHK015**: Navigation within filtered results is specified — US3-AS5

## Scrolling

- [x] **CHK016**: Viewport scrolling to keep selected item visible is specified — FR-011, US4-AS1/AS2
- [x] **CHK017**: Behavior for lists exceeding visible area is covered — US4-AS1/AS2/AS3

## Section Collapse

- [x] **CHK018**: Section collapse/expand toggle is specified (as SHOULD) — FR-016
- [x] **CHK019**: Collapsed section display (header + indicator) is defined — US5-AS1
- [x] **CHK020**: Navigation behavior with collapsed sections is specified — US5-AS3

## Integration

- [x] **CHK021**: Integration with existing ContentModel is specified (replaces placeholder) — FR-017
- [x] **CHK022**: Data sources are identified (SQLite state DB for Running/Finished, DiscoverPipelines for Available) — FR-013/014/015
- [x] **CHK023**: Message types for inter-component communication are referenced (PipelineSelectedMsg) — FR-008
- [x] **CHK024**: NO_COLOR compliance is specified — FR-018

## Edge Cases

- [x] **CHK025**: Missing/inaccessible SQLite database behavior is specified — Edge Case 1
- [x] **CHK026**: Missing/malformed wave.yaml behavior is specified — Edge Case 2
- [x] **CHK027**: Terminal resize below minimum width behavior is specified — Edge Case 3
- [x] **CHK028**: Long pipeline name truncation is specified — Edge Case 5
- [x] **CHK029**: All-empty sections state is specified — Edge Case 6
- [x] **CHK030**: Runtime status change (running → finished) behavior is specified — Edge Case 4

## Quality Gates

- [x] **CHK031**: All user stories have acceptance scenarios in Given/When/Then format — 5/5 stories verified
- [x] **CHK032**: All functional requirements use MUST/SHOULD language — FR-001 to FR-015 (MUST), FR-016 (SHOULD)
- [x] **CHK033**: Success criteria are measurable and technology-agnostic — SC-001 to SC-007 verified
- [x] **CHK034**: No more than 3 [NEEDS CLARIFICATION] markers remain — 0 markers present
- [x] **CHK035**: Spec focuses on WHAT/WHY, not HOW (no implementation details) — verified, no code or architecture specifics

## Notes

- Section collapse is specified as SHOULD (FR-016) since it is P3 priority
- The right pane detail view is explicitly out of scope per issue #254
- Pipeline launching and live output streaming are out of scope
- Data refresh mechanism (polling interval for Running section elapsed time updates) is left to implementation discretion
