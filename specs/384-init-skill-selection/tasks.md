# Tasks: Wave Init Interactive Skill Selection

**Feature Branch**: `384-init-skill-selection`
**Generated**: 2026-03-14
**Spec**: `specs/384-init-skill-selection/spec.md`
**Plan**: `specs/384-init-skill-selection/plan.md`

---

## Phase 1: Setup & Scaffolding

- [X] T001 [P1] Create `internal/onboarding/skill_step.go` with `EcosystemDef` struct, `ecosystems` package-level slice (tessl, BMAD, OpenSpec, Spec-Kit), `lookPathFunc`/`commandRunner` type aliases, and `SkillSelectionStep` struct with injectable dependencies
- [X] T002 [P1] Implement `SkillSelectionStep.Name()` returning `"Skill Selection"` and stub `Run()` method returning empty `StepResult` in `internal/onboarding/skill_step.go`

## Phase 2: Foundational — WizardResult Extension & Step Renumbering

- [X] T003 [P1] Add `Skills []string` field to `WizardResult` in `internal/onboarding/onboarding.go`
- [X] T004 [P1] Renumber all step labels from `"Step N of 5"` to `"Step N of 6"` in `internal/onboarding/steps.go` (6 occurrences: lines 72, 141, 335, 392, 486, 525)
- [X] T005 [P1] Add Step 6 invocation block in `RunWizard()` in `internal/onboarding/onboarding.go` — create `SkillSelectionStep`, call `Run()`, extract `skills` from `StepResult.Data["skills"]` into `result.Skills`
- [X] T006 [P1] Update `buildManifest()` in `internal/onboarding/onboarding.go` — add `m["skills"] = result.Skills` when `len(result.Skills) > 0`

## Phase 3: User Story 1 — Ecosystem Selection During Init (P1)

- [X] T007 [P1] [US1] Implement non-interactive skip path in `SkillSelectionStep.Run()` — when `!cfg.Interactive`, return immediately with empty skills in `internal/onboarding/skill_step.go`
- [X] T008 [P1] [US1] Implement ecosystem selection form using `huh.NewSelect[string]()` with options for tessl, BMAD, OpenSpec, Spec-Kit, and Skip, themed with `tui.WaveTheme()`, labeled `"Step 6 of 6 — Skill Selection"` in `internal/onboarding/skill_step.go`
- [X] T009 [P1] [US1] Implement "Skip" selection handling — when user selects skip, return immediately with empty `StepResult.Data["skills"]` in `internal/onboarding/skill_step.go`

## Phase 4: User Story 2 — Skill Browsing and Multi-Selection (P1)

- [X] T010 [P1] [US2] Implement `parseTesslSearchOutput()` function in `internal/onboarding/skill_step.go` — duplicate from `cmd/wave/commands/skills.go`, parses tab/space-separated tessl output into `skillSearchResult{Name, Description}` structs
- [X] T011 [P1] [US2] Implement tessl skill listing — run `tessl search ""` via `commandRunner`, parse output with `parseTesslSearchOutput()`, build `huh.NewMultiSelect[string]()` options with skill names and descriptions in `internal/onboarding/skill_step.go`
- [X] T012 [P1] [US2] Implement install-all ecosystem confirmation — for BMAD/OpenSpec/Spec-Kit, show `huh.NewConfirm()` describing bulk install behavior, return empty skills if declined in `internal/onboarding/skill_step.go`

## Phase 5: User Story 3 — Skill Installation with Progress Feedback (P2)

- [X] T013 [P2] [US3] Implement tessl per-skill installation loop — for each selected skill, print "Installing <name>..." to stderr, call `SourceRouter.Install(ctx, "tessl:<name>", store)`, print success/failure status in `internal/onboarding/skill_step.go`
- [X] T014 [P2] [US3] Implement install-all ecosystem execution — call `SourceRouter.Install(ctx, "<prefix>:", store)`, print per-skill results from `InstallResult.Skills`, handle warnings in `internal/onboarding/skill_step.go`
- [X] T015 [P2] [US3] Collect installed skill names from `InstallResult.Skills` and return as `StepResult.Data["skills"]` (bare names, not prefixed) in `internal/onboarding/skill_step.go`

## Phase 6: User Story 4 — Graceful Handling of Missing CLI (P2)

- [X] T016 [P2] [US4] Implement CLI dependency check before installation — use `lookPathFunc` to check `EcosystemDef.Dep.Binary`, show missing dependency message with `EcosystemDef.Dep.Instructions`, offer "Skip" or "Show install instructions" choice via `huh.NewSelect[string]()` in `internal/onboarding/skill_step.go`

## Phase 7: User Story 5 — Reconfigure Preserves Skill Context (P3)

- [X] T017 [P3] [US5] Implement reconfigure context display — when `cfg.Reconfigure && cfg.Existing != nil`, show currently installed skill names from `cfg.Existing.Skills` as context before ecosystem selection in `internal/onboarding/skill_step.go`
- [X] T018 [P3] [US5] Merge existing skills with newly installed skills — when reconfiguring, final `result.Skills` is the union of `cfg.Existing.Skills` and newly installed skills (deduped) in `internal/onboarding/skill_step.go`

## Phase 8: Tests

- [X] T019 [P1] [P] Create `internal/onboarding/skill_step_test.go` with `TestSkillSelectionStep_Name` verifying `Name()` returns `"Skill Selection"`
- [X] T020 [P1] [P] Add `TestSkillSelectionStep_NonInteractive_Skips` — non-interactive mode returns empty `skills` in `StepResult.Data` in `internal/onboarding/skill_step_test.go`
- [X] T021 [P1] [P] Add `TestParseTesslSearchOutput` — table-driven tests verifying parsing of tessl search output (empty, single, multi-line, malformed) in `internal/onboarding/skill_step_test.go`
- [X] T022 [P1] [P] Add `TestSkillSelectionStep_MissingCLI` — inject `lookPathFunc` that returns error, verify step handles gracefully (returns empty skills in non-interactive) in `internal/onboarding/skill_step_test.go`
- [X] T023 [P2] [P] Add `TestBuildManifest_WithSkills` — verify `buildManifest()` includes `skills` key when `result.Skills` is non-empty in `internal/onboarding/onboarding_test.go`
- [X] T024 [P2] [P] Add `TestBuildManifest_NoSkills` — verify `buildManifest()` omits `skills` key when `result.Skills` is empty in `internal/onboarding/onboarding_test.go`
- [X] T025 [P3] [P] Add `TestSkillSelectionStep_ReconfigureShowsExisting` — verify reconfigure mode with existing skills doesn't error in non-interactive path in `internal/onboarding/skill_step_test.go`

## Phase 9: Polish & Cross-Cutting Concerns

- [X] T026 [P2] Handle `huh.ErrUserAborted` (Ctrl+C) at all form interaction points — return `fmt.Errorf("wizard cancelled by user")` consistently in `internal/onboarding/skill_step.go`
- [X] T027 [P2] Handle network failure during `tessl search` — if command fails, show error and offer skip/retry choice in `internal/onboarding/skill_step.go`
- [X] T028 [P3] Ensure `.wave/skills/` directory is created before installation if it doesn't exist — use `os.MkdirAll()` in `internal/onboarding/skill_step.go`

---

## Dependency Graph

```
T001 → T002 → T007, T008, T009
T003 → T005 → T006
T004 (independent)
T008 → T010 → T011
T008 → T012
T011 → T013
T012 → T014
T013, T014 → T015
T008 → T016
T007 → T017 → T018
T002 → T019, T020
T010 → T021
T016 → T022
T006 → T023, T024
T017 → T025
T008 → T026
T011 → T027
T015 → T028
```

## Parallelization Opportunities

Tasks marked [P] can run in parallel within their phase:
- **Phase 2**: T003 and T004 are independent of each other
- **Phase 3**: T007 and T008 can start in parallel after T002
- **Phase 8**: T019, T020, T021, T022 can all run in parallel; T023 and T024 can run in parallel

## Summary

| Metric | Value |
|--------|-------|
| Total tasks | 28 |
| P1 (critical) | 14 |
| P2 (important) | 9 |
| P3 (nice-to-have) | 5 |
| Files created | 2 (`skill_step.go`, `skill_step_test.go`) |
| Files modified | 2 (`onboarding.go`, `steps.go`) |
| Parallel opportunities | 8 tasks |
