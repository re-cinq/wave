# Tasks

## Phase 1: Extend Core Types

- [X] Task 1.1: Add `Remediation` field to `Result` struct in `internal/preflight/preflight.go`
- [X] Task 1.2: Add `AuthCheck` field to `Adapter` struct in `internal/manifest/types.go` for configurable adapter health commands
- [X] Task 1.3: Update existing `CheckTools` and `CheckSkills` methods to populate `Remediation` field with actionable guidance

## Phase 2: Adapter Health Checks

- [X] Task 2.1: Add `CheckAdapterHealth(adapters map[string]manifest.Adapter)` method to `Checker` that verifies binary reachability AND authentication status per adapter
- [X] Task 2.2: Implement adapter-specific auth probes (claude --version, opencode --version, generic --version fallback with configurable `AuthCheck` command)
- [X] Task 2.3: Write table-driven tests for `CheckAdapterHealth` covering found+authenticated, found+unauthenticated, and not-found scenarios

## Phase 3: Forge CLI Detection

- [X] Task 3.1: Create `internal/preflight/forge.go` with `DetectForge(remoteURL string) (forgeType string, cliBinary string)` function that maps git remote URLs to forge types and expected CLI binaries [P]
- [X] Task 3.2: Add `CheckForgeCLI(remoteURL string)` method to `Checker` that detects the forge from the remote URL and verifies the corresponding CLI is on PATH [P]
- [X] Task 3.3: Create `internal/preflight/forge_test.go` with table-driven tests for all forge URL patterns (github.com, gitlab.com, gitea.*, bitbucket.org) in both SSH and HTTPS formats [P]

## Phase 4: Wave Initialization Check

- [X] Task 4.1: Add `CheckWaveInit(waveDir string)` method to `Checker` that reads onboarding state and reports completed/not-completed with last-update timestamp
- [X] Task 4.2: Write tests for `CheckWaveInit` covering initialized, not-initialized, and corrupted state file scenarios

## Phase 5: Structured Report and Orchestrator

- [X] Task 5.1: Create `internal/preflight/report.go` with `SystemReadinessReport` struct (Timestamp, AllPassed, Checks, Summary) and JSON serialization
- [X] Task 5.2: Add `RunSystemReadiness(opts SystemReadinessOpts)` orchestrator function to `Checker` that runs all check categories (adapter health, forge CLI, skills, tools, wave init) and returns a `SystemReadinessReport`
- [X] Task 5.3: Create `internal/preflight/report_test.go` with tests for report construction, JSON marshaling, and overall pass/fail logic

## Phase 6: Testing and Validation

- [X] Task 6.1: Run `go test ./internal/preflight/...` to verify all existing tests pass
- [X] Task 6.2: Run `go test -race ./internal/preflight/...` to check for race conditions
- [X] Task 6.3: Run `go test ./internal/manifest/...` to verify manifest type changes don't break existing tests
- [X] Task 6.4: Run `go test ./...` full suite to verify no regressions

## Phase 7: Polish

- [X] Task 7.1: Verify `SystemReadinessReport` JSON output is consumable as a pipeline artifact (valid JSON, matches documented schema)
- [X] Task 7.2: Ensure remediation hints are actionable and reference correct install URLs/commands for each dependency type
