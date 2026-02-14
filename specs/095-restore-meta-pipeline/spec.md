# feat: restore and stabilize `wave meta` dynamic pipeline generation

**Issue**: [#95](https://github.com/re-cinq/wave/issues/95)
**Labels**: enhancement, needs-design, priority: medium
**Author**: nextlevelshit
**Feature Branch**: `095-restore-meta-pipeline`
**Status**: Draft

## Summary

The `wave meta` command — which dynamically generates and executes multi-step pipelines via the philosopher persona — needs to be restored to working condition. The command exists in the codebase (`cmd/wave/commands/meta.go`) with full implementation including dry-run, save, and execution modes, but is currently non-functional or degraded.

## Background

`wave meta` was introduced in commit `164a20f` and allows users to describe a task in natural language and have the philosopher persona design an appropriate pipeline with steps, personas, and contracts, then execute it.

### Current Implementation

- **Command**: `wave meta [task description]`
- **Flags**: `--dry-run`, `--save <path>`, `--manifest <path>`, `--mock`
- **Flow**: User input → philosopher persona generates pipeline YAML → pipeline executor runs generated steps
- **Key files**: `cmd/wave/commands/meta.go`, `internal/pipeline/meta.go`, `internal/pipeline/meta_test.go`

## Root Cause Analysis

Investigation reveals a critical bug in `internal/pipeline/meta.go:282-285`:

```go
buf := make([]byte, 1024*1024) // 1MB buffer
n, _ := result.Stdout.Read(buf)
output := string(buf[:n])
```

**Issue 1 — Wrong data source**: The meta executor reads from `result.Stdout` (raw NDJSON stream) instead of `result.ResultContent` (parsed, clean content). With the `ClaudeAdapter`, `Stdout` contains the full NDJSON event stream, not extracted pipeline YAML.

**Issue 2 — Fragile read**: A single `Read()` call on an `io.Reader` is not guaranteed to return all bytes. Large outputs may be silently truncated.

**Issue 3 — Mock adapter masks the bug**: `MockAdapter` sets both `Stdout` and `ResultContent` to identical content, so tests pass even though the real adapter path is broken.

**Issue 4 — Mock output is wrong for meta**: `generateRealisticOutput` for persona `"philosopher"` returns JSON docs-phase output, not pipeline YAML with `--- PIPELINE ---` / `--- SCHEMAS ---` sections, so `--mock` mode cannot produce valid meta pipeline output.

## Acceptance Criteria

- [ ] `wave meta "<task>" --dry-run` generates a valid pipeline and displays the step plan
- [ ] `wave meta "<task>"` executes the generated pipeline end-to-end
- [ ] `wave meta "<task>" --save <name>` persists the generated pipeline YAML for reuse
- [ ] `wave meta` with `--mock` adapter works for testing without live LLM calls
- [ ] All existing tests in `cmd/wave/commands/meta_test.go` pass
- [ ] All existing tests in `internal/pipeline/meta_test.go` pass
- [ ] Philosopher persona is properly configured in the default manifest
- [ ] Error messages are clear when prerequisites are missing (e.g., no philosopher persona)

## Edge Cases

- What happens when the philosopher generates invalid YAML? (Already handled with parse error + raw YAML dump)
- What happens when the philosopher generates a pipeline that fails validation? (Already handled)
- What happens when stdout exceeds the current 1MB buffer?
- What happens when the philosopher generates schemas with invalid JSON?
- What happens when `--save` path has no parent directory?

## Requirements

### Functional Requirements

- **FR-001**: Meta executor MUST use `result.ResultContent` from the adapter instead of raw `result.Stdout`
- **FR-002**: Meta executor MUST handle the case where `ResultContent` is empty by falling back to reading all of `Stdout` via `io.ReadAll`
- **FR-003**: MockAdapter MUST generate valid meta-pipeline output (with `--- PIPELINE ---` and `--- SCHEMAS ---` sections) when invoked as the philosopher persona for meta-pipeline generation
- **FR-004**: All existing meta pipeline tests MUST continue to pass
- **FR-005**: New integration-level tests MUST verify the full `--mock` flow end-to-end

## Related History

- `164a20f` — feat: implement meta-pipeline for self-designing pipelines
- `c6e0870` — feat(do): add --meta flag for dynamic pipeline generation
- `5b9ab1d` — fix(meta): remove redundant schema instructions from generated prompts
- `5e7b2af` — feat: add standalone wave meta command
- `6d24cc9` — refactor(pipeline): default memory.strategy to fresh
