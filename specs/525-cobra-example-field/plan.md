# Implementation Plan: Cobra Example Field Migration

## Objective

Move embedded examples from cobra `Long` descriptions to the dedicated `Example` field across CLI commands, improving `--help` output formatting and consistency.

## Approach

Mechanical refactoring: for each affected command, extract example lines from `Long` and place them in the `Example` field. Where a command already has an `Example` field, merge content. Where `Long` has no actual examples (just documentation), add an `Example` field with representative usage patterns.

## File Mapping

| File | Action |
|------|--------|
| `cmd/wave/commands/run.go` | Modify — move argument pattern lines from `Long` to existing `Example` |
| `cmd/wave/commands/resume.go` | Modify — extract examples from `Long`, add `Example` field |
| `cmd/wave/commands/cancel.go` | Modify — extract examples from `Long`, add `Example` field |
| `cmd/wave/commands/list.go` | Modify — add `Example` field with usage examples |
| `cmd/wave/commands/logs.go` | Modify — extract examples from `Long`, add `Example` field |
| `cmd/wave/commands/bench.go` | Modify — add `Example` to parent bench command |

No action needed:
- `doctor.go` — already has proper `Example` field
- `pause` — command does not exist in the codebase

## Architecture Decisions

- Keep `Long` descriptions focused on explanation (what the command does, how flags interact)
- `Example` field should show 2-line-indented examples (cobra convention)
- Preserve all existing examples verbatim — just relocate them
- For commands with no explicit examples in `Long` (list, bench parent), synthesize examples from the documented usage patterns

## Risks

- **Low risk**: Purely cosmetic change to help output formatting
- **No behavioral change**: Only `Long` and `Example` string fields are modified
- **Test impact**: Existing tests that assert on help output may need updating (unlikely — help text tests are uncommon)

## Testing Strategy

- Run `go build ./...` to verify compilation
- Run `go test ./...` to verify no test regressions
- Manual verification: `wave <cmd> -h` shows examples in the "Examples:" section
