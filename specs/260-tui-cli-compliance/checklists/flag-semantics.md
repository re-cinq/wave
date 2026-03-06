# Flag Semantics Quality Checklist

**Feature**: #260 — CLI Compliance Polish (clig.dev)  
**Date**: 2026-03-07  
**Scope**: Quality of flag interaction, conflict detection, and resolution specifications

## Flag Interaction Matrix

- [ ] CHK101 - Is every pairwise combination of the 3 new flags (`--json`, `--quiet`, `--no-color`) explicitly classified as either "conflict", "orthogonal", or "override" in the spec? [Completeness]
- [ ] CHK102 - Does the spec define the full interaction matrix between new flags and ALL existing flags (`--output`, `--debug`, `--verbose`, `--no-tui`), not just the conflict cases? [Completeness]
- [ ] CHK103 - Is the `--json` + `--debug` combination specified — does `--debug` add extra fields to JSON output, or only affect text mode? [Completeness]
- [ ] CHK104 - Is the behavior of `--no-color` + `--json` specified — JSON output has no ANSI codes regardless, so is `--no-color` redundant in JSON mode? [Clarity]

## Conflict Detection

- [ ] CHK105 - Is the conflict detection rule for `--quiet` + `--output json` justified — why is this a conflict when `--json` + `--quiet` is orthogonal? [Clarity]
- [ ] CHK106 - Does the conflict error message format include BOTH conflicting flag names and their values, so the user knows exactly what to fix? [Completeness]
- [ ] CHK107 - Is the conflict detection timing specified — does it happen in `PersistentPreRunE` before any subcommand logic, or could a subcommand override it? [Clarity]
- [ ] CHK108 - Are the conflict detection rules testable without executing actual subcommands — is `ResolveOutputConfig` a pure function of flag values? [Coverage]

## Flag Persistence and Scope

- [ ] CHK109 - Is the persistent flag inheritance model clear — do persistent root flags propagate to ALL subcommands including `cobra.Command` children added by plugins or future commands? [Clarity]
- [ ] CHK110 - Does the spec address whether `--json` applies to the `help` subcommand output — should `wave --json help` produce JSON-formatted help? [Coverage]
- [ ] CHK111 - Is the `--no-color` flag scope defined for child processes — when Wave launches adapter subprocesses, does `NO_COLOR=1` propagate through the environment? [Coverage]
- [ ] CHK112 - Does the spec address the case where a user sets `--output json` via a shell alias or config file — does `--json` conflict with a config-sourced `--output json`, or only with explicit CLI flags? [Coverage]
