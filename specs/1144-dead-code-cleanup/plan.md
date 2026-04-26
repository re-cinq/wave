# Implementation Plan — 1144 dead code cleanup

## 1. Objective

Remove 2 HIGH-confidence dead exported message types in `internal/tui` (`ComposeFocusDetailMsg`, `HealthTransitionMsg`) along with their orphan case handlers in `content.go`.

## 2. Approach

Pure deletion. No replacement, no refactor. Both types are unused exports with handlers that never fire. Verify with build + vet + tests.

## 3. File Mapping

| File | Change | Detail |
|------|--------|--------|
| `internal/tui/compose_messages.go` | modify | Delete `ComposeFocusDetailMsg` type (lines 26-27) |
| `internal/tui/guided_messages.go` | modify | Delete `HealthTransitionMsg` type (lines 8-9) |
| `internal/tui/content.go` | modify | Delete `case ComposeFocusDetailMsg:` block (~lines 922-930) and `case HealthTransitionMsg:` block (~lines 1266-1267) |

No new files. No deletions of whole files (sibling types in both message files remain in use).

## 4. Architecture Decisions

- **Delete, don't deprecate.** Pre-1.0; no backward-compat for unused exports.
- **Leave doc-only specs alone.** Stale references in `specs/261-tui-compose-ui/` and `specs/248-guided-tui-orchestrator/` are historical planning artifacts — not load-bearing.
- **No replacement message wiring.** Audit flagged both as "incomplete features." Restoring them is out of scope; if reintroduced later, the type can be re-added with an emitter.

## 5. Risks

| Risk | Mitigation |
|------|------------|
| Hidden producer in non-Go file or generated code | `grep -r` confirmed only declarations + handler match. Build will catch any miss. |
| External consumer (downstream import) | `internal/` package — unimportable externally. Safe. |
| Lint failure on whitespace after deletion | Run `gofmt -w` after edits. |

## 6. Testing Strategy

- `go build ./...` — compile sanity
- `go vet ./...` — orphan reference check
- `go test ./internal/tui/...` — package tests still pass
- `go test ./...` — full suite green
- `golangci-lint run` — no new warnings

No new tests required: dead-code removal does not change behavior. Existing TUI tests cover the surviving message handlers.
