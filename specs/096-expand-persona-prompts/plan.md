# Implementation Plan: Expand Persona Definitions with Detailed System Prompts

**Branch**: `096-expand-persona-prompts` | **Date**: 2026-02-13 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/096-expand-persona-prompts/spec.md`

## Summary

Refine the 13 already-expanded Wave persona definitions to fix language-specific references (FR-008 violations in 4 files: `craftsman.md`, `reviewer.md`, `auditor.md`, `debugger.md`) and sync all 13 persona files from `.wave/personas/` to `internal/defaults/personas/` for byte-identical parity (FR-010). This is a content-only change — no Go source code, `wave.yaml`, or JSON schemas are modified.

## Technical Context

**Language/Version**: Go 1.25+ (project language, but this feature modifies only Markdown files)
**Primary Dependencies**: None — content-only change to `.md` files
**Storage**: Filesystem (Markdown files in two directories)
**Testing**: `go test ./...` for regression validation (no new tests needed)
**Target Platform**: N/A — persona files are platform-independent content
**Project Type**: Single Go binary project
**Performance Goals**: N/A — no runtime performance impact
**Constraints**: Each persona file must be 30-200 lines; zero language-specific toolchain references
**Scale/Scope**: 13 persona files × 2 locations = 26 files total; 4 files need FR-008 fixes, 13 files need parity sync

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No binary changes; Markdown files embedded via existing `//go:embed` |
| P2: Manifest as SSOT | PASS | No `wave.yaml` changes; persona loading unchanged |
| P3: Persona-Scoped Execution | PASS | Persona content improves behavioral clarity; execution boundaries unchanged |
| P4: Fresh Memory | PASS | Expanded personas are designed for fresh-memory contexts (self-contained) |
| P5: Navigator-First | PASS | Navigator persona expanded like all others; no architectural change |
| P6: Contracts at Handover | PASS | No contract changes; output format sections note contract schema precedence |
| P7: Relay via Summarizer | PASS | Summarizer persona expanded; relay mechanism unchanged |
| P8: Ephemeral Workspaces | PASS | No workspace mechanism changes |
| P9: Credentials Never Touch Disk | PASS | No credential handling changes |
| P10: Observable Progress | PASS | No event system changes |
| P11: Bounded Recursion | PASS | No recursion or resource limit changes |
| P12: Minimal Step State Machine | PASS | No state machine changes |
| P13: Test Ownership | PASS | `go test ./...` will be run after changes; content-only changes should not break tests |

**Result**: All 13 principles pass. No constitutional violations.

## Project Structure

### Documentation (this feature)

```
specs/096-expand-persona-prompts/
├── plan.md              # This file
├── research.md          # Phase 0 output — current state analysis and FR-008 violations
├── data-model.md        # Phase 1 output — persona file structure and validation rules
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Files Modified (repository root)

```
.wave/personas/
├── craftsman.md          # FR-008 fix: remove Go-specific references
├── reviewer.md           # FR-008 fix: remove go test/npm test references
├── auditor.md            # FR-008 fix: remove Go-specific identity and tools
└── debugger.md           # FR-008 fix: remove Go-specific identity and tools

internal/defaults/personas/
├── navigator.md          # Parity sync from .wave/personas/
├── philosopher.md        # Parity sync from .wave/personas/
├── planner.md            # Parity sync from .wave/personas/
├── craftsman.md          # Parity sync from .wave/personas/ (after FR-008 fix)
├── implementer.md        # Parity sync from .wave/personas/
├── reviewer.md           # Parity sync from .wave/personas/ (after FR-008 fix)
├── auditor.md            # Parity sync from .wave/personas/ (after FR-008 fix)
├── debugger.md           # Parity sync from .wave/personas/ (after FR-008 fix)
├── researcher.md         # Parity sync from .wave/personas/
├── summarizer.md         # Parity sync from .wave/personas/
├── github-analyst.md     # Parity sync from .wave/personas/
├── github-commenter.md   # Parity sync from .wave/personas/
└── github-enhancer.md    # Parity sync from .wave/personas/
```

**Structure Decision**: No new directories or files are created. This feature modifies existing Markdown files in two existing directories. The `.wave/personas/` directory is the canonical source; `internal/defaults/personas/` is the sync target.

## Implementation Tasks

### Task 1: Fix FR-008 Violations in craftsman.md

Edit `.wave/personas/craftsman.md`:
- Line 12: `Go conventions including effective Go practices, formatting, and idiomatic patterns` → `Language conventions and idiomatic patterns for the target codebase`
- Line 46: `go test, go build, go vet, etc.` → `build, test, and static analysis commands for the project's toolchain`

### Task 2: Fix FR-008 Violations in reviewer.md

Edit `.wave/personas/reviewer.md`:
- Line 35: `` Run available tests (`go test`, `npm test`) to verify passing state `` → `Run the project's test suite to verify passing state`
- Lines 46-47: Replace `Bash(go test*)` and `Bash(npm test*)` with language-agnostic description: `Bash(...)`: Run the project's test suite to validate implementation behavior

### Task 3: Fix FR-008 Violations in auditor.md

Edit `.wave/personas/auditor.md`:
- Lines 2-3: `specializing in Go systems` → `specializing in software systems`
- Line 16: `Go-specific security concerns: unsafe pointer usage, race conditions, path traversal` → `Language-specific security concerns: memory safety, race conditions, path traversal, type confusion`
- Line 33: `` Run static analysis tools (`go vet`) `` → `Run static analysis tools available in the project's toolchain`
- Lines 43-44: Replace `Bash(go vet*)` and `Bash(npm audit*)` with language-agnostic tool descriptions

### Task 4: Fix FR-008 Violations in debugger.md

Edit `.wave/personas/debugger.md`:
- Lines 2-3: `specializing in Go systems` → `specializing in software systems`
- Line 9: `concurrent Go programs` → `concurrent programs`
- Line 13: `Go-specific debugging: goroutine leaks, race conditions, deadlocks, channel misuse` → `Concurrency debugging: race conditions, deadlocks, resource leaks, and synchronization issues`
- Line 51: Replace `Bash(go test*)` with language-agnostic test description

### Task 5: Sync All 13 Persona Files to internal/defaults/personas/

Copy each file from `.wave/personas/{name}.md` to `internal/defaults/personas/{name}.md` for all 13 personas. Validate with `diff -r .wave/personas/ internal/defaults/personas/` — must produce zero differences.

### Task 6: Validate All Requirements

Run validation checks:
1. `wc -l` on all 26 persona files — each must be ≥30 and ≤200 lines
2. Grep for language-specific toolchain references — must find zero matches
3. Verify all 7 structural concepts present in each file
4. `diff -r .wave/personas/ internal/defaults/personas/` — zero differences
5. `go test ./...` — zero failures
6. Confirm no `.go`, `wave.yaml`, or `.json` schema files were modified

## Complexity Tracking

_No constitutional violations to justify._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| (none) | — | — |
