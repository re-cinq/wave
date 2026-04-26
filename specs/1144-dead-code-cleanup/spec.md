# chore: dead code report (2 findings)

**Issue:** [#1144](https://github.com/re-cinq/wave/issues/1144)
**Repository:** re-cinq/wave
**Labels:** code-quality
**Author:** nextlevelshit
**State:** OPEN

## Body

# Dead Code Audit Report

**Scan date:** 2026-04-20
**Scan type:** dead-code
**Total findings:** 2

Scan of `internal/tui` (73 Go files) found 2 HIGH-confidence dead message types: `ComposeFocusDetailMsg` and `HealthTransitionMsg`. Both declared and case-handled but never instantiated.

## Summary by Type

| Type | Count |
|------|-------|
| `unused_export` | 2 |

## Summary by Suggested Action

| Action | Count |
|--------|-------|
| `remove` | 2 |

## Summary by Confidence

| Confidence | Count |
|------------|-------|
| HIGH | 2 |

## Task List

- [ ] **[DC-001]** (`unused_export`, `HIGH`) `internal/tui/compose_messages.go:27` -- Exported message type defined and handled in a switch case in content.go:922, but never instantiated anywhere in the codebase. Handler exists but no code path emits this message, suggesting an incomplete feature. Safe to remove both the type declaration and its case handler.
  Action: `remove` | Safe to remove: `true`
- [ ] **[DC-002]** (`unused_export`, `HIGH`) `internal/tui/guided_messages.go:9` -- Exported message type defined with an empty (no-op) case handler at content.go:1266 and never instantiated anywhere. Doc comment says it 'triggers the auto-transition from health to proposals' but implementation is incomplete. Safe to remove both the type and its empty case branch.
  Action: `remove` | Safe to remove: `true`

## Acceptance Criteria

- `ComposeFocusDetailMsg` type declaration removed from `internal/tui/compose_messages.go`
- `ComposeFocusDetailMsg` case handler removed from `internal/tui/content.go` (~line 922)
- `HealthTransitionMsg` type declaration removed from `internal/tui/guided_messages.go`
- `HealthTransitionMsg` empty case handler removed from `internal/tui/content.go` (~line 1266)
- `go build ./...` passes
- `go test ./...` passes
- `go vet ./...` clean
- `golangci-lint run` clean
