# Data Integration & State Management Quality: TUI Header Bar

**Feature**: 253-tui-header-bar | **Date**: 2026-03-05

## Data Source Specification Quality

- [ ] CHK044 - Does FR-007 specify the exact git commands used for each data field (branch, dirty, commit, remote), or just the data sources generically? [Clarity]
- [ ] CHK045 - Is the manifest loading path clearly specified — does `manifest.Load()` search from CWD, or from a specific path? Is the search behavior defined? [Clarity]
- [ ] CHK046 - Does the spec define the expected format of the `gh api` response for issues count, or does it assume a specific schema? [Completeness]
- [ ] CHK047 - Is the distinction between "auth not configured" (edge case 7) and "API unreachable" (edge case 3) testable from the spec description alone? [Clarity]
- [ ] CHK048 - Does the spec define how the repo identifier is resolved when `manifest.Metadata.Repo` is empty — which git remote URL format is parsed (HTTPS, SSH, both)? [Completeness]

## State Transition Quality

- [ ] CHK049 - Are all state transitions for the logo animation (idle→active, active→idle) defined with their trigger conditions and resulting visual state? [Completeness]
- [ ] CHK050 - Does the spec define what happens to the override branch when the selected pipeline's state changes (e.g., from finished to re-running)? [Coverage]
- [ ] CHK051 - Is the health status aggregation rule unambiguous? If one pipeline is WARN and another is ERR, does ERR take precedence? [Clarity]
- [ ] CHK052 - Does FR-013 define whether event-driven refresh and periodic refresh can conflict, and if so, which takes priority? [Coverage]

## Schema Migration Quality

- [ ] CHK053 - Does C-004 define the migration's backward compatibility — what happens if old code reads a record without `branch_name`? [Completeness]
- [ ] CHK054 - Is the `DEFAULT ''` for `branch_name` semantically meaningful? Does the spec define whether empty string means "not yet resolved" vs "no worktree"? [Clarity]
- [ ] CHK055 - Does the spec define when `UpdateRunBranch` is called relative to pipeline execution lifecycle events? Is the timing precise enough to avoid races? [Clarity]

## Async Loading Quality

- [ ] CHK056 - Does FR-012 define the order in which async fetches are initiated, or can they all fire concurrently? [Clarity]
- [ ] CHK057 - Is there a timeout or error budget for async metadata loading (SC-007 says 2 seconds), and is the behavior defined when this budget is exceeded? [Completeness]
- [ ] CHK058 - Does the spec define whether failed async fetches are retried, and if so, with what backoff strategy? [Coverage]
