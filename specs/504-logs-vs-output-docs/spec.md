# audit: partial — logs vs output docs section (#22)

**Issue**: [#504](https://github.com/re-cinq/wave/issues/504)
**Labels**: audit
**Author**: nextlevelshit
**Source**: #22 — docs: Guidelines for logs vs progress output parameters

## Summary

The CLI reference documentation (`docs/reference/cli.md`) documents both `wave logs` and `--output` modes individually, but lacks an explicit comparison section explaining the difference between these two distinct observability mechanisms.

## Current State

- `docs/reference/cli.md:242-268`: `wave logs` command documented with `--step`, `--errors`, `--tail`, `--follow`, `--since`, `--level`, `--format` flags
- `docs/reference/cli.md:955-958`: `--output` flag with auto/json/text/quiet modes documented in Global Options
- `cmd/wave/commands/run.go:133`: output flags registered in code

## Acceptance Criteria

1. Add a "Logs vs Progress Output" section to `docs/reference/cli.md`
2. Explain the difference between `wave logs` (event history from state DB) and `--output` modes (real-time progress rendering)
3. Include 3 use-case examples demonstrating when to use each
4. Section should be placed logically near the existing `wave logs` section or in a dedicated "Concepts" area within the CLI reference
