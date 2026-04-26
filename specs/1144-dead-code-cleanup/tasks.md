# Work Items

## Phase 1: Setup

- [X] Item 1.1: Confirm feature branch `1144-dead-code-cleanup` checked out from clean `main`
- [X] Item 1.2: Re-verify no other producers exist via `grep -r "ComposeFocusDetailMsg\|HealthTransitionMsg" --include="*.go"`

## Phase 2: Core Implementation

- [X] Item 2.1: Remove `ComposeFocusDetailMsg` type from `internal/tui/compose_messages.go` [P]
- [X] Item 2.2: Remove `HealthTransitionMsg` type from `internal/tui/guided_messages.go` [P]
- [X] Item 2.3: Remove `case ComposeFocusDetailMsg:` block (~content.go:922-930)
- [X] Item 2.4: Remove `case HealthTransitionMsg:` block (~content.go:1266-1267)
- [X] Item 2.5: `gofmt -w internal/tui/`

## Phase 3: Testing

- [X] Item 3.1: `go build ./...`
- [X] Item 3.2: `go vet ./...`
- [X] Item 3.3: `go test ./internal/tui/...` [P]
- [X] Item 3.4: `go test ./...` (full suite)
- [ ] Item 3.5: `golangci-lint run` (binary unavailable in workspace; skipped — CI will run it)

## Phase 4: Polish

- [X] Item 4.1: Confirm diff is minimal — only target deletions, no incidental edits
- [X] Item 4.2: Conventional commit: `chore(tui): remove dead ComposeFocusDetailMsg and HealthTransitionMsg`
