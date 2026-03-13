# Implementation Plan: Unify Platform-Specific Pipelines

**Branch**: `241-unify-platform-pipelines` | **Date**: 2026-03-13 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/241-unify-platform-pipelines/spec.md`

## Summary

Replace 25 forge-prefixed pipeline YAML files and 4 platform-specific prompt directories with 6 unified pipeline definitions that use `{{ forge.* }}` template variables for runtime platform adaptation. Extend `ForgeInfo` with `PRTerm`/`PRCommand` fields, inject forge metadata into `PipelineContext` via `SetCustomVariable()`, resolve persona references and tool requirements through the existing `ResolvePlaceholders()` mechanism, and add backward-compatible deprecated name routing.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `gopkg.in/yaml.v3`, `github.com/spf13/cobra`, `embed` (stdlib)
**Storage**: Embedded filesystem (`embed.FS`) for default pipelines/prompts
**Testing**: `go test ./...`, `go test -race ./...`, `golangci-lint run`
**Target Platform**: Linux/macOS/Windows (single static binary)
**Project Type**: Single Go binary with embedded assets
**Performance Goals**: No runtime overhead — template resolution is string replacement
**Constraints**: No new dependencies, no YAML schema changes, no breaking changes to manifest
**Scale/Scope**: 25 files deleted, 6 new pipeline files, 4 new prompt files, ~10 Go files modified

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new dependencies. Embedded assets only |
| P2: Manifest as SSOT | PASS | No manifest schema changes. `wave.yaml` unchanged |
| P3: Persona-Scoped Execution | PASS | Forge-specific personas remain separate, resolved dynamically |
| P4: Fresh Memory | PASS | No change to step boundary behavior |
| P5: Navigator-First | N/A | Pipeline structure unchanged — navigator still first where applicable |
| P6: Contracts at Every Handover | PASS | Same contracts, same validation. No schema changes |
| P7: Relay via Summarizer | N/A | No changes to relay/compaction |
| P8: Ephemeral Workspaces | PASS | Workspace behavior unchanged |
| P9: Credentials Never Touch Disk | PASS | Token handling stays in persona prompts, not pipeline templates |
| P10: Observable Progress | PASS | Events unchanged. Deprecation warnings go to stderr |
| P11: Bounded Recursion | N/A | No recursion changes |
| P12: Minimal Step State Machine | PASS | No state machine changes |
| P13: Test Ownership | PASS | Full test suite must pass. Forge detection tests updated |

**Post-Phase 1 re-check**: All principles continue to pass. The template variable approach adds no new architectural patterns — it reuses the existing `CustomVariables` mechanism.

## Project Structure

### Documentation (this feature)

```
specs/241-unify-platform-pipelines/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 research findings
├── data-model.md        # Phase 1 entity definitions
└── tasks.md             # Phase 2 task breakdown (from /speckit.tasks)
```

### Source Code (repository root)

```
internal/
├── forge/
│   ├── detect.go          # MODIFY: Add PRTerm, PRCommand to ForgeInfo; extend forgeMetadata()
│   └── detect_test.go     # MODIFY: Update tests for new fields
├── pipeline/
│   ├── context.go         # MODIFY: Add InjectForgeVariables() helper
│   ├── context_test.go    # MODIFY: Test forge variable injection + resolution
│   ├── deprecated.go      # NEW: ResolveDeprecatedName() function
│   ├── deprecated_test.go # NEW: Test deprecated name resolution
│   └── executor.go        # MODIFY: Inject forge vars, resolve persona/tools/source_path
├── preflight/
│   └── preflight.go       # MODIFY: Skip empty strings in CheckTools()
├── suggest/
│   └── engine.go          # MODIFY: Handle unified pipeline names in suggestions
├── doctor/
│   └── optimize.go        # MODIFY: Update pipeline classification for unified names
└── defaults/
    ├── embed.go           # No change needed (embed patterns stay the same)
    ├── pipelines/
    │   ├── implement.yaml   # NEW: Unified implement pipeline
    │   ├── scope.yaml       # NEW: Unified scope pipeline
    │   ├── research.yaml    # NEW: Unified research pipeline
    │   ├── rewrite.yaml     # NEW: Unified rewrite pipeline
    │   ├── refresh.yaml     # NEW: Unified refresh pipeline
    │   └── pr-review.yaml   # NEW: Unified pr-review pipeline
    │   # DELETE: bb-implement.yaml, gh-implement.yaml, gl-implement.yaml, gt-implement.yaml
    │   # DELETE: bb-scope.yaml, gh-scope.yaml, gl-scope.yaml, gt-scope.yaml
    │   # DELETE: bb-research.yaml, gh-research.yaml, gl-research.yaml, gt-research.yaml
    │   # DELETE: bb-rewrite.yaml, gh-rewrite.yaml, gl-rewrite.yaml, gt-rewrite.yaml
    │   # DELETE: bb-refresh.yaml, gh-refresh.yaml, gl-refresh.yaml, gt-refresh.yaml
    │   # DELETE: gh-pr-review.yaml
    └── prompts/
        └── implement/       # NEW: Unified prompt directory
            ├── fetch-assess.md
            ├── plan.md
            ├── implement.md
            └── create-pr.md
        # DELETE: bb-implement/, gh-implement/, gl-implement/, gt-implement/
```

**Structure Decision**: This follows the existing Go project structure. All changes are within existing packages. The only new file is `internal/pipeline/deprecated.go` for the backward-compatibility resolver. No new packages are needed.

## Implementation Phases

### Phase 1: Forge Infrastructure (Go code changes)

Extend the forge detection and pipeline context to support forge template variables.

**1.1 Extend `ForgeInfo` struct** (`internal/forge/detect.go`)
- Add `PRTerm string` and `PRCommand string` fields
- Update `forgeMetadata()` to return all 4 values: `cli, prefix, prTerm, prCommand`
- Update `Detect()` to populate the new fields
- Update tests in `detect_test.go`

**1.2 Add `InjectForgeVariables()` helper** (`internal/pipeline/context.go`)
- New function that takes `*PipelineContext` and `forge.ForgeInfo`
- Calls `ctx.SetCustomVariable()` for each of the 8 forge variables
- Add tests for injection and round-trip resolution

**1.3 Resolve persona template variables** (`internal/pipeline/executor.go`)
- In `runStepExecution()` (line ~1040), resolve `step.Persona` through `ResolvePlaceholders()` before calling `GetPersona()`
- In prompt loading (line ~1695), resolve `step.Exec.SourcePath` before `os.ReadFile()`

**1.4 Inject forge variables in executor** (`internal/pipeline/executor.go`)
- After `newContextWithProject()` (line ~276) and before preflight, call `forge.DetectFromGitRemotes()` and `InjectForgeVariables()`
- Resolve `requires.tools` entries through `ResolvePlaceholders()` before passing to preflight checker
- Skip empty strings from unresolved template variables

**1.5 Update preflight checker** (`internal/preflight/preflight.go`)
- In `CheckTools()`, skip empty strings in the tools list

### Phase 2: Unified Pipeline Definitions (YAML + prompt files)

Create unified pipeline files and prompt files.

**2.1 Create unified `implement.yaml`**
- Template based on `gh-implement.yaml` structure
- Replace `persona: github-commenter` → `persona: "{{ forge.prefix }}-commenter"`
- Replace `source_path: .wave/prompts/gh-implement/...` → `.wave/prompts/implement/...`
- Use `create-pr` as step ID (3-of-4 convention)
- Add `requires.tools: ["{{ forge.cli_tool }}", "git"]`

**2.2 Create unified prompt files** (`internal/defaults/prompts/implement/`)
- `fetch-assess.md`: Merge all 4 variants into one using `{{ forge.cli_tool }}`, `{{ forge.type }}`, `{{ forge.pr_term }}`
- `plan.md`: Copy from `gh-implement/plan.md` (already identical across platforms), change header
- `implement.md`: Copy from `gh-implement/implement.md` (already identical), change header
- `create-pr.md`: Merge all 4 variants using `{{ forge.cli_tool }}`, `{{ forge.pr_command }}`, `{{ forge.pr_term }}`

**2.3 Create unified pipeline files for remaining families**
- `scope.yaml`: Merge 4 scope pipelines, use `{{ forge.prefix }}-analyst` / `{{ forge.prefix }}-scoper`
- `research.yaml`: Merge 4 research pipelines, use `{{ forge.prefix }}-analyst` / `{{ forge.prefix }}-commenter`
- `rewrite.yaml`: Merge 4 rewrite pipelines, use `{{ forge.prefix }}-analyst` / `{{ forge.prefix }}-enhancer`
- `refresh.yaml`: Merge 4 refresh pipelines, use `{{ forge.prefix }}-analyst` / `{{ forge.prefix }}-enhancer`
- `pr-review.yaml`: Extend from `gh-pr-review.yaml`, use `{{ forge.prefix }}-commenter` for publish step

### Phase 3: Backward Compatibility & Cleanup

**3.1 Add deprecated name resolver** (`internal/pipeline/deprecated.go`)
- `ResolveDeprecatedName(name string) (string, bool)` function
- Tests in `deprecated_test.go`

**3.2 Integrate deprecated resolver in CLI** (`cmd/wave/commands/run.go`)
- Call `ResolveDeprecatedName()` before pipeline lookup
- Log deprecation warning to stderr

**3.3 Update suggest engine** (`internal/suggest/engine.go`)
- `resolvePipeline()` should check for unified names (no prefix) first
- `FilterByForge()` updated: unified pipelines match all forges

**3.4 Update doctor optimize** (`internal/doctor/optimize.go`)
- `classifyPipeline()` updated to recognize unified pipeline names
- Remove forge-prefix-specific logic for unified names

**3.5 Update `FilterPipelinesByForge()`** (`internal/forge/detect.go`)
- Return all pipelines when unified names are present (no forge prefix = universal)
- Keep existing behavior for any remaining prefixed pipelines during transition

### Phase 4: Delete Legacy Files

**4.1 Delete 25 pipeline YAML files**
- `bb-implement.yaml`, `gh-implement.yaml`, `gl-implement.yaml`, `gt-implement.yaml`
- `bb-scope.yaml`, `gh-scope.yaml`, `gl-scope.yaml`, `gt-scope.yaml`
- `bb-research.yaml`, `gh-research.yaml`, `gl-research.yaml`, `gt-research.yaml`
- `bb-rewrite.yaml`, `gh-rewrite.yaml`, `gl-rewrite.yaml`, `gt-rewrite.yaml`
- `bb-refresh.yaml`, `gh-refresh.yaml`, `gl-refresh.yaml`, `gt-refresh.yaml`
- `gh-pr-review.yaml`

**4.2 Delete 4 prompt directories**
- `internal/defaults/prompts/bb-implement/`
- `internal/defaults/prompts/gh-implement/`
- `internal/defaults/prompts/gl-implement/`
- `internal/defaults/prompts/gt-implement/`

### Phase 5: Testing & Validation

**5.1 Unit tests**: Forge variable injection, persona resolution, deprecated names, preflight empty-string skip
**5.2 Integration tests**: End-to-end pipeline parsing with forge variables, template resolution round-trips
**5.3 Existing test suite**: `go test ./...` must pass, `go test -race ./...` must pass
**5.4 Manual validation**: `wave validate` on the unified pipelines

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Inline prompts in scope/research/rewrite/refresh contain forge-specific CLI commands that can't simply be templated | Medium | Research shows inline prompts can use `{{ forge.cli_tool }}` and `{{ forge.type }}` for conditional sections. For complex Bitbucket API patterns, use `{{ forge.type }}` with conditional text blocks in the prompt |
| `embed.FS` glob patterns may break when directory names change | Low | The `//go:embed` patterns are `pipelines/*.yaml` and `prompts/**/*.md` — these match by extension, not by name. New directories are automatically picked up |
| Suggest engine and doctor assume forge-prefixed names | Medium | Update both to handle unified names. The `stripForgePrefix()` helper already exists and handles the base name extraction |
| Tests reference forge-prefixed pipeline names | Medium | Search and update all test references. Most tests use mock pipelines, not embedded defaults |

## Complexity Tracking

No constitution violations. All changes use existing mechanisms (`CustomVariables`, `ResolvePlaceholders`, `SetCustomVariable`). No new patterns, no new dependencies, no new architectural concepts.
