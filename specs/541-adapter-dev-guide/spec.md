# docs: add adapter development guide

**Issue**: [#541](https://github.com/re-cinq/wave/issues/541)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: none

## Problem Statement

No documentation exists for building a new first-class adapter. Adding a new adapter requires implementing `AdapterRunner` and adding a case to `ResolveAdapter()` in `opencode.go`.

The existing `docs/examples/custom-adapter.md` only covers the `ProcessGroupRunner` fallback — wrapping arbitrary CLIs via manifest configuration. It does not explain how to add a native Go adapter to the Wave codebase.

## Requirements

Create a "Building a new adapter" guide covering:

1. **The AdapterRunner interface** — the single `Run(ctx, cfg) (*AdapterResult, error)` method, `AdapterRunConfig` fields, and `AdapterResult` structure
2. **Streaming events** — emitting `StreamEvent` via `cfg.OnStreamEvent`, event types (`tool_use`, `tool_result`, `text`, `result`, `system`), and real-time progress reporting
3. **Sandbox integration** — how adapters interact with `internal/sandbox`, Docker sandbox wrapping, domain filtering, curated environment via `BuildCuratedEnvironment()`
4. **Testing patterns** — `MockAdapter` with functional options, `configCapturingAdapter` pattern, integration tests with `ProcessGroupRunner`

## Acceptance Criteria

- [ ] New guide at `docs/guides/adapter-development.md`
- [ ] Covers all four sections listed in the issue
- [ ] References actual source files and interfaces from the codebase
- [ ] Includes code examples showing a skeleton adapter implementation
- [ ] Cross-links to existing adapter docs (`concepts/adapters.md`, `reference/adapters.md`, `examples/custom-adapter.md`)
- [ ] Does not duplicate content from existing docs — focuses on the code-level "how to build" perspective
