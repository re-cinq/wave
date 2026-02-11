# Add unique hash suffix to pipeline IDs for collision prevention

**Feature Branch**: `025-pipeline-id-hash`
**Issue**: [re-cinq/wave#25](https://github.com/re-cinq/wave/issues/25)
**Created**: 2026-02-11
**Status**: Draft
**Labels**: enhancement, ready-for-impl
**Author**: nextlevelshit

## Summary

Add a unique hash suffix to pipeline IDs to prevent naming collisions when multiple pipelines are executed concurrently or when pipelines are re-run.

## Problem

Currently, pipeline IDs are derived directly from `Pipeline.Metadata.Name` (e.g., `github-issue-enhancer`). This means:
- State management conflicts in SQLite when the same pipeline runs concurrently
- Workspace directory collisions (`.wave/workspaces/<pipeline_id>/`) between runs
- Ambiguous audit logs and tracing since multiple runs share the same ID

## Proposed Solution

Append a short hash suffix (e.g., 8-character hex) to pipeline IDs based on:
- Timestamp of execution
- Input parameters hash
- Random entropy

### Example
```
Before: github-issue-enhancer
After:  github-issue-enhancer-a3b2c1d4
```

## User Scenarios & Testing

### User Story 1 - Unique Pipeline Run IDs (Priority: P1)

When a pipeline executes, its runtime ID includes a unique hash suffix so that concurrent and repeated runs never collide in state storage or workspace directories.

**Why this priority**: This is the core problem. Without unique IDs, concurrent runs corrupt each other's state and workspaces.

**Independent Test**: Execute the same pipeline twice rapidly and verify both runs have distinct IDs, distinct workspace directories, and distinct state records.

**Acceptance Scenarios**:

1. **Given** a pipeline `my-pipeline`, **When** it executes, **Then** its runtime ID is `my-pipeline-<8-char-hex>` (e.g., `my-pipeline-a3b2c1d4`).
2. **Given** two concurrent executions of `my-pipeline`, **When** both start, **Then** they receive different hash suffixes and use separate workspace directories.
3. **Given** the same pipeline re-runs, **When** it starts, **Then** it gets a new unique suffix, not reusing the previous one.

---

### User Story 2 - State Queries Work with New ID Format (Priority: P1)

Existing state queries (GetPipelineState, GetStepStates, ListRecentPipelines) continue to function correctly with the new pipeline ID format.

**Why this priority**: Breaking state queries would render the ops commands unusable.

**Independent Test**: Run a pipeline, then query its state by run ID and by pipeline name, confirming both work.

**Acceptance Scenarios**:

1. **Given** a completed pipeline with ID `my-pipeline-a3b2c1d4`, **When** querying by exact ID, **Then** the state record is returned.
2. **Given** multiple runs of `my-pipeline`, **When** listing recent pipelines, **Then** all runs appear and can be distinguished by their hash suffixes.

---

### User Story 3 - Configurable Hash Length (Priority: P2)

The hash suffix length is configurable, defaulting to 8 characters.

**Why this priority**: Nice-to-have flexibility; 8 chars provides sufficient uniqueness for most scenarios.

**Independent Test**: Configure `runtime.pipeline_id_hash_length: 4` in the manifest and verify the suffix is 4 hex characters.

**Acceptance Scenarios**:

1. **Given** no configuration override, **When** a pipeline runs, **Then** the hash suffix is 8 hex characters.
2. **Given** `runtime.pipeline_id_hash_length: 12` in the manifest, **When** a pipeline runs, **Then** the hash suffix is 12 hex characters.

---

### Edge Cases

- What happens when `crypto/rand` is unavailable? Fall back to timestamp-based entropy.
- How does resume work with the new ID format? Resume must use the full suffixed ID.
- What about the `pipeline_state` table which uses `pipeline_id` as primary key with UPSERT? Each run now has a unique key, so no conflicts.
- What about workspace cleanup (`os.RemoveAll`) that currently targets `<wsRoot>/<pipelineID>`? Each run cleans only its own directory.

## Requirements

### Functional Requirements

- **FR-001**: System MUST generate a unique runtime pipeline ID by appending a hash suffix to `Pipeline.Metadata.Name`.
- **FR-002**: The hash suffix MUST be generated using cryptographic randomness (`crypto/rand`) with timestamp-based fallback.
- **FR-003**: The default hash suffix length MUST be 8 hex characters (4 bytes of entropy).
- **FR-004**: The hash suffix length MUST be configurable via `runtime.pipeline_id_hash_length` in the manifest.
- **FR-005**: All state storage operations MUST use the suffixed runtime ID as the pipeline identifier.
- **FR-006**: Workspace paths MUST use the suffixed runtime ID for isolation.
- **FR-007**: Event emissions MUST use the suffixed runtime ID for traceability.
- **FR-008**: Resume operations MUST accept and use the full suffixed ID.
- **FR-009**: The original pipeline name (`Metadata.Name`) MUST remain accessible for display and filtering purposes.

### Key Entities

- **Pipeline Runtime ID**: `{pipeline_name}-{hash_suffix}` - the unique execution identifier
- **Pipeline Name**: `Metadata.Name` - the logical pipeline name, unchanged from the manifest

## Acceptance Criteria (from issue)

- [ ] Pipeline IDs include a unique hash suffix
- [ ] Hash suffix is deterministic based on execution context (or configurable to be random)
- [ ] Existing state queries continue to work with new ID format
- [ ] Hash length is configurable (default: 8 characters)
- [ ] Documentation updated to reflect new ID format

## Success Criteria

### Measurable Outcomes

- **SC-001**: No state collisions when running the same pipeline concurrently (verified by test).
- **SC-002**: No workspace directory collisions between concurrent runs (verified by test).
- **SC-003**: All existing tests pass with the new ID generation.
- **SC-004**: Pipeline state can be queried by both full runtime ID and pipeline name.
