# Implementation Plan: Fix Forge Template Variables in Display Persona Labels

## Objective

Resolve `{{ forge.* }}` template variables in persona names before they reach the display/progress layer, so that TUI, verbose CLI output, and WebUI all show resolved persona labels (e.g., `github-analyst`) instead of raw template syntax.

## Approach

Two-pronged fix:

1. **Source fix**: Resolve persona names at display registration time by calling `forge.DetectFromGitRemotes()` and substituting `{{ forge.type }}` (and other forge variables) in persona names before passing them to `AddStep`.

2. **Defense-in-depth fix**: Update display event handlers to refresh the stored persona from incoming "started"/"running" events, which already contain the resolved persona name from the executor.

The source fix ensures "not started" steps display correctly from the beginning. The defense fix handles edge cases and ensures the display always reflects the latest resolved persona.

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `cmd/wave/commands/output.go` | modify | Resolve forge variables in persona names before `AddStep` calls |
| `internal/display/bubbletea_progress.go` | modify | Update persona from "started"/"running" events in `updateFromEvent` |
| `internal/display/progress.go` | modify | Update persona from "started"/"running" events in `EmitProgress` |
| `internal/webui/handlers_runs.go` | modify | Resolve persona names for DAG display and step detail |
| `internal/webui/handlers_compose.go` | modify | Resolve persona in compose step view |

## Architecture Decisions

1. **Forge detection at emitter creation**: Call `forge.DetectFromGitRemotes()` in `CreateEmitter`/`createAutoEmitter`. This is lightweight (reads git remote URL) and already called in the executor. The result is used to build a simple string replacer for persona names.

2. **Simple string replacement**: Rather than building a full `PipelineContext`, use a targeted replacement of `{{ forge.type }}` (and `{{ forge.type }}` without spaces) in persona names. This avoids coupling the display layer to the full pipeline context system.

3. **Event-based persona refresh**: The display handlers already receive the resolved persona in `ev.Persona`. Adding a single line to update `step.Persona` when a non-empty `ev.Persona` arrives is minimal and safe.

4. **WebUI uses same forge detection**: The webui handlers read from the manifest which has unresolved step personas. Apply the same forge variable resolution when building response DTOs.

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| `forge.DetectFromGitRemotes()` fails (no git remote) | Low | Falls back to empty string, which strips template vars — acceptable degradation |
| Performance of extra `DetectFromGitRemotes` call | Negligible | Already called in executor; git remote parsing is fast |
| Persona names with other template variables | N/A | Only `{{ forge.* }}` variables appear in persona names per pipeline analysis |

## Testing Strategy

1. **Unit tests for display persona update**: Verify that `BubbleTeaProgressDisplay` and `ProgressDisplay` update persona when receiving "started"/"running" events with a resolved persona.

2. **Unit tests for output.go persona resolution**: Verify that `createAutoEmitter` resolves `{{ forge.type }}` in persona names before registration.

3. **Existing tests**: Run `go test ./internal/display/... ./cmd/wave/commands/... ./internal/webui/...` to ensure no regressions.
