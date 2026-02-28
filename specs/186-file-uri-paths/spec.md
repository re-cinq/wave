# feat(display): prefix absolute file paths with file:// URI scheme for clickable terminal links

**Feature Branch**: `186-file-uri-paths`
**Created**: 2026-02-28
**Status**: Draft
**Issue**: [#186](https://github.com/re-cinq/wave/issues/186)
**Labels**: enhancement
**Complexity**: medium

## Summary

When Wave outputs absolute file paths in CLI error messages, recovery hints, and workspace artifact references, they should be prefixed with the `file://` URI scheme so they become clickable hyperlinks in modern terminal emulators (iTerm2, Kitty, Windows Terminal, GNOME Terminal, etc.).

## Motivation

Currently, paths like `/home/user/.wave/workspaces/.../issue-assessment.json` are displayed as plain text. Adding the `file://` prefix (e.g., `file:///home/user/.wave/workspaces/.../issue-assessment.json`) makes them ctrl-clickable in terminals that support URI detection, significantly improving developer experience when debugging pipeline failures.

### Example (current output)
```
- File: /home/mwc/Coding/recinq/wave/.wave/workspaces/gh-implement-20260228-050356-9682/__wt_gh-implement-20260228-050356-9682/.wave/output/issue-assessment.json
```

### Example (desired output)
```
- File: file:///home/mwc/Coding/recinq/wave/.wave/workspaces/gh-implement-20260228-050356-9682/__wt_gh-implement-20260228-050356-9682/.wave/output/issue-assessment.json
```

## Scope

Absolute paths should be prefixed with `file://` in the following CLI output locations:

- **Contract validation error details** — the `File:` field and schema reference paths in `internal/contract/validation_error_formatter.go` and `internal/contract/jsonschema.go`
- **Recovery hint commands** — workspace paths in `ls` suggestions in `internal/recovery/recovery.go`
- **Artifact path references** — any absolute path displayed to the user referencing local files in `internal/deliverable/types.go` (the `String()` method) and `internal/display/outcome.go` (workspace inspect hints)
- **Progress display artifact paths** — artifact paths shown during handover in `internal/display/progress.go` and `internal/display/bubbletea_model.go`

Paths that are already using a URI scheme (e.g., `https://`, `file://`) should not be double-prefixed.

## User Scenarios & Testing

### User Story 1 - Clickable Paths in Error Output (Priority: P1)

A developer runs a Wave pipeline that fails contract validation. The error output shows the artifact file path with `file://` prefix, making it ctrl-clickable in their terminal emulator.

**Why this priority**: This is the most common path display location — contract validation errors are the primary failure output users see.

**Independent Test**: Can be tested by verifying `FormatJSONSchemaError` and jsonschema validation errors output paths with `file://` prefix.

**Acceptance Scenarios**:

1. **Given** a contract validation fails, **When** the error is formatted, **Then** the `File:` detail line contains `file:///absolute/path` instead of `/absolute/path`
2. **Given** a contract validation fails with a relative path, **When** the error is formatted, **Then** the path is NOT prefixed (only absolute paths get the prefix)

---

### User Story 2 - Clickable Paths in Recovery Hints (Priority: P1)

A developer sees recovery hints after a pipeline failure. The workspace inspection command shows `file://` prefixed paths.

**Why this priority**: Recovery hints are shown alongside errors and are the primary way users inspect failed workspaces.

**Independent Test**: Can be tested by verifying `BuildRecoveryBlock` produces workspace paths with `file://` prefix when the path is absolute.

**Acceptance Scenarios**:

1. **Given** a pipeline step fails with an absolute workspace path, **When** recovery hints are generated, **Then** the workspace inspection command uses `file://` prefixed path
2. **Given** a pipeline step fails with a relative workspace path, **When** recovery hints are generated, **Then** the path is NOT prefixed

---

### User Story 3 - Clickable Paths in Outcome Display (Priority: P2)

A developer views pipeline outcome summary. Workspace inspection paths and artifact file paths use `file://` prefix.

**Why this priority**: Outcome display is the post-run summary; less urgent than error paths but still important for DX.

**Independent Test**: Can be tested by verifying `GenerateNextSteps` and `Deliverable.String()` output `file://` prefixed absolute paths.

**Acceptance Scenarios**:

1. **Given** a pipeline completes with a workspace path, **When** next steps are generated, **Then** the inspect workspace command uses `file://` prefix
2. **Given** deliverables are rendered, **When** a file deliverable has an absolute path, **Then** the rendered string uses `file://` prefix

---

### Edge Cases

- Paths already prefixed with `file://` must NOT be double-prefixed
- Paths prefixed with `https://` or other URI schemes must NOT be modified
- Relative paths (e.g., `.wave/workspaces/...`) must NOT be prefixed — only absolute paths starting with `/`
- Empty paths must not be modified
- Windows-style paths are not in scope (Wave targets Unix systems)

## Requirements

### Functional Requirements

- **FR-001**: System MUST prefix absolute file paths (`/...`) with `file://` in all CLI error output displayed to the user
- **FR-002**: System MUST NOT modify paths that already contain a URI scheme (e.g., `https://`, `file://`)
- **FR-003**: System MUST NOT prefix relative paths — only paths starting with `/`
- **FR-004**: System MUST provide a shared utility function for path formatting to ensure consistency
- **FR-005**: System MUST NOT modify paths used in internal processing — only display/output formatting is affected
- **FR-006**: Existing tests MUST continue to pass; new tests MUST cover the path formatting logic

## Acceptance Criteria

- [ ] All absolute file paths (`/...`) in CLI error output use `file:///...` URI scheme
- [ ] Paths already containing a URI scheme are not modified
- [ ] Recovery hint output includes clickable `file://` paths for workspace inspection
- [ ] Existing tests pass; new tests cover the path formatting logic
- [ ] No changes to paths used in internal processing — only display/output formatting is affected

## Success Criteria

### Measurable Outcomes

- **SC-001**: All absolute paths in user-facing CLI output contain `file://` prefix
- **SC-002**: Zero double-prefixed paths in any output
- **SC-003**: All existing tests pass without modification (except updating expected output strings)
- **SC-004**: New utility function has >90% test coverage including edge cases
