# Implementation Plan: Release-Gated Pipeline Embedding

**Branch**: `029-release-gated-embedding` | **Date**: 2026-02-11 | **Spec**: `specs/029-release-gated-embedding/spec.md`
**Input**: Feature specification from `/specs/029-release-gated-embedding/spec.md`

## Summary

Add a `metadata.release` boolean field to pipeline definitions that controls whether a pipeline is distributed to end users via `wave init`. By default, `release` is `false` (explicit opt-in). The `wave init` command filters embedded pipelines to only extract `release: true` pipelines, along with their transitively referenced contracts and prompts. A `--all` flag bypasses filtering for contributors. Personas are never transitively excluded.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `gopkg.in/yaml.v3`, `github.com/spf13/cobra`
**Storage**: N/A (embedded filesystem via `go:embed`, no persistence changes)
**Testing**: `go test ./...` with table-driven tests
**Target Platform**: Linux/macOS/Windows (single binary)
**Project Type**: Single Go project
**Performance Goals**: N/A (init is a one-time command, no performance-critical path)
**Constraints**: Single static binary, no runtime dependencies
**Scale/Scope**: 3 files modified, ~150 lines of new code, ~50 lines modified

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new dependencies. All filtering uses existing `go:embed` FS. |
| P2: Manifest as SSOT | PASS | `release` field lives in pipeline YAML (part of the manifest ecosystem), not a separate config file. |
| P3: Persona-Scoped Execution | N/A | Feature does not affect execution boundaries. |
| P4: Fresh Memory | N/A | Feature does not affect pipeline execution. |
| P5: Navigator-First | N/A | Feature does not affect pipeline step ordering. |
| P6: Contracts at Handovers | N/A | Feature does not change contract validation. |
| P7: Relay | N/A | Feature does not affect relay/compaction. |
| P8: Ephemeral Workspaces | N/A | Feature does not affect workspace management. |
| P9: Credentials | PASS | No credential handling introduced. |
| P10: Observable Progress | PASS | Warning emitted when zero release pipelines found. `printInitSuccess` shows accurate filtered counts. |
| P11: Bounded Recursion | N/A | No recursion introduced. |
| P12: Step State Machine | N/A | No state machine changes. |
| P13: Test Ownership | PASS | Existing tests must pass. New tests added for release filtering, transitive exclusion, and `--all` flag behavior. |

**Result**: All applicable principles pass. No violations.

## Project Structure

### Documentation (this feature)

```
specs/029-release-gated-embedding/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── defaults-api.md
│   ├── init-filtering.md
│   └── types-extension.md
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
internal/
├── pipeline/
│   └── types.go              # Add Release, Disabled to PipelineMetadata
├── defaults/
│   ├── embed.go              # Add GetReleasePipelines(), ReleasePipelineNames()
│   └── embed_test.go         # Add release filtering tests

cmd/wave/commands/
└── init.go                   # Add --all flag, release filtering, transitive exclusion,
                              #   update printInitSuccess display
```

**Structure Decision**: This feature modifies existing files in the established Go project structure. No new packages or directories needed (apart from the contracts directory in the spec folder). All changes are confined to 3 existing Go files and their tests.

## Complexity Tracking

_No constitution violations. Table intentionally empty._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| (none)    |           |                                      |
