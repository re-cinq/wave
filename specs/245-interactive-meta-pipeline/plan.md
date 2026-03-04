# Implementation Plan: Interactive Meta-Pipeline Orchestrator (`wave run wave`)

**Branch**: `245-interactive-meta-pipeline` | **Date**: 2026-03-04 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/245-interactive-meta-pipeline/spec.md`

## Summary

Implement `wave run wave` as an interactive meta-orchestrator that performs parallel health checks (init, deps, codebase, platform), proposes pipelines based on codebase state, and dispatches selected pipelines — including parallel multi-pipeline and sequential chained execution. This replaces the need for users to manually select pipelines based on their repository state. Three legacy backward-compatibility shims are removed as part of the breaking-change release.

## Technical Context

**Language/Version**: Go 1.25+ (existing project)  
**Primary Dependencies**: `golang.org/x/sync/errgroup` (existing), `github.com/charmbracelet/huh` (existing), `github.com/spf13/cobra` (existing), `gopkg.in/yaml.v3` (existing), `golang.org/x/term` (existing)  
**Storage**: SQLite for pipeline state (existing `internal/state/`), filesystem for workspaces and artifacts  
**Testing**: `go test -race ./...`  
**Target Platform**: Linux (primary), macOS, Windows  
**Project Type**: Single binary CLI  
**Performance Goals**: Health checks complete within 30 seconds (SC-001). Interactive selection in ≤3 steps (SC-005).  
**Constraints**: Single static binary. No new runtime dependencies. Breaking change — no backward compatibility.  
**Scale/Scope**: ~46 existing pipelines, 4 platform families (gh/gl/bb/gt)

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ Pass | No new runtime dependencies — all imports are existing Go dependencies |
| P2: Manifest as SSOT | ✅ Pass | `wave run wave` reads from `wave.yaml` for adapter/persona/skill configuration |
| P3: Persona-Scoped Execution | ✅ Pass | Dispatched pipelines still execute persona-scoped steps. The meta-orchestrator itself doesn't invoke adapters |
| P4: Fresh Memory | ✅ Pass | Each dispatched pipeline starts fresh via `NewChildExecutor()` |
| P5: Navigator-First | ⚠️ N/A | The meta-orchestrator is not a pipeline — it's a CLI handler. Dispatched pipelines retain navigator-first requirements |
| P6: Contracts at Handover | ✅ Pass | Dispatched pipelines retain contract validation. Cross-pipeline artifact handoff validates file existence |
| P7: Relay via Summarizer | ✅ Pass | No impact — dispatched pipelines retain relay behavior |
| P8: Ephemeral Workspaces | ✅ Pass | Each dispatched pipeline gets its own workspace. SequenceExecutor copies artifacts between independent workspaces |
| P9: Credentials Never Touch Disk | ✅ Pass | GitHub token via env var (`GITHUB_TOKEN`), passed to existing `internal/github/` client |
| P10: Observable Progress | ✅ Pass | Health check phases emit structured events. Meta-orchestrator emits start/selection/dispatch events |
| P11: Bounded Recursion | ✅ Pass | Meta-orchestrator does not recurse. Dispatched pipelines inherit existing bounds from `MetaConfig` |
| P12: Minimal Step State Machine | ✅ Pass | No changes to the 5-state machine |
| P13: Test Ownership | ✅ Pass | All changes require `go test -race ./...` to pass. Legacy removal must fix or delete affected tests |

**Post-design re-check**: No violations found. P5 (Navigator-First) is N/A because the meta-orchestrator is a CLI-level dispatcher, not a pipeline step.

## Project Structure

### Documentation (this feature)

```
specs/245-interactive-meta-pipeline/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── contracts/           # Phase 1 output (JSON schemas for new types)
│   ├── health-report.schema.json
│   └── pipeline-proposal.schema.json
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```
internal/
├── meta/                      # NEW — Meta-pipeline orchestrator
│   ├── health.go              # HealthReport, RunHealthChecks()
│   ├── health_test.go
│   ├── proposal.go            # ProposalEngine, GenerateProposals()
│   ├── proposal_test.go
│   ├── sequence.go            # SequenceExecutor, Execute()
│   ├── sequence_test.go
│   ├── tuning.go              # CodebaseProfile, AnalyzeCodebase() (Phase 3)
│   └── tuning_test.go
├── platform/                  # NEW — Platform detection
│   ├── detect.go              # Detect(), PlatformProfile, PlatformType
│   └── detect_test.go
├── tui/
│   ├── proposal_selector.go   # NEW — RunProposalSelector()
│   ├── proposal_selector_test.go
│   └── health_report.go       # NEW — RenderHealthReport()
├── pipeline/
│   ├── meta.go                # MODIFIED — remove extractYAMLLegacy()
│   ├── context.go             # MODIFIED — remove legacy template vars
│   └── resume.go              # MODIFIED — remove legacy workspace lookup
cmd/wave/commands/
├── run.go                     # MODIFIED — add `wave` special-case handling
└── wave.go                    # NEW — RunWaveCommand(), interactive orchestrator
```

**Structure Decision**: Follows the existing `internal/<package>/` convention. New `meta/` and `platform/` packages are peers to existing `pipeline/`, `github/`, `tui/`. The `meta/` package depends on `platform/`, `pipeline/`, `github/`, and `tui/` — consistent with the existing dependency graph where CLI commands compose internal packages.

## Implementation Phases

### Phase 1: Foundation — Platform Detection + Legacy Removal (FR-005, FR-020)

**Scope**: Create `internal/platform/` package, remove legacy code.

1. **`internal/platform/detect.go`**: `Detect(remoteURL string) PlatformProfile` — regex-based URL matching for GitHub, GitLab, Bitbucket, Gitea. `DetectFromGit()` executes `git remote -v` and calls `Detect()` on the origin.
2. **`internal/platform/detect_test.go`**: Table-driven tests for SSH, HTTPS, self-hosted URLs per platform.
3. **Remove `extractYAMLLegacy()`** from `internal/pipeline/meta.go` — replace fallback call with error return.
4. **Remove legacy template variables** from `internal/pipeline/context.go` — delete bare `pipeline_id`/`pipeline_name`/`step_id` replacements.
5. **Remove legacy workspace lookup** from `internal/pipeline/resume.go` — delete exact-name directory check.
6. **Fix or update affected tests** — any test relying on legacy behavior must be updated.

### Phase 2: Health Check Infrastructure (FR-001 – FR-004, FR-006, FR-007)

**Scope**: Create `internal/meta/health.go` with parallel health check execution.

1. **`internal/meta/health.go`**: Define `HealthReport`, `RunHealthChecks(ctx, manifest, platform)` using `errgroup.WithContext()`. Each check gets `context.WithTimeout()`:
   - `checkInit()`: Validate `wave.yaml` presence and parse validity
   - `checkDependencies()`: Use existing `preflight.Checker.CheckTools/CheckSkills` against all discoverable pipeline requirements
   - `checkCodebase()`: GitHub API via `internal/github/client.go` when platform is GitHub + token available; git-local fallback otherwise
   - `checkPlatform()`: Call `platform.DetectFromGit()`
2. **`internal/meta/health_test.go`**: Unit tests with mocked dependencies (mock GitHub client, mock preflight checker).
3. **`internal/tui/health_report.go`**: `RenderHealthReport(report HealthReport) string` — formatted text output using lipgloss styling consistent with existing TUI.

### Phase 3: Proposal Engine (FR-008, FR-009, FR-013, FR-014)

**Scope**: Create `internal/meta/proposal.go` with rule-based pipeline recommendation.

1. **`internal/meta/proposal.go`**: `ProposalEngine.GenerateProposals(report HealthReport, pipelines []tui.PipelineInfo) []PipelineProposal` — rule-based mapping from health signals to pipeline recommendations. Platform-aware: selects `gh-*` vs `gl-*` vs `bb-*` vs `gt-*` based on `PlatformProfile.PipelineFamily`.
2. **`internal/meta/proposal_test.go`**: Table-driven tests with various health report scenarios.
3. Rules:
   - Open issues → single implementation pipeline (`{family}-implement`)
   - Multiple open issues → epic pipeline (`{family}-implement-epic`)
   - Pending PRs → review pipeline (`{family}-pr-review` or generic `gh-pr-review`)
   - Recent test failures → `wave-bugfix` (if tests not passing)
   - Stale documentation → `doc-audit`
   - Generic → `wave-evolve`, `improve`, `refactor`
4. Sequence proposals: combine related pipelines (e.g., `{family}-research → {family}-implement → wave-land`)

### Phase 4: Interactive Selection (FR-010, FR-011, FR-019)

**Scope**: TUI proposal selector and non-interactive mode.

1. **`internal/tui/proposal_selector.go`**: `RunProposalSelector(proposals []PipelineProposal) (*ProposalSelection, error)` using `huh` forms:
   - Single select for choosing a proposal
   - Confirm dialog showing pre-filled input (editable)
   - Multi-select option for parallel execution
2. **Non-interactive mode**: When `!isInteractive()`, serialize `HealthReport` as JSON to stdout. Accept `--proposal` flag to auto-select.
3. **`cmd/wave/commands/wave.go`**: Orchestrator function `runWave(opts)` that ties health checks → proposals → selection → dispatch.

### Phase 5: Pipeline Dispatch (FR-011, FR-012, FR-018)

**Scope**: Wire `wave run wave` into the CLI and implement dispatch.

1. **`cmd/wave/commands/run.go`**: Intercept `pipeline == "wave"` before `loadPipeline()`. Dispatch to `runWave()`.
2. **Single pipeline dispatch**: Load pipeline YAML, create executor, run.
3. **Parallel dispatch**: Spawn `NewChildExecutor()` per pipeline in `errgroup`, following `MatrixExecutor` pattern.
4. **`internal/meta/sequence.go`**: `SequenceExecutor.Execute(ctx, pipelineNames, manifest, input)`:
   - Load each pipeline
   - Run sequentially via `NewChildExecutor()`
   - After each pipeline: copy output artifacts from workspace to next pipeline's `.wave/artifacts/`
   - Halt on failure with options (retry/skip/abort)

### Phase 6: Auto-Tuning (FR-015, FR-016, FR-017) — Phase 3 Priority

**Scope**: Codebase analysis and profile generation.

1. **`internal/meta/tuning.go`**: `AnalyzeCodebase() CodebaseProfile`:
   - Detect language from marker files (`go.mod`, `package.json`, etc.)
   - Detect framework from dependencies
   - Classify project size by file count
   - Detect test infrastructure
2. Uses profile to augment persona prompts and suggest `wave.yaml` `project` section updates.
3. **Respects existing user config**: Never overwrites `wave.yaml` values — only suggests additions.

### Phase 7: Observability & Polish (FR-021)

**Scope**: Structured events and edge case handling.

1. Emit `event.Event` for: health_check_started, health_check_completed, proposal_generated, proposal_selected, pipeline_dispatched.
2. Edge cases:
   - Missing `wave.yaml` → clear error with `wave init` suggestion
   - No git remote → platform detection returns `PlatformUnknown`
   - All pipelines filtered → health report + "no runnable pipelines" message
   - Ctrl+C → clean exit via existing signal handling
   - Non-TTY → JSON health report output
   - Health check timeout → partial results with timeout indicators

## Dependency Order

```
Phase 1 (Platform + Legacy) ─────┐
                                  ├──→ Phase 2 (Health Checks) ──→ Phase 3 (Proposals) ──→ Phase 4 (Selection)
                                  │                                                           │
                                  │                                                           ▼
                                  └─────────────────────────────────────────────────────→ Phase 5 (Dispatch)
                                                                                              │
                                                                                              ▼
                                                                                         Phase 6 (Tuning)
                                                                                              │
                                                                                              ▼
                                                                                         Phase 7 (Polish)
```

## Key Risks

| Risk | Mitigation |
|------|------------|
| Legacy removal breaks tests | Run `go test -race ./...` after each removal. Fix or delete with justification per P13. |
| GitHub API rate limiting during health check | Use existing `internal/github/ratelimit.go`. Fall back to git-local data on rate limit hit. |
| `huh` form rendering issues with proposals | Follow existing patterns in `run_selector.go`. Test manually. |
| Cross-pipeline artifact handoff loses data | SequenceExecutor validates artifact existence after copy. Test with real pipeline output. |
| Non-interactive mode missing edge cases | Table-driven tests with various TTY/flag combinations. |

## Complexity Tracking

_No constitution violations found — no entries needed._
