# Rename Pipeline Identifiers for Clarity and Consistency

**Feature Branch**: `136-pipeline-rename`
**Issue**: [#136](https://github.com/re-cinq/wave/issues/136)
**Labels**: enhancement, ux, cleanup, pipeline
**Status**: Draft

## Summary

Pipeline identifiers should be renamed so that each name clearly conveys the pipeline's purpose without needing to read the description. Several current names are ambiguous (e.g., `doc-loop` vs `doc-sync`) or unnecessarily verbose (e.g., `gh-issue-impl`).

## Motivation

- `doc-loop` and `doc-sync` sound similar but serve different purposes -- their names don't distinguish them.
- `gh-issue-impl` could be shortened to `gh-implement` without losing meaning.
- Consistent naming conventions make pipelines easier to discover and remember.

## Current Pipeline Inventory

The following 26 pipelines exist in `.wave/pipelines/`:

| Current Name | Proposed Name | Rationale |
|---|---|---|
| `adr` | `adr` | Already clear (Architecture Decision Record) |
| `changelog` | `changelog` | Already clear |
| `code-review` | `code-review` | Already clear |
| `dead-code` | `dead-code` | Already clear |
| `debug` | `debug` | Already clear |
| `doc-loop` | `doc-audit` | Audits docs for consistency, creates issues for drift |
| `doc-sync` | `doc-fix` | Scans docs, fixes inconsistencies, commits and creates PR |
| `explain` | `explain` | Already clear |
| `feature` | `feature` | Already clear |
| `gh-issue-impl` | `gh-implement` | Shorter, clearer (original suggestion) |
| `gh-issue-research` | `gh-research` | Shorter, clearer gh- prefix grouping |
| `gh-issue-rewrite` | `gh-rewrite` | Shorter, clearer gh- prefix grouping |
| `gh-issue-update` | `gh-refresh` | "update" is vague; "refresh" conveys stale-issue renewal |
| `hello-world` | `hello-world` | Example/demo pipeline, name is fine |
| `hotfix` | `hotfix` | Already clear |
| `improve` | `improve` | Already clear |
| `onboard` | `onboard` | Already clear |
| `plan` | `plan` | Already clear |
| `prototype` | `prototype` | Already clear |
| `recinq` | `recinq` | Trademark — not renamed |
| `refactor` | `refactor` | Already clear |
| `security-scan` | `security-scan` | Already clear |
| `smoke-test` | `smoke-test` | Already clear |
| `speckit-flow` | `speckit-flow` | Trademark — not renamed |
| `supervise` | `supervise` | Already clear |
| `test-gen` | `test-gen` | Already clear |

## Naming Conventions

1. **Pattern**: `<verb>` or `<scope>-<verb>` in kebab-case
2. **Max length**: 20 characters
3. **Clarity rule**: A developer should be able to guess what the pipeline does from the name alone
4. **Grouping**: Related pipelines share a common prefix (e.g., `gh-` for GitHub operations, `doc-` for documentation)
5. **No abbreviations** unless universally understood (e.g., `adr`, `gh`)

## Schema Enhancements

In addition to renaming, each pipeline YAML should populate:

- `description`: One-sentence explanation of what the pipeline does
- `examples`: At least one usage example showing a typical invocation
- Any other schema fields that are currently unused but available

**Note**: The current `PipelineMetadata` struct in `internal/pipeline/types.go` does not have an `examples` field. The `input.example` field already exists on `InputConfig`. The issue's requirement for "examples" should be satisfied by ensuring every pipeline has `input.example` populated, rather than adding a new schema field.

## Acceptance Criteria

- [ ] Every pipeline name clearly conveys its function without reading the description
- [ ] `doc-loop` and `doc-sync` have distinct, self-explanatory names
- [ ] `gh-issue-*` pipelines follow a consistent, shorter naming pattern
- [ ] A naming convention document or comment is established for future pipelines
- [ ] Each renamed pipeline YAML includes a `description` and at least one `input.example`
- [ ] All references to old names are updated (code, docs, tests)
- [ ] `go test ./...` passes after renaming

## Scope of Changes

The rename touches three categories of files:

### 1. Pipeline YAML files (two copies)
- `.wave/pipelines/<name>.yaml` -- user-facing defaults
- `internal/defaults/pipelines/<name>.yaml` -- embedded defaults compiled into the binary

### 2. Prompt files (for pipelines with `source_path` references)
- `.wave/prompts/github-issue-impl/*.md` -- must be renamed to match new pipeline name
- `internal/defaults/prompts/github-issue-impl/*.md` -- embedded copies (if they exist)

### 3. All references across the codebase
- Go source files (string literals, comments, test fixtures)
- Documentation files (docs/, README.md)
- VitePress config (docs/.vitepress/config.ts)
- Contract schema files (if they reference pipeline names)
- Install scripts
