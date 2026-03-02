# Tasks

## Phase 1: Foundation — Types and Forge Detection

- [X] Task 1.1: Create `internal/health/types.go` — Define `ForgeType` enum (`GitHub`, `GitLab`, `Bitbucket`, `Gitea`, `Unknown`), `ForgeAnalyzer` interface, `HealthReport` struct with sub-types (`CommitAnalysis`, `PRSummary`, `IssueSummary`, `CIStatus`), and configuration constants (commit window 30 days, staleness threshold 14 days, max items 100)
- [X] Task 1.2: Create `internal/health/detect.go` — Implement `DetectForge(repoPath string) (ForgeType, string, error)` that runs `git remote -v` and pattern-matches remote URLs to forge types, returning the forge type and repository identifier (e.g., `owner/repo`)
- [X] Task 1.3: Create `internal/health/detect_test.go` — Table-driven tests for forge detection covering: GitHub HTTPS, GitHub SSH, GitLab HTTPS, Bitbucket SSH, Gitea/Codeberg, self-hosted instances, unknown forges, repos with multiple remotes, repos with no remotes

## Phase 2: Core Implementation — GitHub Analyzer

- [X] Task 2.1: Create `internal/health/github.go` — Implement `GitHubAnalyzer` struct with methods: `AnalyzeCommits()` (uses `gh api` to fetch recent commits, compute frequency/authors/areas), `AnalyzePRs()` (fetch open PRs, categorize by review state/staleness/activity), `AnalyzeIssues()` (fetch open issues, categorize by labels/priority/actionability), `AnalyzeCIStatus()` (fetch recent workflow runs, compute pass rate) [P]
- [X] Task 2.2: Create `internal/health/stubs.go` — Implement `GitLabAnalyzer`, `BitbucketAnalyzer`, `GiteaAnalyzer` structs that implement `ForgeAnalyzer` interface with all methods returning a "not implemented" error containing a TODO marker and forge name [P]
- [X] Task 2.3: Create `internal/health/analyzer.go` — Implement top-level `Analyze(ctx context.Context, repoPath string, opts AnalyzeOptions) (*HealthReport, error)` that orchestrates: detect forge → select analyzer → gather all sections → assemble report

## Phase 3: Contract and Schema

- [X] Task 3.1: Create `.wave/contracts/health-analysis.schema.json` — Define JSON Schema (draft-07) for the health report artifact, with required fields: `forge_type`, `repository`, `analyzed_at`, `commits`, `pull_requests`, `issues`; optional fields: `ci_status`, `summary`
- [X] Task 3.2: Create `internal/health/schema_test.go` — Test that a complete `HealthReport` marshaled to JSON validates against the contract schema using `internal/contract.Validate()`

## Phase 4: Testing

- [X] Task 4.1: Create `internal/health/github_test.go` — Unit tests for GitHub analyzer with mocked `gh` CLI subprocess output: test commit parsing, PR categorization (review states, staleness), issue categorization (labels, priorities), CI status parsing [P]
- [X] Task 4.2: Create `internal/health/stubs_test.go` — Verify each stub analyzer returns appropriate "not implemented" errors and that the error messages include TODO markers [P]
- [X] Task 4.3: Create `internal/health/analyzer_test.go` — Integration test for `Analyze()` with mocked forge detection and subprocess; verify end-to-end artifact generation

## Phase 5: Polish

- [X] Task 5.1: Run `go test ./internal/health/...` and fix any failures
- [X] Task 5.2: Run `go vet ./internal/health/...` and address any issues
- [X] Task 5.3: Verify the health report schema matches all type fields and that the generated artifact is consumable by the `inject_artifacts` system
