# Implementation Plan: Wave Init Interactive Skill Selection

**Branch**: `384-init-skill-selection` | **Date**: 2026-03-14 | **Spec**: `specs/384-init-skill-selection/spec.md`
**Input**: Feature specification from `/specs/384-init-skill-selection/spec.md`

## Summary

Add a new Step 6 to the `wave init` onboarding wizard that lets users select a skill ecosystem (tessl, BMAD, OpenSpec, Spec-Kit, or Skip), browse/select skills, and install them into `.wave/skills/` with progress feedback. Skills are recorded as bare names in `wave.yaml` under the `skills:` key. The implementation follows the existing `WizardStep` interface pattern using `huh` forms with `WaveTheme()`, reuses the `SourceRouter` and `SourceAdapter` infrastructure from `internal/skill/`, and supports non-interactive skip (`--yes`) and reconfiguration (`--reconfigure`).

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `github.com/charmbracelet/huh v0.8.0`, `internal/skill` package (SourceAdapter, SourceRouter, DirectoryStore)
**Storage**: Filesystem — `.wave/skills/` for skill files, `wave.yaml` for manifest
**Testing**: `go test ./...` with `testify/assert`, `testify/require`; table-driven tests
**Target Platform**: Linux/macOS CLI
**Project Type**: Single Go binary
**Performance Goals**: N/A — interactive wizard, not performance-sensitive
**Constraints**: Must not add new dependencies; must work without ecosystem CLIs installed (graceful degradation)
**Scale/Scope**: ~300 lines new code, ~50 lines modified code across 4 files

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new runtime dependencies. Ecosystem CLIs are external prerequisites checked at runtime |
| P2: Manifest as SSOT | PASS | Skills recorded in `wave.yaml` under `skills:` key, matching `Manifest.Skills []string` |
| P3: Persona-Scoped Execution | N/A | This feature does not modify persona execution |
| P4: Fresh Memory | N/A | No pipeline step changes |
| P5: Navigator-First | N/A | No pipeline changes |
| P6: Contracts at Handover | N/A | No pipeline changes |
| P7: Relay | N/A | No relay changes |
| P8: Ephemeral Workspaces | N/A | No workspace changes |
| P9: Credentials Never Touch Disk | PASS | No credentials involved |
| P10: Observable Progress | PASS | Per-skill installation progress printed to stderr |
| P11: Bounded Recursion | N/A | No meta-pipelines |
| P12: Minimal State Machine | N/A | No state machine changes |
| P13: Test Ownership | PASS | All new code will have unit tests; existing tests must continue to pass |

**Post Phase-1 Re-check**: All principles remain PASS/N/A. No violations introduced.

## Project Structure

### Documentation (this feature)

```
specs/384-init-skill-selection/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model output
└── tasks.md             # Phase 2 task breakdown (created by /speckit.tasks)
```

### Source Code (repository root)

```
internal/onboarding/
├── onboarding.go        # MODIFY: Add Skills field to WizardResult, Step 6 call in RunWizard(), skills key in buildManifest()
├── steps.go             # MODIFY: Renumber step labels from "N of 5" to "N of 6"
├── skill_step.go        # NEW: SkillSelectionStep implementation with EcosystemDef, tessl search, install orchestration
├── skill_step_test.go   # NEW: Unit tests for SkillSelectionStep
├── onboarding_test.go   # MODIFY: Update test assertions for Skills field and step count
└── steps_test.go        # (no changes needed — step tests don't assert on label text)
```

**Structure Decision**: All new code lives in `internal/onboarding/` package. The `skill_step.go` file is a new file in the existing package, following the pattern of `steps.go` containing all step implementations. Separating into its own file keeps `steps.go` focused on the original 5 steps and makes the new step self-contained.

## Implementation Components

### Component 1: SkillSelectionStep (internal/onboarding/skill_step.go)

**New file** implementing the `WizardStep` interface.

**Key elements**:
- `EcosystemDef` struct and `ecosystems` package-level slice defining the 4 ecosystems
- `SkillSelectionStep` struct with injectable `lookPath` and `runCommand` for testability
- `Name()` returns `"Skill Selection"`
- `Run()` orchestrates the full flow:
  1. Non-interactive → skip immediately
  2. Show reconfiguration context if `cfg.Reconfigure`
  3. Ecosystem selection via `huh.Select[string]`
  4. "Skip" → return empty
  5. Check CLI dependency → offer skip or show install instructions
  6. For tessl: `tessl search ""` → `parseTesslSearchOutput()` → `huh.MultiSelect[string]` → install each
  7. For install-all: `huh.Confirm` → run adapter bulk install
  8. Print per-skill progress to stderr
  9. Return skill names via `StepResult.Data["skills"]`

**Dependencies**: `internal/skill` (SourceRouter, DirectoryStore, DependencyError, CLIDependency), `charmbracelet/huh`, `internal/tui` (WaveTheme)

**Functions to extract from skills.go for reuse**: `parseTesslSearchOutput()` — currently in `cmd/wave/commands/skills.go`. Since the onboarding package shouldn't import the commands package, the function should be either:
- (a) Duplicated in `skill_step.go` (simple, ~25 lines), or
- (b) Moved to `internal/skill/` as an exported function

Recommendation: (a) Duplicate. The function is small, self-contained, and the onboarding context may diverge from CLI context (e.g., different field handling). Moving it to `internal/skill/` would create coupling for a simple parser.

### Component 2: WizardResult Extension (internal/onboarding/onboarding.go)

**Modifications**:
- Add `Skills []string` field to `WizardResult`
- Add Step 6 invocation in `RunWizard()` between model selection and `writeManifest()`
- Extract skills from `StepResult.Data["skills"]` and set `result.Skills`
- In `buildManifest()`: add `m["skills"] = result.Skills` when `len(result.Skills) > 0`

### Component 3: Step Renumbering (internal/onboarding/steps.go)

**Modifications**:
- Change all `"Step N of 5"` labels to `"Step N of 6"` (6 occurrences in existing steps)

### Component 4: Tests (internal/onboarding/skill_step_test.go, onboarding_test.go)

**New file** `skill_step_test.go` with table-driven tests:

| Test | Scenario |
|------|----------|
| `TestSkillSelectionStep_Name` | Returns "Skill Selection" |
| `TestSkillSelectionStep_NonInteractive_Skips` | Non-interactive mode returns empty skills |
| `TestSkillSelectionStep_Skip_Ecosystem` | User selects "Skip" → empty skills |
| `TestSkillSelectionStep_MissingCLI` | CLI not found → returns empty skills (simulates skip path) |
| `TestSkillSelectionStep_TesslSearchParse` | Parses tessl search output correctly |
| `TestSkillSelectionStep_Reconfigure_ShowsExisting` | Reconfigure mode shows existing skills as context |

**Modified** `onboarding_test.go`:
- `TestBuildManifest_WithSkills` — verify `skills` key in manifest when skills present
- `TestBuildManifest_NoSkills` — verify no `skills` key when empty
- `TestRunWizard_NonInteractive` — verify `result.Skills` is empty (non-interactive skip)

## Complexity Tracking

_No constitution violations. No complexity justifications needed._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| (none) | — | — |
