# Requirements Quality Review Checklist

**Feature**: #260 — CLI Compliance Polish (clig.dev)  
**Date**: 2026-03-07  
**Scope**: Overall requirements quality validation across spec.md, plan.md, tasks.md

## Completeness

- [ ] CHK001 - Are all 7 persistent root flags (`--json`, `-q`/`--quiet`, `--no-color`, `--debug`, `--verbose`, `--no-tui`, `--output`) explicitly listed with their type, default value, and short-form alias? [Completeness]
- [ ] CHK002 - Does the spec define behavior for every subcommand under `--json` mode, including subcommands beyond the 5 listed (e.g., `wave init`, `wave clean`, `wave run`, `wave validate`)? [Completeness]
- [ ] CHK003 - Are all 13 error codes in the data model mapped to at least one trigger scenario in the spec or plan's actionable error messages? [Completeness]
- [ ] CHK004 - Does the spec define what JSON output looks like for `wave cancel --json` when there is nothing to cancel? [Completeness]
- [ ] CHK005 - Is the quiet mode behavior defined for ALL subcommands, not just `wave run` and `wave status`? (e.g., `wave init -q`, `wave clean -q`, `wave validate -q`) [Completeness]
- [ ] CHK006 - Does the spec define the exit code semantics — which exit codes map to which error categories (flag conflict, pipeline failure, contract violation)? [Completeness]
- [ ] CHK007 - Are requirements defined for `--json` interaction with interactive prompts (e.g., `wave init`, `wave clean` confirmation)? Interactive commands must either skip prompts or error in JSON mode. [Completeness]
- [ ] CHK008 - Does the spec cover the `--json` output schema for each subcommand, or are formats left implicit? [Completeness]

## Clarity

- [ ] CHK009 - Is the resolution precedence between `--json`, `--quiet`, `--output`, and subcommand `--format` unambiguously defined with a single ordered rule? [Clarity]
- [ ] CHK010 - Is the distinction between "structured JSON" (single document) and "NDJSON" (line-delimited) clearly associated with specific commands, or could a developer misinterpret which format applies? [Clarity]
- [ ] CHK011 - Is the `--quiet` + `--json` combination semantics clear enough that two independent implementers would produce the same behavior? [Clarity]
- [ ] CHK012 - Is the term "non-essential output" in FR-007 precisely defined — is a completion summary "essential" or "non-essential"? [Clarity]
- [ ] CHK013 - Does C5 (subcommand `--format` coexistence) clearly define what "explicitly set" means — does `--output auto` count as explicitly set, or only non-default values? [Clarity]
- [ ] CHK014 - Is the `ErrorResponse` debug field requirement clear — does "only when `--debug` is set" mean the field is omitted from JSON, or present with null/empty value? [Clarity]

## Consistency

- [ ] CHK015 - Is the naming convention for the `--json` flag consistent with `--output json` — does `--json` set Format to exactly "json" (same string), or a different internal value? [Consistency]
- [ ] CHK016 - Are the error code strings in the data model (`pipeline_not_found`, `manifest_missing`) consistent with the recovery package's `ErrorClass` enum values, or is a mapping layer needed? [Consistency]
- [ ] CHK017 - Is the `--no-color` flag behavior consistent with the existing `NO_COLOR` env var detection in `display.SelectColorPalette()` — do both code paths converge at the same point? [Consistency]
- [ ] CHK018 - Are the FR numbers (FR-001 through FR-015) consistently referenced in tasks, plan phases, and success criteria, or are some requirements orphaned from tasks? [Consistency]
- [ ] CHK019 - Is the `CLIError` JSON field naming (`"error"` not `"message"`) consistent with the clig.dev recommendation and common CLI tools like `gh` and `kubectl`? [Consistency]
- [ ] CHK020 - Are the task priority labels ([P1], [P2], [P3]) consistent with the user story priorities they reference? [Consistency]

## Coverage

- [ ] CHK021 - Does the spec address backward compatibility for existing scripts that parse `wave status --format json` output — will the output schema change when using root `--json` vs subcommand `--format json`? [Coverage]
- [ ] CHK022 - Are the interactions between `--no-color` and the TUI's Lipgloss renderer covered beyond the high-level "monochrome mode" description — is there a specific Lipgloss API call or color profile specified? [Coverage]
- [ ] CHK023 - Does the spec address what happens when `--json` is passed to a command that currently produces no structured output (e.g., `wave validate`, `wave init`)? [Coverage]
- [ ] CHK024 - Is the `TERM=dumb` edge case fully covered — does it trigger both `--no-color` and `--no-tui`, and is this documented in the spec or only in the plan? [Coverage]
- [ ] CHK025 - Does the spec cover signal handling (SIGINT, SIGTERM) in JSON mode — should interrupted pipelines still emit a final JSON error object? [Coverage]
- [ ] CHK026 - Are the stream discipline requirements (FR-011, FR-012) covered by specific task items for ALL commands, not just the 5 explicitly listed in Phase 7? [Coverage]
- [ ] CHK027 - Does the spec define whether `--json` affects the TUI web dashboard output (`internal/webui/`), or is it scoped strictly to CLI stdout? [Coverage]
- [ ] CHK028 - Is there coverage for the case where `NO_COLOR` env var is set AND `--no-color` flag is passed — are they idempotent or could double-application cause issues? [Coverage]
