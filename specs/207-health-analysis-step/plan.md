# Implementation Plan: Codebase Health Analysis Step

## 1. Objective

Add a new pipeline step type that performs forge-aware codebase health analysis, producing a structured JSON artifact consumed by the proposal engine. GitHub is the primary forge target; other forges get stub implementations.

## 2. Approach

### Architecture: New `internal/health` package

Create a new `internal/health` package that encapsulates the forge abstraction and health analysis logic. This follows Wave's single-responsibility-per-package convention (like `internal/github`, `internal/preflight`, etc.).

**Forge abstraction pattern:**
- Define a `ForgeAnalyzer` interface with methods for gathering commits, PRs, issues, and CI status
- Implement `GitHubAnalyzer` as the primary concrete type using the `gh` CLI
- Add `GitLabAnalyzer`, `BitbucketAnalyzer`, `GiteaAnalyzer` stubs with TODO markers
- Provide a `DetectForge()` function that parses `git remote -v` to identify the forge type

**Data flow:**
1. Pipeline step invokes health analysis with repo context
2. `DetectForge()` identifies the forge from git remotes
3. Appropriate `ForgeAnalyzer` is instantiated
4. Analyzer gathers data via forge CLI (e.g., `gh api` for GitHub)
5. Results are assembled into a structured `HealthReport` and written as JSON artifact
6. Contract schema validates the output

### Integration: Pipeline step and contract

- Create a new pipeline YAML that includes a health-analysis step
- The step uses a persona with permissions to run `gh` CLI commands
- Output is a JSON artifact validated by a new `health-analysis.schema.json` contract
- The artifact is injectable into downstream steps via `inject_artifacts`

## 3. File Mapping

### New Files (create)

| Path | Description |
|------|-------------|
| `internal/health/types.go` | Health report types, forge enum, `ForgeAnalyzer` interface |
| `internal/health/detect.go` | Forge detection from git remote URLs |
| `internal/health/detect_test.go` | Table-driven tests for forge detection |
| `internal/health/github.go` | GitHub health analyzer using `gh` CLI |
| `internal/health/github_test.go` | Tests for GitHub analyzer (mocked `gh` output) |
| `internal/health/stubs.go` | Stub analyzers for GitLab, Bitbucket, Gitea |
| `internal/health/stubs_test.go` | Tests verifying stubs return "not implemented" errors |
| `internal/health/analyzer.go` | Top-level `Analyze()` orchestrator function |
| `internal/health/analyzer_test.go` | Integration tests for the orchestrator |
| `.wave/contracts/health-analysis.schema.json` | JSON Schema for the health report artifact |

### Modified Files (modify)

| Path | Description |
|------|-------------|
| _None_ | This feature is additive — no existing files need modification |

### Notes

- No modifications to `internal/pipeline/executor.go` or `internal/manifest/types.go` are needed. The health analysis step will be consumed as a regular pipeline step with a persona that invokes the health analysis. The pipeline YAML and prompt define how the persona gathers and writes the artifact.
- However, the health analysis logic itself lives in a Go package so it can be unit-tested independently and potentially invoked programmatically (not just via pipeline persona).

## 4. Architecture Decisions

### AD-1: New package vs. extending `internal/github`
**Decision**: New `internal/health` package.
**Rationale**: The `internal/github` package is specific to GitHub API types and client. Health analysis spans multiple forges and has different concerns (data aggregation, categorization). Mixing forge-agnostic analysis with GitHub-specific types would violate single responsibility.

### AD-2: `gh` CLI vs. Go HTTP client for GitHub data
**Decision**: Use `gh` CLI via subprocess.
**Rationale**: Wave's security model requires forge CLI validation via preflight (`requires: tools: [gh]`). The `gh` CLI handles authentication, token management, and pagination. Using the existing `internal/github` Go client would bypass the sandbox model. The `gh api` command provides direct access to any GitHub API endpoint.

### AD-3: Forge detection strategy
**Decision**: Parse `git remote -v` output to match URL patterns.
**Rationale**: This is the simplest reliable approach — no config needed, works with any repo. Patterns:
- `github.com` → GitHub
- `gitlab.com` or self-hosted with `/gitlab` → GitLab
- `bitbucket.org` → Bitbucket
- `gitea` or `codeberg.org` → Gitea
Falls back to "unknown" if no pattern matches.

### AD-4: Commit history window
**Decision**: Last 30 days or 100 commits (whichever is smaller).
**Rationale**: This provides a meaningful activity snapshot without excessive API calls. The window is defined as a constant in the types so it can be easily adjusted.

### AD-5: PR staleness threshold
**Decision**: PRs with no activity for 14 days are considered "stale".
**Rationale**: This is a common industry convention. Defined as a configurable constant.

## 5. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `gh` CLI not authenticated | Medium | Step fails | Preflight check validates `gh auth status` before pipeline runs (dependency #206) |
| Rate limiting on large repos | Low | Slow/incomplete data | Use `--paginate` where needed; cap results with `--limit` flags |
| Git remote URL patterns vary | Medium | Wrong forge detected | Comprehensive test suite with URL variants; fallback to "unknown" |
| Schema evolution | Low | Breaking changes | Version the schema; artifact consumers handle missing optional fields gracefully |
| Large monorepo with thousands of issues/PRs | Low | Memory/timeout | Cap at 100 items per category; use `--limit` in `gh` queries |

## 6. Testing Strategy

### Unit Tests
- **Forge detection** (`detect_test.go`): Table-driven tests with various remote URL formats (SSH, HTTPS, self-hosted, custom domains)
- **GitHub analyzer** (`github_test.go`): Mock `gh` CLI output (JSON fixtures) to test parsing, categorization, staleness calculation
- **Stub analyzers** (`stubs_test.go`): Verify each stub returns appropriate "not implemented" error
- **Type validation** (`types_test.go`): Ensure health report types serialize to valid JSON matching the schema

### Integration Tests
- **Analyzer orchestrator** (`analyzer_test.go`): End-to-end test with mocked subprocess execution
- **Schema compliance**: Generate a health report and validate against `health-analysis.schema.json` using the existing `internal/contract` package

### Contract Tests
- The pipeline contract validates the output artifact at runtime
- Test that a well-formed artifact passes schema validation
- Test that malformed artifacts fail schema validation
