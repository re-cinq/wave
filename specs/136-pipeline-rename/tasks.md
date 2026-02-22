# Tasks

## Phase 1: File Renames

- [X] Task 1.1: Rename pipeline YAMLs in `.wave/pipelines/` (6 files: doc-loop->doc-audit, doc-sync->doc-fix, gh-issue-impl->gh-implement, gh-issue-research->gh-research, gh-issue-rewrite->gh-rewrite, gh-issue-update->gh-refresh) [P]
- [X] Task 1.2: Rename pipeline YAMLs in `internal/defaults/pipelines/` (same 6 renames) [P]
- [X] Task 1.3: Rename prompt directory `.wave/prompts/github-issue-impl/` -> `.wave/prompts/gh-implement/`
- [X] Task 1.4: Rename prompt directory `internal/defaults/prompts/github-issue-impl/` -> `internal/defaults/prompts/gh-implement/`
- [ ] Task 1.5: SKIPPED — `speckit-flow` is a trademark, not renamed
- [ ] Task 1.6: SKIPPED — `speckit-flow` is a trademark, not renamed
- [X] Task 1.7: Rename doc files: `docs/use-cases/doc-loop.md` -> `docs/use-cases/doc-audit.md`

## Phase 2: YAML Internal Updates

- [X] Task 2.1: Update `metadata.name` in all 8 renamed pipeline YAMLs (both `.wave/` and `internal/defaults/` copies) [P]
- [X] Task 2.2: Update `source_path` references in gh-implement YAMLs: `github-issue-impl/` -> `gh-implement/` (both copies) [P]
- [ ] Task 2.3: SKIPPED — `speckit-flow` is a trademark, not renamed
- [X] Task 2.4: Ensure all 26 pipeline YAMLs have `metadata.description` populated (both copies) [P]
- [X] Task 2.5: Ensure all 26 pipeline YAMLs have `input.example` populated (both copies) [P]
- [X] Task 2.6: Update internal prompt references in doc-audit YAML (footer references `doc-loop pipeline` -> `doc-audit pipeline`) [P]

## Phase 3: Codebase-wide Reference Updates

- [X] Task 3.1: Update all Go source files referencing `doc-loop` -> `doc-audit` [P]
- [X] Task 3.2: Update all Go source files referencing `doc-sync` -> `doc-fix` [P]
- [X] Task 3.3: Update all Go source files referencing `gh-issue-impl` -> `gh-implement` [P]
- [X] Task 3.4: Update all Go source files referencing `gh-issue-research` -> `gh-research` [P]
- [X] Task 3.5: Update all Go source files referencing `gh-issue-rewrite` -> `gh-rewrite` [P]
- [X] Task 3.6: Update all Go source files referencing `gh-issue-update` -> `gh-refresh` [P]
- [ ] Task 3.7: SKIPPED — `recinq` is a trademark, not renamed
- [ ] Task 3.8: SKIPPED — `speckit-flow` is a trademark, not renamed

## Phase 4: Documentation Updates

- [X] Task 4.1: Update `README.md` with new pipeline names [P]
- [X] Task 4.2: Update `docs/use-cases/index.md` (VueJS gallery component pipeline IDs, links) [P]
- [X] Task 4.3: Update content in renamed doc files (`doc-audit.md`) [P]
- [X] Task 4.4: Update `docs/.vitepress/config.ts` sidebar/nav references [P]
- [X] Task 4.5: Update `docs/guide/quick-start.md` and `docs/guide/pipelines.md` [P]
- [X] Task 4.6: Update `docs/reference/*.md` files [P]
- [X] Task 4.7: Update `docs/guides/*.md` files [P]
- [X] Task 4.8: Update `docs/future/use-cases/*.md` files [P]
- [X] Task 4.9: Update `docs/examples/hotfix-pipeline.md` if it references renamed pipelines [P]
- [X] Task 4.10: Update `install.sh` if it references pipeline names [P]
- [X] Task 4.11: Create `docs/reference/pipeline-naming.md` naming convention document

## Phase 5: Testing & Validation

- [X] Task 5.1: Run `go build ./...` and fix any compilation errors
- [X] Task 5.2: Run `go test ./...` and fix any test failures
- [X] Task 5.3: Run `go test -race ./...` to verify no race conditions
- [X] Task 5.4: Grep for all old pipeline names to verify zero remaining references outside specs/
- [X] Task 5.5: Verify `.wave/` and `internal/defaults/` copies are in sync
