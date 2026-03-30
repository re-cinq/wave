# Wave Scope Definition

## What Wave IS

1. **A multi-agent pipeline orchestrator** that composes LLM-powered personas into reproducible, contract-validated workflows
2. **A subprocess executor** that wraps Claude Code and other LLM CLIs — it does not embed models or run inference directly
3. **A DAG engine** with composition primitives (iterate, aggregate, branch, loop, gate) for fan-out, conditional routing, and feedback loops
4. **A forge-agnostic tool** that works with GitHub, GitLab, Gitea, Forgejo, Codeberg, Bitbucket, and local git
5. **An observable system** with structured events, audit logging, WebUI dashboard, and retrospective generation

## What Wave IS NOT

- **Not an LLM framework** — does not train, fine-tune, or serve models
- **Not a CI/CD system** — orchestrates AI agents, not build pipelines (though it can gate on CI)
- **Not a browser automation tool** — the browser adapter is for vision-based UX audits, not general web scraping
- **Not a deployment tool** — produces PRs and artifacts, does not deploy code

## Package Inventory

### Core Engine (must-have)

| Package | Lines | Purpose | Status |
|---------|-------|---------|--------|
| `pipeline` | 17,240 | DAG executor, composition primitives, step orchestration | Active — primary engine |
| `adapter` | 4,626 | Subprocess execution (Claude, browser, mock) | Active |
| `contract` | 4,405 | Output validation (JSON schema, test suite, LLM judge) | Active |
| `state` | 4,076 | SQLite persistence, run records, event log | Active |
| `manifest` | 1,316 | Configuration loading and validation | Active |
| `event` | 249 | Progress event types and emission | Active |
| `workspace` | 356 | Ephemeral workspace management | Active |
| `worktree` | 134 | Git worktree lifecycle | Active |

### User Interface

| Package | Lines | Purpose | Status |
|---------|-------|---------|--------|
| `tui` | 15,131 | Bubble Tea terminal UI (guided mode, fleet view) | Active |
| `webui` | 8,339 | Embedded web dashboard | Active |
| `display` | 4,788 | Terminal progress display and formatting | Active |

### Supporting Infrastructure

| Package | Lines | Purpose | Status |
|---------|-------|---------|--------|
| `onboarding` | 3,066 | `wave init` flow, flavour detection, metadata extraction | Active |
| `doctor` | 2,782 | Project health checking | Active |
| `skill` | 2,328 | Skill discovery and provisioning | Active |
| `forge` | 855 | Git forge detection (GitHub/GitLab/Gitea/etc.) | Active |
| `hooks` | 835 | Lifecycle hooks and webhook delivery | Active |
| `security` | 1,155 | Input sanitization and path validation | Active |
| `defaults` | 257 | Embedded default personas/pipelines/contracts | Active |
| `preflight` | 647 | Pipeline dependency validation and auto-install | Active |
| `recovery` | 295 | Error guidance and resume hints | Active |
| `suggest` | 518 | Pipeline suggestion engine | Active |
| `audit` | 388 | Audit logging and credential scrubbing | Active |
| `scope` | 604 | Persona token scope validation | Active |
| `cost` | 237 | Cost ledger and budget enforcement | Active |
| `deliverable` | 513 | Pipeline deliverable tracking | Active |
| `retro` | 659 | Run retrospective generation | Active |
| `pathfmt` | 22 | Path formatting utilities | Active |
| `timeouts` | 29 | Timeout constants | Active |

### Specialized / Niche

| Package | Lines | Purpose | Status |
|---------|-------|---------|--------|
| `relay` | 444 | Context compaction and summarization | Active — wired 2026-03-30 |
| `bench` | 749 | SWE-bench benchmarking | Active — used by `wave bench` |
| `continuous` | 491 | Continuous execution mode (`--continuous`) | Active — used by `wave run --continuous` |
| `github` | 1,310 | GitHub API integration for issue enhancement | Active |
| `sandbox` | 198 | Docker and bubblewrap sandbox backends | Active |

### Test-only

| Package | Lines | Purpose | Status |
|---------|-------|---------|--------|
| `testutil` | 770 | Test helpers (event collector, manifest factory) | Test infrastructure |

## Dead Code Removed (2026-03-30)

### Cleaned this session
- `continuous`: `IterationStatus` type, `IterationResult.RunID/Error/Status`, `SourceConfig.RawURI`, `GitHubSource.Sort/Direction`
- `bench`: `LoadSWEBenchLite`, `RunConfig.CacheDir/WaveBinary/ClaudeBinary`, `BenchTask.Version/ExpectedPatch`, `BenchResult.TokensUsed`

### Remaining dead code (low priority)
- `relay`: Sentinel errors (`ErrInvalidCheckpoint`, `ErrWriteCheckpointFailed`) never wrapped by return values, `Checkpoint.Context` field always empty, `validateConfig` only called from tests
- `pipeline/composition_state.go`: `IterationState`, `GateState`, `SaveIterationState`, `LoadIterationState` — file-based state persistence never used (executor uses SQLite)

## Unwired Features Found and Fixed

| Feature | Package | Status |
|---------|---------|--------|
| Context compaction | `relay` | **Wired** — `WithRelayMonitor` now called when `relay.token_threshold_percent > 0` |
| Composition primitives | `pipeline` | **Wired** — iterate/aggregate/branch/loop now dispatched in `executeCompositionStep` |
| Child run records | `pipeline` | **Fixed** — child sub-pipelines now create `pipeline_run` records via `createRunID` |
| Child event logging | `cmd/wave` | **Fixed** — `dbLoggingEmitter` uses `ev.PipelineID` instead of hardcoded parent ID |

## Duplicate Implementations

| Concern | Implementations | Resolution |
|---------|----------------|------------|
| Pipeline state | `pipeline_state` table + `pipeline_run` table | Historical — `pipeline_run` is the source of truth for WebUI, `pipeline_state` for executor internals. Merging would be a large refactor. |
| Template resolution | `PipelineContext.ResolvePlaceholders` + `TemplateContext` (ResolveTemplate) | Different scopes — `PipelineContext` for pipeline-level vars, `TemplateContext` for composition step expressions. Bridged via `resolveStepOutputRef`. |
| Sandbox backends | bubblewrap (`sandbox/bwrap.go`) + Docker (`sandbox/docker.go`) | By design — different deployment targets. No duplication. |
| Composition executor | `CompositionExecutor` + `DefaultPipelineExecutor` composition methods | `CompositionExecutor` is legacy dead code — only used in tests. The real execution goes through `DefaultPipelineExecutor`. Consider removing. |

## ADR Status

No ADRs directory exists (`docs/adr/` is empty or missing). Architecture decisions are documented in:
- `CLAUDE.md` — primary source of truth for development guidelines
- `docs/guides/` — operational guides
- Memory files in `.claude/projects/` — session learnings

Recommendation: formalize key decisions as ADRs using the `plan-adr` pipeline.
