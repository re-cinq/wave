# Implementation Plan — #1171

## Objective

Eliminate duplicated Docker-daemon and skill check logic across `internal/doctor` and `internal/preflight` by extracting a single shared implementation, with test coverage on the shared layer.

## Approach

Introduce a new low-level package `internal/checks` housing forge-agnostic, side-effecting host probes:

- `checks.DockerDaemon(runCmd) DockerStatus`
- `checks.Skill(runCmd, cfg) SkillStatus`

Both `internal/doctor` and `internal/preflight` import `internal/checks` and adapt the typed status into their own result type (`doctor.CheckResult` / `preflight.Result`). No probe logic remains inside doctor or preflight — only translation + framing.

Doctor gains a Docker-daemon health check (`checkDockerDaemon`) that consumes the shared primitive. This makes both packages real consumers (acceptance #1) and adds genuine diagnostic value rather than only eliminating duplication.

## File Mapping

### Create

- `internal/checks/checks.go` — `DockerStatus`, `SkillStatus`, `DockerDaemon`, `Skill`, `SkillInstalled`, `SkillInstalledWithToolBin`. Accepts an injectable `RunCmdFunc func(name string, args ...string) error` for testability. Exports a `DefaultRunCmd` for production use.
- `internal/checks/checks_test.go` — covers: docker-not-on-PATH, docker-on-PATH-daemon-down, docker-up; skill-empty-check, skill-check-passes, skill-check-fails, skill-installed-with-tool-bin fallback (via `RunCmdWithEnv` injection point).

### Modify

- `internal/preflight/preflight.go`
  - `CheckDockerDaemon` reduces to: call `checks.DockerDaemon(c.runCmd)`, translate `DockerStatus` → `Result` with the same install-hint copy.
  - `isSkillInstalled` / `isSkillInstalledWithToolBin` delegate to `checks.SkillInstalled` / `checks.SkillInstalledWithToolBin`. `CheckSkills` orchestration (auto-install, optional handling, output capture) stays here — it is preflight-specific policy.
  - Drop now-duplicate `runCmdWithEnv` if it moves to `checks`.
- `internal/doctor/checks_deps.go`
  - `checkRequiredSkills` delegates the per-skill probe to `checks.Skill`, then maps `SkillStatus` → `CheckResult`. Empty-check warning, fix-string framing stay here.
- `internal/doctor/checks_infra.go`
  - Add `checkDockerDaemon(opts *Options) CheckResult` wrapping `checks.DockerDaemon(opts.runCmd)`. Returns `StatusWarn` (not `StatusErr`) when docker absent — many wave projects don't use docker; absence is informational.
- `internal/doctor/doctor.go`
  - Append `report.Results = append(report.Results, checkDockerDaemon(&opts))` after the existing infra checks.

### Delete

None.

## Architecture Decisions

1. **New `internal/checks` sub-package** rather than promoting one of doctor/preflight as host. Keeps both packages free of cross-imports, allows independent test coverage of the host probes, and matches existing `internal/tools` precedent (low-level host-probe utilities).
2. **Typed `*Status` structs returned**, not pre-formatted `Result` strings. Lets each consumer attach its own messaging, fix-strings, and severity (e.g., doctor uses Warn for missing docker, preflight uses OK=false). Resolves the assessment's `missing_info` Q2 in favour of typed results.
3. **`RunCmdFunc` injection** kept as the single seam (mirrors existing `Checker.runCmd` and `Options.runCmd`). No interface explosion.
4. **Doctor adds a docker daemon check** rather than skipping it. Required to make acceptance #1 ("shared between the two") substantive — otherwise only one consumer exists.
5. **Auto-install logic stays in preflight.** Doctor is read-only diagnostic; preflight is gating + remediation. Sharing the *probe* not the *policy* is the correct boundary.

## Risks

| Risk | Mitigation |
|---|---|
| Behaviour drift in error messages breaks downstream parsers | Keep exact wording for preflight Docker / skill messages — only the probe moves. |
| Test fakes in preflight stop working after delegation | Inject `runCmd` through `checks` API; preflight tests already inject `runCmd` on the Checker — propagate it. |
| Doctor adding a docker check fails on machines without docker | Use `StatusWarn` not `StatusErr` so wave projects without docker still see a green doctor report. |
| `checks.SkillInstalledWithToolBin` reads `$HOME` / `$PATH` env directly, hard to test | Accept env-resolver in package-level helper or stub via existing `runCmdWithEnv` — keep current behaviour, add table-driven test using `t.Setenv`. |
| Cyclic import risk (preflight already pulls `internal/skill`) | `internal/checks` depends only on `internal/skill` types (or a minimal local config struct) and stdlib. No import of doctor/preflight/pipeline. |

## Testing Strategy

- **Shared layer (`internal/checks/checks_test.go`)**: table-driven tests for both probes using injected `runCmd` fakes. Cover empty check command, success, failure, env-fallback paths. Target ≥85% coverage on new file.
- **Preflight (`internal/preflight/preflight_test.go`)**: existing `CheckDockerDaemon` and `CheckSkills` tests must keep passing unchanged — proves the message-level contract is preserved. Adjust internal mocks only if signature changes.
- **Doctor (`internal/doctor/doctor_test.go`)**: add a case for the new `checkDockerDaemon` covering binary-missing → Warn and binary+daemon-up → OK. Update `checks_deps` tests if any assert on phrasing.
- **Race**: full `go test -race ./...` per repo convention.
