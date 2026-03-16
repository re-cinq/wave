# Implementation Plan: Init Cold-Start Fix, Flavour Auto-Detection

**Branch**: `403-init-cold-start-flavour` | **Date**: 2026-03-16 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/403-init-cold-start-flavour/spec.md`

## Summary

Fix `wave init` cold-start failures (no `.git`, no commits, no remote) and introduce automatic language/flavour detection for 25+ languages. Adds a new `internal/flavour` package for structured project detection, extends the manifest `Project` struct with `flavour` and `format_command` fields, integrates forge-filtered pipeline selection during init, extracts project metadata from manifest files, and provides first-run pipeline suggestions.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `gopkg.in/yaml.v3`, `github.com/spf13/cobra`, `github.com/charmbracelet/huh`
**Storage**: Filesystem (`wave.yaml`, `.wave/` directory tree)
**Testing**: `go test ./...` with table-driven tests
**Target Platform**: Linux, macOS, Windows (single static binary)
**Project Type**: Single Go module
**Constraints**: No runtime dependencies beyond adapter binaries
**Scale/Scope**: ~8 files modified, 1 new package created

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new dependencies added. `internal/flavour` is pure Go. |
| P2: Manifest as Source of Truth | PASS | New fields (`flavour`, `format_command`) are in `wave.yaml`. |
| P3: Persona-Scoped Execution | N/A | Init flow does not involve persona execution. |
| P4: Fresh Memory | N/A | No pipeline steps involved. |
| P5: Navigator-First | N/A | Not a pipeline change. |
| P6: Contracts at Handovers | N/A | Not a pipeline change. |
| P7: Relay via Summarizer | N/A | Not relevant. |
| P8: Ephemeral Workspaces | N/A | Not relevant. |
| P9: Credentials Never Touch Disk | PASS | No credential handling in init. |
| P10: Observable Progress | PASS | Init already prints structured output. |
| P11: Bounded Recursion | N/A | Not relevant. |
| P12: Step State Machine | N/A | Not a pipeline change. |
| P13: Test Ownership | PASS | All new code has corresponding tests. Existing test suites must pass. |

**Post-Phase 1 Re-check**: All principles still pass. No new violations introduced by the design.

## Project Structure

### Documentation (this feature)

```
specs/403-init-cold-start-flavour/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 research
├── data-model.md        # Phase 1 data model
└── tasks.md             # Phase 2 output (not yet created)
```

### Source Code (repository root)

```
internal/
├── flavour/             # NEW: Flavour detection package
│   ├── detect.go        # Detect() and detection rules matrix
│   ├── metadata.go      # DetectMetadata() for name/description extraction
│   └── detect_test.go   # Table-driven tests for all 25+ flavours
├── manifest/
│   └── types.go         # MODIFIED: Add Flavour, FormatCommand to Project
├── onboarding/
│   ├── onboarding.go    # MODIFIED: Add Flavour, FormatCommand to WizardResult + buildManifest
│   └── steps.go         # MODIFIED: Replace detectProjectType() with flavour.Detect()
└── forge/
    └── detect.go        # EXISTING: FilterPipelinesByForge (no changes needed)

cmd/wave/commands/
└── init.go              # MODIFIED: Cold-start git bootstrap, forge filtering,
                         #           metadata extraction, first-run suggestion,
                         #           replace detectProject() with flavour.Detect()
```

**Structure Decision**: This is a modification to an existing Go module structure. The only new directory is `internal/flavour/`. All other changes are modifications to existing files.

## Implementation Phases

### Phase A: Foundation — `internal/flavour` Package (Stories 2, 3, 13)

**Goal**: Create the flavour detection engine as a standalone testable package.

**Files**:
1. `internal/flavour/detect.go` — `DetectionResult` type, `DetectionRule` type, ordered rules slice, `Detect(dir string)` function
2. `internal/flavour/metadata.go` — `MetadataResult` type, `DetectMetadata(dir string)` function
3. `internal/flavour/detect_test.go` — Table-driven tests: one test case per flavour, specificity ordering tests, no-match test, metadata extraction tests

**Key Design Decisions**:
- Marker matching: `os.Stat` for exact filenames, `filepath.Glob` for patterns (`*.csproj`, `*.cabal`)
- Node refinement: After base detection, check `tsconfig.json` for language override and `package.json` scripts for command override
- Priority: Rules are checked in slice order. More specific markers (e.g., `bun.lock`) come before generic ones (`package.json`)

### Phase B: Manifest Extension (Story 3)

**Goal**: Add `Flavour` and `FormatCommand` to the type system and template variables.

**Files**:
1. `internal/manifest/types.go` — Add fields to `Project` struct, extend `ProjectVars()`
2. `internal/onboarding/onboarding.go` — Add fields to `WizardResult`, extend `buildManifest()`

**Backward Compatibility**: Both new fields are `yaml:"...,omitempty"`. Existing manifests without these fields load without error (FR-011).

### Phase C: Cold-Start Git Bootstrap (Story 1)

**Goal**: Ensure `wave init` works in directories without `.git`, without commits, and without remotes.

**Files**:
1. `cmd/wave/commands/init.go` — Add `ensureGitRepo()` helper called from both `runInit()` and `runWizardInit()` before any other logic

**Logic**:
```
ensureGitRepo():
  if .git not exists:
    git init
  if git rev-parse --verify HEAD fails (no commits):
    defer: after wave files written, create initial commit
  forge detection: already handles no-remote gracefully
```

**Initial Commit**: Stage only `wave.yaml` and `.wave/` contents. Message: `chore: initialize wave project`. Only created when the repo had no prior commits.

### Phase D: Integration — Wire Flavour Detection into Init (Stories 2, 5, 6, 7)

**Goal**: Replace duplicate detection logic with `flavour.Detect()`, add metadata extraction, forge filtering, required pipeline safeguard, and first-run suggestions.

**Files**:
1. `cmd/wave/commands/init.go`:
   - Replace `detectProject()` body with `flavour.Detect(".")` conversion
   - Replace `detectNodeProject()` — logic moves to flavour package
   - Add `flavour.DetectMetadata(".")` call to populate `metadata.name`/`metadata.description`
   - Add `forge.DetectFromGitRemotes()` + `forge.FilterPipelinesByForge()` in `getFilteredAssets()`
   - Add `requiredPipelines` safeguard for `impl-issue`
   - Update `printInitSuccess()` and `printWizardSuccess()` with dynamic first-run suggestion
2. `internal/onboarding/steps.go`:
   - Replace `detectProjectType()` body with `flavour.Detect(".")` conversion
   - Add `format_command` to `TestConfigStep` result

### Phase E: Tests (All Stories)

**Goal**: Comprehensive test coverage.

**Files**:
1. `internal/flavour/detect_test.go` — Already created in Phase A
2. `cmd/wave/commands/init_test.go` — Tests for:
   - Cold-start in empty directory (no `.git`)
   - Cold-start with `.git` but no commits
   - Cold-start with `.git`, commits, no remote
   - Flavour detection integration (go.mod → go flavour in manifest)
   - Forge-filtered pipeline selection
   - Required pipeline safeguard
   - Metadata extraction
   - Backward compatibility (existing manifest without new fields)
3. `internal/manifest/types_test.go` — Tests for `ProjectVars()` with new fields

## Complexity Tracking

No constitution violations to justify.

## Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Node detection order (bun vs npm) | Strict priority ordering in rules slice; tested individually |
| Glob patterns for markers (`*.csproj`) | `filepath.Glob` is well-tested; fallback to no-match |
| Git operations in init failing | `ensureGitRepo` wraps all git calls with clear error messages |
| Existing tests breaking | `go test ./...` required before PR; new fields are `omitempty` |
