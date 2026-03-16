# Implementation Plan: Cold-Start Fix, Flavour Auto-Detection, and Smart Init

## Objective

Overhaul `wave init` to (1) handle greenfield projects with no git repo, (2) auto-detect project language/flavour from 25+ marker file patterns, and (3) intelligently select pipelines/personas based on detected forge and language.

## Approach

Three sequential phases building on each other:

1. **Cold-start fix**: Add git init + initial commit logic to `runInit` and `runWizardInit` before any worktree/forge operations. Guard `DetectFromGitRemotes()` calls against missing `.git`.
2. **Flavour system**: Extract the detection logic into a dedicated `internal/onboarding/flavour.go` with a `FlavourInfo` struct and a table-driven detection function. Add `Flavour` and `FormatCommand` fields to `manifest.Project`. Unify the duplicate `detectProject()`/`detectProjectType()` functions into the new flavour system.
3. **Smart init**: Use `forge.DetectFromGitRemotes()` + `forge.FilterPipelinesByForge()` to filter personas at init time. Extract project metadata (name, description) from language-specific manifest files. Add first-run suggestions.

## File Mapping

### New Files
| File | Purpose |
|------|---------|
| `internal/onboarding/flavour.go` | `FlavourInfo` struct, `DetectFlavour()` function, 25+ language detection matrix |
| `internal/onboarding/flavour_test.go` | Table-driven tests for flavour detection |
| `internal/onboarding/metadata.go` | `ExtractProjectMetadata()` — reads name/description from go.mod, Cargo.toml, package.json, etc. |
| `internal/onboarding/metadata_test.go` | Tests for metadata extraction |

### Modified Files
| File | Changes |
|------|---------|
| `internal/manifest/types.go` | Add `Flavour string` and `FormatCommand string` fields to `Project` struct; update `ProjectVars()` |
| `cmd/wave/commands/init.go` | Cold-start git init + commit; replace `detectProject()` with `onboarding.DetectFlavour()`; use forge filtering for personas; use extracted metadata for manifest `name`/`description`; add first-run suggestion |
| `internal/onboarding/steps.go` | Replace `detectProjectType()` with `DetectFlavour()`; add `FormatCommand` to `TestConfigStep` result |
| `internal/onboarding/onboarding.go` | Add `Flavour` and `FormatCommand` fields to `WizardResult`; pass them through from step results; include flavour/format in `buildManifest()` |
| `internal/defaults/embed.go` | Add `FilterPersonasByForge()` function that filters persona configs by forge type (using naming convention or metadata) |

### Deleted
- `detectProject()` in `init.go` (replaced by `onboarding.DetectFlavour()`)
- `detectNodeProject()` in `init.go` (absorbed into flavour system)
- `detectProjectType()` in `steps.go` (replaced by `onboarding.DetectFlavour()`)

## Architecture Decisions

1. **Table-driven detection**: Use a `[]FlavourRule` slice checked sequentially (first match wins) rather than a switch statement. Each rule has marker files (all must exist), exclude files (none must exist), and the resulting `FlavourInfo`. This makes the 25+ language matrix maintainable and testable.

2. **Unified detection function**: Both `init.go`'s `detectProject()` and `steps.go`'s `detectProjectType()` are duplicated logic. Consolidate into a single `onboarding.DetectFlavour()` that returns a `FlavourInfo` struct. Both callers convert to their needed format.

3. **Flavour vs Language**: `flavour` captures the toolchain variant (e.g., `node-yarn` vs `node-pnpm`), while `language` captures the programming language (e.g., `javascript` or `typescript`). Both stored in manifest for different use cases.

4. **Cold-start sequence**: `git init` → write wave files → `git add -A` → `git commit -m "chore: initialize wave project"`. This ensures worktree operations have at least one commit to work with.

5. **Persona forge filtering**: Use the existing `forge.FilterPipelinesByForge()` pattern but applied to persona names. Personas without a forge prefix are always included. This reuses the existing convention.

6. **Glob-based marker detection**: For markers like `*.csproj` and `*.sln`, use `filepath.Glob()` to check for matches rather than exact filenames.

## Risks

| Risk | Mitigation |
|------|-----------|
| Cold-start `git init` in non-empty dir could be unexpected | Only run when `.git` does NOT exist; print clear message |
| Auto-commit could include unwanted files | Use targeted `git add wave.yaml .wave/` not `git add -A` |
| Ambiguous detection (e.g., both `go.mod` and `package.json`) | First-match-wins with priority order documented |
| Metadata extraction may fail on malformed files | Return empty metadata on parse errors; never block init |
| Forge detection fails without remote | Already handled — `DetectFromGitRemotes()` returns `ForgeUnknown`, which passes all personas through |

## Testing Strategy

1. **Unit tests for `DetectFlavour()`**: Create temp dirs with various marker file combinations, verify correct flavour/commands returned. Test priority ordering, glob patterns, edge cases.
2. **Unit tests for `ExtractProjectMetadata()`**: Create temp dirs with minimal go.mod, Cargo.toml, package.json files; verify name/description extraction.
3. **Integration tests for cold-start**: Test `runInit` in an empty temp dir (no `.git`); verify git repo created with initial commit.
4. **Manifest field tests**: Verify `Flavour` and `FormatCommand` fields round-trip through YAML marshal/unmarshal.
5. **Forge filtering tests**: Verify persona filtering works for each forge type.
