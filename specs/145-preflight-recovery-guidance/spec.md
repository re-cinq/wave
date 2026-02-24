# Preflight recovery message lacks actionable guidance and shows malformed workspace path

**Issue**: [#145](https://github.com/re-cinq/wave/issues/145)
**Feature Branch**: `145-preflight-recovery-guidance`
**Created**: 2026-02-24
**Status**: Implementation
**Labels**: bug, ux, priority: medium, pipeline

## Summary

When a pipeline fails preflight checks due to missing required skills, the recovery options shown to the user are insufficient and contain a malformed path (double trailing slash). The message does not suggest how to actually resolve the problem.

## Current Behavior

Running a pipeline that requires the `speckit` skill produces:

```
pipeline execution failed: preflight check failed: preflight check failed: missing required skills: speckit

Recovery options:
  Inspect workspace artifacts:
    ls .wave/workspaces/speckit-flow-20260223-114229-0e8a//
```

**Problems:**
1. The workspace path contains a double trailing slash (`//`), suggesting a path-join bug
2. The only recovery option is "inspect workspace artifacts" which does not help the user fix the missing skill
3. The error message says "preflight check failed" twice (redundant nesting)
4. No actionable guidance is provided (e.g., `wave skill install speckit` or equivalent)

## User Scenarios & Testing

### User Story 1 - Missing Skill Recovery (Priority: P1)

When a pipeline fails due to a missing skill, the user receives actionable recovery hints that guide them to install the missing skill.

**Why this priority**: This is the most common preflight failure and the one that most needs actionable guidance. Currently users are stuck with no clear path forward.

**Independent Test**: Run a pipeline with a missing skill dependency and verify the error message suggests installing the skill.

**Acceptance Scenarios**:

1. **Given** a pipeline requires the `speckit` skill, **When** the skill is not installed and the pipeline runs, **Then** the recovery options include "Install missing skill: wave skill install speckit"
2. **Given** a pipeline requires multiple missing skills, **When** the pipeline runs, **Then** the recovery options suggest installing each missing skill separately
3. **Given** a preflight check fails for missing skills, **When** the error is displayed, **Then** the error message does NOT redundantly repeat "preflight check failed"

---

### User Story 2 - Missing Tool Recovery (Priority: P2)

When a pipeline fails due to a missing CLI tool, the user receives guidance on how to install the tool or configure it properly.

**Why this priority**: Tool installation varies by system, so this is lower priority than skill installation which has a consistent command.

**Independent Test**: Run a pipeline with a missing tool dependency and verify the error message provides helpful guidance.

**Acceptance Scenarios**:

1. **Given** a pipeline requires a CLI tool not on PATH, **When** the pipeline runs, **Then** the recovery options suggest checking PATH or installing the tool
2. **Given** multiple tools are missing, **When** preflight fails, **Then** each missing tool is listed clearly

---

### User Story 3 - Clean Workspace Paths (Priority: P1)

When any error recovery options include workspace paths, those paths are correctly formatted without double slashes or other malformations.

**Why this priority**: Path bugs undermine user trust and can cause copy-paste errors. This is a simple bug fix with high visibility.

**Independent Test**: Trigger any preflight failure and verify workspace paths do not contain `//`.

**Acceptance Scenarios**:

1. **Given** preflight check fails before a step executes (no stepID), **When** workspace path is shown, **Then** it does not contain double trailing slashes
2. **Given** any recovery hint includes a workspace path, **When** displayed to the user, **Then** the path is properly formatted

---

### Edge Cases

- What happens when preflight fails but stepID is empty (no step started yet)?
- How does the system handle preflight failures when workspaceRoot has a trailing slash?
- What if both tools and skills are missing simultaneously?

## Requirements

### Functional Requirements

- **FR-001**: System MUST provide context-aware recovery hints based on preflight failure type (missing skill vs. missing tool)
- **FR-002**: System MUST suggest `wave skill install <skill-name>` for missing skills
- **FR-003**: System MUST NOT generate workspace paths with double slashes
- **FR-004**: Error messages MUST NOT redundantly repeat "preflight check failed" in the error chain
- **FR-005**: Recovery hints for preflight failures MUST handle empty stepID gracefully (since preflight runs before step execution)

### Key Entities

- **PreflightError**: A new error class to distinguish preflight failures from other runtime errors
- **RecoveryHint**: Enhanced to support preflight-specific hints with skill/tool names

## Success Criteria

### Measurable Outcomes

- **SC-001**: Users encountering missing skill errors can resolve them in one command without documentation lookup
- **SC-002**: Zero instances of double-slash paths in recovery messages
- **SC-003**: Error messages are concise and non-redundant
- **SC-004**: All preflight failure types provide at least one actionable recovery hint
