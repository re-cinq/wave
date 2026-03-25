# Tasks

## Phase 1: Defense-in-depth — Update persona from events

- [X] Task 1.1: In `internal/display/bubbletea_progress.go`, update `updateFromEvent` to set `step.Persona = evt.Persona` when a "started"/"running" event arrives with a non-empty `Persona` field
- [X] Task 1.2: In `internal/display/progress.go`, update `EmitProgress` to set `step.Persona = ev.Persona` when a "started"/"running" event arrives with a non-empty `Persona` field (auto-registered steps already receive the persona; pre-registered ones need the update)

## Phase 2: Source fix — Resolve personas at registration time

- [X] Task 2.1: In `cmd/wave/commands/output.go`, add a `resolveForgePersona` helper function that replaces `{{ forge.type }}` and `{{forge.type}}` (both spaced and unspaced variants) with the detected forge type string
- [X] Task 2.2: In `createAutoEmitter`, call `forge.DetectFromGitRemotes()` and use `resolveForgePersona` to resolve persona names before passing them to `btpd.AddStep` [P]
- [X] Task 2.3: Apply the same resolution in `CreateEmitter` for the "text" format path (BasicProgressDisplay doesn't pre-register steps, but the emitted events already have resolved personas — verify this is correct) [P]

## Phase 3: WebUI fix — Resolve personas in web handlers

- [X] Task 3.1: In `internal/webui/handlers_runs.go`, resolve forge variables in `step.Persona` before assigning to `DAGStepInput` and `StepDetail` structs [P]
- [X] Task 3.2: In `internal/webui/handlers_compose.go`, resolve forge variables in `step.Persona` before assigning to compose step DTO [P]

## Phase 4: Testing

- [X] Task 4.1: Add unit test in `internal/display/bubbletea_progress_resume_test.go` verifying that a "running" event updates the stored persona from an unresolved to a resolved value
- [X] Task 4.2: Add unit test in `internal/display/progress_test.go` verifying that `EmitProgress` updates the stored persona when receiving a "running" event with a non-empty persona [P]
- [X] Task 4.3: Run `go test ./internal/display/... ./cmd/wave/commands/... ./internal/webui/...` to verify no regressions [P]

## Phase 5: Validation

- [X] Task 5.1: Run `go build ./...` to confirm compilation
- [X] Task 5.2: Run `go vet ./...` to check for issues
