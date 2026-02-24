# Implementation Plan: Preflight Recovery Guidance

## Objective

Fix two related UX issues in preflight error handling:
1. Replace generic "inspect workspace artifacts" hints with actionable, context-aware recovery suggestions (e.g., `wave skill install <skill>` for missing skills)
2. Eliminate the double trailing slash bug in workspace paths shown in recovery messages

## Approach

### High-Level Strategy

The fix involves three key changes:

1. **Add preflight error classification**: Create a new `ClassPreflight` error class in the recovery package to distinguish preflight failures from runtime errors
2. **Enhance recovery hint generation**: Modify `BuildRecoveryBlock` to accept preflight-specific metadata (missing skills/tools) and generate appropriate hints
3. **Fix path construction**: Ensure workspace paths are built without trailing slashes when stepID is empty

The solution follows the existing recovery hint architecture from issue #86 (pipeline-recovery-hints) but extends it to handle preflight-specific failures.

### Key Design Decision

**Where to intercept preflight errors?**

Option A: Modify `internal/pipeline/executor.go` to wrap preflight errors with metadata before returning
Option B: Pass preflight results through the error chain and extract them in `cmd/wave/commands/run.go`

**Chosen**: Option A - wrap at the source in `executor.go`. This keeps the metadata close to where the error originates and avoids parsing error strings in the command layer.

### Path Bug Root Cause

The double slash occurs in `internal/recovery/recovery.go:54`:
```go
WorkspacePath: fmt.Sprintf("%s/%s/%s/", workspaceRoot, runID, stepID)
```

When `stepID` is empty (preflight fails before any step executes), this produces:
```
.wave/workspaces/run-id-123//
```

**Fix**: Check if `stepID` is empty and omit the trailing component.

## File Mapping

### Files to Modify

1. **internal/recovery/recovery.go**
   - Add `ClassPreflight` to `ErrorClass` enum
   - Add `PreflightMetadata` struct to carry missing skills/tools
   - Modify `BuildRecoveryBlock` signature to accept optional preflight metadata
   - Fix workspace path construction to handle empty stepID
   - Add logic to generate skill/tool-specific hints

2. **internal/recovery/classify.go**
   - Add preflight error detection in `ClassifyError` function

3. **internal/preflight/preflight.go**
   - Create custom error types: `SkillError` and `ToolError`
   - Modify `CheckSkills` and `CheckTools` to return typed errors with metadata
   - Modify `Run` to return a wrapped error that preserves skill/tool lists

4. **internal/pipeline/executor.go** (around line 191)
   - Detect preflight errors and extract metadata
   - Pass metadata to recovery block generation in the error path

5. **cmd/wave/commands/run.go** (around line 303-304)
   - Update `BuildRecoveryBlock` call to pass preflight metadata when available

### Files to Add

None - all changes are modifications to existing files.

### Files to Test

1. **internal/recovery/recovery_test.go** - Add tests for preflight class and metadata
2. **internal/recovery/classify_test.go** - Add tests for preflight error classification
3. **internal/preflight/preflight_test.go** - Add tests for typed errors
4. **Integration test** - End-to-end test for missing skill scenario

## Architecture Decisions

### 1. Error Type Design

**Decision**: Create specific error types (`SkillError`, `ToolError`) in the preflight package rather than generic wrappers.

**Rationale**:
- Type-safe metadata extraction using `errors.As()`
- Clear ownership (preflight owns its error types)
- Follows Go error wrapping best practices

### 2. Recovery Hint Priority

When generating hints for preflight failures:
1. Install missing skills/tools (most actionable)
2. Workspace inspection (useful for debugging)
3. Re-run with debug (only if helpful)

**No resume hint**: Preflight failures happen before step execution, so there's no step to resume from.

### 3. Metadata Structure

```go
type PreflightMetadata struct {
    MissingSkills []string
    MissingTools  []string
}
```

**Rationale**: Simple, focused structure. Skills and tools are handled separately because they have different recovery actions.

## Risks

### 1. Breaking Changes to Recovery API

**Risk**: Changing `BuildRecoveryBlock` signature could break existing callers.

**Mitigation**: Use optional parameters pattern or add a new function. Review all call sites first.

### 2. Preflight Errors Without Metadata

**Risk**: If some code path returns preflight errors without proper wrapping, hints won't be generated.

**Mitigation**: Comprehensive testing of all preflight failure paths. Add fallback to generic hints if metadata is missing.

### 3. Double Error Wrapping

**Risk**: The redundant "preflight check failed: preflight check failed" suggests nested error wrapping.

**Mitigation**: Audit error wrapping in executor.go and ensure we only wrap once. Consider unwrapping before re-wrapping.

## Testing Strategy

### Unit Tests

1. **recovery package**:
   - `TestClassifyError_Preflight` - Verify preflight errors are classified correctly
   - `TestBuildRecoveryBlock_PreflightSkills` - Verify skill install hints are generated
   - `TestBuildRecoveryBlock_PreflightTools` - Verify tool hints are generated
   - `TestBuildRecoveryBlock_EmptyStepID` - Verify path doesn't have `//`

2. **preflight package**:
   - `TestSkillError_Type` - Verify error type wrapping
   - `TestCheckSkills_ReturnsSkillError` - Verify metadata is preserved
   - `TestCheckTools_ReturnsToolError` - Verify metadata is preserved

### Integration Tests

1. **Missing skill scenario**:
   - Setup: Pipeline with `required_skills: [speckit]`, skill not installed
   - Run: Execute pipeline
   - Verify: Error message contains `wave skill install speckit`
   - Verify: No `//` in workspace path
   - Verify: "preflight check failed" appears only once

2. **Missing tool scenario**:
   - Setup: Pipeline with `required_tools: [nonexistent-tool]`
   - Run: Execute pipeline
   - Verify: Error message suggests PATH or installation
   - Verify: No `//` in workspace path

3. **Mixed failures**:
   - Setup: Missing both skills and tools
   - Verify: Both types of hints are shown

### Manual Testing Checklist

- [ ] Run pipeline with missing skill, verify `wave skill install <name>` hint
- [ ] Run pipeline with missing tool, verify helpful guidance
- [ ] Verify workspace path has no `//` in any error scenario
- [ ] Verify "preflight check failed" appears only once in error chain
- [ ] Verify JSON output mode includes preflight hints correctly
