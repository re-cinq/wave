# Implementation Plan — Issue #1294

## Objective

Replace production panics in three library constructors/validators (`NewStallWatchdog`, `NewGitHubClient`, `ProvisionSkills`) with returned errors so CLI, webui, and tests can handle bad input gracefully. Add a `forbidigo` lint rule that flags new `panic(` calls inside `internal/` outside test files.

## Approach

1. Change each offending function's signature to return `error` (or for `ProvisionSkills`, replace the inner `panic` with the existing `error` return path).
2. Update every caller — production and test — to handle the new error.
3. Sweep `internal/` for residual `panic(` calls; allowlist the embedded-FS init panics in `internal/contract/schemas/shared/registry.go` with `//nolint:forbidigo` and a justification comment.
4. Wire `forbidigo` into `.golangci.yml` with a deny pattern for `panic` scoped via `exclusions` to test files only.
5. Run `go build ./...`, `go test ./...`, and `golangci-lint run` to verify.

## File Mapping

### Modified — production signatures

| File | Change |
|------|--------|
| `internal/pipeline/watchdog.go` | `NewStallWatchdog(timeout) (*StallWatchdog, error)`; return `errors.New(...)` instead of panic; update doc comment. |
| `internal/forge/github.go` | `NewGitHubClient(client) (*GitHubClient, error)`; return error if nil. |
| `internal/forge/token.go` | `NewClient(info)` propagates error from `NewGitHubClient`; signature becomes `(Client, error)`. |
| `internal/adapter/skills.go` | Replace `panic` in workspace-scope guard with `return fmt.Errorf("refusing skill provisioning outside workspace: ...")`. |

### Modified — production callers

| File | Change |
|------|--------|
| `internal/pipeline/executor.go` (line 1808) | Handle error from `NewStallWatchdog`; propagate as step setup failure. |
| `internal/webui/server.go` (line 150) | Handle error from `forge.NewClient`. |
| `internal/tui/app.go` (line 160) | Handle error from `forge.NewClient`. |

### Modified — test callers

| File | Change |
|------|--------|
| `internal/pipeline/watchdog_test.go` | Update four call sites to use `t.Helper`/error-checking; add a new test asserting the error path for invalid timeout. |
| `internal/forge/github_test.go` | Update three call sites; rewrite `NewGitHubClient(nil)` test to assert error rather than recover-from-panic. |
| `internal/doctor/codebase_test.go` (line 78) | Handle error. |
| `internal/adapter/skills_test.go` (line 113) | Update path-traversal test to assert returned error rather than recovered panic. |

### Modified — lint config

| File | Change |
|------|--------|
| `.golangci.yml` | Enable `forbidigo` linter with pattern `^panic$` (non-test, non-main). Add exclusion preset so the rule only fires in `internal/`. Justified `nolint` allowed because `nolintlint` is already enabled. |
| `internal/contract/schemas/shared/registry.go` | Add `//nolint:forbidigo // package-init guard; embedded FS read cannot fail at runtime` to the two panic lines. |

### Created

None — no new files required.

### Deleted

None.

## Architecture Decisions

- **AD1: Return `(T, error)` over `(T, error)` with options-pattern.** Single positional error is consistent with the rest of the codebase (`github.NewClient`, `pipeline.NewExecutor`).
- **AD2: Promote `forge.NewClient` to return `error`.** The current `NewClient` swallows the configuration outcome by returning `nil`. Since we have to thread the new `NewGitHubClient` error somewhere, surface it through `NewClient` so `webui`/`tui` callers can show a real reason. Keeps the API uniformly error-returning.
- **AD3: Use `forbidigo` over a custom analyzer.** It is part of the `default: standard` linter set in golangci-lint v2 and supports per-pattern messages. Scoping it to non-test files via the existing `exclusions.rules` matrix avoids new build-system surface area.
- **AD4: Allowlist embedded-FS init panics.** They are unreachable at runtime by Go's embed contract. Refactoring to lazy init would change error semantics across the contract package; deferred to a tracked follow-up.
- **AD5: Keep `ProvisionSkills` signature stable.** It already returns `error`; only the inner `panic` site changes. No caller updates required for that function.

## Risks

| Risk | Mitigation |
|------|------------|
| Cascading signature change on `forge.NewClient` touches webui/tui | Both callers are nil-guarded today; conversion to error-return is mechanical. Type-check via `go build` will catch any miss. |
| Tests that asserted-on-panic break silently | Audit `recover()` and `defer func()` patterns in the affected test files; rewrite to error assertions. |
| `forbidigo` flags allowed `panic` calls (e.g. registry.go) | Allowlist via `//nolint:forbidigo` with explanation; `nolintlint` already mandates explanations. |
| New lint rule blocks unrelated PRs | Scope rule strictly to `internal/`; verify on local lint run before pushing. |
| `forbidigo` v2 syntax differs from v1 | Project already uses golangci-lint v2 (`version: "2"` in `.golangci.yml`); use v2 syntax. |
| `NewClient` returning `(Client, error)` is a breaking change for any external caller | Wave is pre-1.0; per memory policy, no compatibility shims. Update internal callers and proceed. |

## Testing Strategy

### Unit tests
- `internal/pipeline/watchdog_test.go`: add `TestNewStallWatchdog_InvalidTimeout` covering zero and negative.
- `internal/forge/github_test.go`: replace recover-on-panic with `_, err := NewGitHubClient(nil); require.Error(...)`.
- `internal/forge/token_test.go` (existing or new): add a test that `NewClient` returns an error path when the token resolves but client construction fails (rare; smoke-only).
- `internal/adapter/skills_test.go`: convert path-traversal test to assert `err != nil` rather than recovering.

### Integration / build
- `go build ./...`
- `go test ./...` (no `-race` for contract paths per project convention; full race for unit tests where it currently runs).
- `golangci-lint run` — verifies `forbidigo` is wired and produces no false positives.

### Manual verification
- Run `wave run` on a trivial pipeline to confirm executor still constructs the watchdog correctly under valid input.
- `wave doctor` to confirm forge client construction surface still reports configuration cleanly.

## Sequencing

1. Library signature changes (watchdog, github, skills, token).
2. Production caller updates (executor, server, app).
3. Test updates.
4. Lint rule + allowlist comments.
5. Final `go build` / `go test` / `golangci-lint run` sweep.
