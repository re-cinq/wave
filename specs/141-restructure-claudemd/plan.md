# Implementation Plan: Restructure CLAUDE.md

## Objective

Restructure the project root `CLAUDE.md` (285 lines) to reduce noise by 30%+, surface critical directives early, and add missing documentation about Wave's core runtime behavior (pipeline steps, contracts, artifacts).

## Approach

This is a **documentation-only change** to a single file (`CLAUDE.md`). No Go code changes are required. The strategy is:

1. **Audit** every section of the current CLAUDE.md against its value-to-length ratio
2. **Promote** critical directives (constraints, security model, test ownership) to the top
3. **Add** a concise "How Wave Works at Runtime" section covering pipelines, contracts, and artifacts
4. **Condense** verbose recipe-style sections (Common Tasks, Database Migrations, Adding New X) into brief references to source files
5. **Remove** information that duplicates what's in code comments or is trivially discoverable
6. **Preserve** structural markers (`MANUAL ADDITIONS`) and auto-updated sections (`Recent Changes`)

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `CLAUDE.md` | modify | Restructure, trim, and add runtime documentation |

No other files are created, modified, or deleted.

## Section-by-Section Analysis

### Current Structure (285 lines)

| Section | Lines | Verdict |
|---------|-------|---------|
| Wave Development Guidelines (header + overview) | ~10 | **Keep** — condense slightly |
| Architecture Principles | ~25 | **Keep** — high signal |
| Development Guidelines > Code Standards | ~8 | **Keep** — condense to bullet list |
| Development Guidelines > Critical Constraints | ~10 | **Promote** — move to top of file |
| File Structure | ~20 | **Keep** — useful reference |
| Key Implementation Patterns | ~15 | **Replace** — fold into new Runtime section |
| Testing Requirements | ~8 | **Keep** — condense |
| Test Ownership | ~12 | **Promote** — move to top alongside constraints |
| Constitutional Compliance | ~8 | **Keep** — merge into constraints |
| Security Considerations (~40 lines) | ~40 | **Condense** — remove sub-sections, keep essentials |
| Common Tasks (~25 lines) | ~25 | **Remove** — reference `cmd/wave/commands/` and `internal/contract/` instead |
| Performance Considerations | ~5 | **Remove** — generic, not actionable |
| Database Migrations (~25 lines) | ~25 | **Condense** — single line referencing `docs/migrations.md` |
| Testing (CLI commands) | ~12 | **Condense** — keep only `go test ./...` and `go test -race ./...` |
| Code Style | ~6 | **Keep** — already concise |
| Git Commits | ~6 | **Keep** — critical for AI agents |
| Versioning (~20 lines) | ~20 | **Condense** — keep table, remove prose |
| Debugging | ~5 | **Keep** — useful |
| Recent Changes | ~5 | **Keep** — auto-updated |
| Manual Additions | ~2 | **Keep** — structural markers |

### Proposed New Structure (~190 lines target)

```
# Wave Development Guidelines
  (2-sentence overview)

## Critical Constraints               ← PROMOTED to top
  (single static binary, test ownership, security first, constitutional compliance)

## How Wave Works at Runtime           ← NEW section
  Pipeline execution → workspace → persona → contract
  Artifact injection for inter-step communication
  Runtime CLAUDE.md assembly (base-protocol + persona + contract + restrictions)

## Architecture
  Active technologies, core components, security model

## File Structure
  (existing tree, trimmed)

## Security
  (condensed: key principles only, no sub-headed recipes)

## Development
  Code standards, testing (condensed CLI commands), code style

## Git & Versioning
  Commits, conventional prefixes, semver table

## Debugging
  (existing, compact)

## Recent Changes
  (preserved as-is)

## Manual Additions
  (preserved as-is)
```

## Architecture Decisions

1. **No code changes**: The issue is about the project root CLAUDE.md, not the runtime-generated per-step CLAUDE.md. The runtime assembly in `internal/adapter/claude.go` is unaffected.

2. **Critical directives first**: Agents need to see constraints, test ownership, and security rules immediately. Currently these are buried ~60 lines in.

3. **New "How Wave Works at Runtime" section**: This addresses the missing core concepts (pipeline environments, contracts, artifacts). It will be ~20 lines of high-density content, not a tutorial.

4. **Reference over duplication**: Instead of documenting migration CLI commands or "how to add a new contract type" in CLAUDE.md, reference the source files. This prevents staleness.

5. **Preserve auto-updated sections**: The `Recent Changes` section and `MANUAL ADDITIONS` markers are maintained exactly as-is for pipeline compatibility.

## Risks

| Risk | Mitigation |
|------|------------|
| Removing too much context causes agents to miss important patterns | Keep all critical constraints and security model intact; only remove recipe-style instructions |
| New runtime section becomes stale as code evolves | Write it at the conceptual level (not implementation detail); reference source files for specifics |
| Breaking pipeline-injected content in MANUAL ADDITIONS section | Preserve markers and any content between them exactly |
| Tests referencing CLAUDE.md content | Check for any tests that parse CLAUDE.md content (unlikely but verify) |

## Testing Strategy

1. **Line count verification**: `wc -l CLAUDE.md` should show ≤200 lines (down from 285)
2. **Content audit**: Verify all acceptance criteria sections are present
3. **Test suite**: Run `go test ./...` to confirm no regressions
4. **Marker preservation**: Verify `MANUAL ADDITIONS` markers are intact
5. **Grep for removed content**: Confirm no critical directives were accidentally removed
