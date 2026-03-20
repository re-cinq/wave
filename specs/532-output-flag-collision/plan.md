# Implementation Plan

## Objective

Resolve the `--output` flag semantic collision by renaming subcommand-level `--output` flags to descriptive, purpose-specific names while keeping the root-level `--output`/`-o` (output format) as the canonical usage.

## Approach

Straightforward flag renaming across three commands:
1. `init --output` → `--manifest-path`
2. `agent export --output`/`-o` → `--export-path` (no short form)
3. `bench run --output` → `--results-path`

Each rename touches the flag definition and its corresponding tests. No architectural changes needed.

## File Mapping

| File | Action | Change |
|------|--------|--------|
| `cmd/wave/commands/init.go` | modify | Rename `--output` flag to `--manifest-path` |
| `cmd/wave/commands/init_test.go` | modify | Update test to use `--manifest-path` |
| `cmd/wave/commands/agent.go` | modify | Rename `--output`/`-o` to `--export-path` (no short) |
| `cmd/wave/commands/agent_test.go` | modify | Update test to use `--export-path` |
| `cmd/wave/commands/bench.go` | modify | Rename `--output` to `--results-path` |

## Architecture Decisions

- **No deprecation aliases**: The issue is filed during prototype phase where backward compatibility is not required (per CLAUDE.md). Clean rename without aliases.
- **No `-o` short form on subcommands**: Only root keeps `-o` to prevent future collisions.
- **bench --output included**: While not mentioned in the issue, it has the same semantic collision pattern and should be fixed for consistency.

## Risks

- **Low**: Flag renaming is straightforward with no behavioral changes.
- **User scripts**: Any scripts using `wave init --output` or `wave agent export --output` will break. Acceptable during prototype phase.

## Testing Strategy

- Update existing tests that reference the old flag names.
- Run `go test ./cmd/wave/commands/...` to verify all tests pass.
- Run `go test ./...` for full regression.
