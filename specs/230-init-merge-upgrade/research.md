# Research: Init Merge & Upgrade Workflow

**Feature**: #230 — Init Merge & Upgrade Workflow  
**Date**: 2026-03-04

## Decision 1: Change Summary Display Format

**Decision**: Use a tabular format with ANSI coloring, consistent with Wave's existing TUI patterns (`internal/tui/` and `internal/display/`).

**Rationale**: Wave already uses `tui.WaveLogo()` and formatted output in `printInitSuccess` and `printMergeSuccess` (init.go:473-527). A table with columns `[Status] [Category] [Path]` matches the existing style and is scannable. Categories: "new" (green), "preserved" (yellow), "up to date" (dim/gray).

**Alternatives Rejected**:
- JSON output only: Not human-friendly for interactive use. Could be added later as `--json` flag.
- Tree view: More complex to implement, harder to scan for decision-making.
- Simple list: Loses category distinction needed for user decision.

## Decision 2: File Comparison Strategy

**Decision**: Use `bytes.Equal` for byte-for-byte content comparison between existing files and `go:embed` defaults.

**Rationale**: Per clarification C-5, byte-for-byte comparison is deterministic, simple, and avoids false positives. The embedded defaults (`internal/defaults/embed.go`) are always available via `go:embed`, making the comparison source stable.

**Alternatives Rejected**:
- Content hashing (SHA256): Adds complexity for the same result — would need to hash both sides and compare.
- Normalized comparison (strip whitespace): Violates spec requirement for exact byte match; any intentional whitespace change should count as user-modified.

## Decision 3: Manifest Merge Approach

**Decision**: Preserve the existing `mergeMaps` recursive approach (init.go:449-471) with atomic array handling per clarification C-2. Add a diff-tracking layer that records what changed during the merge.

**Rationale**: The existing `mergeMaps` implementation already handles the deep-merge correctly — user values take precedence, new default keys are added. The only addition needed is tracking _what_ changed for the summary display.

**Alternatives Rejected**:
- Replace with a YAML diff library: Would add an external dependency, violating Principle 1 (minimal dependencies). The merge logic is simple enough to track changes inline.
- Two-pass merge (diff then apply): Unnecessarily complex. A single pass that records changes is sufficient.

## Decision 4: Confirmation Prompt Architecture

**Decision**: Use a pre-mutation confirmation pattern — compute all changes first, display summary, then prompt. Only write files after confirmation.

**Rationale**: This is the core safety requirement from FR-001/FR-002. The current `runMerge` writes files immediately. The new flow must: (1) compute change summary, (2) display it, (3) prompt if interactive, (4) write only on confirmation.

**Implementation Pattern**:
```
computeChangeSummary() → ChangeSummary struct
displayChangeSummary(summary)
if needsConfirmation(opts) { confirmOrAbort() }
applyChanges(summary)
```

## Decision 5: Non-Interactive Terminal Detection

**Decision**: Reuse the existing `isInitInteractive()` function (init.go:839-843) which checks `term.IsTerminal` and `WAVE_FORCE_TTY` env var.

**Rationale**: Consistent with existing behavior. Per FR-014 and edge case 4, non-interactive terminals must require `--yes` or `--force` to proceed.

## Decision 6: Post-Merge Migration Guidance

**Decision**: Per clarification C-4, add `wave migrate up` to the "Next steps" section in `printMergeSuccess`.

**Rationale**: The current `printMergeSuccess` (init.go:509-527) already has a "Next steps" section. Adding migration guidance follows the existing pattern.

## Decision 7: Integration Test Strategy

**Decision**: Use table-driven tests in `cmd/wave/commands/init_test.go` covering the full upgrade lifecycle. Reuse the existing `testEnv` pattern (init_test.go:20-50).

**Rationale**: The existing test infrastructure (`newTestEnv`, `executeInitCmd`, `readYAML`) provides isolation and cleanup. The lifecycle test will: init → write custom files → init --merge → verify preservation and additions → verify summary output.

## Decision 8: `--yes` Flag Semantic in Merge Context

**Decision**: Per clarification C-3, `--yes` and `--force` are functionally identical in merge context — both skip the confirmation prompt while retaining merge-safe behavior. The `--merge` flag is the primary mode selector.

**Rationale**: `--merge` constrains the operation to merge behavior. When `--merge` is present, `--force` does not switch to overwrite-all semantics. This matches the existing code path where `opts.Merge` is checked before `opts.Force` (init.go:248-249).
