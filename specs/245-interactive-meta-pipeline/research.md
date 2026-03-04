# Research: Interactive Meta-Pipeline Orchestrator

**Feature**: `245-interactive-meta-pipeline`  
**Date**: 2026-03-04  
**Phase**: 0 — Outline & Research

## R-001: Special-Case Handler vs Pipeline YAML (FR-018)

**Decision**: Implement `wave run wave` as a special-case handler in `cmd/wave/commands/run.go`.

**Rationale**: The interactive meta-orchestrator requires:
- User input mid-flow (interactive proposal selection via `charmbracelet/huh`)
- Parallel Go-native health check operations (goroutines + `errgroup`)
- Dynamic pipeline dispatch based on user selection
- Non-TTY fallback for CI/CD (JSON health report output)

None of these fit the step-based execution model (static DAG of persona-driven steps with fresh memory at each boundary). The existing `wave meta` command provides precedent for dedicated orchestrator commands.

**Alternatives Rejected**:
- **Pipeline YAML with custom step type**: Would require a new `exec.type: interactive` that breaks the core assumption of non-interactive, AI-driven execution.
- **Two-stage pipeline (health → execute)**: Can't handle user selection between stages without violating fresh-memory constraints.

## R-002: Parallel Health Check Architecture (FR-001, FR-006)

**Decision**: Use `errgroup.WithContext()` with per-check timeouts via `context.WithTimeout()`.

**Rationale**: The codebase already uses `errgroup` extensively (see `internal/pipeline/matrix.go:126`). Each health check runs as an independent goroutine:
1. **Init check**: `os.Stat("wave.yaml")` + YAML parse — purely local, <100ms
2. **Dependency audit**: `exec.LookPath()` for tools, skill check commands — local, <2s
3. **Codebase health**: GitHub API calls via `internal/github/client.go` or git-local fallback — network, needs 10s timeout
4. **Platform detection**: `git remote -v` parse — local, <500ms

Each check gets an independent `context.WithTimeout()` so a slow GitHub API doesn't block the report. Results are collected into a `HealthReport` struct via channels or mutex-guarded map.

**Alternatives Rejected**:
- **Sequential execution**: Unnecessarily slow. Health checks are independent.
- **Worker pool**: Over-engineering for 4 checks. `errgroup` is simpler and proven.

## R-003: Platform Detection Strategy (FR-005)

**Decision**: Parse `git remote -v` output with regex patterns for GitHub, GitLab, Bitbucket, and Gitea.

**Rationale**: Git remote URLs have well-known patterns:
- GitHub: `github.com[:/]owner/repo`
- GitLab: `gitlab.com[:/]owner/repo` or self-hosted with `/gitlab/` path
- Bitbucket: `bitbucket.org[:/]owner/repo`
- Gitea: Detected by `.gitea.` domain or `/gitea/` path segment

The origin remote is checked first. If multiple remotes exist pointing to different platforms, the system reports the primary (origin) and notes alternatives.

**Implementation**: New package `internal/platform/` with `Detect(remoteURL string) Platform` function. Returns `Platform{Type, Owner, Repo, APIURL}`. The existing `internal/github/` package already knows how to derive owner/repo from URLs — generalize this pattern.

**Alternatives Rejected**:
- **API probing**: Slow, requires network, may hit rate limits before health check even starts.
- **Config file declaration**: Forces users to declare their platform — defeats the purpose of auto-detection.

## R-004: Interactive Menu Architecture (FR-010, FR-011)

**Decision**: Use `charmbracelet/huh` for interactive selection, consistent with existing TUI in `internal/tui/run_selector.go`.

**Rationale**: The existing TUI uses `huh.NewSelect`, `huh.NewMultiSelect`, and `huh.NewConfirm`. The pipeline proposal menu should follow the same patterns:
1. Display health report summary (plain text, not interactive)
2. `huh.NewSelect` for single pipeline selection
3. `huh.NewMultiSelect` for parallel execution
4. Sequence proposals shown as single selectable items

The existing `internal/tui/` package provides `PipelineInfo`, `DiscoverPipelines()`, and theming via `WaveTheme()`. Extend with a new `RunProposalSelector()` function.

**Alternatives Rejected**:
- **Bubble Tea full model**: Over-engineering — `huh` forms are sufficient and match existing patterns.
- **Simple stdin prompt**: Lacks the multi-select capability needed for parallel execution.

## R-005: Pipeline Proposal Engine (FR-008, FR-009)

**Decision**: Rule-based proposal engine that maps health check signals to pipeline recommendations.

**Rationale**: The proposal engine analyzes `HealthReport` and generates ranked `PipelineProposal` items. Rules:
- Open issues + GitHub platform → propose `gh-implement` (or `gl-implement` etc.)
- Pending PRs → propose `gh-pr-review`
- Failed tests (detected via `go test` exit code in health check) → propose `wave-bugfix`
- Stale docs → propose `doc-audit`
- No recent commits → propose `wave-evolve`
- Multiple open issues → propose `gh-implement-epic`

Each proposal includes: pipeline name(s), rationale string, pre-filled input (e.g., issue URL), and dependency status (all deps met vs. missing).

**Implementation**: New `internal/meta/` package with `ProposalEngine` struct. Takes `HealthReport` + discovered pipelines → returns `[]PipelineProposal`. Rules are hardcoded for v1 (not configurable) — configuration can be added post-release.

**Alternatives Rejected**:
- **AI-generated proposals**: Requires adapter invocation, adds latency and cost. Rule-based is deterministic and fast.
- **User-configurable rules**: Premature complexity. Hardcoded rules cover the 80% case.

## R-006: Cross-Pipeline Artifact Handoff (FR-012)

**Decision**: `SequenceExecutor` component that copies output artifacts between pipeline workspaces.

**Rationale**: The existing `DefaultPipelineExecutor` manages artifacts within a single pipeline via `ArtifactPaths` tracking. For cross-pipeline handoff:
1. Run pipeline A → collect `execution.ArtifactPaths`
2. Create pipeline B workspace
3. Copy pipeline A's output files into pipeline B's `.wave/artifacts/` directory
4. Run pipeline B — it reads `.wave/artifacts/` as if they were injected normally

This reuses the filesystem-based artifact injection pattern without modifying the single-pipeline executor.

**Implementation**: `internal/meta/sequence.go` with `SequenceExecutor.Execute(ctx, []Pipeline, manifest, input)`. Each pipeline uses `NewChildExecutor()` for independent state.

**Alternatives Rejected**:
- **Combined DAG**: Merging pipeline DAGs risks step ID collisions and conflicting workspace configs.
- **Shared workspace**: Violates ephemeral workspace isolation (Constitution Principle 8).

## R-007: Legacy Code Removal (FR-020)

**Decision**: Remove three specific backward-compatibility shims.

**Items**:
1. **`extractYAMLLegacy`** (`internal/pipeline/meta.go:604-630`): Fallback for old meta-pipeline output format. The new `--- PIPELINE ---`/`--- SCHEMAS ---` format has been stable since initial implementation. Remove function and the fallback call at line 579.

2. **Legacy template variables** (`internal/pipeline/context.go:95-98`): The `replaceBoth` calls at lines 96-98 handle `{{pipeline_id}}`, `{{pipeline_name}}`, `{{step_id}}` without the `pipeline_context.` prefix. These are legacy spaced variants. Remove the three `replaceBoth` calls for bare variable names.

3. **Legacy workspace directory lookup** (`internal/pipeline/resume.go:211-213`): The exact-name directory check (without hash suffix) at line 212 is a legacy fallback. All current runs use hashed run IDs. Remove the `os.Stat` check and the `append` that prepends it.

**Risk Assessment**: Low — all three have been superseded by newer implementations. Removing them will cause test failures if any tests depend on the legacy behavior, which must be fixed or removed per Constitution Principle 13.

## R-008: Non-Interactive Mode (FR-019)

**Decision**: Detect non-TTY via `term.IsTerminal()` (already used in `run.go:460`). Output JSON health report to stdout and exit.

**Rationale**: The existing `isInteractive()` function in `cmd/wave/commands/run.go` provides the detection pattern. In non-interactive mode:
- Health checks still run in parallel
- Results are serialized as JSON to stdout
- No proposal selection — the `--proposal` flag allows specifying a proposal index or pipeline name
- Exit code 0 for success, 1 for errors

**Alternatives Rejected**:
- **Always require TTY**: Breaks CI/CD integration.
- **Separate `wave health` command**: Fragments the user experience. `wave run wave` should handle both modes.

## R-009: Parallel Multi-Pipeline Execution (FR-011, C-005)

**Decision**: Use `NewChildExecutor()` + `errgroup.WithContext()`, following the `MatrixExecutor` pattern.

**Rationale**: `internal/pipeline/matrix.go` already demonstrates spawning independent `DefaultPipelineExecutor` instances via `NewChildExecutor()` (line 957). Each child executor has:
- Shared adapter runner (connection pooling)
- Shared event emitter (concurrent-safe via mutex)
- Independent execution state (`pipelines` map)
- Independent workspace

For parallel multi-pipeline execution, the meta-orchestrator spawns one goroutine per selected pipeline, each with its own child executor. `errgroup` handles coordination and cancellation propagation.

**Alternatives Rejected**:
- **New concurrency primitive**: Unnecessary when `errgroup` + `NewChildExecutor()` is proven.
- **Sequential execution with flag**: Would negate the performance benefit of parallel execution.

## R-010: Auto-Tuning Architecture (FR-015, FR-016, FR-017)

**Decision**: Analyze repository via file system inspection and populate runtime `CodebaseProfile`.

**Rationale**: Auto-tuning creates a `CodebaseProfile` by:
1. Detecting language: check for `go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, etc.
2. Detecting framework: parse dependency files for known frameworks
3. Detecting structure: count top-level directories, check for monorepo markers
4. Detecting test infrastructure: look for test files, config files (`jest.config`, `pytest.ini`, etc.)

The profile is used to:
- Populate `project` section suggestions in `wave.yaml` (but never overwrite existing values per FR-017)
- Add runtime context to persona prompts via template variables
- Create platform-specific pipeline variants (new files, not modifying generic ones per FR-016)

**Implementation**: New `internal/meta/tuning.go`. Returns `CodebaseProfile` struct. Auto-tuning is Phase 3 (P3 priority) — implementation can be deferred while the health check and proposal infrastructure is P1.
