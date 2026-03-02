# Implementation Plan: Forge Detection

## 1. Objective

Auto-detect the repository's forge type (GitHub, GitLab, Bitbucket, Gitea) from git remote URLs and optional `wave.yaml` domain mappings, then filter the pipeline catalog to the correct forge family (`gh-*`, `gl-*`, `bb-*`, `gt-*`) so that `wave run` only proposes relevant pipelines.

## 2. Approach

Create a new `internal/forge/` package that:

1. **Parses git remote URLs** — executes `git remote -v` and extracts hostnames from SSH and HTTPS URLs.
2. **Classifies forge type** — matches hostnames against built-in defaults (github.com, gitlab.com, bitbucket.org) and user-configured domain mappings from `wave.yaml`.
3. **Exposes a structured `ForgeDetection` result** — containing the detected forge type, remote URL, confidence level, and CLI tool name.
4. **Filters the pipeline catalog** — provides a function that takes a list of pipeline names and returns only those matching the detected forge prefix.
5. **Handles multi-forge repos** — when multiple remotes point to different forges, returns all detected forges with a method to pick a primary.

Integration points:
- **Manifest types** — add a `Forge` configuration section to `wave.yaml` for domain mappings.
- **TUI pipeline selector** — pass forge filter into `RunPipelineSelector` to pre-filter the pipeline list.
- **Run command** — invoke forge detection before pipeline loading and pass the result to the selector.
- **Preflight** — expose forge detection result so the preflight checker can validate the correct CLI tool (e.g., `gh` for GitHub).

## 3. File Mapping

### New files (create)
| Path | Purpose |
|------|---------|
| `internal/forge/forge.go` | Core types: `ForgeType`, `ForgeDetection`, `ForgeConfig` |
| `internal/forge/detect.go` | Detection logic: git remote parsing, hostname classification |
| `internal/forge/filter.go` | Pipeline filtering by forge prefix |
| `internal/forge/detect_test.go` | Unit tests for detection logic |
| `internal/forge/filter_test.go` | Unit tests for pipeline filtering |

### Modified files (modify)
| Path | Change |
|------|--------|
| `internal/manifest/types.go` | Add `Forge *ForgeConfig` field to `Manifest` struct |
| `internal/tui/run_selector.go` | Accept optional forge filter to pre-filter pipeline list |
| `cmd/wave/commands/run.go` | Call forge detection before pipeline selection; pass filter to TUI |
| `cmd/wave/commands/list.go` | Optionally show forge-filtered pipelines |

## 4. Architecture Decisions

### AD-1: New `internal/forge/` package (not `internal/manifest/`)
The issue suggests either location. A dedicated package is better because:
- Forge detection requires subprocess execution (`git remote -v`) which is not configuration-adjacent.
- Keeps manifest types clean — the manifest only holds the domain mapping config.
- Easier to test in isolation.

### AD-2: Built-in defaults with configurable overrides
Default domain-to-forge mappings are hardcoded for the well-known forges (github.com, gitlab.com, bitbucket.org). Self-hosted instances and Gitea use `wave.yaml` `forge.domains` config.

### AD-3: Forge prefix convention
The existing pipeline naming convention (`gh-*`, `gl-*`, `bb-*`, `gt-*`) is the filtering mechanism. No metadata changes to pipeline YAML needed — filtering is purely name-based prefix matching. Non-prefixed pipelines (e.g., `prototype`, `hotfix`) are always included.

### AD-4: Git remote parsing without forge CLI
Detection uses `git remote -v` output, not forge-specific CLIs. This ensures detection works even if `gh`, `glab`, or `tea` aren't installed. The result includes which CLI *should* be available, so preflight can validate it.

### AD-5: ForgeType as a string type
Use `type ForgeType string` with constants (`GitHub`, `GitLab`, `Bitbucket`, `Gitea`, `Unknown`). This is idiomatic Go for the existing codebase patterns and easy to serialize to JSON for the readiness artifact.

## 5. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| SSH URLs with custom ports/paths may not parse cleanly | Medium | Low | Use `net/url` with SSH prefix stripping; add test cases for edge formats |
| Self-hosted Gitea instances have no default domain pattern | High | Medium | Require explicit `wave.yaml` config; document in forge section |
| Multi-forge repos cause confusion if auto-selected wrong | Low | Medium | When ambiguous, return all detected forges; let TUI prompt user |
| Dependency on #206 system readiness not yet implemented | Medium | Low | Design forge detection as standalone; add a `ToReadinessResult()` method that can integrate later |
| Pipeline naming convention drift | Low | Medium | Document the prefix convention; add validation that warns if forge-prefixed pipelines use wrong personas |

## 6. Testing Strategy

### Unit tests (`internal/forge/`)
- **URL parsing**: SSH (`git@github.com:org/repo.git`), HTTPS (`https://github.com/org/repo.git`), custom ports, enterprise domains
- **Classification**: Each forge type correctly detected from its canonical domain; unknown domains return `Unknown`
- **Custom domain mapping**: User-configured domains override defaults correctly
- **Pipeline filtering**: Only matching-prefix pipelines returned; non-prefixed pipelines always included
- **Multi-forge**: Multiple remotes to different forges detected and returned
- **Edge cases**: No remotes, empty config, malformed URLs

### Integration tests
- **Run command flow**: Forge detection integrates with pipeline selection (mock git remotes)
- **Preflight integration**: Detected forge CLI requirement feeds into preflight checks

### Test approach
- Table-driven tests per the codebase convention
- Mock `git remote -v` output via dependency injection (pass a `gitRemoteFn` function)
- No external dependencies needed — all tests use test fixtures
