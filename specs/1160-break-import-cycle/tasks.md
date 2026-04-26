# Work Items

## Phase 1: Setup
- [ ] 1.1: Create `internal/suggest/types.go` with promoted types: `Status` (+`String`,`MarshalJSON`), `CheckResult`, `Report`, `CodebaseHealth`, `PRSummary`, `IssueSummary`, `CIStatus`, `HealthCheckMsg`.
- [ ] 1.2: Add `internal/suggest/types_test.go` covering `Status.String` and `MarshalJSON` for OK/Warn/Err.

## Phase 2: Doctor Migration
- [ ] 2.1: Delete duplicated types from `internal/doctor/types.go`; replace with `suggest.*` references. Drop `onboarding` import; remove default `isOnboarded` fallback (require `Options.CheckOnboarded`).
- [ ] 2.2: Move `CodebaseHealth`/`PRSummary`/`IssueSummary`/`CIStatus` declarations out of `internal/doctor/codebase.go`; update doctor builders to populate `suggest.*` types. [P]
- [ ] 2.3: Sweep `internal/doctor/{checks_config,checks_deps,checks_infra,checks_ontology,doctor,optimize,profile,scan}.go` and matching `_test.go` for `Status*` / `CheckResult` / `Report` references → `suggest.*`. [P]
- [ ] 2.4: Fix doctor tests (`doctor_test.go`, `codebase_test.go`, `optimize_test.go`, `profile_test.go`) to construct `suggest.*` literals. [P]

## Phase 3: Suggest Engine Cleanup
- [ ] 3.1: Drop `import "internal/doctor"` from `internal/suggest/engine.go`; switch `EngineOptions.Report` to local `*Report`.
- [ ] 3.2: Update `internal/suggest/engine_test.go` to construct `&Report{Codebase: &CodebaseHealth{…}}` directly.

## Phase 4: TUI Migration
- [ ] 4.1: In `internal/tui/health_provider.go`, delete `HealthCheckStatus` enum + `HealthCheckResultMsg`; switch all `RunCheck` returns to `suggest.HealthCheckMsg`. Add `type HealthCheckResultMsg = suggest.HealthCheckMsg` alias for switch-case stability.
- [ ] 4.2: Replace `HealthCheckOK/Warn/Err` constants with `suggest.StatusOK/Warn/Err` across `health_list.go`, `health_detail.go`, `content.go`. [P]
- [ ] 4.3: Update `internal/tui/health_list_test.go` and any other touched tests. [P]
- [ ] 4.4: Confirm/remove `HealthCheckChecking` (TUI spinner uses absence-of-result, not enum).

## Phase 5: Webui + cmd Migration
- [ ] 5.1: Update `internal/webui/handlers_health.go` `healthStatusString` to switch on `suggest.Status*`; update return type bindings. [P]
- [ ] 5.2: Update `internal/webui/handlers_health_test.go` for new types. [P]
- [ ] 5.3: Update `cmd/wave/commands/doctor.go` — `doctor.Status*` → `suggest.Status*`, `*doctor.Report` → `*suggest.Report`. Inject `CheckOnboarded: onboarding.IsOnboarded` into `doctor.Options`. [P]
- [ ] 5.4: Update `cmd/wave/commands/doctor_test.go`, `analyze.go`, `analyze_test.go`, `suggest.go`, `run.go` references. [P]

## Phase 6: Verification
- [ ] 6.1: `go build ./...` — clean.
- [ ] 6.2: `go vet ./...` — clean.
- [ ] 6.3: `go test -race ./...` — green.
- [ ] 6.4: `go list -deps ./internal/suggest ./internal/doctor ./internal/tui ./internal/onboarding` — confirm suggest has no doctor/tui import; doctor has no onboarding import; tui has no doctor import.
- [ ] 6.5: `wave doctor` smoke run — output identical to pre-change.
- [ ] 6.6: `wave compose` TUI smoke — health view loads, statuses render.

## Phase 7: Polish
- [ ] 7.1: Update any package doc comments that reference the old type locations.
- [ ] 7.2: Commit with `refactor(suggest): promote shared health types to break tui↔doctor cycle`.
- [ ] 7.3: Open PR linking #1160.
