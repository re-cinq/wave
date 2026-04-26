# Work Items

## Phase 1: Setup

- [X] Item 1.1: Confirm branch `1171-dedupe-docker-skill-checks` checked out and clean
- [X] Item 1.2: Sketch `internal/checks` API (`DockerStatus`, `SkillStatus`, `RunCmdFunc`) in a scratch comment block before coding

## Phase 2: Core Implementation

- [X] Item 2.1: Create `internal/checks/checks.go` with `DockerDaemon`, `Skill`, `SkillInstalled`, `SkillInstalledWithToolBin`, `DefaultRunCmd`, and typed `*Status` results [P]
- [X] Item 2.2: Refactor `internal/preflight/preflight.go::CheckDockerDaemon` to delegate to `checks.DockerDaemon`, preserving exact message strings [P]
- [X] Item 2.3: Refactor `internal/preflight/preflight.go::isSkillInstalled` and `isSkillInstalledWithToolBin` to delegate to `checks.SkillInstalled` / `checks.SkillInstalledWithToolBin`
- [X] Item 2.4: Refactor `internal/doctor/checks_deps.go::checkRequiredSkills` per-skill probe to call `checks.Skill`
- [X] Item 2.5: Add `internal/doctor/checks_infra.go::checkDockerDaemon` wrapping `checks.DockerDaemon`, register it in `internal/doctor/doctor.go::Run`

## Phase 3: Testing

- [X] Item 3.1: Write `internal/checks/checks_test.go` covering both probes with injected `runCmd` fakes [P]
- [X] Item 3.2: Verify existing `internal/preflight/preflight_test.go` passes unchanged; adjust only if internal seams shifted [P]
- [X] Item 3.3: Add `doctor_test.go` case for `checkDockerDaemon` (binary missing → Warn; daemon up → OK)
- [X] Item 3.4: `go test -race ./...` green across repo

## Phase 4: Polish

- [X] Item 4.1: Run `golangci-lint run ./...`; fix any issues  *(binary unavailable in this sandbox; `go vet ./...` clean, `gofmt` clean for changed files)*
- [X] Item 4.2: Verify no orphaned helpers (`runCmdWithEnv` etc.) left in preflight
- [X] Item 4.3: Commit with `refactor(checks): extract shared docker + skill probes (#1171)` and open PR
