# Implementation Plan — #1160

## Objective

Move shared health-check types out of `internal/doctor` and `internal/tui` into `internal/suggest` so the three packages no longer duplicate definitions and no longer risk the tui↔doctor↔onboarding import cycle.

## Approach

Invert the current `suggest → doctor` dependency. `internal/suggest` becomes the canonical home for the cross-package types (`Status`, `CheckResult`, `Report`, `CodebaseHealth`, `PRSummary`, `IssueSummary`, `CIStatus`). `doctor` and `tui` both depend on `suggest` for those types instead of redefining them. The `doctor → onboarding` default callback is dropped (or moved) so the only edge between the three packages is `onboarding → tui`, which is not a cycle.

Rename map (package-qualified):

| Old | New |
|-----|-----|
| `doctor.Status` | `suggest.Status` |
| `doctor.StatusOK/Warn/Err` | `suggest.StatusOK/Warn/Err` |
| `doctor.CheckResult` | `suggest.CheckResult` |
| `doctor.Report` | `suggest.Report` |
| `doctor.CodebaseHealth` | `suggest.CodebaseHealth` |
| `doctor.PRSummary/IssueSummary/CIStatus` | `suggest.PRSummary/IssueSummary/CIStatus` |
| `tui.HealthCheckStatus` | `suggest.Status` (drop `Checking` — see decision 3) |
| `tui.HealthCheckResultMsg` | new `suggest.HealthCheckMsg` (Bubbletea msg form, includes `Details`) |

## File Mapping

### Create
- `internal/suggest/types.go` — new file housing the promoted types (`Status` + `String`/`MarshalJSON`, `CheckResult`, `Report`, `CodebaseHealth`, `PRSummary`, `IssueSummary`, `CIStatus`, optional `HealthCheckMsg` for tui async).

### Modify
- `internal/doctor/types.go` — delete `Status`, `CheckResult`, `Report`; alias to `suggest` types or replace usages. Drop `onboarding` import + default `isOnboarded`. Caller must inject `CheckOnboarded` (cmd/wave/commands/doctor.go is the one production caller).
- `internal/doctor/codebase.go` — `CodebaseHealth`, `PRSummary`, `IssueSummary`, `CIStatus` move to `suggest`; doctor refs become `suggest.X`.
- `internal/doctor/checks_*.go`, `doctor.go`, `optimize.go`, `profile.go`, `scan.go` — replace `Status*`, `CheckResult`, `Report` with `suggest.*`.
- `internal/doctor/*_test.go` — same replacements.
- `internal/suggest/engine.go` — `EngineOptions.Report` becomes `*Report` (local); `opts.Report.ForgeInfo` field path unchanged.
- `internal/suggest/engine_test.go` — drop `doctor` import; construct `&Report{Codebase: &CodebaseHealth{…}}`.
- `internal/tui/health_provider.go` — drop local `HealthCheckStatus`, `HealthCheckResultMsg`. Provider returns `suggest.HealthCheckMsg` (or accepts `tui.HealthCheckResultMsg = suggest.HealthCheckMsg` type alias for minimal call-site churn).
- `internal/tui/content.go` — `case HealthCheckResultMsg:` → updated alias.
- `internal/tui/health_list*.go`, `health_detail.go` — use `suggest.Status*` constants.
- `internal/webui/handlers_health.go` — `healthStatusString` switches on `suggest.Status*`. Provider call still routes through `tui.NewDefaultHealthDataProvider`, but its return type changes.
- `internal/webui/handlers_health_test.go` — match new types.
- `cmd/wave/commands/doctor.go` — replace `doctor.Status*`, `*doctor.Report` with `suggest.Status*`, `*suggest.Report`. Pass `CheckOnboarded: onboarding.IsOnboarded` explicitly when constructing `doctor.Options` (cmd already imports onboarding indirectly — verify).
- `cmd/wave/main.go` — verify suggest/doctor imports unchanged.

### Delete
- None.

## Architecture Decisions

1. **Home = `internal/suggest`** — required by issue acceptance ("promote shared types here"). Trade-off: the package now mixes pipeline-suggestion logic with health primitives. Acceptable because suggest's `Engine` already consumes a `Report`; the types are its input shape, not foreign concepts.
2. **No new package** — considered `internal/health` as a leaner home, but the issue is explicit. Stick with suggest, document the dual purpose with file-level naming (`types.go` for primitives, `engine.go` for suggestion logic).
3. **Drop `HealthCheckChecking`** — the `Checking` enum value exists in tui only and is never set in any `RunCheck` path (grep confirms). The "checking" UX is rendered via spinner in `health_list.go` based on absence-of-result, not via a dedicated status. Dropping simplifies the unified enum.
4. **Sever `doctor → onboarding`** — `Options.CheckOnboarded` already accepts an injection. Default fallback `onboarding.IsOnboarded(waveDir)` deletes; callers (only `cmd/wave/commands/doctor.go`) must pass it. This kills the doctor→onboarding edge without requiring `IsOnboarded` to relocate.
5. **Type aliases for tui shim** — `type HealthCheckResultMsg = suggest.HealthCheckMsg` keeps Bubbletea message switches unchanged at first; can be removed in a follow-up after callers migrate.

## Risks

- **Test churn** — `internal/suggest/engine_test.go` constructs `doctor.Report` literals 12+ times. Mass replace; verify no straggler imports.
- **JSON shape** — `Status.MarshalJSON` emits string form. Ensure the moved method preserves byte-for-byte identical output (`"ok"|"warn"|"error"`). Doctor JSON consumers (CLI, webui) must see no diff.
- **`HealthCheckChecking` removal** — verify no production read; only tests/UI may reference. If any does, keep enum but document it as tui-only sentinel via a separate `tui` const.
- **Webui handler regression** — `handlers_health_test.go` exercises status-string mapping; cover with assertions on each enum value.
- **cmd/wave/main.go** wires doctor at startup — confirm `Options.CheckOnboarded` injection point and don't drop the onboarded check silently.

## Testing Strategy

- **Unit:** existing `internal/suggest/engine_test.go`, `internal/doctor/*_test.go`, `internal/tui/health_*_test.go`, `internal/webui/handlers_health_test.go` updated to new types. Add a tiny test in `internal/suggest` for `Status.String()` + `MarshalJSON`.
- **Build:** `go build ./...` clean, `go vet ./...` clean.
- **Race:** `go test -race ./...` — covers any concurrent test in tui or webui.
- **Manual:** `wave doctor` against this repo — confirm output unchanged. `wave compose` (TUI) loads health view without panic.
- **Cycle check:** `go list -deps -f '{{.ImportPath}}: {{.Imports}}' ./internal/...` — eyeball that suggest imports neither doctor nor tui, doctor imports neither tui nor onboarding (ideally).
