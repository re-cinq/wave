# Implementation Plan: Preflight System Readiness

## Objective

Extend the existing `internal/preflight/` package to serve as a comprehensive system readiness gate for `wave run wave`. The current preflight system checks per-pipeline tool/skill dependencies; this extension adds adapter health verification (binary + authentication), forge CLI detection, Wave initialization status, and structured artifact output with remediation hints.

## Approach

Extend the existing `Checker` and `Result` types in `internal/preflight/preflight.go` rather than creating a parallel system. Add a `Remediation` field to `Result`, introduce new check methods for adapter health, forge CLIs, and Wave initialization, and add a `SystemReadiness` orchestrator function that runs all checks and produces a structured report.

The implementation follows the existing pattern: `Checker` methods return `[]Result` and typed errors, with injectable command runners for testability.

## File Mapping

### Modified Files

| File | Action | Purpose |
|------|--------|---------|
| `internal/preflight/preflight.go` | modify | Add `Remediation` field to `Result`; add `CheckAdapterHealth`, `CheckForgeCLI`, `CheckWaveInit` methods; add `SystemReadinessReport` type and `RunSystemReadiness` orchestrator |
| `internal/preflight/preflight_test.go` | modify | Add tests for all new check methods and the orchestrator |
| `internal/manifest/types.go` | modify | Add optional `auth_check` field to `Adapter` struct for adapter health verification commands |

### New Files

| File | Action | Purpose |
|------|--------|---------|
| `internal/preflight/forge.go` | create | Forge CLI detection logic: map git remote URL to forge type (github/gitlab/gitea/bitbucket), return the expected CLI binary name |
| `internal/preflight/forge_test.go` | create | Tests for forge detection and CLI mapping |
| `internal/preflight/report.go` | create | `SystemReadinessReport` struct with JSON marshaling for structured artifact output |
| `internal/preflight/report_test.go` | create | Tests for report serialization and overall pass/fail logic |

## Architecture Decisions

### 1. Extend `Result` with `Remediation` field

The existing `Result` struct has `Name`, `Kind`, `OK`, `Message`. Add a `Remediation` string field with actionable install/fix guidance. This is backward-compatible — existing code that doesn't set it gets empty string.

### 2. Adapter health = binary + auth check

- **Binary check**: Already covered by `CheckTools` (LookPath).
- **Auth check**: Run a lightweight probe command per adapter:
  - `claude`: `claude --version` (exits 0 if authenticated, non-zero otherwise)
  - `opencode`: `opencode --version`
  - Generic: Run the binary with `--version` or a configurable `auth_check` command from the manifest `Adapter.AuthCheck` field.
- The `Checker` gains a `CheckAdapterHealth(adapters map[string]manifest.Adapter)` method.

### 3. Forge CLI detection from git remote

Detect the forge type by parsing the git remote URL:
- `github.com` → needs `gh` CLI
- `gitlab.com` or self-hosted GitLab → needs `glab` CLI
- `gitea.*` or Forgejo → needs `tea` CLI
- `bitbucket.org` → needs `bb` CLI

Use `git remote get-url origin` (injectable for testing) to get the remote, then pattern-match. The `Checker` gains a `CheckForgeCLI(remoteURL string)` method that returns the expected CLI and checks if it's on PATH.

### 4. Wave initialization status

Read `onboarding.ReadState(waveDir)` to check:
- Whether onboarding is completed
- When it was completed (`CompletedAt` timestamp)
- Display as "Wave initialized: yes (last updated: 2025-01-15)" or "Wave not initialized — run `wave init`"

### 5. Structured artifact output

Define a `SystemReadinessReport` struct:
```go
type SystemReadinessReport struct {
    Timestamp   time.Time `json:"timestamp"`
    AllPassed   bool      `json:"all_passed"`
    Checks      []Result  `json:"checks"`
    Summary     string    `json:"summary"`
}
```

This serializes to JSON for downstream pipeline steps to consume.

### 6. Remediation hints per check

Each `Result` gets a `Remediation` string:
- Missing tool: "Install with: `brew install gh` or visit https://cli.github.com"
- Missing skill: "Install with: `wave skill install speckit`"
- Adapter not authenticated: "Run `claude` to complete authentication"
- Wave not initialized: "Run `wave init` to set up your project"

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| `claude --version` behavior may differ across versions | Medium | Use a simple binary invocation that's unlikely to change; make the command configurable via `auth_check` manifest field |
| Forge detection from git remote may miss non-standard hosting | Low | Fall back gracefully — if no forge detected, skip the check rather than fail |
| Adding `Remediation` field to `Result` breaks existing JSON serialization consumers | Low | Field is additive (empty string default), no breaking change |
| Git remote parsing may fail in detached-HEAD CI environments | Low | Return a "skipped" result when git remote is unavailable, rather than failing |

## Testing Strategy

1. **Unit tests for each new check method**: Mock `runCmd` and git remote to test all code paths
2. **Forge detection tests**: Table-driven tests for GitHub, GitLab, Gitea, Bitbucket URL patterns (SSH and HTTPS)
3. **Report serialization tests**: Verify JSON output matches expected schema
4. **Integration with existing tests**: Ensure `go test ./internal/preflight/...` passes with zero regressions
5. **Backward compatibility**: Existing `CheckTools`/`CheckSkills`/`Run` behavior unchanged; new methods are additive
