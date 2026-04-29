# Work Items

## Phase 1: Setup
- [X] Item 1.1: Inspect `internal/defaults/` embed mechanism — confirmed glob `personas/*.md`, `pipelines/*.yaml`, `contracts/*.json contracts/*.md`, `prompts/**/*.md` auto-pick up new files (see `internal/defaults/embed.go`)
- [X] Item 1.2: Inspect existing contract loader — `detection.schema.json` placed in `internal/defaults/contracts/` is auto-discovered; mirrored copy required in `.agents/contracts/` per `TestSchemaSync`
- [X] Item 1.3: Inspect persona loader — `onboarder.md` + `onboarder.yaml` auto-discovered by `GetPersonas` / `GetPersonaConfigs`

## Phase 2: Core Implementation
- [X] Item 2.1: Wrote `internal/defaults/contracts/detection.schema.json` (flavour, build_system, test_command, package_managers, signals[], additional_signals object, project_intent, frameworks). Mirror copy at `.agents/contracts/detection.schema.json`.
- [X] Item 2.2: Wrote `internal/defaults/personas/onboarder.{md,yaml}` — workspace-scoped persona, full-tool access, hard rule: never write outside `.agents/`
- [X] Item 2.3: Wrote `internal/defaults/prompts/onboard/detect.md`
- [X] Item 2.4: Wrote `internal/defaults/prompts/onboard/propose.md`
- [X] Item 2.5: Wrote `internal/defaults/prompts/onboard/generate.md`
- [X] Item 2.6: Wrote `internal/defaults/prompts/onboard/finalize.md`
- [X] Item 2.7: Wrote `internal/defaults/pipelines/onboard-project.yaml` — 4-step (detect → propose → generate → finalize) meta-pipeline
- [X] Item 2.8: Embed registration not required (glob pattern picks up new paths)

## Phase 3: Testing
- [X] Item 3.1: Schema validation test with valid + invalid fixtures (`TestDetectionSchemaValidatesFixtures` in `internal/defaults/onboarder_test.go`)
- [X] Item 3.2: Defaults-registry test asserting `onboard-project` and `onboarder` load (`TestOnboardProjectPipelineRegistered`, `TestOnboarderPersonaRegistered`)
- [X] Item 3.3: Pipeline structure validates via `go test ./internal/defaults/...` and full `go test ./...` (existing `TestSchemaSync` reverse-check passes)
- [ ] Item 3.4: Smoke run on throwaway Go repo (real LLM, `--model cheapest`) — deferred; requires live adapter and lies outside the impl-issue pipeline scope. Will run before merge.
- [ ] Item 3.5: Smoke run on throwaway Node repo (real LLM, `--model cheapest`) — deferred; same reason as 3.4.
- [ ] Item 3.6: Verify sentinel `.agents/.onboarding-done` written after each smoke run — deferred with 3.4/3.5.

## Phase 4: Polish
- [X] Item 4.1: Marked `1.1 ✅` in `docs/scope/onboarding-as-session-plan.md` Phase 1 row 1.1 with the actual file list shipped
- [ ] Item 4.2: Tick acceptance boxes on issue #1576 (PR step)
- [ ] Item 4.3: Open PR with conventional title (PR step)
- [X] Item 4.4: Final check — only project files staged; `.agents/artifacts/`, `.agents/output/`, `.claude/`, `CLAUDE.md` excluded via the documented `git reset HEAD --` filter
