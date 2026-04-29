# 1.3: CLI driver: wave init calls OnboardingService

**Issue:** [re-cinq/wave#1578](https://github.com/re-cinq/wave/issues/1578)
**Epic:** [#1565](https://github.com/re-cinq/wave/issues/1565)
**Phase:** 1, after PRE-2 (#1569) and 1.1 (#1576)
**Branch:** `1578-init-onboarding-service`
**Labels:** none
**State:** open
**Author:** nextlevelshit

## Body

Part of Epic #1565. Phase 1, depends on PRE-2 (#1569 ✓), 1.1 (#1576).

Rewrite `cmd/wave/commands/init.go` to delegate through PRE-2's
`OnboardingService` instead of the legacy `RunWizard` path. After this, the
wizard machinery can finally be deleted.

**Files:**
- Modify: `cmd/wave/commands/init.go` — wizard path replaced by service call
- Delete: legacy `RunWizard` machinery from `internal/onboarding/onboarding.go`,
  `steps.go`, `flow.go` (keep `flavour.go` + `metadata.go`)

**Acceptance:**
- [ ] `wave init` non-interactive path uses service
- [ ] `wave init --interactive` (if kept) routes through service
- [ ] Existing tests adapted, `RunWizard` removed
- [ ] Smoke run on greenfield repo (real verification per memory)

**Pipeline:** `impl-finding` (`--adapter claude --model cheapest`)

## Acceptance Criteria

| # | Criterion | How verified |
|---|-----------|--------------|
| AC1 | Non-interactive `wave init --yes` writes wave.yaml + sentinel via `BaselineService.StartSession` | unit test asserts service call path; smoke run on `t.TempDir()` |
| AC2 | Interactive default `wave init` (TTY) routes through `BaselineService` (stop-gap until 1.2 InteractiveService) | unit test, init.go cyclomatic path check |
| AC3 | `--reconfigure` clears sentinel, then re-runs service | unit test |
| AC4 | `RunWizard`, `WizardConfig`, `WizardResult`, `WizardStep`, `StepResult`, all wizard `*Step` types and tests removed | `grep -r RunWizard` returns no Go matches outside `specs/` and `docs/` |
| AC5 | `flavour.go` + `metadata.go` retained intact (re-used by `BaselineService`) | diff inspection |
| AC6 | Greenfield smoke: `wave init --yes` in empty `t.TempDir()` produces `wave.yaml`, `.agents/` tree, `.agents/.onboarding-done` | manual smoke + integration test |
| AC7 | `go build ./... && go vet ./... && go test ./...` green | CI gates |

## Open Questions (resolved here)

1. **`--interactive` flag**: Keep the existing default-interactive behaviour
   (`!opts.Yes && IsInteractive()`) for now. Until 1.2 lands `InteractiveService`,
   the interactive path also routes through `BaselineService` — i.e. the wizard
   prompts go away in this issue. Acceptable per epic plan: 1.2 will replace
   the prompt-less stop-gap with a real `UI` implementation.
2. **Error behaviour from `StartSession`**: Bubble up unchanged (`return err`
   from `RunE`). Wrap with context `"onboarding: %w"` so the cobra error layer
   stays informative.
