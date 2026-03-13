# feat: establish naming taxonomy for built-in pipelines with category prefixes

**Issue**: [#310](https://github.com/re-cinq/wave/issues/310)
**Labels**: enhancement
**Author**: nextlevelshit

## Summary

Establish a consistent naming taxonomy for all built-in pipelines using category prefixes so they sort together logically and are easier to discover and navigate.

Currently, only `gh-*` pipelines follow a prefix convention. All other built-in pipelines should be renamed with a category prefix that communicates their phase (pre/post implementation) and destructiveness.

## Motivation

- Pipelines are listed alphabetically in CLI output and documentation — prefixes group related pipelines together
- Users can quickly identify a pipeline's purpose and safety characteristics from its name alone
- Consistent naming reduces cognitive overhead when browsing `wave list`

## Proposed Taxonomy

The issue research proposes 7 categories: `plan-`, `impl-`, `audit-`, `doc-`, `test-`, `gh-`, `ops-`.

| Prefix | Phase | Destructive? | Description | Examples |
|--------|-------|-------------|-------------|----------|
| `audit-*` | Pre or post impl | No | Analysis and reporting pipelines | `audit-security`, `audit-test-coverage` |
| `plan-*` | Pre impl | No | Planning pipelines producing issues, markdown, specs | `plan-feature`, `plan-refactor` |
| `impl-*` | Implementation | Yes | Code-writing pipelines | `impl-feature`, `impl-hotfix` |
| `doc-*` | Any | Varies | Documentation pipelines | `doc-adr`, `doc-changelog` |
| `test-*` | Post impl | No | Test generation and verification | `test-gen`, `test-smoke` |
| `ops-*` | Any | Varies | Operational/meta pipelines | `ops-hello-world` |

### Excluded from rename

- **Forge-prefixed pipelines** (`gh-*`, `gl-*`, `bb-*`, `gt-*`): Per author comment, these will be replaced by "wave flavours" (#241) with forge auto-detection. Renaming now would be wasted effort.
- **`wave-*` pipelines** (in `.wave/pipelines/` only, not in defaults): Already follow a clear prefix convention for Wave self-evolution. These are user-space pipelines, not built-in defaults.

## Acceptance Criteria

- [ ] Decide on final set of prefix categories and document the taxonomy
- [ ] Rename all built-in pipelines in `internal/defaults/pipelines/` to follow the chosen prefix convention
- [ ] Rename corresponding prompt subdirectories in `internal/defaults/prompts/`
- [ ] Update metadata.name inside each renamed pipeline YAML
- [ ] Update any references to old pipeline names in Go source (hardcoded string comparisons)
- [ ] Update references in docs, specs, and CLI help text
- [ ] Update test fixtures that reference pipeline names by string
- [ ] Ensure `wave list` output groups pipelines logically by prefix
- [ ] Add taxonomy documentation to `docs/guide/pipelines.md`

## Key Constraints

- `prototype` pipeline has hardcoded references in `internal/pipeline/validation.go` (PhaseSkipValidator) and `internal/pipeline/resume.go` — these must be updated
- `speckit-flow` is referenced across ~15 test files, docs, and composition configs
- Pipeline names flow through: YAML filename → `metadata.name` → Go string comparisons → suggest engine → doctor → TUI → docs
- The `internal/defaults/prompts/` directory structure keys off pipeline names (subdirectory = pipeline name)
- `.wave/pipelines/` (user-space) files should be updated too for consistency but are NOT embedded in the binary
