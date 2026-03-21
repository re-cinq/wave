# fix(cli): resolve --output flag semantic collision across commands

**Issue**: [#532](https://github.com/re-cinq/wave/issues/532)
**Author**: nextlevelshit
**State**: OPEN

## Problem

Root `--output` = output format. `init --output` = manifest file path. `agent export --output`/`-o` = export file path. Three different meanings; the `-o` short flag is registered on both root and `agent export`.

## Proposed Solution

Rename `init --output` to `--manifest-path` and `agent export --output` to `--export-path` or remove the `-o` short form.

## Current Flag Inventory

| Flag | Location | Short | Default | Semantic |
|------|----------|-------|---------|----------|
| Root `--output` | `main.go:131` | `-o` | `auto` | Output format (auto/json/text/quiet) |
| `init --output` | `init.go:60` | None | `wave.yaml` | Manifest file output path |
| `agent export --output` | `agent.go:210` | `-o` | `<name>.agent.md` | Export file path |
| `bench run --output` | `bench.go:173` | None | `""` | JSON results file path |

## Acceptance Criteria

1. `init --output` renamed to `init --manifest-path`
2. `agent export --output` renamed to `agent export --export-path` (remove `-o` short form)
3. `bench run --output` renamed to `bench run --results-path` (consistency)
4. Root `--output`/`-o` remains unchanged (canonical usage)
5. All existing tests updated to use new flag names
6. No `-o` collision between root and subcommands
