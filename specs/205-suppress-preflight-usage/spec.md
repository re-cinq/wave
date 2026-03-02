# fix(cli): preflight failure shows full help text, confusing the user

**Issue**: [#205](https://github.com/re-cinq/wave/issues/205)
**Labels**: bug, pipeline
**Author**: nextlevelshit
**State**: OPEN

## Summary

When a pipeline fails preflight validation (e.g., missing tool), the CLI prints the full usage/help text alongside the error. This is noisy and confusing — the user sees a wall of flags and examples when they just need to know which tool is missing.

Example output when `gh` is not on PATH:

```
Error: pipeline execution failed: missing required tools: gh

Recovery options:
  Install missing tool:
    gh is required but not on PATH — install it using your package manager

Usage:
  wave run [pipeline] [input] [flags]

Examples:
  wave run gh-pr-review "Review the authentication changes"
  ...

Flags:
      --dry-run            Show what would be executed without running
  ...
```

The recovery hints are good. The full help text is not — it buries the actual error.

## Proposed Fix

Suppress the `Usage:`, `Examples:`, and `Flags:` sections when the error is a preflight failure. Only show the error message and recovery options.

This likely requires changing how the error is returned from the `run` command to cobra — cobra auto-appends usage on error by default. Use `cmd.SilenceUsage = true` or `RunE` error handling to suppress it for known error types.

## Location

- `cmd/wave/commands/run.go` — where the pipeline is invoked
- `internal/preflight/preflight.go` — `ToolError` / `SkillError` types

## Acceptance Criteria

- [ ] When a pipeline fails preflight validation, the CLI shows ONLY the error message and recovery options — no `Usage:`, `Examples:`, or `Flags:` sections
- [ ] Non-preflight errors (contract validation, runtime errors, etc.) continue to show usage text as before
- [ ] Recovery hints remain visible and correctly formatted
- [ ] Existing tests continue to pass
- [ ] New test verifies usage suppression for preflight failures
