# Tasks

## Phase 1: Configuration Updates

- [X] Task 1.1: Update `wave.yaml` reviewer persona deny patterns — add `Write(*.py)`, `Write(*.rs)`, `Bash(rm *)`, `Bash(git push*)`, `Bash(git commit*)` to the deny list (lines 233-236)
- [X] Task 1.2: Update `internal/defaults/personas/reviewer.yaml` embedded default — add `Write(*.py)`, `Write(*.rs)`, `Bash(rm *)` to align with `wave.yaml` [P]

## Phase 2: Test Updates

- [X] Task 2.1: Update `createTestManifestWithPersonas` test fixture in `internal/manifest/permissions_test.go` — add new deny patterns to reviewer persona entry
- [X] Task 2.2: Extend `TestPersonaPermission_ReviewerCannotWriteSourceFiles` — add checks for `Write(*.py)` deny and `Write(*.rs)` deny [P]
- [X] Task 2.3: Add `TestPersonaPermission_ReviewerCannotRunDestructiveCommands` — new test covering `Bash(rm *)`, `Bash(git push*)`, `Bash(git commit*)` deny enforcement and verifying safe commands (`go test`, `git log`) still pass [P]
- [X] Task 2.4: Add reviewer `.py` and `.rs` scenarios to `TestPersonaPermission_ArtifactCreationScenarios` [P]
- [X] Task 2.5: Add `Bash(rm *)`, `Bash(git push*)`, `Bash(git commit*)` cases to `TestPersonaPermission_DenyPatternTakesPrecedence` [P]
- [X] Task 2.6: Extend `TestLoadWaveYAML_PersonaPermissions` integration test — verify new deny patterns present in real `wave.yaml`

## Phase 3: Validation

- [X] Task 3.1: Run `go test ./internal/manifest/ -v -run TestPersona` to verify permission tests pass
- [X] Task 3.2: Run `go test ./...` full test suite
- [X] Task 3.3: Run `go test -race ./...` with race detector

## Phase 4: Polish

- [X] Task 4.1: Verify no unintended side effects — reviewer can still run `go test*`, `npm test*`, read files, write artifacts
