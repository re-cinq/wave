# Research: Wave Init Interactive Skill Selection

**Feature Branch**: `384-init-skill-selection`
**Date**: 2026-03-14

## Research Questions

### RQ-1: How does the existing wizard step architecture work?

**Decision**: Implement `SkillSelectionStep` as a `WizardStep` interface conformant struct.

**Rationale**: All existing wizard steps follow the same pattern:
- Struct implementing `WizardStep` interface (`Name() string`, `Run(cfg *WizardConfig) (*StepResult, error)`)
- Non-interactive path returns defaults immediately
- Interactive path uses `huh` form library with `tui.WaveTheme()`
- Results passed back via `StepResult.Data` map
- Orchestrator in `RunWizard()` extracts typed values from `Data` map into `WizardResult` fields

**Evidence**: `internal/onboarding/steps.go` — all 5 steps follow this pattern. Step labels hardcoded as `"Step N of 5"`.

**Alternatives Rejected**:
- Custom step outside the `WizardStep` interface — would break the consistent architecture
- Post-wizard hook — would not participate in manifest generation

### RQ-2: How do ecosystem source adapters work?

**Decision**: Two interaction modes — multi-select for tessl, confirm/skip for install-all ecosystems.

**Rationale**: The `SourceAdapter` interface exposes only `Install(ctx, ref, store)` — no `List()` method. Only tessl has an external registry search capability:
- **tessl**: `tessl search ""` lists all skills, `tessl install <ref>` installs individual skills
- **BMAD**: `npx bmad-method install --tools claude-code --yes` installs everything
- **OpenSpec**: `openspec init` installs everything
- **Spec-Kit**: `specify init` installs everything

The tessl adapter installs to a temp dir, discovers `SKILL.md` files, then parses and writes to the store. Install-all adapters do the same but don't support individual skill references.

**Evidence**: `internal/skill/source_cli.go` — `TesslAdapter.Install()` takes a `ref` param, while `BMADAdapter.Install()`, `OpenSpecAdapter.Install()`, and `SpecKitAdapter.Install()` ignore their `ref` parameter.

**Alternatives Rejected**:
- Adding `List()` to `SourceAdapter` — invasive change to an interface with 7 implementations, not all of which support listing
- Treating all ecosystems as install-all — loses the tessl-specific individual selection capability

### RQ-3: How should tessl search results populate the multi-select?

**Decision**: Reuse the `parseTesslSearchOutput()` function from `cmd/wave/commands/skills.go` and the same `tessl search ""` subprocess pattern.

**Rationale**: `parseTesslSearchOutput()` already parses tessl's tab/space-separated output into `SkillSearchResult{Name, Rating, Description}` structs. The same `exec.CommandContext(ctx, "tessl", "search", query)` pattern is used in `runSkillsSearch()`.

For the onboarding step, we'll call `tessl search ""` to get all available skills, parse them into `SkillSearchResult` items, then build `huh.MultiSelect` options from them.

**Evidence**: `cmd/wave/commands/skills.go:401` — `parseTesslSearchOutput` and `cmd/wave/commands/skills.go:389` — subprocess invocation.

**Alternatives Rejected**:
- Creating a new parser — duplication of working code
- Calling through the `SourceRouter` — the router only does install, not search

### RQ-4: How should `WizardResult` flow skills to `buildManifest()`?

**Decision**: Add `Skills []string` field to `WizardResult`. `buildManifest()` adds `"skills"` key when non-empty.

**Rationale**: The `Manifest.Skills` field is `[]string` (bare names, not source-prefixed). The existing `WizardResult` → `buildManifest()` pattern is straightforward: typed fields on `WizardResult` map directly to manifest keys. Adding `Skills []string` follows the same pattern as `Pipelines []string`.

**Evidence**: `internal/manifest/types.go:23` — `Skills []string yaml:"skills,omitempty"`. `internal/onboarding/onboarding.go:162` — `buildManifest()` constructs a `map[string]interface{}`.

**Alternatives Rejected**:
- Storing source-prefixed names — doesn't match `Manifest.Skills` format
- Writing skills to manifest separately after wizard — breaks the single `writeManifest()` call pattern

### RQ-5: How to handle missing CLI dependencies gracefully?

**Decision**: Use existing `checkDependency()` / `DependencyError` pattern. When detected, show a choice: "Skip" or "Show install instructions".

**Rationale**: The `skill` package already has `checkDependency(dep CLIDependency, lookPath lookPathFunc)` which returns `*DependencyError` with `Binary` and `Instructions` fields. The CLI layer already classifies these via `classifySkillError()`. In the wizard context, we check before attempting install and present a user-friendly choice.

**Evidence**: `internal/skill/source_cli.go:18-27` — `checkDependency`, `internal/skill/source.go:37-51` — `DependencyError`, `CLIDependency`.

**Alternatives Rejected**:
- Silently skipping — violates FR-007 requirement for user notification
- Failing the wizard — too harsh for an optional feature

### RQ-6: Step numbering strategy

**Decision**: Renumber all existing step labels from "Step N of 5" to "Step N of 6". New skill selection step is "Step 6 of 6".

**Rationale**: Step labels are hardcoded strings in each step's `Run()` method. Six occurrences to update (Steps 1-5 in their respective `Run()` methods). The new step inserts between model selection (Step 5) and `writeManifest()`.

**Evidence**: `internal/onboarding/steps.go` — grep for `"Step N of 5"`: lines 72, 141, 335, 394, 487, 525.

**Alternatives Rejected**:
- Dynamic step numbering — over-engineering for 6 steps; adds complexity without clear benefit
- Not renumbering — confusing UX ("Step 5 of 5" followed by "Step 6")

### RQ-7: huh v0.8.0 MultiSelect API

**Decision**: Use `huh.NewMultiSelect[string]()` with `.Options()`, `.Value()`, and `.Height()`.

**Rationale**: The codebase already uses `huh.NewMultiSelect[string]()` in `PipelineSelectionStep` (steps.go:330). The same pattern works for skill selection. `huh.MultiSelect` supports built-in keyboard filtering in v0.8.0.

**Evidence**: `internal/onboarding/steps.go:330-334` — existing `huh.NewMultiSelect[string]()` usage. `go.mod:8` — `github.com/charmbracelet/huh v0.8.0`.

**Alternatives Rejected**:
- Custom list rendering — loses huh theme integration and filtering
- huh.Select for single skill at a time — poor UX for multi-selection

### RQ-8: Installation progress feedback

**Decision**: Use `fmt.Fprintf(os.Stderr, ...)` with per-skill status updates during installation, consistent with existing wizard step output patterns.

**Rationale**: The wizard already writes progress to stderr (see `DependencyStep.Run()` which uses `fmt.Fprintf(os.Stderr, ...)`). For skill installation, iterate over selected skills, print "Installing <name>...", then print success/failure. The `SourceAdapter.Install()` return value includes `InstallResult.Skills` and `InstallResult.Warnings` for status tracking.

**Evidence**: `internal/onboarding/steps.go:72-81` — stderr output pattern in `DependencyStep`.

**Alternatives Rejected**:
- Spinner/progress bar — overkill for sequential installs; adds external dependency
- Silent installation — violates FR-004 requirement for progress feedback
