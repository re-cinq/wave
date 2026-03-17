# Implementation Plan: WebUI Test Coverage

## Objective

Increase `internal/webui/` test coverage from 53.3% to at least 70% by adding tests for uncovered handlers, SSE edge cases, artifact serving, template helpers, and control handlers.

## Approach

Add test files targeting the largest coverage gaps. The existing test infrastructure (`testTemplates`, `testServer`, mock state store) is well-established — extend it rather than creating new patterns.

**Coverage gap analysis** (0% coverage functions to target):
1. `handlers_artifacts.go` — `handleArtifact`, `detectMimeType` (0%)
2. `handlers_sse.go` — `matchesRunID` (0%), `handleSSE` partial gaps
3. `handlers_runs.go` — `handleRunDetailPage`, `buildStepDetails`, `eventToSummary`, `artifactToSummary`, `formatDurationValue` partial
4. `handlers_personas.go` — `handlePersonasPage` (0%)
5. `handlers_pipelines.go` — `handlePipelinesPage` (0%)
6. `embed.go` — `statusClass`, `formatDuration`, `formatDurationShort`, `formatMinSec`, `formatTime`, `formatTokensFunc` (all 0%)
7. `server.go` — `resolveGitHubToken`, `detectRepoSlug` (0%, but depend on external commands)

**Strategy**: Focus on pure functions and HTTP handler tests using httptest. Skip functions that depend on external commands (server.go) as they provide minimal coverage gain for high complexity.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/webui/handlers_artifacts_test.go` | create | Tests for artifact handler: success, missing params, not found, path traversal, raw download, truncation, MIME detection |
| `internal/webui/handlers_sse_test.go` | create | Tests for SSE handler: connection setup, Last-Event-ID reconnection, matchesRunID, client disconnect |
| `internal/webui/handlers_runs_test.go` | create | Tests for run detail page, buildStepDetails, eventToSummary, artifactToSummary, formatDurationValue |
| `internal/webui/handlers_control_test.go` | create | Tests for start, cancel, retry, resume handlers — valid/invalid/conflict states |
| `internal/webui/embed_test.go` | create | Tests for template helper functions: statusClass, formatDuration, formatTime, formatTokensFunc |
| `internal/webui/handlers_test.go` | modify | May need minor updates to testServer helper if additional mock methods are needed |

## Architecture Decisions

1. **Use existing test patterns**: `testServer` helper with `testTemplates` and mock state store — no new test framework
2. **httptest for HTTP handlers**: Standard library `httptest.NewRecorder` and `httptest.NewRequest`
3. **Table-driven tests**: Consistent with codebase conventions
4. **No external dependencies**: All tests use in-memory mocks, no network/filesystem side effects
5. **SSE testing**: Use a flushing `httptest.ResponseRecorder` wrapper for SSE stream assertions
6. **Skip `server.go` functions**: `resolveGitHubToken`/`detectRepoSlug` depend on `exec.Command` — not worth mocking for coverage target

## Risks

| Risk | Mitigation |
|------|------------|
| Template rendering tests brittle if templates change | Use minimal assertions (no error, status 200), not content matching |
| SSE handler tests may be flaky with goroutine timing | Use buffered channels and short timeouts with `select` |
| Mock state store may need new methods | Extend existing mock incrementally |
| `loadPipelineYAML` reads filesystem | Create temp pipeline files in test setup |

## Testing Strategy

- Run `go test -cover ./internal/webui/` before and after to verify coverage delta
- Run `go test -race ./internal/webui/` to catch SSE-related race conditions
- Target: each new test file covers one handler file's 0% functions to reach 70%+ total
- No `t.Skip()` without linked issue per project policy
