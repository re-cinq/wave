# Work Items — Issue #1294

## Phase 1: Setup
- [ ] 1.1: Confirm baseline `go build ./...` and `go test ./...` are green on the new branch.
- [ ] 1.2: Confirm `golangci-lint run` baseline is green and `forbidigo` is not yet enabled.

## Phase 2: Library Signature Changes
- [ ] 2.1: Convert `NewStallWatchdog(timeout)` to `(*StallWatchdog, error)` in `internal/pipeline/watchdog.go`. [P]
- [ ] 2.2: Convert `NewGitHubClient(client)` to `(*GitHubClient, error)` in `internal/forge/github.go`. [P]
- [ ] 2.3: Convert `forge.NewClient(info)` to `(Client, error)` in `internal/forge/token.go`, propagating the new `NewGitHubClient` error.
- [ ] 2.4: Replace `panic` in `ProvisionSkills` workspace-scope guard with `fmt.Errorf` in `internal/adapter/skills.go`. [P]

## Phase 3: Production Caller Updates
- [ ] 3.1: Update `internal/pipeline/executor.go:1808` to handle `NewStallWatchdog` error (treat as step-setup failure).
- [ ] 3.2: Update `internal/webui/server.go:150` to handle `forge.NewClient` error.
- [ ] 3.3: Update `internal/tui/app.go:160` to handle `forge.NewClient` error.

## Phase 4: Test Updates
- [ ] 4.1: Update `internal/pipeline/watchdog_test.go` four call sites + add `TestNewStallWatchdog_InvalidTimeout`. [P]
- [ ] 4.2: Update `internal/forge/github_test.go` three call sites + rewrite nil-client test to assert error. [P]
- [ ] 4.3: Update `internal/doctor/codebase_test.go:78` for new error return. [P]
- [ ] 4.4: Update `internal/adapter/skills_test.go:113` path-traversal test to assert returned error. [P]

## Phase 5: Lint Rule
- [ ] 5.1: Enable `forbidigo` in `.golangci.yml` with pattern `^panic$`, scoped to `internal/` non-test files via `exclusions.rules`.
- [ ] 5.2: Add `//nolint:forbidigo // package-init guard, embedded FS read cannot fail at runtime` to both panic lines in `internal/contract/schemas/shared/registry.go`.
- [ ] 5.3: Sweep `internal/` once more for any remaining panics; document or convert.

## Phase 6: Validation
- [ ] 6.1: `go build ./...` clean.
- [ ] 6.2: `go test ./...` clean.
- [ ] 6.3: `golangci-lint run` clean (verifies `forbidigo` rule is in place and no unintended hits).
- [ ] 6.4: Smoke-run `wave run` on a trivial pipeline + `wave doctor` to verify executor and forge construction paths.

## Phase 7: Polish
- [ ] 7.1: Update doc comments on each changed function to reflect new error return.
- [ ] 7.2: Final review of diff for stray `panic` introductions.
