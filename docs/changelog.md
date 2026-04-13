# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Semantic convergence tracking for rework loops — aborts early when LLM judge scores plateau (#772)
- `wave analyze --decisions` provenance CLI showing orchestration decision table (#772)
- Toast notifications in WebUI via attention SSE (needs_review, failed, completed transitions) (#772)
- Gate smoke test pipeline (`wave-smoke-gates`) for validating gate step execution (#772)
- LoreProvider interface for historical classification enrichment (`internal/classify/lore.go`) (#772)
- V&V Patterns documentation guide (`docs/guides/vv-patterns.md`) with contract types reference (#772)
- Sub-pipeline diff fix: child worktree branch propagated to parent for diff endpoint (#772)
- Attention classifier with brand logo state transitions (running, needs_review, failed, winddown) (#772)
- Wave orchestrator pipeline (`wave-orchestrate`) for task classification and pipeline routing (#772)
- Orchestration decision tracking in `orchestration_decision` SQLite table (#772)
- TUI live output dashboard view with structured per-step progress display
- Dashboard shows completed/running/failed steps with spinners, durations, and token counts
- `l` key toggle between dashboard and event log in live output
- Header completion counts (e.g. "1 ok, 1 fail") in live output
- Handover metadata (artifacts, contracts, targets) in verbose dashboard mode
- Pipeline taxonomy with mandatory prefixes (`audit-`, `doc-`, `impl-`, `ops-`, `plan-`, `test-`, `wave-`)
- Forge-agnostic template variables in pipeline prompts
- Skill management CLI (`wave skills list|install|remove|search|sync`)
- Project ontology system with telos, bounded contexts, invariants, and conventions (#590)
- `wave analyze` CLI for deterministic ontology generation and `--deep` AI-assisted enrichment
- `wave analyze --evolve` self-evolution feedback loop using decision lineage
- Ontology lineage badges in WebUI and TUI (total runs, success rate, last used)
- Staleness detection and warning banners in WebUI and TUI ontology views
- Centralized `runtime.timeouts` configuration with 16 tunable values
- Full `Timeouts` section in manifest schema documentation

### Changed
- Renamed all pipelines to use taxonomy prefixes (e.g., `implement` → `impl-issue`, `pr-review` → `ops-pr-review`, `speckit-flow` → `impl-speckit`)
- Unified forge-specific pipelines into forge-agnostic pipelines with template variables
- `github_api_seconds` → `forge_api_seconds` in timeout configuration
- All hardcoded timeout values replaced with configurable constants via `internal/timeouts`

### Removed
- Deprecated pipeline name resolution (`ResolveDeprecatedName`) — no backward-compat shims pre-1.0.0
- Timeout constant re-exports from `manifest` package
- Stale multiplatform pipeline tests referencing non-existent gl-*/gt-* YAML files
- Nonexistent `timeout` and `retry` persona fields from custom-personas guide

## [0.69.0] - 2026-03-10

### Added
- Pipeline failure context loading on resume from prior run
- Pipeline relevance scoring and assignee filter in TUI
- Detached pipeline output redirect to `.wave/logs/`

## [0.68.1] - 2026-03-10

### Fixed
- Documentation inaccuracies found in review of #273
- 17 documentation inconsistencies from #264
- Pipeline status/event-logging gaps and craftsman commit block

## [0.68.0] - 2026-03-10

### Added
- Pre-merge change summary and upgrade workflow to `wave init`

## [0.67.1] - 2026-03-10

### Added
- ADR (Architectural Decision Record) template and README

## [0.67.0] - 2026-03-10

### Added
- Per-step `timeout_minutes` support (#247)

### Fixed
- Hardcoded test command and stale schemas in pipelines (#241)
- 10 bugs from #241 audit
- Artifact path resolution from prior run when using `--run` flag

## [0.66.1] - 2026-03-10

### Fixed
- TUI filter, stale run, and UX quality issues (#295, #250)

## [0.66.0] - 2026-03-10

### Added
- AI-steered project optimization with `wave doctor --optimize` flag (#296)
- Interactive pipeline orchestration (epic #184)

## [0.65.5] - 2026-03-10

### Fixed
- TUI repo detection from git remote for issue provider

## [0.65.4] - 2026-03-10

### Fixed
- TUI GitHub token resolution via `gh auth` and live output on hover

## [0.65.3] - 2026-03-10

### Fixed
- TUI live output state and polling on pipeline hover

## [0.65.2] - 2026-03-10

### Fixed
- TUI launch flags restoration and subprocess passthrough
- TUI finished detail alignment with spec labels and artifact paths
- TUI header metadata grid layout

### Changed
- Delegated token and duration formatting to display package
- Unified duration display and item indicators in TUI

## [0.65.1] - 2026-03-10

### Changed
- Added `tea` and `glab` to Nix devShell packages

## [0.65.0] - 2026-03-10

### Added
- Sequential pipeline composition via `SequenceExecutor`

## [0.64.0] - 2026-03-10

### Added
- Retry with prompt adaptation and step attempt tracking

## [0.63.0] - 2026-03-09

### Added
- TUI issue browser and pipeline chooser dialog

## [0.62.0] - 2026-03-09

### Added
- Detached pipeline execution from TUI process lifecycle

## [0.61.0] - 2026-03-09

### Added
- Rich handover metadata in TUI live output (parity with CLI)

## [0.60.0] - 2026-03-09

### Fixed
- Huh form input lost across Bubble Tea value-copy cycles
- Event logs preserved across refresh ticks; skip duplicate started line
- Empty input rejection in research pipelines instead of using examples
- Live output alignment with CLI, zombie run cleanup, dead flag removal
- Cross-process cancellation via DB polling
- Live output buffer rebuild when toggling v/d/o display flags
- Pipeline events persisted to SQLite; stale run dismiss enabled
- Stale pending runs cleaned on startup with improved messaging
- Cancel with `c` key from both left and right pane
- Top padding calculation in child model height
- Pending runs limit in `GetRunningRuns` to last 5 minutes
- Live output buffer wired on pipeline hover
- Header reorder, status divider, collapse defaults, launch focus, repo detection
- Performance metrics recorded on step completion
- Status bar hints for compose, cancel, and Tab/Shift+Tab
- Provider wiring, pane layout, section reorder, and launch fixes

### Added
- Walking glow logo animation and 3-row header metadata grid

## [0.59.0] - 2026-03-07

### Added
- Pipeline composition UI with sequence builder and artifact flow visualization

## [0.58.0] - 2026-03-07

### Added
- CLI compliance polish per clig.dev guidelines

## [0.57.2] - 2026-03-06

### Fixed
- Commit-constraint fix applied to implementer persona
- Pipeline status/event-logging gaps and craftsman commit block

## [0.57.1] - 2026-03-06

### Fixed
- Label create permissions and `glab issue update` command in personas

## [0.57.0] - 2026-03-06

### Added
- Alternative master-detail views for personas, contracts, skills, and health in TUI

## [0.56.1] - 2026-03-06

### Changed
- Documentation synced with implementation

## [0.56.0] - 2026-03-06

### Added
- Finished pipeline actions in TUI — chat, branch checkout, diff view

## [0.55.0] - 2026-03-06

### Added
- Live output streaming for running pipelines in TUI

## [0.54.0] - 2026-03-06

### Added
- Pipeline launch flow with argument form, executor integration, and cancellation

## [0.53.0] - 2026-03-06

### Added
- Pipeline detail right pane with navigation, data providers, and focus management

## [0.52.0] - 2026-03-06

### Added
- Pipeline list left pane with navigation, filtering, and sections
- Header bar with animated logo and project metadata

## [0.51.0] - 2026-03-05

### Added
- Bubble Tea TUI scaffold with 3-row layout

## [0.50.0] - 2026-03-05

### Added
- Skill definitions moved from manifest to pipeline YAML

### Fixed
- Bare template variables in worktree branch names
- Default step timeout increased to 90 minutes

## [0.49.2] - 2026-03-04

### Fixed
- Added `uv` to Nix flake and fixed speckit skill commands

## [0.49.1] - 2026-03-04

### Fixed
- Missing matrix strategy and workspace properties in pipeline JSON schema

## [0.49.0] - 2026-03-04

### Added
- Bitbucket (`bb-implement-epic`), GitLab (`gl-implement-epic`), and Gitea (`gt-implement-epic`) epic pipelines

## [0.48.0] - 2026-03-04

### Added
- `--model` flag to override adapter model per run

## [0.46.0] - 2026-03-03

### Added
- Progress summary on third header line beside logo
- Unified TUI color palette and logo shimmer animation

### Fixed
- Display formatting: colons after step IDs, blank lines, top margin, shimmer rune indexing

## [0.45.0] - 2026-03-03

### Changed
- Redesigned pipeline TUI — deduplicated logo, model visibility, token split, collapsible tools

## [0.44.5] - 2026-03-03

### Changed
- Removed Write auto-grant workaround, superseded cascade/executor_enhanced code, speculative validators

## [0.44.4] - 2026-03-03

### Fixed
- Blanket deny rules blocking tool availability in personas
- Workspace path resolution anchored with `git init`
- Bare Write/Edit subsumption of scoped permissions in adapter
- Distill step rewritten to make JSON output primary task

## [0.44.3] - 2026-03-03

### Fixed
- impl-issue pipeline flaws from parallel run audit

## [0.44.2] - 2026-03-03

### Fixed
- Bare Write permission in personas instead of scoped paths
- Synthesizer JSON-only output constraints
- Write/Edit preservation in `normalizeAllowedTools`

## [0.44.1] - 2026-03-03

### Fixed
- Batch fallback prevention when specific issue not found

## [0.44.0] - 2026-03-02

### Added
- Array extraction in outcome `json_path` for multi-link results (#191)
- `[*]` wildcard support in scope/rewrite outcome `json_path`

## [0.43.2] - 2026-03-02

### Fixed
- Noisy outcome warnings replaced with friendly messages for empty arrays (#204)

## [0.43.1] - 2026-03-02

### Fixed
- Suppressed usage text on pipeline execution errors (#205)
- Silenced cobra error printing to prevent triple output

## [0.43.0] - 2026-03-02

### Added
- Enriched trace entries with step lifecycle context (#189)

## [0.42.0] - 2026-03-02

### Added
- `impl-issue-epic` pipeline and artifacts
- Child pipeline invocation in matrix executor
- Dependency tiers in matrix executor

## [0.41.4] - 2026-03-02

### Fixed
- Stripped Write/Edit tools, disallowed TodoWrite, removed false JSON validation

## [0.41.3] - 2026-03-01

### Fixed
- Embedded persona configs synced (dev, commenter, enhancer, analyst)
- Scope verify-report steps made read-only
- Persona tool requirements expanded across gl-/gt-/gh-/bb-* pipelines

## [0.41.2] - 2026-03-01

### Fixed
- Bitbucket pipeline prompts rewritten for REST API
- Bitbucket persona permissions rewritten for curl+jq
- TodoWrite avoidance instruction added to base protocol

## [0.41.1] - 2026-03-01

### Fixed
- `wave-land` pipeline creates feature branch before committing

## [0.41.0] - 2026-03-01

### Added
- `wave-land` pipeline for commit-and-ship workflow

### Fixed
- Templatized test command; removed stale script reference in impl-issue
- Tightened persona permissions with granular tool controls
- Real-execution-only constraint added to base protocol

### Changed
- Streamlined refresh and scope pipeline prompts
- Removed redundant version checks from persona prompts

## [0.40.0] - 2026-03-01

### Added
- Ollama adapter and free-text input for adapter/model in onboarding
- Interactive onboarding wizard for first-time setup (#163)
- Token display in web UI dashboard (#98)

### Fixed
- PersonaConfigs threaded through CLI and onboarding manifest emission

## [0.39.0] - 2026-02-28

### Added
- Token counting fix and token display in TUI (#98)

## [0.38.1] - 2026-02-28

### Fixed
- All pipeline configs audited and optimized

## [0.38.0] - 2026-02-28

### Added
- `plan-scope` pipeline for epic decomposition across all 4 forges
- Scope contracts and pipelines added to `.wave/` runtime directory

## [0.37.0] - 2026-02-28

### Added
- `file://` URI scheme prefix for absolute file paths in display (#186)

## [0.36.0] - 2026-02-28

### Changed
- Replaced hardcoded persona map with embedded YAML configs in `wave init`

## [0.35.1] - 2026-02-27

### Changed
- README install/quickstart and contributor guidance updated for public repo (#174)

## [0.35.0] - 2026-02-27

### Fixed
- Table width truncation adapted to terminal width (#167)

## [0.34.0] - 2026-02-27

### Added
- Bitbucket platform support with `bb-*` pipelines
- GitLab and Gitea platform support pipelines (#168)

## [0.33.2] - 2026-02-27

### Changed
- Default model for wave changed to opus

## [0.33.1] - 2026-02-26

### Fixed
- Responsive terminal header with logo/meta variant switching
- Side-by-side logo+metadata for desktop terminal

## [0.33.0] - 2026-02-27

### Added
- Bitbucket platform support with `bb-*` pipelines

## [0.32.0] - 2026-02-24

### Added
- Pipeline identifier renaming for improved clarity (#136)
  - `gh-issue-impl` → `gh-implement`
  - `gh-pr-comment` → `gh-pr-review`
  - 6 additional pipelines renamed for consistency

### Fixed
- Documentation references to stale `gh-issue-impl` updated to `gh-implement`
- Preserved recinq and speckit-flow trademark names in pipeline identifiers

## [0.31.0] - 2026-02-24

### Added
- `wave list skills` CLI subcommand for skill discovery

## [0.30.0] - 2026-02-24

### Added
- Comprehensive persona improvements with anti-patterns, quality checklists, and scope boundaries

### Changed
- Enhanced persona system prompts with structured guidance and constraints

## [0.29.0] - 2026-02-24

### Added
- Verbose handover display for pipeline steps (#154)
- Schema filename display instead of contract type in verbose output
- Artifact paths in step completion events
- Contract validation status in step metadata

### Fixed
- Handover display format for non-TTY output
- Contract passed logic in pipeline execution

## [0.28.1] - 2026-02-23

### Fixed
- CLAUDE.md documentation restructuring to reduce noise and improve runtime clarity (#141)

### Changed
- Replaced `skill_mounts` references with `skills` terminology across all documentation
- Added speckit skill declaration to wave.yaml configuration

### Removed
- Deprecated `skill_mounts` from test fixtures and JSON schema
- Dead SkillMount type and validation code

## [0.28.0] - 2026-02-23

### Added
- Structured dead-code detection pipeline with multi-mode output (#135)
- Preflight recovery guidance and path fix (#145)
- Auto-prepend artifact references into step prompts
- Handover contracts to all JSON output steps
- JSON schemas for all output artifact gaps

### Fixed
- Prompts to remove inline JSON schemas and hardcoded paths
- Pipeline artifact guidance in contract prompt
- Recovery hints addressing code review findings

### Changed
- Extracted validation infrastructure abstraction
- Removed backwards-compatibility shims from prototype codebase

## [0.27.0] - 2026-02-22

### Added
- Token display to web UI dashboard
- Specification for webui token display (issue #98)

### Fixed
- Always populate step ID in pipeline failure errors
- False-positive rate limit detection on persona output
- Auto-generate output guidance from output_artifacts metadata

### Changed
- Ignore `.wave/chat/` and `.wave/wave.db` in gitignore

## [0.26.0] - 2026-02-21

### Added
- `wave chat` command for interactive pipeline analysis
- Step manipulation and cascade control to wave chat
- Validated-findings schema for recinq converge step

### Fixed
- Remove conflicting schema injection from buildStepPrompt
- Archive artifacts per-step to prevent shared-path collision
- Normalize artifact paths to `.wave/` directory structure
- Contract compliance auto-injection into CLAUDE.md

### Changed
- Removed hardcoded paths from all pipeline prompts
- Remove file paths from prompts, using contract compliance instead

## [0.25.0] - 2026-02-20

### Added
- DAG-level concurrent step execution
- Outcome extraction from step artifacts
- Outcomes to all PR/issue pipelines
- Publish step to code-review pipeline

### Fixed
- Deduplicate artifacts and surface outcome extraction warnings
- Sort artifacts chronologically by step execution order
- Remove stdout URL scanning, only track declared artifacts

## [0.24.0] - 2026-02-19

### Added
- Failure modes validation contracts and pipelines
- Comprehensive end-to-end pipeline failure mode tests
- Pipeline failure mode tests for DAG, permissions, workspace
- Adapter tests for non-zero exit code and error handling
- Contract validation failure tests
- Pipeline failure mode test coverage specification

### Fixed
- Report missing artifacts with clear error message

## [0.23.0] - 2026-02-18

### Added
- Typed artifact composition and input validation
- Structured pipeline outcome summary with scannable UX
- Optional step support for non-blocking failures
- Base protocol preamble and persona quality guardrails

### Fixed
- Move runtime artifacts under `.wave/` to prevent worktree pollution

## [0.22.0] - 2026-02-17

### Added
- Auto-recover input on `--from-step` resume
- Publish steps to pipelines

### Fixed
- Remove harmful "Do NOT push" instructions from pipeline prompts

### Removed
- `wave resume` command (replaced by `--from-step` flag)
- Unused DisplayConfig fields and dead display functions (#66)

## [0.21.0] - 2026-02-16

### Added
- Supervisor and provocateur personas
- `wave-recinq` pipeline for convergent/divergent thinking
- `wave-supervise` pipeline for quality supervision
- `gh-issue-update` pipeline for GitHub issue updates
- Supervision evidence and evaluation schemas
- Divergent findings and convergent proposals schemas
- Issue-update JSON schema contracts

### Changed
- Eliminate duplicate default content from `.wave/`
- Unify template variable replacement
- Delete unused executor methods (Resume, GetStatus, injectCheckpointIfExists)
- Complete StrictMode deprecation

### Fixed
- Use pipeline name not run ID in recovery hints
- Add provocateur, validator, synthesizer personas to default manifest
- Exclude cumulative cache_read tokens from result total
- Add validator/synthesizer personas, fix distill step failure
- Enforce JSON output in recinq distill step
- Worktree sharing and resume artifact discovery

### Removed
- ConcurrencyValidator (doubly broken)
- executor_enhanced.go dead code cascade
- Redundant injectArtifacts pre-call from MatrixExecutor

## [0.20.0] - 2026-02-15

### Added
- Git-native worktree workspaces across landing page, concepts, and reference documentation
- Worktree workspace sharing across steps using same branch
- Workspace ref for shared worktrees
- Pipeline step visibility in TUI showing all steps with status
- Specification and plan for pipeline step visibility

### Changed
- Convert `root:./` to worktree workspace, remove cd hack
- Decompose pipeline executor runStepExecution into focused helpers
- Consolidate CLI boilerplate into shared helpers
- Replace 12-strategy JSON recovery with minimal cleanup
- Gut json_recovery.go from 814 to 194 lines

### Fixed
- Preserve worktree artifacts and filter deliverable noise
- Use detached HEAD at base ref, remove sidecar artifact dirs
- Redirect worktree artifact writes to numbered sidecar dirs
- Detect rate limit errors and use worktree workspaces in defaults

## [0.19.0] - 2026-02-14

### Added
- Per-persona model and temperature settings in manifest
- Create-pr step to doc-sync and dead-code pipelines
- Quality checklists for various features

### Changed
- Register supervisor and provocateur personas

### Fixed
- SSH config permission error in Nix dev shell

## [0.18.0] - 2026-02-13

### Added
- Contextual recovery hints on pipeline failure (#86)
- Handle Claude context window exhaustion gracefully (#60)
- Implementation plan and research for pipeline recovery hints
- Specification and tasks for context exhaustion handling

### Fixed
- Detect shell metacharacters in input sanitizer
- Use worktree isolation for gh-issue-impl implement step
- Address review feedback on context exhaustion handling and recovery hints

## [0.17.0] - 2026-02-12

### Added
- Web-based pipeline operations dashboard (`wave serve`) (#85)
- Dark/light mode toggle with Wave brand colors
- Static analysis CI specification
- Feature specs for web operations dashboard and app

### Fixed
- Dashboard styling aligned with VitePress docs theme
- Remove hover shift effects, add SVG logo to navbar
- Rename "Deliverables" to "Artifacts" in pipeline output
- Register artifacts in DB so dashboard step cards show them
- Unify run IDs, fix dashboard bugs, improve layout
- Skip homebrew tap upload, add private-repo install warnings

### Changed
- Documentation fixes for 6 inconsistencies (#89)

## [0.16.0] - 2026-02-11

### Added
- Rotating pipeline demos with typewriter effect in hero terminal
- Web dashboard with Preact SPA and Go HTTP server (#81)

### Fixed
- Clean up stale worktrees before creating new ones
- Resolve 21 dead links in VitePress site
- Remove extra closing div tag in code-review.md
- Support spaced template variables in ResolvePlaceholders

### Changed
- Remove dead code from contract, display, manifest, pipeline, relay
- Default memory.strategy to fresh, add JSON schemas for IDE support
- Switch feature/doc-sync/dead-code to worktree workspaces
- Per-step timeouts instead of shared pipeline timeout

## [0.15.0] - 2026-02-10

### Added
- Skill dependency management (#76)
- Worktree workspaces (#76)
- Preflight validation (#76)
- Schema reference and worktree cleanup documentation

### Changed
- Sync documentation with implementation

[Unreleased]: https://github.com/re-cinq/wave/compare/v0.69.0...HEAD
[0.69.0]: https://github.com/re-cinq/wave/compare/v0.68.1...v0.69.0
[0.68.1]: https://github.com/re-cinq/wave/compare/v0.68.0...v0.68.1
[0.68.0]: https://github.com/re-cinq/wave/compare/v0.67.1...v0.68.0
[0.67.1]: https://github.com/re-cinq/wave/compare/v0.67.0...v0.67.1
[0.67.0]: https://github.com/re-cinq/wave/compare/v0.66.1...v0.67.0
[0.66.1]: https://github.com/re-cinq/wave/compare/v0.66.0...v0.66.1
[0.66.0]: https://github.com/re-cinq/wave/compare/v0.65.5...v0.66.0
[0.65.5]: https://github.com/re-cinq/wave/compare/v0.65.4...v0.65.5
[0.65.4]: https://github.com/re-cinq/wave/compare/v0.65.3...v0.65.4
[0.65.3]: https://github.com/re-cinq/wave/compare/v0.65.2...v0.65.3
[0.65.2]: https://github.com/re-cinq/wave/compare/v0.65.1...v0.65.2
[0.65.1]: https://github.com/re-cinq/wave/compare/v0.65.0...v0.65.1
[0.65.0]: https://github.com/re-cinq/wave/compare/v0.64.0...v0.65.0
[0.64.0]: https://github.com/re-cinq/wave/compare/v0.63.0...v0.64.0
[0.63.0]: https://github.com/re-cinq/wave/compare/v0.62.0...v0.63.0
[0.62.0]: https://github.com/re-cinq/wave/compare/v0.61.0...v0.62.0
[0.61.0]: https://github.com/re-cinq/wave/compare/v0.60.0...v0.61.0
[0.60.0]: https://github.com/re-cinq/wave/compare/v0.59.0...v0.60.0
[0.59.0]: https://github.com/re-cinq/wave/compare/v0.58.0...v0.59.0
[0.58.0]: https://github.com/re-cinq/wave/compare/v0.57.2...v0.58.0
[0.57.2]: https://github.com/re-cinq/wave/compare/v0.57.1...v0.57.2
[0.57.1]: https://github.com/re-cinq/wave/compare/v0.57.0...v0.57.1
[0.57.0]: https://github.com/re-cinq/wave/compare/v0.56.1...v0.57.0
[0.56.1]: https://github.com/re-cinq/wave/compare/v0.56.0...v0.56.1
[0.56.0]: https://github.com/re-cinq/wave/compare/v0.55.0...v0.56.0
[0.55.0]: https://github.com/re-cinq/wave/compare/v0.54.0...v0.55.0
[0.54.0]: https://github.com/re-cinq/wave/compare/v0.53.0...v0.54.0
[0.53.0]: https://github.com/re-cinq/wave/compare/v0.52.0...v0.53.0
[0.52.0]: https://github.com/re-cinq/wave/compare/v0.51.0...v0.52.0
[0.51.0]: https://github.com/re-cinq/wave/compare/v0.50.0...v0.51.0
[0.50.0]: https://github.com/re-cinq/wave/compare/v0.49.2...v0.50.0
[0.49.2]: https://github.com/re-cinq/wave/compare/v0.49.1...v0.49.2
[0.49.1]: https://github.com/re-cinq/wave/compare/v0.49.0...v0.49.1
[0.49.0]: https://github.com/re-cinq/wave/compare/v0.48.0...v0.49.0
[0.48.0]: https://github.com/re-cinq/wave/compare/v0.46.0...v0.48.0
[0.46.0]: https://github.com/re-cinq/wave/compare/v0.45.0...v0.46.0
[0.45.0]: https://github.com/re-cinq/wave/compare/v0.44.5...v0.45.0
[0.44.5]: https://github.com/re-cinq/wave/compare/v0.44.4...v0.44.5
[0.44.4]: https://github.com/re-cinq/wave/compare/v0.44.3...v0.44.4
[0.44.3]: https://github.com/re-cinq/wave/compare/v0.44.2...v0.44.3
[0.44.2]: https://github.com/re-cinq/wave/compare/v0.44.1...v0.44.2
[0.44.1]: https://github.com/re-cinq/wave/compare/v0.44.0...v0.44.1
[0.44.0]: https://github.com/re-cinq/wave/compare/v0.43.2...v0.44.0
[0.43.2]: https://github.com/re-cinq/wave/compare/v0.43.1...v0.43.2
[0.43.1]: https://github.com/re-cinq/wave/compare/v0.43.0...v0.43.1
[0.43.0]: https://github.com/re-cinq/wave/compare/v0.42.0...v0.43.0
[0.42.0]: https://github.com/re-cinq/wave/compare/v0.41.4...v0.42.0
[0.41.4]: https://github.com/re-cinq/wave/compare/v0.41.3...v0.41.4
[0.41.3]: https://github.com/re-cinq/wave/compare/v0.41.2...v0.41.3
[0.41.2]: https://github.com/re-cinq/wave/compare/v0.41.1...v0.41.2
[0.41.1]: https://github.com/re-cinq/wave/compare/v0.41.0...v0.41.1
[0.41.0]: https://github.com/re-cinq/wave/compare/v0.40.0...v0.41.0
[0.40.0]: https://github.com/re-cinq/wave/compare/v0.39.0...v0.40.0
[0.39.0]: https://github.com/re-cinq/wave/compare/v0.38.1...v0.39.0
[0.38.1]: https://github.com/re-cinq/wave/compare/v0.38.0...v0.38.1
[0.38.0]: https://github.com/re-cinq/wave/compare/v0.37.0...v0.38.0
[0.37.0]: https://github.com/re-cinq/wave/compare/v0.36.0...v0.37.0
[0.36.0]: https://github.com/re-cinq/wave/compare/v0.35.1...v0.36.0
[0.35.1]: https://github.com/re-cinq/wave/compare/v0.35.0...v0.35.1
[0.35.0]: https://github.com/re-cinq/wave/compare/v0.34.0...v0.35.0
[0.34.0]: https://github.com/re-cinq/wave/compare/v0.33.2...v0.34.0
[0.33.2]: https://github.com/re-cinq/wave/compare/v0.33.1...v0.33.2
[0.33.1]: https://github.com/re-cinq/wave/compare/v0.33.0...v0.33.1
[0.33.0]: https://github.com/re-cinq/wave/compare/v0.32.0...v0.33.0
[0.32.0]: https://github.com/re-cinq/wave/compare/v0.31.0...v0.32.0
[0.31.0]: https://github.com/re-cinq/wave/compare/v0.30.0...v0.31.0
[0.30.0]: https://github.com/re-cinq/wave/compare/v0.29.0...v0.30.0
[0.29.0]: https://github.com/re-cinq/wave/compare/v0.28.1...v0.29.0
[0.28.1]: https://github.com/re-cinq/wave/compare/v0.28.0...v0.28.1
[0.28.0]: https://github.com/re-cinq/wave/compare/v0.27.0...v0.28.0
[0.27.0]: https://github.com/re-cinq/wave/compare/v0.26.0...v0.27.0
[0.26.0]: https://github.com/re-cinq/wave/compare/v0.25.0...v0.26.0
[0.25.0]: https://github.com/re-cinq/wave/compare/v0.24.0...v0.25.0
[0.24.0]: https://github.com/re-cinq/wave/compare/v0.23.0...v0.24.0
[0.23.0]: https://github.com/re-cinq/wave/compare/v0.22.0...v0.23.0
[0.22.0]: https://github.com/re-cinq/wave/compare/v0.21.0...v0.22.0
[0.21.0]: https://github.com/re-cinq/wave/compare/v0.20.0...v0.21.0
[0.20.0]: https://github.com/re-cinq/wave/compare/v0.19.0...v0.20.0
[0.19.0]: https://github.com/re-cinq/wave/compare/v0.18.0...v0.19.0
[0.18.0]: https://github.com/re-cinq/wave/compare/v0.17.0...v0.18.0
[0.17.0]: https://github.com/re-cinq/wave/compare/v0.16.0...v0.17.0
[0.16.0]: https://github.com/re-cinq/wave/compare/v0.15.0...v0.16.0
[0.15.0]: https://github.com/re-cinq/wave/releases/tag/v0.15.0
