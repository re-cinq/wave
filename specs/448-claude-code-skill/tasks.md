# Tasks

## Phase 1: Core Implementation

- [X] Task 1.1: Create `internal/onboarding/wave_command_step.go` with `WaveCommandStep` struct implementing `WizardStep` interface
  - `Name()` returns "Wave Command Registration"
  - `Run(cfg *WizardConfig)` generates `.claude/commands/wave.md`
  - Embedded template string for the command file content
  - Creates `.claude/commands/` directory if needed
  - Writes file using `os.WriteFile` (overwrite = idempotent)
  - Returns `StepResult` with `wave_command_generated: true`

- [X] Task 1.2: Define the wave command file template content
  - YAML frontmatter with `description: "Run Wave multi-agent pipelines"`
  - Markdown body with `$ARGUMENTS` placeholder
  - Instructions for `/wave run <pipeline> -- <input>`, `/wave status`, `/wave list`
  - Agent-friendly format with clear subcommand routing

## Phase 2: Wizard Integration

- [X] Task 2.1: Modify `internal/onboarding/onboarding.go` — add Step 7 to `RunWizard()` [P]
  - Add `WaveCommandGenerated bool` field to `WizardResult`
  - Insert `WaveCommandStep` execution after skill selection (Step 6)
  - Capture result into `WizardResult`

- [X] Task 2.2: Modify `cmd/wave/commands/init.go` — ensure `.claude/commands/` in directory creation [P]
  - Add `.claude/commands` to the `waveDirs` slice in `runWizardInit()`
  - Also handle in `runMergeInit()` path if applicable

## Phase 3: Testing

- [X] Task 3.1: Create `internal/onboarding/wave_command_step_test.go`
  - Test file generation at correct path
  - Test YAML frontmatter validity (has `description` field)
  - Test markdown body contains `$ARGUMENTS` placeholder
  - Test body references `wave run`, `wave status`, `wave list`
  - Test idempotency (run twice, same output)
  - Test non-interactive mode generates file
  - Test custom `WaveDir` path is respected

## Phase 4: Validation

- [X] Task 4.1: Run `go test ./internal/onboarding/...` to verify all tests pass
- [X] Task 4.2: Run `go vet ./...` for static analysis
