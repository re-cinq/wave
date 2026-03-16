# Tasks

## Phase 1: Path Validation Tests
- [X] Task 1.1: Create `internal/security/path_test.go` with `TestValidatePath` — table-driven tests for valid paths, traversal sequences (../, ..\, encoded %2e%2e), max path length, absolute paths with no approved dirs
- [X] Task 1.2: Add `TestContainsTraversal` — test all traversal patterns including encoded variants
- [X] Task 1.3: Add `TestValidateApprovedDirectory` — test paths within/outside approved dirs, empty approved dirs, relative vs absolute
- [X] Task 1.4: Add `TestContainsSymlinks` — create real symlinks in temp dir, test detection, skip if symlinks unsupported
- [X] Task 1.5: Add `TestIsWithinDirectory` — test parent/child/sibling/identical directory relationships
- [X] Task 1.6: Add `TestSanitizePathForDisplay` — test short paths, long paths (>50 chars), paths with traversal sequences

## Phase 2: Sanitization Tests [P]
- [X] Task 2.1: Add `TestSanitizeSchemaContent` to `sanitize_test.go` — test script tag removal, event handler removal, javascript: URL removal, content size limit exceeded, prompt injection in schema [P]
- [X] Task 2.2: Add `TestSanitizeInput_StrictMode` — test strict mode rejects prompt injection patterns, returns error [P]
- [X] Task 2.3: Add `TestValidateInputLength` — test within limit, at limit, over limit [P]
- [X] Task 2.4: Add `TestIsHighRisk` — test records at/below/above threshold (score 50) [P]
- [X] Task 2.5: Add `TestRemoveSuspiciousContent` via `SanitizeSchemaContent` — script tags, onclick handlers, javascript: hrefs [P]

## Phase 3: Workspace Test Fix
- [X] Task 3.1: Fix `TestWorkspaceIsolation_NoPathTraversal` in `internal/workspace/workspace_test.go` — replace `t.Logf` with assertions that the traversal path does NOT resolve to the sensitive file content

## Phase 4: Validation
- [X] Task 4.1: Run `go test -race ./internal/security/ ./internal/workspace/` and verify all tests pass
- [X] Task 4.2: Run `golangci-lint run ./internal/security/ ./internal/workspace/` and fix any lint errors
