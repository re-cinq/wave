# Tasks: Skill Dependency Installation in Pipeline Steps

**Branch**: `102-skill-deps-pipeline` | **Date**: 2026-02-14
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Phase 1: Setup — Foundational Constants

- [X] T001 [P1] [US5] Add `StatePreflight` constant to event package — `internal/event/emitter.go`
  - Add `StatePreflight = "preflight"` to the `const` block at line 55-69, after `StateStreamActivity`
  - This replaces the string literal `"preflight"` used in executor.go:173
  - Verify it compiles: `go build ./internal/event/...`

## Phase 2: Core Preflight Enhancement (US1 + US2 — P1)

These tasks implement the emitter callback in the preflight Checker so that per-dependency
progress events are emitted during tool and skill checks.

- [X] T002 [P1] [US1,US5] Add emitter callback field and `WithEmitter` option to Checker — `internal/preflight/preflight.go`
  - Add `emitter func(name, kind, message string)` field to `Checker` struct (line 20-23)
  - Add `type CheckerOption func(*Checker)` type
  - Add `func WithEmitter(fn func(name, kind, message string)) CheckerOption` function
  - Add `func WithRunCmd(fn func(name string, args ...string) error) CheckerOption` function
  - Modify `NewChecker` signature to `NewChecker(skills map[string]manifest.SkillConfig, opts ...CheckerOption) *Checker`
  - Apply options in `NewChecker`; default emitter is nil (no-op)
  - **Depends on**: T001

- [X] T003 [P1] [US2,US5] Add emitter calls to `CheckTools` — `internal/preflight/preflight.go`
  - Before each `exec.LookPath` call, emit: `c.emitter(tool, "tool", "checking tool \"<tool>\"")`
  - On success, emit: `c.emitter(tool, "tool", "tool \"<tool>\" found")`
  - On failure, emit: `c.emitter(tool, "tool", "tool \"<tool>\" not found on PATH")`
  - Guard all emitter calls with nil check: `if c.emitter != nil { ... }`
  - **Depends on**: T002

- [X] T004 [P1] [US1,US5] Add emitter calls to `CheckSkills` — `internal/preflight/preflight.go`
  - Emit at each decision point in the skill check flow:
    - `"checking skill \"<name>\""` — before check command
    - `"skill \"<name>\" installed"` — when check passes (already installed)
    - `"skill \"<name>\" not declared in wave.yaml"` — undeclared skill
    - `"skill \"<name>\" not installed, no install command"` — check fails, no install
    - `"installing skill \"<name>\""` — before install command
    - `"skill \"<name>\" install failed: <err>"` — install command fails
    - `"initializing skill \"<name>\""` — before init command
    - `"skill \"<name>\" init failed: <err>"` — init command fails
    - `"skill \"<name>\" installed successfully"` — re-check passes after install
    - `"skill \"<name>\" still not detected after install"` — re-check fails
  - Guard all emitter calls with nil check
  - **Depends on**: T002

- [X] T005 [P1] [US1,US2] Update existing `NewChecker` call sites for new signature — `internal/preflight/preflight_test.go`
  - All existing `NewChecker(skills)` calls must still compile since `opts` is variadic
  - Verify no compilation breakage: `go build ./internal/preflight/...`
  - **Depends on**: T002
  - [P] Parallelizable with T003, T004

## Phase 3: Executor Integration (US1 + US2 — P1)

Wire the enhanced Checker with emitter into the pipeline executor.

- [X] T006 [P1] [US1,US2,US5] Wire emitter callback in executor preflight section — `internal/pipeline/executor.go`
  - At line 160, change `preflight.NewChecker(m.Skills)` to:
    ```go
    preflight.NewChecker(m.Skills, preflight.WithEmitter(func(name, kind, msg string) {
        e.emit(event.Event{
            Timestamp:  time.Now(),
            PipelineID: pipelineID,
            State:      event.StatePreflight,
            Message:    msg,
        })
    }))
    ```
  - Note: `pipelineID` is not yet assigned at line 160 (assigned at line 184). Move the preflight block to after `pipelineID` assignment (after line 188), or compute `pipelineID` earlier.
  - Recommended: Move `pipelineID` computation (lines 183-187) to before the preflight block (before line 159)
  - **Depends on**: T002, T001

- [X] T007 [P1] [US5] Replace string literal with `StatePreflight` constant — `internal/pipeline/executor.go`
  - At line 173, replace `State: "preflight"` with `State: event.StatePreflight`
  - Note: If T006 removes the post-hoc emission loop (lines 170-175), this becomes part of that change
  - **Depends on**: T001, T006

- [X] T008 [P1] [US1,US2,US5] Remove post-hoc event emission loop from executor — `internal/pipeline/executor.go`
  - Lines 170-175 emit events after all checks complete. Since the Checker now emits events inline via the callback, this loop is redundant.
  - Remove the `for _, r := range results { ... }` loop
  - Keep the `if err != nil { return ... }` error check
  - **Depends on**: T006

## Phase 4: Provisioner Fix (US3 — P2)

Fix the repoRoot bug so skill commands are discoverable in step workspaces.

- [X] T009 [P2] [US3] Fix repoRoot parameter in executor skill provisioning — `internal/pipeline/executor.go`
  - At line 512, change `skill.NewProvisioner(execution.Manifest.Skills, "")` to pass the actual repo root
  - Derive repo root: use the current working directory (`os.Getwd()`) or resolve from the manifest's location
  - Simplest approach: add `repoRoot, _ := os.Getwd()` before the provisioner block (line 509) and pass it
  - Verify skill command glob resolution works with the correct root
  - **Depends on**: None (independent of Phase 2/3)
  - [P] Parallelizable with T002-T008

## Phase 5: Manifest Configuration (US4 — P2)

Add skill definitions to wave.yaml for the three target skills.

- [X] T010 [P2] [US4] Add `skills` section to wave.yaml — `wave.yaml`
  - Add after the `skill_mounts` section (after line 221):
    ```yaml
    skills:
      speckit:
        check: "test -d .specify"
        install: "npx -y @anthropic/speckit init"
        commands_glob: ".claude/commands/speckit.*.md"
      bmad:
        check: "test -f .claude/commands/bmad.*.md"
        commands_glob: ".claude/commands/bmad.*.md"
      openspec:
        check: "test -d .openspec"
        commands_glob: ".claude/commands/openspec.*.md"
    ```
  - Note: `bmad` and `openspec` have no `install` command (check-only skills per FR-008)
  - **Depends on**: None (independent config change)
  - [P] Parallelizable with all other tasks

## Phase 6: Test Coverage (US1 + US2 + US5)

Extend existing tests to cover event emission, edge cases, and the new functional options.

- [X] T011 [P1] [US5] Test nil emitter doesn't panic — `internal/preflight/preflight_test.go`
  - Create a test that creates a Checker without `WithEmitter` and runs `CheckTools`/`CheckSkills`
  - Verify no panic occurs and results are correct (existing behavior preserved)
  - **Depends on**: T002

- [X] T012 [P1] [US5] Test emitter callback is called during tool checks — `internal/preflight/preflight_test.go`
  - Create a Checker with `WithEmitter` that captures emitted messages
  - Run `CheckTools([]string{"sh"})` and verify emitter was called with kind="tool"
  - Verify message contains "checking" and "found"
  - **Depends on**: T003

- [X] T013 [P1] [US5] Test emitter callback is called during skill checks — `internal/preflight/preflight_test.go`
  - Create a Checker with `WithEmitter` and mock `runCmd`
  - Run `CheckSkills` for a skill that is already installed (check succeeds)
  - Verify emitter was called with kind="skill", messages include "checking" and "installed"
  - **Depends on**: T004

- [X] T014 [P1] [US1,US5] Test emitter callback for install+init sequence — `internal/preflight/preflight_test.go`
  - Create a Checker with `WithEmitter` and mock `runCmd`
  - Configure a skill with check (fails first time), install, init, and re-check (succeeds)
  - Verify emitter messages follow sequence: checking → installing → initializing → installed
  - **Depends on**: T004

- [X] T015 [P1] [US1] Test edge case: install succeeds but re-check fails — `internal/preflight/preflight_test.go`
  - Mock `runCmd` so check always fails even after install succeeds
  - Verify result is `OK: false` with message "still not detected after install"
  - Verify emitter captures the "still not detected" message
  - **Depends on**: T004

- [X] T016 [P1] [US1] Test edge case: init fails after successful install — `internal/preflight/preflight_test.go`
  - Mock `runCmd` so check fails, install succeeds, init fails
  - Verify result is `OK: false` with message about init failure
  - Verify emitter captures the init failure message
  - **Depends on**: T004

- [X] T017 [P2] [US4] Test edge case: skill declared in requires but not in manifest — `internal/preflight/preflight_test.go`
  - Create a Checker with empty skills map
  - Run `CheckSkills([]string{"undeclared"})` and verify error message mentions manifest
  - Already partially covered by `TestCheckSkills_Undeclared` but verify emitter also fires
  - **Depends on**: T004
  - [P] Parallelizable with T015, T016

- [X] T018 [P1] [US5] Test `WithRunCmd` option works for test injection — `internal/preflight/preflight_test.go`
  - Create a Checker using `WithRunCmd` instead of directly setting `c.runCmd`
  - Verify mock command execution works through the functional option
  - **Depends on**: T002
  - [P] Parallelizable with T011-T017

## Phase 7: Polish & Cross-Cutting

- [X] T019 [P1] Verify all existing tests pass with changes — `go test -race ./...`
  - Run full test suite including race detector
  - Existing tests in `preflight_test.go` must pass with the new variadic `NewChecker` signature
  - Executor tests must pass with the restructured preflight block
  - **Depends on**: T001-T018

- [X] T020 [P2] [US3] Verify skill command provisioning end-to-end — manual verification
  - Confirm that with `repoRoot` fixed (T009), `Provisioner.Provision()` correctly resolves `commands_glob` patterns
  - Confirm files land in `.wave-skill-commands/.claude/commands/` within the workspace
  - Confirm adapter `copySkillCommands()` picks them up
  - **Depends on**: T009, T010

- [X] T021 [P3] [US5] Verify preflight events are visible in progress output — manual verification
  - Run a pipeline with `requires.skills` and verify NDJSON output includes events with `state: "preflight"`
  - Verify the enhanced progress display shows per-dependency status
  - **Depends on**: T006, T007, T008
