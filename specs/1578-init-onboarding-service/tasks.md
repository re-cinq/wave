# Work Items

## Phase 1: Setup
- [ ] Item 1.1: Verify branch `1578-init-onboarding-service` checked out
- [ ] Item 1.2: Sanity sweep — `rg "RunWizard|WizardConfig|WizardResult|PrepareWizard"` to enumerate every consumer that will need adaptation; cross-check against plan.md File Mapping

## Phase 2: Core Implementation
- [ ] Item 2.1: Rewrite `cmd/wave/commands/init.go` — replace `runWizardInit`, `runReconfigure`, and `runInit`'s wizard branch with `onboarding.NewBaselineService(stderr).StartSession(ctx, ".", StartOptions{...})`. Keep all cobra flags. Drop `runWizardInit` helper, `printWizardSuccess`, `WaveLogo` injection
- [ ] Item 2.2: Update `cmd/wave/commands/validate_test.go` line 1196 — replace `onboarding.RunWizard(cfg)` with `BaselineService` setup that produces the same artifacts the test asserts on
- [ ] Item 2.3: Delete `internal/onboarding/onboarding.go` content (or whole file if no surviving symbols) — remove `WizardConfig`, `WizardResult`, `WizardStep`, `StepResult`, `DependencyStatus`, `RunWizard`, `writeManifest`, `buildManifest`, `inferTokenScopes`, `getDefaultTierModels`, `suggestSkills`, `autoInstallSkills`
- [ ] Item 2.4: Delete `internal/onboarding/steps.go` [P]
- [ ] Item 2.5: Delete `internal/onboarding/skill_step.go` [P]
- [ ] Item 2.6: Delete `internal/onboarding/wave_command_step.go` [P]
- [ ] Item 2.7: Trim `internal/onboarding/flow.go` — remove `PrepareWizard` + `WizardOpts` + `WizardWaveDirs` (if exported only for wizard) [P]
- [ ] Item 2.8: Trim `internal/onboarding/output.go` — remove `PrintWizardSuccess` [P]

## Phase 3: Testing
- [ ] Item 3.1: Delete `internal/onboarding/onboarding_test.go` (all tests target deleted functions); port any `buildManifest` assertions still relevant to `manifest_build_test.go` first [P]
- [ ] Item 3.2: Delete `internal/onboarding/steps_test.go` [P]
- [ ] Item 3.3: Delete `internal/onboarding/skill_step_test.go` [P]
- [ ] Item 3.4: Delete `internal/onboarding/wave_command_step_test.go` [P]
- [ ] Item 3.5: Add `cmd/wave/commands/init_test.go` (or extend existing) — table-driven coverage of `--yes`, `--reconfigure`, default-interactive (TTY stub), `--merge`. Assert file artifacts + sentinel
- [ ] Item 3.6: `go build ./...` — confirm no stale references
- [ ] Item 3.7: `go vet ./...` clean
- [ ] Item 3.8: `golangci-lint run ./...` clean
- [ ] Item 3.9: `go test -race ./...` green
- [ ] Item 3.10: Smoke run — `mktemp -d`, `cd`, `git init`, `<wave-binary> init --yes`, inspect `wave.yaml`, `.agents/`, `.agents/.onboarding-done`. Confirm parity with previous behaviour minus interactive prompts

## Phase 4: Polish
- [ ] Item 4.1: Final `rg "RunWizard|WizardConfig|WizardResult|WizardStep|DependencyStep|TestConfigStep|PipelineSelectionStep|AdapterConfigStep|ModelSelectionStep|SkillSelectionStep|WaveCommandStep|PrepareWizard|PrintWizardSuccess"` over `cmd/`, `internal/`, `pkg/` — must return zero hits
- [ ] Item 4.2: Commit with conventional prefix `refactor(onboarding):` — note in body that interactive prompts are deliberately removed pending issue 1.2
- [ ] Item 4.3: Push branch, open PR referencing #1578 + epic #1565, paste smoke output in PR description
