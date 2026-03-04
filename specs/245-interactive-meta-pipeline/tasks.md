# Tasks: Interactive Meta-Pipeline Orchestrator (`wave run wave`)

**Feature**: `245-interactive-meta-pipeline`
**Generated**: 2026-03-04
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

---

## Phase 1: Legacy Removal (FR-020) — Breaking Changes

These tasks remove backward-compatibility shims. Must be done first so all subsequent work builds on the clean codebase.

- [X] T001 [P1] [FR-020] Remove `extractYAMLLegacy()` function and fallback call in `internal/pipeline/meta.go`
  - Delete the function at lines 604-630
  - Replace the fallback at line 579 with an error return: `return nil, fmt.Errorf("missing PIPELINE section marker in meta-pipeline output")`
  - Update/fix any tests in `internal/pipeline/meta_test.go` that rely on the legacy extraction path

- [X] T002 [P1] [FR-020] Remove legacy bare template variables in `internal/pipeline/context.go`
  - Delete lines 95-98 (the three `replaceBoth` calls for bare `pipeline_id`, `pipeline_name`, `step_id`)
  - Update/fix any tests in `internal/pipeline/context_test.go` that assert on bare variable substitution

- [X] T003 [P1] [FR-020] Remove legacy exact-name workspace directory lookup in `internal/pipeline/resume.go`
  - Delete lines 211-214 (the `os.Stat` check and prepend for exact-name directory without hash suffix)
  - Update/fix any tests in `internal/pipeline/resume_test.go` that rely on legacy directory matching

- [X] T004 [P1] [FR-020] Run `go test -race ./...` and fix all regressions from legacy removal
  - Every failing test must be fixed or deleted with justification (per Constitution P13)
  - No `t.Skip()` without a linked issue

---

## Phase 2: Foundation — Platform Detection (US3: Platform-Aware Routing)

- [X] T005 [P1] [US3] Create `internal/platform/detect.go` — `PlatformType` enum and `PlatformProfile` struct
  - Define `PlatformType` constants: `PlatformGitHub`, `PlatformGitLab`, `PlatformBitbucket`, `PlatformGitea`, `PlatformUnknown`
  - Define `PlatformProfile` struct per data-model.md (Type, Owner, Repo, APIURL, CLITool, PipelineFamily, AdditionalRemotes)
  - Define `RemoteInfo` struct

- [X] T006 [P1] [US3] Implement `Detect(remoteURL string) PlatformProfile` in `internal/platform/detect.go`
  - Regex-based URL matching for SSH (`git@github.com:owner/repo`) and HTTPS (`https://github.com/owner/repo`) patterns
  - Support GitHub (`github.com`), GitLab (`gitlab.com`), Bitbucket (`bitbucket.org`), Gitea (`.gitea.` domain or `/gitea/` path)
  - Extract owner/repo from URL, set APIURL, CLITool, and PipelineFamily for each platform
  - Return `PlatformUnknown` for unrecognized URLs

- [X] T007 [P1] [US3] Implement `DetectFromGit() (PlatformProfile, error)` in `internal/platform/detect.go`
  - Execute `git remote -v` and parse output
  - Use origin remote as primary; collect additional remotes into `AdditionalRemotes`
  - Handle: no git repo, no remotes, multiple remotes on different platforms

- [X] T008 [P1] [US3] Create `internal/platform/detect_test.go` — comprehensive table-driven tests
  - Test SSH and HTTPS URLs for all 4 platforms
  - Test self-hosted URLs (e.g., `git.example.com`)
  - Test edge cases: `.git` suffix, port numbers, subgroups (GitLab), malformed URLs
  - Test `PlatformUnknown` fallback
  - Test multiple remotes scenario

---

## Phase 3: Health Check Infrastructure (US1: System Health Dashboard)

- [X] T009 [P1] [US1] Create `internal/meta/health.go` — define core health check types
  - Define `HealthReport`, `InitCheckResult`, `DependencyReport`, `DependencyStatus`, `CodebaseMetrics`, `HealthCheckError` structs per data-model.md
  - Define `HealthCheckConfig` with per-check timeouts (default: init 5s, deps 10s, codebase 15s, platform 5s)
  - Define `HealthChecker` interface for testability

- [X] T010 [P1] [US1] Implement `checkInit(ctx context.Context, manifestPath string) InitCheckResult` in `internal/meta/health.go`
  - Check `wave.yaml` existence with `os.Stat`
  - Parse with `manifest.LoadManifest()` to validate
  - Populate WaveVersion from build info, LastConfigDate from file modtime
  - Return error string on failure (not panic)

- [X] T011 [P1] [US1] Implement `checkDependencies(ctx context.Context, manifest *manifest.Manifest) DependencyReport` in `internal/meta/health.go`
  - Iterate all pipeline definitions to collect required tools and skills
  - Use `exec.LookPath()` for CLI tools
  - Use existing `preflight.Checker` patterns for skill availability
  - Set `AutoInstallable` based on whether skill has an `install` command configured

- [X] T012 [P1] [US1] Implement `checkCodebase(ctx context.Context, platform PlatformProfile) CodebaseMetrics` in `internal/meta/health.go`
  - If platform is GitHub and `GITHUB_TOKEN` is set: use `internal/github/client.go` for issue count, PR count, PR status distribution
  - Otherwise: git-local fallback — `git log --oneline --since=7.days.ago`, `git branch -r | wc -l`, `git log -1 --format=%ci`
  - Set `Source` field to `"github_api"` or `"git_local"` accordingly
  - Set `APIAvailable` based on whether API call succeeded

- [X] T013 [P1] [US1] Implement `RunHealthChecks(ctx context.Context, opts HealthCheckConfig) (*HealthReport, error)` in `internal/meta/health.go`
  - Use `errgroup.WithContext()` to run all 4 checks in parallel
  - Each check gets `context.WithTimeout()` per its configured timeout (FR-006)
  - Collect results into `HealthReport`; timed-out checks produce `HealthCheckError` with `Timeout: true`
  - Set overall `Timestamp` and `Duration`

- [X] T014 [P1] [US1] Create `internal/meta/health_test.go` — unit tests for health checks
  - Test `checkInit` with valid manifest, missing manifest, invalid YAML
  - Test `checkDependencies` with mock tool/skill availability
  - Test `checkCodebase` with mocked GitHub client and git-local fallback
  - Test `RunHealthChecks` with timeout scenarios (one check times out, others succeed)
  - Test parallel execution doesn't introduce data races (`-race`)

---

## Phase 4: Health Report Display (US1: System Health Dashboard)

- [X] T015 [P] [P1] [US1] Create `internal/tui/health_report.go` — `RenderHealthReport(report *meta.HealthReport) string`
  - Format init status: manifest found/valid, Wave version, config date
  - Format dependency table: tool/skill name, status (available / missing / auto-installable)
  - Format codebase metrics: commits, issues, PRs, branches, data source
  - Format platform: detected platform, owner/repo, pipeline family
  - Format errors/timeouts if present
  - Use lipgloss styling consistent with existing `internal/tui/theme.go`

- [X] T016 [P] [P1] [US1] Create `internal/tui/health_report_test.go` — test health report rendering
  - Test with full health report (all fields populated)
  - Test with minimal health report (errors, timeouts)
  - Test with empty/zero values

---

## Phase 5: Proposal Engine (US2: Interactive Pipeline Proposal & Selection)

- [X] T017 [P1] [US2] Create `internal/meta/proposal.go` — define proposal types and engine
  - Define `ProposalType` constants: `ProposalSingle`, `ProposalParallel`, `ProposalSequence`
  - Define `PipelineProposal` struct per data-model.md
  - Define `ProposalEngine` struct with dependency on `HealthReport` and discovered pipeline list
  - Define `ProposalSelection` struct: `Proposals []PipelineProposal`, `ModifiedInputs map[string]string`, `ExecutionMode ProposalType`

- [X] T018 [P1] [US2] Implement `GenerateProposals(report *HealthReport, pipelines []string) []PipelineProposal` in `internal/meta/proposal.go`
  - Rule: open issues → `{family}-implement` (single) with issue URL as prefilled input
  - Rule: multiple open issues (>3) → `{family}-implement-epic` (single)
  - Rule: pending PRs with reviews → `{family}-pr-review` or generic review pipeline
  - Rule: recent test failures detected → `wave-bugfix` (single)
  - Rule: low recent commits → `wave-evolve` (single)
  - Rule: open issues + implementation → `{family}-research → {family}-implement` (sequence)
  - Filter proposals where dependencies are not met (`DepsReady: false`, `MissingDeps` populated)
  - Rank by priority (lower number = higher priority)
  - Use platform `PipelineFamily` to select correct pipeline variants (FR-014)

- [X] T019 [P1] [US2] Create `internal/meta/proposal_test.go` — table-driven tests
  - Test GitHub repo with open issues → proposes `gh-implement`
  - Test GitLab repo with open issues → proposes `gl-implement`
  - Test repo with no issues and no PRs → proposes `wave-evolve`
  - Test repo with missing dependencies → proposals marked `DepsReady: false`
  - Test proposal ranking/priority ordering
  - Test sequence proposal generation

---

## Phase 6: Interactive Selection TUI (US2: Interactive Pipeline Proposal & Selection)

- [X] T020 [P1] [US2] Create `internal/tui/proposal_selector.go` — `RunProposalSelector(proposals []PipelineProposal) (*ProposalSelection, error)`
  - Use `huh.NewSelect` for single proposal selection (consistent with `run_selector.go` patterns)
  - Display each proposal with: pipeline name(s), type indicator (single/sequence/parallel), rationale
  - After selection: show pre-filled input with option to edit via `huh.NewText`
  - Support multi-select mode via `huh.NewMultiSelect` for parallel execution
  - Return `ProposalSelection` with execution mode and modified inputs

- [X] T021 [P1] [US2] Create `internal/tui/proposal_selector_test.go` — test proposal selector
  - Test rendering with various proposal sets
  - Test empty proposals list handling (should display helpful message)

---

## Phase 7: CLI Integration & Single Pipeline Dispatch (US2, FR-018)

- [X] T022 [P1] [US2] Create `cmd/wave/commands/wave.go` — `runWave()` orchestrator function
  - Wire together: `RunHealthChecks()` → `RenderHealthReport()` → `GenerateProposals()` → `RunProposalSelector()` → dispatch
  - Accept `manifest *manifest.Manifest` and CLI options
  - Emit structured progress events for each phase (FR-021)

- [X] T023 [P1] [US2] Modify `cmd/wave/commands/run.go` — intercept `pipeline == "wave"` (FR-018)
  - Before `loadPipeline()`, check if pipeline argument is literally `"wave"`
  - If so, dispatch to `runWave()` instead of the normal pipeline execution path
  - Reserve `"wave"` as a keyword — reject `wave.yaml` pipelines named `"wave"`

- [X] T024 [P1] [US2] Implement single pipeline dispatch in `cmd/wave/commands/wave.go`
  - Load selected pipeline YAML via existing `loadPipeline()` mechanism
  - Create executor with `NewDefaultPipelineExecutor()`
  - Set pre-filled/modified input from `ProposalSelection`
  - Run pipeline and report result

---

## Phase 8: Non-Interactive Mode (FR-019, Edge Cases)

- [X] T025 [P] [P1] [FR-019] Add non-interactive mode to `cmd/wave/commands/wave.go`
  - Detect non-TTY using existing `isInteractive()` function
  - In non-interactive mode: run health checks, serialize `HealthReport` as JSON to stdout, exit
  - Add `--proposal` flag: auto-select a specific proposal by index or pipeline name without interactive menu

- [X] T026 [P] [P1] [FR-019] Add edge case handling to meta-orchestrator
  - Missing `wave.yaml` → clear error: "No wave.yaml found. Run `wave init` to initialize."
  - No git remote → platform detection returns `PlatformUnknown`, proposals use generic pipelines
  - All pipelines filtered (deps missing) → display health report + "No runnable pipelines" message
  - Ctrl+C / ESC → clean exit via existing signal handling (no extra work needed)
  - Health check timeout → partial results with timeout indicators in report

---

## Phase 9: Dependency Auto-Installation (US4)

- [X] T027 [P2] [US4] Add auto-install capability to `internal/meta/install.go`
  - After dependency audit, for each missing dependency with `AutoInstallable: true`:
    prompt user for confirmation, then run the install command
  - Report install success/failure
  - On failure: continue with degraded proposal set (exclude pipelines requiring the missing dep)

- [X] T028 [P2] [US4] Integrate auto-install into `cmd/wave/commands/wave.go`
  - After health report display, before proposal generation
  - Offer to install missing auto-installable dependencies
  - Re-run dependency check after installations to update `DependencyReport`

---

## Phase 10: Pipeline Composition & Chaining (US5)

- [X] T029 [P2] [US5] Create `internal/meta/sequence.go` — `SequenceExecutor` struct and types
  - Define `SequenceExecutor`, `SequenceResult`, `SequencePipelineResult` per data-model.md
  - Accept `executorFactory func() *pipeline.DefaultPipelineExecutor` for testability

- [X] T030 [P2] [US5] Implement `SequenceExecutor.Execute(ctx, pipelineNames []string, manifest, input)` in `internal/meta/sequence.go`
  - Run each pipeline sequentially via `NewChildExecutor()`
  - After each pipeline completes: copy output artifacts from its workspace to next pipeline's `.wave/artifacts/`
  - Track `ArtifactPaths` from each `PipelineExecution` for handoff
  - On failure: halt sequence, report which pipeline and step failed, return `SequenceResult` with `FailedAt` index
  - Validate artifact existence after copy (no silent loss)

- [X] T031 [P2] [US5] Implement parallel multi-pipeline dispatch in `cmd/wave/commands/wave.go`
  - When `ProposalSelection.ExecutionMode == ProposalParallel`:
    spawn `NewChildExecutor()` per pipeline in `errgroup.WithContext()`
  - Each pipeline gets independent workspace and state tracking
  - Follow `MatrixExecutor` pattern from `internal/pipeline/matrix.go`
  - Report results after all pipelines complete (or fail)

- [X] T032 [P2] [US5] Create `internal/meta/sequence_test.go` — test sequence executor
  - Test successful two-pipeline sequence with artifact handoff
  - Test failure in first pipeline halts sequence
  - Test artifact copy validation (missing artifact detected)
  - Test with mocked executors to avoid real adapter calls

---

## Phase 11: Codebase Auto-Tuning (US6)

- [X] T033 [P3] [US6] Create `internal/meta/tuning.go` — `CodebaseProfile` struct and `AnalyzeCodebase()` function
  - Define `SizeClass` constants and `CodebaseProfile` struct per data-model.md
  - Detect primary language from marker files (`go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, `Gemfile`, `pom.xml`)
  - Detect framework from dependency files
  - Classify project size by file count / LOC estimation
  - Detect test infrastructure (test commands, config files)
  - Detect monorepo structure (multiple `go.mod`, workspace files)

- [X] T034 [P3] [US6] Implement auto-tuning integration in `cmd/wave/commands/wave.go`
  - After health checks: run `AnalyzeCodebase()` to generate `CodebaseProfile`
  - Use profile to augment persona prompts with project-specific context (FR-015)
  - Suggest `wave.yaml` `project` section additions (never overwrite existing per FR-017)
  - Create platform-specific configs rather than modifying generics (FR-016)

- [X] T035 [P3] [US6] Create `internal/meta/tuning_test.go` — test auto-tuning
  - Test Go project detection (finds `go.mod`, sets `TestCommand: "go test ./..."`)
  - Test Python project detection (finds `pyproject.toml`, sets pytest conventions)
  - Test monorepo detection (multiple `go.mod` files)
  - Test unknown project (no recognized markers → sensible defaults)

---

## Phase 12: Observability & Polish (FR-021, Cross-Cutting)

- [X] T036 [P] [P2] [FR-021] Add structured progress events to meta-orchestrator
  - Emit events: `meta.health_started`, `meta.health_completed`, `meta.proposals_generated`, `meta.proposal_selected`, `meta.pipeline_dispatched`, `meta.sequence_started`, `meta.sequence_completed`
  - Use existing `event.EventEmitter` interface
  - Include relevant data in each event (duration, pipeline names, selection details)

- [X] T037 [P] [P2] Run final `go test -race ./...` and verify all tests pass
  - Ensure no data races in parallel health checks
  - Ensure no data races in parallel pipeline dispatch
  - Ensure legacy removal hasn't introduced regressions
  - Verify test coverage for new packages (`internal/meta/`, `internal/platform/`)

---

## Task Summary

| Phase | Tasks | User Story | Priority |
|-------|-------|------------|----------|
| 1: Legacy Removal | T001-T004 | FR-020 | P1 |
| 2: Platform Detection | T005-T008 | US3 | P1 |
| 3: Health Checks | T009-T014 | US1 | P1 |
| 4: Health Report TUI | T015-T016 | US1 | P1 |
| 5: Proposal Engine | T017-T019 | US2 | P1 |
| 6: Selection TUI | T020-T021 | US2 | P1 |
| 7: CLI + Dispatch | T022-T024 | US2 | P1 |
| 8: Non-Interactive | T025-T026 | FR-019 | P1 |
| 9: Auto-Install | T027-T028 | US4 | P2 |
| 10: Composition | T029-T032 | US5 | P2 |
| 11: Auto-Tuning | T033-T035 | US6 | P3 |
| 12: Polish | T036-T037 | FR-021 | P2 |

**Total**: 37 tasks
**Parallelizable**: T015-T016 (Phase 4) can run parallel with Phase 3. T025-T026 (Phase 8) can run parallel with Phase 7. T036-T037 (Phase 12) can run parallel with Phase 11.

## Dependency Graph

```
T001─┐
T002─┼─→ T004 ─→ T005─┐
T003─┘              T006─┼─→ T009─┐
                    T007─┤    T010─┤
                    T008─┘    T011─┼─→ T013 ─→ T014 ─→ T017─┐
                              T012─┘                     T018─┼─→ T020─┐
                                        T015─┐           T019─┘    T021─┘
                                        T016─┘                        │
                                                                      ▼
                                                         T022 ─→ T023 ─→ T024
                                                                   │
                                                         ┌─────────┼─────────┐
                                                         ▼         ▼         ▼
                                                     T025─┐   T027─┐    T029─┐
                                                     T026─┘   T028─┘    T030─┤
                                                                        T031─┤
                                                                        T032─┘
                                                                           │
                                                              T033─┐       │
                                                              T034─┤       │
                                                              T035─┘       │
                                                                    ▼      ▼
                                                                T036 ─→ T037
```
