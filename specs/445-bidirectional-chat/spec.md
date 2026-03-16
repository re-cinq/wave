# feat: bidirectional chat with pipeline context and artifacts

**Issue**: [#445](https://github.com/re-cinq/wave/issues/445)
**Labels**: enhancement
**Author**: nextlevelshit

## Description

Issue #276 was closed but only one-way context injection exists (`internal/pipeline/chatworkspace.go`, ~90 lines). Artifacts are injected into the workspace but there is no interactive chat interface.

### What exists
- `chatworkspace.go`: builds workspace with artifact content, suggested questions, focus areas
- `stepcontroller.go`: has `ContinueStep`, `ExtendStep`, `RevertStep` but only in test code

### What's missing
- No `wave chat <run-id>` CLI command that opens an interactive session with injected pipeline context
- No TUI chat view that lets you discuss artifacts with a persona
- No way to ask follow-up questions about pipeline results

### Minimum viable
- `wave chat <run-id> [--step <name>]` that launches Claude with pipeline artifacts pre-loaded

## Current State Analysis

The `wave chat` command already exists (`cmd/wave/commands/chat.go`) with:
- Run ID resolution (latest or explicit)
- `--step`, `--artifact`, `--model`, `--prompt`, `--list` flags
- `--continue`, `--extend`, `--rewrite` flags (Phase 2 manipulation)
- Chat context building from state store
- Workspace preparation with CLAUDE.md and settings.json
- Interactive Claude session launch via `adapter.LaunchInteractive`

The issue comments confirm prior implementations were partial — "bidirectional chat not implemented." The key gap is that the current chat is one-way: context is injected but the user cannot send follow-up questions or have a multi-turn conversation that feeds back into the pipeline context.

## Acceptance Criteria

1. `wave chat <run-id>` opens an interactive multi-turn session with pipeline context pre-loaded
2. `wave chat --step <name>` scopes the session to a specific step's context
3. The session supports follow-up questions about pipeline results and artifacts
4. The session allows the user to ask clarifying questions and get contextual answers
5. Chat session can optionally persist conversation history for resume (`--resume`)
6. Bidirectional: user prompts flow in, responses flow out, and the session maintains conversational context across turns
