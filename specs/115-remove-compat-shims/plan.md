# Implementation Plan: Remove Backwards-Compatibility Shims

**Branch**: `115-remove-compat-shims` | **Date**: 2026-02-20 | **Spec**: `specs/115-remove-compat-shims/spec.md`
**Input**: Feature specification from `/specs/115-remove-compat-shims/spec.md`

## Summary

Remove all backwards-compatibility shims from the Wave codebase as identified in issue #115. This is a pure cleanup operation: remove deprecated fields, collapse dual-path code to single paths, delete dead fallback functions, and update stale comments. The net result is fewer lines of code, no dual-path conditionals in `contract`, `pipeline`, and `state` packages, and clearer documentation of current behavior. Zero new features, zero behavioral changes for current consumers.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `gopkg.in/yaml.v3`, `github.com/spf13/cobra`, `modernc.org/sqlite`, `github.com/santhosh-tekuri/jsonschema/v6`
**Storage**: SQLite (via migration system)
**Testing**: `go test -race ./...`, `go vet ./...`
**Target Platform**: Linux (single static binary)
**Project Type**: Single Go project (CLI tool)
**Performance Goals**: N/A (no performance-sensitive changes)
**Constraints**: Single static binary, all tests must pass
**Scale/Scope**: ~15 files modified, ~1 file deleted, net negative LOC

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new dependencies, no runtime changes |
| P2: Manifest as SSOT | PASS | No manifest changes required |
| P3: Persona-Scoped Execution | PASS | No persona changes |
| P4: Fresh Memory at Boundaries | PASS | No memory model changes |
| P5: Navigator-First | PASS | No pipeline structure changes |
| P6: Contracts at Handovers | PASS | Contract validation behavior preserved, only internal field consolidation |
| P7: Relay via Summarizer | PASS | No relay changes |
| P8: Ephemeral Workspaces | PASS | No workspace model changes |
| P9: Credentials Never Touch Disk | PASS | No credential handling changes |
| P10: Observable Progress | PASS | No event model changes |
| P11: Bounded Recursion | PASS | No recursion changes |
| P12: Minimal Step State Machine | PASS | No state machine changes |
| P13: Test Ownership | PASS | All test updates owned by this change; `go test -race ./...` required before merge |

**Post-Phase 1 Re-check**: All gates still PASS. The only behavioral changes are:
- `WAVE_MIGRATION_ENABLED=false` now errors instead of falling back (improves P10 observability)
- `wave migrate down` always fails with a clear error (simplifies P12 state machine — no unintended rollback path)
- Missing `--- PIPELINE ---` marker in meta-pipeline output now errors (strengthens P6 contract enforcement)

## Project Structure

### Documentation (this feature)

```
specs/115-remove-compat-shims/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 codebase investigation
├── data-model.md        # Entity changes documentation
├── checklists/
│   └── requirements.md  # Requirements checklist
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```
internal/
├── contract/
│   ├── contract.go          # Remove StrictMode field from ContractConfig
│   ├── contract_test.go     # Update StrictMode → MustPass in test fixtures
│   ├── json_cleaner.go      # Remove extractJSONFromTextLegacy()
│   ├── jsonschema.go        # Replace StrictMode checks with MustPass
│   ├── typescript.go        # Remove IsTypeScriptAvailable(), replace StrictMode → MustPass
│   └── typescript_test.go   # Update StrictMode → MustPass, remove IsTypeScriptAvailable test
├── pipeline/
│   ├── context.go           # Update "legacy template variables" comment
│   ├── executor.go          # Remove StrictMode assignment/check, update legacy comment
│   ├── meta.go              # Remove extractYAMLLegacy()
│   ├── resume.go            # Remove legacy exact-name directory lookup
│   └── types.go             # Update WorkspaceConfig.Type comment
├── state/
│   ├── migration_config.go  # (callsite update for error path)
│   ├── migration_definitions.go  # Empty all Down SQL fields
│   ├── schema.sql           # DELETE FILE
│   └── store.go             # Remove schema.sql fallback, embed directive, embed import
└── worktree/
    └── worktree.go          # Update "legacy behavior" comment

cmd/wave/commands/
└── migrate.go               # Update migrate down help text, remove confirmation prompt
```

**Structure Decision**: All changes are within the existing Go project structure. No new files created (except spec documentation). One file deleted (`internal/state/schema.sql`).

## Complexity Tracking

_No constitution violations. Table left empty._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| (none)    | —         | —                                    |
