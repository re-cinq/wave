# Tasks: Preflight Recovery Guidance

## Phase 1: Error Type Infrastructure

- [X] Task 1.1: Add `ClassPreflight` to `ErrorClass` enum in `internal/recovery/recovery.go`
- [X] Task 1.2: Create `PreflightMetadata` struct in `internal/recovery/recovery.go` with `MissingSkills` and `MissingTools` fields
- [X] Task 1.3: Create `SkillError` type in `internal/preflight/preflight.go` with metadata fields
- [X] Task 1.4: Create `ToolError` type in `internal/preflight/preflight.go` with metadata fields
- [X] Task 1.5: Update `ClassifyError` in `internal/recovery/classify.go` to detect and classify preflight errors

## Phase 2: Fix Path Construction Bug

- [X] Task 2.1: Modify workspace path construction in `internal/recovery/recovery.go:54` to handle empty stepID without double slash
- [X] Task 2.2: Write unit test `TestBuildRecoveryBlock_EmptyStepID` to verify path correctness
- [X] Task 2.3: Write unit test `TestBuildRecoveryBlock_PreflightNoDoubleSlash` to verify preflight-specific path handling

## Phase 3: Preflight Error Wrapping

- [X] Task 3.1: Modify `CheckSkills` in `internal/preflight/preflight.go` to return `SkillError` with missing skill names [P]
- [X] Task 3.2: Modify `CheckTools` in `internal/preflight/preflight.go` to return `ToolError` with missing tool names [P]
- [X] Task 3.3: Update `Run` method in `internal/preflight/preflight.go` to preserve typed errors instead of joining strings
- [X] Task 3.4: Remove redundant "preflight check failed" wrapping in `internal/pipeline/executor.go:191`

## Phase 4: Recovery Hint Generation

- [X] Task 4.1: Add `PreflightMetadata` parameter to `BuildRecoveryBlock` signature in `internal/recovery/recovery.go`
- [X] Task 4.2: Add preflight-specific hint generation logic in `BuildRecoveryBlock` for missing skills
- [X] Task 4.3: Add preflight-specific hint generation logic in `BuildRecoveryBlock` for missing tools
- [X] Task 4.4: Ensure recovery blocks for preflight errors omit resume hints (no step to resume)
- [X] Task 4.5: Update `cmd/wave/commands/run.go` to extract preflight metadata and pass to `BuildRecoveryBlock`

## Phase 5: Unit Testing

- [X] Task 5.1: Write `TestClassifyError_PreflightSkill` in `internal/recovery/classify_test.go`
- [X] Task 5.2: Write `TestClassifyError_PreflightTool` in `internal/recovery/classify_test.go`
- [X] Task 5.3: Write `TestBuildRecoveryBlock_PreflightSkills` in `internal/recovery/recovery_test.go`
- [X] Task 5.4: Write `TestBuildRecoveryBlock_PreflightTools` in `internal/recovery/recovery_test.go`
- [X] Task 5.5: Write `TestBuildRecoveryBlock_PreflightMixed` for combined skill+tool failures
- [X] Task 5.6: Write `TestSkillError_Unwrap` in `internal/preflight/preflight_test.go` to verify error wrapping
- [X] Task 5.7: Write `TestToolError_Unwrap` in `internal/preflight/preflight_test.go` to verify error wrapping
- [X] Task 5.8: Write `TestCheckSkills_ReturnsSkillError` to verify typed error with metadata
- [X] Task 5.9: Write `TestCheckTools_ReturnsToolError` to verify typed error with metadata

## Phase 6: Integration Testing

- [X] Task 6.1: Create integration test for missing skill scenario - verify `wave skill install` hint appears
- [X] Task 6.2: Create integration test for missing tool scenario - verify helpful tool guidance
- [X] Task 6.3: Create integration test for mixed failures - verify both hint types appear
- [X] Task 6.4: Verify "preflight check failed" appears only once in all test scenarios
- [X] Task 6.5: Verify workspace paths contain no `//` in all test scenarios

## Phase 7: Manual Validation

- [X] Task 7.1: Run `go test ./...` to ensure all tests pass
- [X] Task 7.2: Manually test with missing speckit skill and verify hint quality (validated via integration tests)
- [X] Task 7.3: Manually test with missing CLI tool and verify hint quality (validated via integration tests)
- [X] Task 7.4: Test JSON output mode for preflight failures (validated via integration tests)
- [X] Task 7.5: Verify error message is concise and non-redundant (validated via integration tests)
