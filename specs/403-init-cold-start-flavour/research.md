# Research: Init Cold-Start Fix, Flavour Auto-Detection

**Date**: 2026-03-16
**Branch**: `403-init-cold-start-flavour`

## Research Areas

### R1: Cold-Start Git Bootstrap

**Decision**: Pre-wizard git init in `runInit`/`runWizardInit`
**Rationale**: The onboarding wizard and forge detection both depend on git state. Bootstrapping `.git` before the wizard matches the existing control flow where `runInit` orchestrates prerequisites and the wizard handles configuration.
**Alternatives Rejected**:
- Moving git init into the wizard: Violates separation of concerns. The wizard collects user preferences; git setup is infrastructure.
- Lazy init on first `wave run`: Too late — `wave init` needs to write `.wave/` and commit.

**Key Findings**:
- `runInit` (init.go:267) and `runWizardInit` (init.go:1165) are the two entry points that need cold-start logic.
- `forge.DetectFromGitRemotes()` already returns `ForgeUnknown` gracefully when git is unavailable or has no remotes (detect.go:64-88). No fix needed there.
- The worktree manager is not invoked during init, so worktree crashes are not relevant to this issue.
- `git rev-parse --verify HEAD` can check for commits; exit code 128 = no commits.

### R2: Flavour Detection Architecture

**Decision**: New `internal/flavour` package with `Detect(dir string) DetectionResult`
**Rationale**: The detection matrix (25+ languages) is too large for inline functions. A dedicated package provides clean testing surface, reuse by `wave doctor` and `wave suggest`, and a structured return type instead of `map[string]string`.
**Alternatives Rejected**:
- Extending `detectProjectType()` in onboarding/steps.go: Returns untyped map, mixes with wizard flow, not reusable.
- Extending `detectProject()` in init.go: Same issues plus it returns `map[string]interface{}`.

**Key Findings**:
- Two duplicate detection functions exist: `detectProjectType()` (onboarding/steps.go:165) and `detectProject()` (init.go:850). Both should be replaced.
- `detectNodeProject()` (init.go:902) has sophisticated lockfile-based package manager detection — this logic should move into the flavour package.
- The detection must be ordered by specificity: `bun.lock` before `package.json`, `deno.json` before `package.json`, etc.
- Current coverage: go, deno, node, rust, python (5 languages). Target: 25+.

### R3: Manifest Field Extension

**Decision**: Add `Flavour` and `FormatCommand` to `manifest.Project` struct
**Rationale**: These are project-level properties that parallel existing `Language`, `TestCommand`, etc. The `ProjectVars()` method already emits template variables for these fields.
**Alternatives Rejected**:
- Separate `flavour` top-level key: Breaks the `project.*` namespace convention.
- Storing flavour only in detection, not manifest: Pipelines need template access at runtime.

**Key Findings**:
- `Project` struct (types.go:8-14) needs two new fields: `Flavour string` and `FormatCommand string`.
- `ProjectVars()` (types.go:171-192) needs two new entries.
- `buildManifest()` (onboarding.go:175) and `createDefaultManifest()` (init.go:994) need to propagate the new fields.
- `WizardResult` (onboarding.go:25) needs `Flavour` and `FormatCommand` fields.

### R4: Forge-Filtered Pipeline Selection During Init

**Decision**: Call `forge.FilterPipelinesByForge()` in `getFilteredAssets()` after release filtering
**Rationale**: The function already exists and handles the exact semantics needed. Pipelines without forge prefixes pass through unchanged.
**Alternatives Rejected**:
- New filter in wizard: Duplicates existing logic.
- Metadata-based forge tags: Over-engineering for current needs.

**Key Findings**:
- `getFilteredAssets()` (init.go:116) is the natural integration point.
- After release filtering, the pipeline name list can be passed to `FilterPipelinesByForge()`.
- For `--all` mode, forge filtering should still apply (you want all release pipelines for your forge, not all forges).
- `DetectFromGitRemotes()` needs to be called once and the result threaded through.

### R5: Project Metadata Extraction

**Decision**: `flavour.DetectMetadata(dir string) MetadataResult` in the new flavour package
**Rationale**: Metadata extraction is tightly coupled to the same marker files used for flavour detection. Same package, separate function.
**Alternatives Rejected**:
- Metadata extraction in onboarding: Not reusable, and the parsing logic belongs with flavour detection.

**Key Findings**:
- `go.mod`: Parse module path, extract repo name from last path segment.
- `package.json`: JSON unmarshal, read `name` and `description`.
- `Cargo.toml`: TOML-like parsing (line scanning for `name = "..."` under `[package]`).
- `pyproject.toml`: Similar line scanning under `[project]` or `[tool.poetry]`.
- Fallback: use directory name as project name.

### R6: First-Run Suggestion

**Decision**: Check for source files in the project after init completes; suggest `audit-dx` if present, `ops-bootstrap` if empty.
**Rationale**: Simple heuristic that covers the two main user flows.
**Alternatives Rejected**:
- Complex suggestion engine: Over-engineering for a hint message.

**Key Findings**:
- `internal/suggest/engine.go` exists but is for runtime pipeline suggestion, not init-time hints.
- The check should look for any non-Wave source files (anything outside `.wave/`, `wave.yaml`).
- Both `printInitSuccess` and `printWizardSuccess` need to include the suggestion.

### R7: Required Pipeline Safeguard (Story 7)

**Decision**: Add `requiredPipelines` list to `getFilteredAssets` as defense-in-depth
**Rationale**: `impl-issue` has no forge prefix so it passes forge filtering. But `getFilteredAssets` also filters by `release: true`. A safeguard ensures essential pipelines survive all filters.
**Alternatives Rejected**:
- Trusting existing filters only: One metadata change could break it.

**Key Findings**:
- `getFilteredAssets()` calls `defaults.GetReleasePipelines()` which filters embedded pipelines by `release: true` metadata.
- `impl-issue.yaml` currently has `release: true`, but this could change.
- A `requiredPipelines` set (just `impl-issue` for now) that forces inclusion after all filtering is cheap insurance.
