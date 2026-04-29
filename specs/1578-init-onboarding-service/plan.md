# Plan: 1578-init-onboarding-service

## Objective

Replace `RunWizard`-driven `wave init` with a `BaselineService.StartSession` call, then delete the legacy wizard machinery (`RunWizard` + `*Step` types + supporting tests). Keep `flavour.go`, `metadata.go`, `manifest_build.go`, `flow.go::Greenfield`, `service.go`, `baseline_service.go`, `state.go`, `scaffold.go`.

## Approach

1. Rewrite `cmd/wave/commands/init.go` so all four code paths (default-interactive, `--yes`, `--merge`, `--reconfigure`) route through `onboarding.NewBaselineService(stderr).StartSession(ctx, ".", StartOptions{...})` (or the existing `runMerge` helper that already lives outside the wizard for merge).
2. Delete `RunWizard` and the `WizardConfig` / `WizardResult` / `WizardStep` / `StepResult` / `DependencyStatus` types from `internal/onboarding/onboarding.go`. Delete supporting helpers that become orphaned: `writeManifest`, `buildManifest`, `inferTokenScopes`, `getDefaultTierModels`, `suggestSkills`, `autoInstallSkills`. (Verified non-shared: only `RunWizard` consumes them.)
3. Delete `internal/onboarding/steps.go`, `skill_step.go`, `wave_command_step.go`, plus their `*_test.go` siblings.
4. Delete `internal/onboarding/onboarding_test.go` (entirely about `RunWizard` + `buildManifest`).
5. Trim `internal/onboarding/flow.go::PrepareWizard` and `WizardOpts` (orphaned after step 2).
6. Trim `internal/onboarding/output.go::PrintWizardSuccess` (orphaned).
7. Update `cmd/wave/commands/validate_test.go` (single `RunWizard` call site at line 1196) to call `BaselineService` instead.
8. Smoke `wave init --yes` on a fresh `t.TempDir()` and inspect resulting `wave.yaml`, `.agents/`, `.agents/.onboarding-done` per memory: real verification = drive the user surface, not just `go test`.

## File Mapping

| Path | Action | Notes |
|------|--------|-------|
| `cmd/wave/commands/init.go` | **modify** | Drop `runWizardInit`, replace `RunWizard` calls in `runInit`/`runReconfigure` with `BaselineService.StartSession`. Keep cobra flags (`--force`, `--merge`, `--all`, `--adapter`, `--workspace`, `--manifest-path`, `--yes`, `--reconfigure`). Drop wizard-specific TUI prompt + huh logo. |
| `cmd/wave/commands/init_test.go` | **modify or add** | Tests the new init paths against `BaselineService` (use `NoopUI`). |
| `cmd/wave/commands/validate_test.go` | **modify** | Replace `onboarding.RunWizard(cfg)` call site with `BaselineService` setup that emits the same on-disk artifacts. |
| `internal/onboarding/onboarding.go` | **modify (gut)** | Remove all wizard types + helpers. File becomes either deleted or trimmed to `// Package doc + nothing`. Prefer deletion. |
| `internal/onboarding/onboarding_test.go` | **delete** | All tests target `RunWizard` and `buildManifest`. |
| `internal/onboarding/steps.go` | **delete** | All wizard `*Step` implementations. |
| `internal/onboarding/steps_test.go` | **delete** | Tests for the deleted steps. |
| `internal/onboarding/skill_step.go` | **delete** | Wizard step. |
| `internal/onboarding/skill_step_test.go` | **delete** | Tests for the deleted step. |
| `internal/onboarding/wave_command_step.go` | **delete** | Wizard step. |
| `internal/onboarding/wave_command_step_test.go` | **delete** | Tests for the deleted step. |
| `internal/onboarding/flow.go` | **modify** | Remove `PrepareWizard` + `WizardOpts`. Keep `Greenfield` + `GreenfieldOpts`. |
| `internal/onboarding/output.go` | **modify** | Remove `PrintWizardSuccess`. Keep `PrintInitSuccess`, `PrintMergeSuccess`, `SuggestFirstRun`. |
| `internal/onboarding/state.go` | **keep** | `MarkOnboarded` / `ClearOnboarding` already used by sentinel write paths. |
| `internal/onboarding/flavour.go` | **keep** | Re-used by service. |
| `internal/onboarding/metadata.go` | **keep** | Re-used by service. |
| `internal/onboarding/scaffold.go` | **keep** | `CreateInitialCommit`, `RemoveDeselectedPipelines` (RemoveDeselectedPipelines becomes orphan candidate — verify no other call sites; if orphan, delete). |
| `internal/onboarding/service.go` | **keep** | Domain interface. |
| `internal/onboarding/baseline_service.go` | **keep** | Already implements `Service`. |
| `internal/onboarding/manifest_build.go` | **keep** | `BuildDefaultManifest` consumed by `Greenfield`. |
| `internal/onboarding/diff.go` | **keep** | `ApplyChanges`/`ComputeChangeSummary`/`DisplayChangeSummary` used by `runMerge`. |
| `internal/onboarding/skill_step.go` orphan helpers? | **verify** | If `suggestSkills` survives in another file (check), preserve. |

## Architecture Decisions

- **Stop-gap for interactive prompts**: until issue 1.2 lands `InteractiveService`, the default-interactive path falls back to `BaselineService` (defaults-only, `NoopUI`). The wizard's per-step `huh` forms vanish in this PR. Trade-off: short window where `wave init` no longer asks the operator anything; mitigated by issue 1.2 landing in the same epic.
- **No `--interactive` flag added**: existing behaviour (TTY-and-not-`--yes` ⇒ interactive) is preserved at the cobra layer; the body simply routes both branches through the service. We do not introduce a flag the issue describes hypothetically.
- **`runMerge` left untouched**: merge already lives outside the wizard machinery (it operates on `LoadAssets` + `BuildDefaultManifest` directly). No changes required there.
- **`getDefaultTierModels` deletion**: `BuildDefaultManifest` already builds tier_models in its own path (`manifest_build.go`); the wizard's copy is redundant. Verified by ripgrep.
- **No backward-compat for the wizard**: per memory, no legacy/deprecated support pre-1.0 — straight delete.

## Risks

| Risk | Mitigation |
|------|------------|
| Hidden cross-package use of deleted symbols | `go build ./...` plus a fresh `rg "RunWizard|WizardConfig|WizardResult|WizardStep|DependencyStep|TestConfigStep|PipelineSelectionStep|AdapterConfigStep|ModelSelectionStep|SkillSelectionStep|WaveCommandStep|PrepareWizard"` after the cuts. |
| Removed wizard prompts surprise users mid-epic | `wave init --yes` already ships defaults; the surprise window is narrow + deliberately bridged by 1.2. Note in commit body. |
| `onboarding_test.go` deletion drops coverage of `buildManifest` semantics | Coverage moves to `BuildDefaultManifest` in `manifest_build_test.go` (already exists). Spot-check that the asserted invariants are mirrored. If any uncovered, port the assertion to `manifest_build_test.go`. |
| `validate_test.go` regression | Adapt the call site to use `BaselineService` so the test still produces the same on-disk artifacts. |
| Sentinel path mismatch (`.agents/.onboarding-done` vs `.wave/.onboarded`) | `MarkDoneAt` writes `.agents/.onboarding-done`; `MarkOnboarded` writes `.wave/.onboarded`. Keep the two distinct — the new path is the future, the old path is legacy until callers migrate. Don't conflate. |

## Testing Strategy

- **Unit**: `cmd/wave/commands/init_test.go` (new) — table-driven: `--yes`, `--reconfigure`, default-interactive (with `IsInteractive` stubbed), `--merge` happy-path. Each variant asserts: file written, sentinel present, no `RunWizard` import.
- **Service-level**: `BaselineService` already has tests in `baseline_service_test.go` — verify they still pass.
- **Integration smoke**: shell-level `wave init --yes` in a fresh `mktemp -d`. Inspect `wave.yaml`, `.agents/personas/`, `.agents/.onboarding-done`, `.git/HEAD`. Required by memory ("real verification").
- **Lint/build**: `go build ./...`, `go vet ./...`, `golangci-lint run ./...`, `go test -race ./...`.

## Out of Scope

- `InteractiveService` implementation (issue 1.2).
- WebUI driver (`internal/webui/handlers_onboard.go`) — separate work item.
- `BuildDefaultManifest` cleanup unrelated to wizard removal.
- Sentinel path consolidation (`.agents/.onboarding-done` vs `.wave/.onboarded`) — explicitly deferred.
