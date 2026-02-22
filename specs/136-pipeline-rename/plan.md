# Implementation Plan: Pipeline Identifier Rename

## Objective

Rename 7 pipeline identifiers for clarity and consistency, update all references across the codebase, and ensure every pipeline YAML has a `description` and `input.example` populated.

## Approach

Work in waves: rename files first (git mv for history preservation), then update internal references within the YAMLs, then do a codebase-wide search-and-replace for each old name, then verify tests pass.

## Rename Mapping

| Old Name | New Name | File Renames |
|---|---|---|
| `doc-loop` | `doc-audit` | `doc-loop.yaml` -> `doc-audit.yaml` |
| `doc-sync` | `doc-fix` | `doc-sync.yaml` -> `doc-fix.yaml` |
| `gh-issue-impl` | `gh-implement` | `gh-issue-impl.yaml` -> `gh-implement.yaml` |
| `gh-issue-research` | `gh-research` | `gh-issue-research.yaml` -> `gh-research.yaml` |
| `gh-issue-rewrite` | `gh-rewrite` | `gh-issue-rewrite.yaml` -> `gh-rewrite.yaml` |
| `gh-issue-update` | `gh-refresh` | `gh-issue-update.yaml` -> `gh-refresh.yaml` |
| `recinq` | `simplify` | `recinq.yaml` -> `simplify.yaml` |
| `speckit-flow` | `spec-develop` | `speckit-flow.yaml` -> `spec-develop.yaml` |

## File Mapping

### Files to Rename (git mv)

Each renamed pipeline has files in two locations:

**Pipeline YAMLs:**
- `.wave/pipelines/<old>.yaml` -> `.wave/pipelines/<new>.yaml` (8 renames)
- `internal/defaults/pipelines/<old>.yaml` -> `internal/defaults/pipelines/<new>.yaml` (8 renames)

**Prompt directories (only for pipelines using `source_path`):**
- `.wave/prompts/github-issue-impl/` -> `.wave/prompts/gh-implement/` (4 files)
- `internal/defaults/prompts/github-issue-impl/` -> `internal/defaults/prompts/gh-implement/` (4 files)
- `.wave/prompts/speckit-flow/` -> `.wave/prompts/spec-develop/` (8 files)
- `internal/defaults/prompts/speckit-flow/` -> `internal/defaults/prompts/spec-develop/` (8 files)

**Documentation files:**
- `docs/use-cases/doc-loop.md` -> `docs/use-cases/doc-audit.md`
- `docs/use-cases/recinq.md` -> `docs/use-cases/simplify.md`
- `docs/examples/speckit-flow.md` -> `docs/examples/spec-develop.md`

### Files to Modify (content updates)

**Inside each renamed YAML:** Update `metadata.name` field to the new name.

**source_path references in YAMLs:** Update directory paths for `gh-issue-impl` -> `gh-implement` and `speckit-flow` -> `spec-develop`.

**Go source files (string literals, comments):**
- `internal/defaults/embed.go` and `embed_test.go`
- `internal/pipeline/*.go` and `*_test.go` (comments, test fixtures)
- `internal/skill/skill.go` and `skill_test.go`
- `internal/recovery/classify.go` and `classify_test.go`
- `internal/preflight/preflight.go` and `preflight_test.go`
- `internal/display/*.go` and `*_test.go`
- `internal/webui/*.go`
- `internal/tui/pipelines.go`
- `cmd/wave/commands/*.go` and `*_test.go`

**Documentation:**
- `README.md`
- `docs/guide/quick-start.md`, `docs/guide/pipelines.md`
- `docs/use-cases/index.md` (VueJS component references)
- `docs/use-cases/doc-loop.md`, `docs/use-cases/recinq.md`, `docs/use-cases/supervise.md`
- `docs/examples/speckit-flow.md`
- `docs/.vitepress/config.ts`
- `docs/reference/*.md`
- `docs/guides/*.md`
- `docs/future/use-cases/*.md`
- `install.sh`

**Contract schemas (if referencing pipeline names):**
- `internal/defaults/contracts/*.schema.json`
- `.wave/contracts/*.schema.json`

### Files to Create

- `docs/reference/pipeline-naming.md` -- naming convention document for future pipelines

## Architecture Decisions

1. **No schema changes for `examples`**: The `PipelineMetadata` struct already has `Description` and the `InputConfig` already has `Example`. Rather than adding a new `Examples` field to the type system, we populate `input.example` on all 26 pipelines and ensure `metadata.description` is filled.

2. **Prompt directory naming**: For `gh-issue-impl`, the prompt directory is named `github-issue-impl` (not matching the pipeline name). The rename should use the new pipeline name directly: `gh-implement/`.

3. **Embedded defaults parity**: Both `.wave/` and `internal/defaults/` copies must always stay in sync. Each file operation is done in both locations.

4. **No backward compatibility**: Per CLAUDE.md, backward compatibility is NOT a constraint during prototype phase.

## Risks

| Risk | Mitigation |
|---|---|
| Missing a reference leads to runtime failure | Run `grep -r` for each old name after rename; run `go test ./...` |
| Embedded defaults diverge from .wave/ copies | Rename both in same task; diff after |
| Pipeline ID in SQLite state DB references old name | Old pipeline runs keep old IDs in DB; new runs get new IDs. No migration needed -- pipeline_id includes a hash suffix. |
| Prompt `source_path` references break | Update all `source_path` values inside renamed YAMLs |
| Doc links break in VitePress | Update all link references in `docs/` and `config.ts` |

## Testing Strategy

1. **Automated**: `go test ./...` must pass with zero failures
2. **Race detection**: `go test -race ./...` must pass
3. **Build verification**: `go build ./...` must compile
4. **Grep verification**: After all renames, `grep -r` for each old name should return zero results outside of `specs/` (the spec itself references old names for documentation)
5. **Manual verification**: `wave list` should show all 26 pipelines with new names and descriptions
