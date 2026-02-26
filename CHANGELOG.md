# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Mobile responsiveness improvements for documentation landing page terminal

### Changed
- Renamed `code-review` pipeline to `gh-pr-review` for clarity and consistency
- Generalized `github-commenter` persona to support multiple GitHub operations
- Updated all documentation references from `code-review` to `gh-pr-review`

### Fixed
- Terminal text alignment on mobile devices in documentation
- Base URL configuration for GitHub Pages subpath hosting
- Premature StateRunning status during artifact injection in pipeline execution
- Resume display showing prior steps as completed when using `--from-step` flag (#151)

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

[Unreleased]: https://github.com/re-cinq/wave/compare/v0.32.0...HEAD
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
