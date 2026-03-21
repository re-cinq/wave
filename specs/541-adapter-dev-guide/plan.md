# Implementation Plan: Adapter Development Guide

## Objective

Create a developer-facing guide (`docs/guides/adapter-development.md`) that explains how to build a new first-class adapter in Go, covering the `AdapterRunner` interface, streaming events, sandbox integration, and testing patterns.

## Approach

Write a single markdown document structured as a step-by-step guide. Use the existing Claude, OpenCode, and Browser adapters as reference implementations. Include a complete skeleton adapter example that developers can copy and modify.

## File Mapping

| Path | Action | Description |
|------|--------|-------------|
| `docs/guides/adapter-development.md` | create | New guide document |

## Architecture Decisions

1. **Single file**: One comprehensive guide rather than splitting across multiple files. The topic is focused enough for a single document.
2. **Skeleton-first approach**: Lead with a complete minimal adapter implementation, then explain each integration point in detail. Developers can get something working fast, then refine.
3. **Reference real code**: Point to actual source files (`internal/adapter/adapter.go`, `claude.go`, `opencode.go`) rather than abstracting away implementation details. This keeps the guide grounded and verifiable.
4. **No code changes**: This is purely documentation — no Go source modifications needed.

## Risks

| Risk | Mitigation |
|------|------------|
| Guide becomes stale as adapter interface evolves | Reference interface definition by file path so readers can verify against current code |
| Overlap with existing docs | Focus exclusively on the Go implementation perspective; link to existing docs for manifest configuration and concepts |

## Testing Strategy

No automated tests needed — this is a documentation-only change. Validation is:
- Markdown renders correctly
- Code examples are syntactically valid Go
- All referenced file paths exist in the codebase
- Cross-links to other docs are correct relative paths
