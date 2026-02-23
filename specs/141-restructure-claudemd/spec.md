# Restructure CLAUDE.md: reduce noise, add core pipeline/contract documentation

**Feature Branch**: `141-restructure-claudemd`
**Issue**: [#141](https://github.com/re-cinq/wave/issues/141)
**Labels**: documentation, enhancement
**Author**: nextlevelshit
**Status**: Draft

## Problem

The project `CLAUDE.md` (285 lines) is a critical component of Wave's agent protocol — it configures every AI persona's behavior at runtime. However, the current file has grown too long and contains low-signal content that dilutes the important instructions. This makes it harder for agents to follow key directives and increases the risk of content becoming outdated.

Additionally, core Wave functionality is not documented in CLAUDE.md, leaving agents without essential context about how the system works.

## Current Issues

- **Too much noise**: The file contains verbose sections that could be condensed without losing meaning
- **Low signal-to-noise ratio**: Critical keywords and directives are buried in walls of text
- **Staleness risk**: The structure makes it easy for sections to become outdated as Wave evolves
- **Missing core concepts**: Key runtime behaviors are not explained, including:
  - Pipeline step environments (ephemeral worktrees with isolated context)
  - Contract injection and validation at step boundaries
  - Artifact injection for inter-step communication
  - Runtime prompt generation from manifest + persona + contract schemas

## Acceptance Criteria

- [ ] CLAUDE.md is shorter overall (target: reduce by 30%+ in line count, from 285 to ~200 or fewer)
- [ ] Every section has a clear, scannable heading
- [ ] Core pipeline/contract/artifact runtime behavior is documented
- [ ] No duplicated information that could become stale — references to canonical sources where appropriate
- [ ] Agent personas can discover critical directives within the first screen of content
- [ ] Existing test suite passes (`go test ./...`) after any related code changes
- [ ] MANUAL ADDITIONS markers are preserved for user-added content

## Context

### Runtime CLAUDE.md Assembly

The project root `CLAUDE.md` is read by Claude Code as project instructions. It is NOT the same as the runtime-generated per-step CLAUDE.md. The runtime CLAUDE.md is assembled by `prepareWorkspace()` in `internal/adapter/claude.go:213-309` from:

1. Base protocol preamble (`.wave/personas/base-protocol.md`)
2. Persona system prompt
3. Contract compliance section (auto-generated from step contract)
4. Restriction section (derived from manifest permissions)

The project root CLAUDE.md serves a different purpose: it provides development guidelines to agents working directly in the Wave codebase (not through pipeline steps).

### Key Source Files

- `CLAUDE.md` — the file to restructure (285 lines)
- `internal/pipeline/executor.go` — pipeline execution, workspace creation, artifact injection, contract validation
- `internal/pipeline/context.go` — template variable resolution, pipeline context
- `internal/contract/contract.go` — contract validation types and interfaces
- `internal/workspace/workspace.go` — ephemeral workspace management
- `internal/adapter/claude.go` — runtime CLAUDE.md assembly, workspace preparation

## Requirements

### Functional Requirements

- **FR-001**: Restructured CLAUDE.md MUST be 30%+ shorter in line count than the current 285 lines
- **FR-002**: Critical directives (constraints, test ownership, security model) MUST appear in the first ~50 lines
- **FR-003**: New section documenting core runtime behavior MUST be added (pipeline steps, contracts, artifacts)
- **FR-004**: Verbose recipe-style sections (Common Tasks, Database Migrations) MUST be condensed or reference source files
- **FR-005**: `<!-- MANUAL ADDITIONS START -->` / `<!-- MANUAL ADDITIONS END -->` markers MUST be preserved
- **FR-006**: Recent Changes section MUST be preserved (it is auto-updated by pipelines)
- **FR-007**: Information that duplicates source code comments or is easily discoverable MUST be removed

### Non-Functional Requirements

- **NFR-001**: No Go code changes required — this is a documentation-only change
- **NFR-002**: All existing tests MUST pass after the change
- **NFR-003**: The restructured content MUST be readable without external context

## Success Criteria

- **SC-001**: CLAUDE.md line count drops from 285 to ≤200 lines
- **SC-002**: A new developer-facing section about pipeline/contract/artifact runtime is present
- **SC-003**: `go test ./...` passes with no regressions
