# Implementation Plan: Bidirectional Chat

## Objective

Make `wave chat` a true bidirectional interactive session where the user can have multi-turn conversations about pipeline results, artifacts, and step outputs. The current implementation launches Claude with context but lacks resume support and proper session tracking.

## Approach

The core infrastructure already exists — `chat.go` builds context, prepares a workspace, and launches Claude interactively. The gap is:

1. **Session resume**: Claude Code supports `--resume <session-id>` but Wave doesn't track or expose this. Adding `--resume` support lets users continue previous chat sessions.
2. **Session tracking**: Record chat sessions in the state store so users can list and resume them.
3. **Prompt piping**: The `--prompt` flag exists but only works as a one-shot. For bidirectional flow, the interactive session (stdin/stdout passthrough) already handles this — Claude Code's native interactive mode IS bidirectional.
4. **Session output capture**: After a chat session ends, capture and persist the session ID returned by Claude Code so it can be resumed later.

### Key Insight

Claude Code's interactive mode (`adapter.LaunchInteractive`) already provides bidirectional chat — stdin flows to Claude, stdout flows to the user. The issue title's "bidirectional" requirement is about making this work reliably with pipeline context AND supporting session persistence for resume.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/adapter/interactive.go` | modify | Add session capture and return session ID from LaunchInteractive |
| `internal/adapter/interactive_test.go` | modify | Test session ID extraction |
| `cmd/wave/commands/chat.go` | modify | Add `--resume` flag, session save/load, session listing |
| `cmd/wave/commands/chat_test.go` | modify | Test resume flag and session management |
| `internal/state/chat_session.go` | create | Chat session record type and state store methods |
| `internal/state/chat_session_test.go` | create | Tests for session persistence |
| `internal/state/store.go` | modify | Add chat session methods to StateStore interface |
| `internal/state/sqlite.go` | modify | Implement chat session SQL methods |
| `internal/state/migrations.go` | modify | Add chat_session table migration |

## Architecture Decisions

1. **Session ID from Claude Code**: Claude Code returns a session ID that can be passed back via `--resume`. We capture this from the process output.
2. **State store persistence**: Chat sessions are stored in SQLite alongside pipeline runs, linking session → run_id for easy discovery.
3. **No TUI chat view in MVP**: The issue mentions a TUI chat view, but the MVP scope is CLI-only. TUI integration can follow.
4. **Workspace reuse on resume**: When resuming, reuse the same workspace (CLAUDE.md + settings.json) from the original session.

## Risks

| Risk | Mitigation |
|------|-----------|
| Claude Code session ID format may change | Parse defensively, fall back to no-resume if ID not captured |
| Workspace cleanup may delete resumed sessions | Only clean workspaces when explicitly requested |
| State migration on existing databases | Use IF NOT EXISTS for table creation |

## Testing Strategy

- Unit tests for session record types and state store methods
- Unit tests for session ID extraction from Claude Code output
- Unit tests for resume flag integration in chat command
- Integration test: full chat session lifecycle (create → list → resume)
- Existing chat tests must continue passing
