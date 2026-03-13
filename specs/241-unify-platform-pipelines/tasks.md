# Tasks: Unify Platform-Specific Pipelines

**Feature**: 241-unify-platform-pipelines
**Generated**: 2026-03-13
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Phase 1: Foundation — Forge Metadata Extension (US-2, US-3)

These tasks extend the forge detection system with the new fields needed by template variables.

- [X] T001 [P1] [US-2] Add `PRTerm` and `PRCommand` fields to `ForgeInfo` struct in `internal/forge/detect.go:20-27`
- [X] T002 [P1] [US-2] Extend `forgeMetadata()` to return 4 values (cli, prefix, prTerm, prCommand) and populate new fields in `Detect()` in `internal/forge/detect.go:181-194` and `internal/forge/detect.go:39-56`
- [X] T003 [P1] [US-2] Update `FilterPipelinesByForge()` to include pipelines without any forge prefix (unified pipelines match all forges) in `internal/forge/detect.go:88-102`
- [X] T004 [P1] [US-2] Update existing forge detection tests for new `PRTerm`/`PRCommand` fields in `internal/forge/detect_test.go`

## Phase 2: Pipeline Context — Forge Variable Injection (US-2)

These tasks enable forge template variables in the pipeline execution context.

- [X] T005 [P1] [US-2] Add `InjectForgeVariables(ctx *PipelineContext, info forge.ForgeInfo)` helper function to `internal/pipeline/context.go`
- [X] T006 [P1] [US-2] Add unit tests for `InjectForgeVariables` — verify all 8 forge variables are injected and round-trip through `ResolvePlaceholders()` in `internal/pipeline/context_test.go`

## Phase 3: Executor Integration — Template Resolution (US-1, US-2, US-3, US-7)

These tasks wire forge variables into the executor and resolve template placeholders at the right points.

- [X] T007 [P1] [US-1] Call `forge.DetectFromGitRemotes()` and `InjectForgeVariables()` in `Execute()` after `newContextWithProject()` (line 276) but before preflight checks in `internal/pipeline/executor.go:276`
- [X] T008 [P1] [US-3] Resolve `step.Persona` through `execution.Context.ResolvePlaceholders()` before `GetPersona()` call in `runStepExecution()` in `internal/pipeline/executor.go:1040`
- [X] T009 [P1] [US-2] Resolve `step.Exec.SourcePath` through `execution.Context.ResolvePlaceholders()` before `os.ReadFile()` in `internal/pipeline/executor.go:1695`
- [X] T010 [P1] [US-7] Resolve `requires.tools` entries through `pipelineContext.ResolvePlaceholders()` and filter empty strings before passing to `preflight.Checker.Run()` in `internal/pipeline/executor.go:248-269`
- [X] T011 [P1] [US-7] Update `CheckTools()` in preflight to silently skip empty strings in the tools list in `internal/preflight/preflight.go:81-86`
- [X] T012 [P1] [US-1] Add executor unit tests for forge variable injection, persona resolution, and tool resolution in `internal/pipeline/executor_test.go`

## Phase 4: Unified Pipeline Definitions — implement Family (US-1, US-4)

Create the unified implement pipeline and its prompt files, replacing 4 platform-specific variants.

- [X] T013 [P1] [US-1] Create unified `implement.yaml` pipeline with `{{ forge.prefix }}-commenter` persona, `{{ forge.cli_tool }}` tools, and `create-pr` step ID in `internal/defaults/pipelines/implement.yaml`
- [X] T014 [P2] [US-4] Create unified `fetch-assess.md` prompt using `{{ forge.cli_tool }}`, `{{ forge.type }}`, `{{ forge.pr_term }}` for platform-specific CLI commands in `internal/defaults/prompts/implement/fetch-assess.md`
- [X] T015 [P] [P2] [US-4] Create unified `plan.md` prompt (already identical across platforms, just update header) in `internal/defaults/prompts/implement/plan.md`
- [X] T016 [P] [P2] [US-4] Create unified `implement.md` prompt (already identical across platforms, just update header) in `internal/defaults/prompts/implement/implement.md`
- [X] T017 [P2] [US-4] Create unified `create-pr.md` prompt using `{{ forge.cli_tool }}`, `{{ forge.pr_command }}`, `{{ forge.pr_term }}` for platform-specific PR/MR creation in `internal/defaults/prompts/implement/create-pr.md`

## Phase 5: Unified Pipeline Definitions — Remaining Families (US-1, US-4, US-5)

Create unified pipelines for scope, research, rewrite, refresh, and pr-review families.

- [X] T018 [P] [P1] [US-1] Create unified `scope.yaml` pipeline with `{{ forge.prefix }}-analyst` / `{{ forge.prefix }}-scoper` personas and inline prompts using `{{ forge.cli_tool }}` in `internal/defaults/pipelines/scope.yaml`
- [X] T019 [P] [P1] [US-1] Create unified `research.yaml` pipeline with `{{ forge.prefix }}-analyst` / `{{ forge.prefix }}-commenter` personas and inline prompts using `{{ forge.cli_tool }}` in `internal/defaults/pipelines/research.yaml`
- [X] T020 [P] [P1] [US-1] Create unified `rewrite.yaml` pipeline with `{{ forge.prefix }}-analyst` / `{{ forge.prefix }}-enhancer` personas and inline prompts using `{{ forge.cli_tool }}` in `internal/defaults/pipelines/rewrite.yaml`
- [X] T021 [P] [P1] [US-1] Create unified `refresh.yaml` pipeline with `{{ forge.prefix }}-analyst` / `{{ forge.prefix }}-enhancer` personas and inline prompts using `{{ forge.cli_tool }}` in `internal/defaults/pipelines/refresh.yaml`
- [X] T022 [P2] [US-5] Create unified `pr-review.yaml` pipeline extending from `gh-pr-review.yaml` with `{{ forge.prefix }}-commenter` for publish step and inline prompts using `{{ forge.cli_tool }}` and `{{ forge.pr_term }}` in `internal/defaults/pipelines/pr-review.yaml`

## Phase 6: Backward Compatibility (US-6)

Implement deprecated name resolution so `gh-implement`, `gl-research`, etc. still work with a deprecation warning.

- [X] T023 [P2] [US-6] Create `ResolveDeprecatedName(name string) (string, bool)` function in new file `internal/pipeline/deprecated.go`
- [X] T024 [P] [P2] [US-6] Add table-driven tests for `ResolveDeprecatedName` covering all forge prefixes and non-prefixed names in `internal/pipeline/deprecated_test.go`
- [X] T025 [P2] [US-6] Integrate `ResolveDeprecatedName()` into pipeline loading in `cmd/wave/commands/run.go` — call before pipeline lookup, log deprecation warning to stderr

## Phase 7: Supporting Systems Update (US-1, US-6)

Update the suggest engine and doctor to handle unified pipeline names.

- [X] T026 [P] [P2] [US-1] Update `resolvePipeline()` in suggest engine to try bare name first (unified pipeline), then forge-prefixed as fallback in `internal/suggest/engine.go:219-231`
- [X] T027 [P] [P2] [US-1] Update `FilterByForge()` in suggest engine to always include non-prefixed pipelines in `internal/suggest/engine.go:244-256`
- [X] T028 [P] [P2] [US-1] Update `classifyPipeline()` in doctor to recognize unified pipeline names as universal in `internal/doctor/optimize.go:276-312`

## Phase 8: Delete Legacy Files (US-1)

Remove the 25 platform-specific pipeline YAMLs and 4 platform-specific prompt directories.

- [X] T029 [P1] [US-1] Delete 4 implement pipeline variants: `bb-implement.yaml`, `gh-implement.yaml`, `gl-implement.yaml`, `gt-implement.yaml` from `internal/defaults/pipelines/`
- [X] T030 [P] [P1] [US-1] Delete 4 scope pipeline variants: `bb-scope.yaml`, `gh-scope.yaml`, `gl-scope.yaml`, `gt-scope.yaml` from `internal/defaults/pipelines/`
- [X] T031 [P] [P1] [US-1] Delete 4 research pipeline variants: `bb-research.yaml`, `gh-research.yaml`, `gl-research.yaml`, `gt-research.yaml` from `internal/defaults/pipelines/`
- [X] T032 [P] [P1] [US-1] Delete 4 rewrite pipeline variants: `bb-rewrite.yaml`, `gh-rewrite.yaml`, `gl-rewrite.yaml`, `gt-rewrite.yaml` from `internal/defaults/pipelines/`
- [X] T033 [P] [P1] [US-1] Delete 4 refresh pipeline variants: `bb-refresh.yaml`, `gh-refresh.yaml`, `gl-refresh.yaml`, `gt-refresh.yaml` from `internal/defaults/pipelines/`
- [X] T034 [P1] [US-1] Delete `gh-pr-review.yaml` from `internal/defaults/pipelines/`
- [X] T035 [P] [P1] [US-1] Delete 4 prompt directories: `bb-implement/`, `gh-implement/`, `gl-implement/`, `gt-implement/` from `internal/defaults/prompts/`

## Phase 9: Testing & Validation (Cross-cutting)

Final validation that all tests pass and unified pipelines work correctly.

- [X] T036 [P1] Run `go test ./...` to verify all existing tests pass with unified pipelines
- [X] T037 [P1] Run `go test -race ./...` to verify no race conditions in forge variable injection
- [X] T038 [P2] Update any existing tests that reference forge-prefixed pipeline names (search for `gh-implement`, `gl-research`, etc. in test files) across `internal/` and `cmd/`
- [X] T039 [P3] [US-7] Verify preflight skips empty strings from unresolved `{{ forge.cli_tool }}` when forge is unknown

## Dependency Graph

```
T001 → T002 → T005 → T006
                ↓
T003 (parallel with T005)
T004 (parallel with T005)
                ↓
T007 → T008, T009, T010 → T012
T011 (parallel with T007)
                ↓
T013 → T014, T015, T016, T017 (parallel)
T018, T019, T020, T021 (parallel, after T007)
T022 (after T007)
                ↓
T023 → T024 (parallel with T025)
T025 (after T023)
T026, T027, T028 (parallel, after T003)
                ↓
T029-T035 (parallel, after T013-T022 unified pipelines created)
                ↓
T036, T037 (after T029-T035)
T038 (parallel with T036)
T039 (after T011)
```

## Task Summary

| Phase | Description | Tasks | Parallelizable |
|-------|-------------|-------|----------------|
| 1 | Forge Metadata Extension | T001-T004 | 2 (T003, T004) |
| 2 | Pipeline Context | T005-T006 | 0 |
| 3 | Executor Integration | T007-T012 | 3 (T008-T010) |
| 4 | Implement Family | T013-T017 | 3 (T015, T016, T017) |
| 5 | Remaining Families | T018-T022 | 4 (T018-T021) |
| 6 | Backward Compatibility | T023-T025 | 1 (T024) |
| 7 | Supporting Systems | T026-T028 | 3 (T026-T028) |
| 8 | Delete Legacy Files | T029-T035 | 6 (T030-T035) |
| 9 | Testing & Validation | T036-T039 | 2 (T038, T039) |
| **Total** | | **39** | **24** |
