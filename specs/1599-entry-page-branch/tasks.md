# Work Items

## Phase 1: Setup
- [X] Item 1.1: Confirm `onboarding.SentinelFile` exported and import path correct
- [X] Item 1.2: Confirm `s.runtime.repoDir` populated in `NewServer`

## Phase 2: Core Implementation
- [X] Item 2.1: Create `internal/webui/handlers_root.go` with `Server.handleRoot` (sentinel stat → 302 `/work` or `/onboard`)
- [X] Item 2.2: Update `internal/webui/routes.go` `GET /{$}` to call `s.handleRoot` (replaces hardcoded `/runs` redirect)

## Phase 3: Testing
- [X] Item 3.1: Add `internal/webui/handlers_root_test.go` — sentinel present → `/work` [P]
- [X] Item 3.2: Add `internal/webui/handlers_root_test.go` — sentinel missing → `/onboard` [P]
- [X] Item 3.3: Run `go test ./internal/webui/... -race`

## Phase 4: Polish
- [X] Item 4.1: Run `golangci-lint run ./internal/webui/...`
- [ ] Item 4.2: Manual smoke: `wave server` → curl `/` with and without sentinel
- [X] Item 4.3: Verify no emojis introduced
