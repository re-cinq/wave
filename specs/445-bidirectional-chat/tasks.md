# Tasks

## Phase 1: State Layer — Chat Session Persistence

- [X] Task 1.1: Create `internal/state/chat_session.go` with `ChatSession` record type (SessionID, RunID, StepFilter, WorkspacePath, Model, CreatedAt, LastResumedAt)
- [X] Task 1.2: Add `chat_session` table migration in `internal/state/migration_definitions.go`
- [X] Task 1.3: Add `SaveChatSession`, `GetChatSession`, `ListChatSessions` methods to `StateStore` interface in `internal/state/store.go`
- [X] Task 1.4: Implement SQLite methods for chat sessions in `internal/state/store.go`
- [X] Task 1.5: Write tests in `internal/state/chat_session_test.go` for CRUD operations

## Phase 2: Adapter — Session ID Capture

- [X] Task 2.1: Modify `LaunchInteractive` in `internal/adapter/interactive.go` to return a session ID (captured from Claude Code's output or `--print-session-id` if available) [P]
- [X] Task 2.2: Update `InteractiveOptions` to include `Resume` session ID support (already partially present — wire it through)
- [X] Task 2.3: Write tests for session ID extraction in `internal/adapter/interactive_test.go` [P]

## Phase 3: CLI — Chat Command Enhancement

- [X] Task 3.1: Add `--resume` flag to `wave chat` command that accepts a session ID or "last"
- [X] Task 3.2: After interactive session completes, save the session record to state store
- [X] Task 3.3: On `--resume`, load the previous session's workspace and pass `--resume <session-id>` to Claude Code
- [X] Task 3.4: Enhance `--list` to show chat sessions alongside runs (session ID, run, step, timestamp)
- [X] Task 3.5: Update `cmd/wave/commands/chat_test.go` with tests for resume flow and session listing

## Phase 4: Testing & Validation

- [X] Task 4.1: Run `go test ./...` to ensure all existing tests pass
- [X] Task 4.2: Run `go test -race ./...` for race condition detection
- [ ] Task 4.3: Run `golangci-lint run ./...` for static analysis
- [ ] Task 4.4: Manual validation: `wave chat --list`, `wave chat <run-id>`, verify bidirectional conversation works
