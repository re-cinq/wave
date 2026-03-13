# docs: documentation consistency report

**Issue**: [#358](https://github.com/re-cinq/wave/issues/358)
**Author**: nextlevelshit
**Labels**: documentation
**Severity levels**: 3 Critical, 5 High, 4 Medium, 2 Low (14 total)

## Issue Description

The report contains 14 inconsistencies (`clean: false`) between documentation and the actual codebase.

## Inconsistencies

### Critical

- **[DOC-001]** Manifest schema documents non-existent `skills` top-level field
  - `docs/reference/manifest-schema.md` documents `skills` as a top-level manifest field
  - The `Manifest` struct in `internal/manifest/types.go` has NO `skills` field
  - Skills are declared at the pipeline level via `Requires.Skills` in `internal/pipeline/types.go`
  - The `SkillConfig` section and examples in the docs are misleading

- **[DOC-002]** Contract types document non-existent `on_failure` field
  - **INVESTIGATION RESULT**: `on_failure` IS present on `ContractConfig` in `internal/pipeline/types.go:256`
  - This item may be a false positive — the field exists in the codebase
  - Implementer should verify and close if confirmed correct

- **[DOC-003]** Three platform-specific implement-epic pipelines documented but files do not exist
  - `docs/guide/pipelines.md` documents gl-*, gt-*, bb-* pipeline variants (18 total)
  - No gl-*, gt-*, or bb-* pipeline YAML files exist in `.wave/pipelines/`
  - Only gh-* pipelines exist as actual files

### High

- **[DOC-004]** Undocumented `--preserve-workspace` flag on `wave run`
  - Flag exists at `cmd/wave/commands/run.go:115`
  - Not listed in `docs/reference/cli.md` wave run options

- **[DOC-005]** Undocumented `wave migrate validate` subcommand
  - Subcommand exists at `cmd/wave/commands/migrate.go:164-165`
  - Not listed in `docs/reference/cli.md` migrate section

- **[DOC-006]** `Persona.sandbox` field not documented in manifest schema
  - Field exists at `internal/manifest/types.go:49` as `*PersonaSandbox`
  - `PersonaSandbox` struct has `AllowedDomains []string` (types.go:52-54)
  - Not in `docs/reference/manifest-schema.md` persona table
  - README does show sandbox usage in examples but not schema docs

- **[DOC-007]** `RoutingRule.match_labels` and `RelayConfig` fields missing from manifest schema
  - `RoutingRule.MatchLabels` exists at `internal/manifest/types.go:136` — undocumented
  - `RelayConfig.ContextWindow` exists at `types.go:142` — undocumented
  - `RelayConfig.SummarizerPersona` exists at `types.go:143` — undocumented

- **[DOC-008]** Pipeline count claims '47 built-in' but actual counts differ
  - 46 pipeline YAML files exist in `.wave/pipelines/`
  - Docs claim 47 in multiple places (README, pipelines guide)

### Medium

- **[DOC-009]** 17+ pipelines in `.wave/pipelines/` absent from pipeline documentation
  - Missing from docs: consolidate, dead-code-issue, dead-code-review, dual-analysis, dx-audit, epic-runner, junk-code, quality-loop, release-harden, research-implement, ux-audit, wave-audit, wave-bugfix, wave-evolve, wave-review, wave-security-audit, wave-test-hardening

- **[DOC-010]** Undocumented `--manifest` flag on `wave do` command
  - `--manifest` is a global flag available on all commands (docs/reference/cli.md Global Options)
  - This may be a false positive — it IS documented globally

- **[DOC-011]** `GITHUB_TOKEN` fallback behavior not documented
  - Code checks `GH_TOKEN` first, then falls back to `GITHUB_TOKEN`
  - `docs/reference/environment.md:123` mentions both but doesn't specify precedence order

- **[DOC-012]** README persona count says 30 but directory has 31 files
  - `.wave/personas/` has 31 files including `base-protocol.md`
  - `base-protocol.md` is NOT a persona — it's a shared preamble
  - 31 files - 1 non-persona = 30 personas — count is technically correct
  - May warrant a clarifying note

### Low

- **[DOC-013]** Incomplete persona documentation coverage for 22 of 30 personas
- **[DOC-014]** Anomalous `reviewer.yaml` file in `internal/defaults/personas/`
  - A `.yaml` config file exists alongside `.md` persona files — may be intentional

## Acceptance Criteria

- [ ] All critical inconsistencies resolved
- [ ] All high-severity inconsistencies resolved
- [ ] Medium-severity items addressed or documented as intentional
- [ ] Low-severity items triaged
- [ ] `go test ./...` passes (no code changes expected, but verify)
