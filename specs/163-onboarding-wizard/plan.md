# Implementation Plan: Interactive Onboarding Wizard

## Objective

Transform `wave init` from a non-interactive heuristic-based command into an interactive onboarding wizard that walks users through 5 setup steps (dependency verification, test command configuration, pipeline selection, adapter configuration, model selection), persists completion state, and gates pipeline execution until onboarding is complete.

## Approach

### Architecture Decision: `charmbracelet/huh` Forms

The project already uses `charmbracelet/huh` for the interactive pipeline selector (`internal/tui/run_selector.go`). The onboarding wizard will use the same library and the existing `WaveTheme()` for visual consistency. Each wizard step maps to a `huh.Group` within a multi-group `huh.Form`.

### Architecture Decision: File-Based Onboarding State

Use a simple marker file (`.wave/.onboarded`) rather than SQLite for onboarding state. Rationale:
- Onboarding state is a binary flag (complete/incomplete) — no need for relational queries.
- The file can be checked cheaply before opening the state DB (which happens later in the run flow).
- `--reconfigure` simply deletes the marker.
- Simpler to reason about and test than adding a new table + migration.

### Architecture Decision: New `internal/onboarding` Package

Create a dedicated package rather than expanding `cmd/wave/commands/init.go` inline. The `init.go` command handler becomes an orchestrator that calls into `internal/onboarding` functions. This keeps the command layer thin and the wizard logic testable via interfaces.

### Architecture Decision: Gating via `checkOnboarding()` Helper

Add a shared helper function that `wave run` and `wave do` call before execution. The helper checks for `.wave/.onboarded` and returns a descriptive error if missing. This is a minimal change to existing command flow.

## File Mapping

### New Files

| Path | Action | Description |
|------|--------|-------------|
| `internal/onboarding/onboarding.go` | create | Core wizard types, step definitions, and orchestration |
| `internal/onboarding/steps.go` | create | Individual step implementations (dependency, test config, pipeline selection, adapter, model) |
| `internal/onboarding/state.go` | create | Onboarding state persistence (marker file read/write) |
| `internal/onboarding/onboarding_test.go` | create | Unit tests for wizard logic |
| `internal/onboarding/steps_test.go` | create | Unit tests for individual steps |
| `internal/onboarding/state_test.go` | create | Unit tests for state persistence |

### Modified Files

| Path | Action | Description |
|------|--------|-------------|
| `cmd/wave/commands/init.go` | modify | Integrate wizard into init flow; add `--reconfigure` flag |
| `cmd/wave/commands/init_test.go` | modify | Add tests for wizard integration and reconfigure flag |
| `cmd/wave/commands/run.go` | modify | Add onboarding gate check before pipeline execution |
| `cmd/wave/commands/run_test.go` | modify | Add tests for onboarding gating |
| `cmd/wave/commands/do.go` | modify | Add onboarding gate check before ad-hoc execution |
| `cmd/wave/commands/helpers.go` | modify | Add `checkOnboarding()` shared helper |
| `internal/manifest/types.go` | modify | Add `Category` field to `PipelineMetadata` |
| `internal/pipeline/types.go` | modify | Add `Category` field to `PipelineMetadata` |
| `internal/tui/theme.go` | modify | May add wizard-specific styles (progress indicator) |

## Detailed Design

### 1. Onboarding State (`internal/onboarding/state.go`)

```go
type State struct {
    Completed   bool      `json:"completed"`
    CompletedAt time.Time `json:"completed_at,omitempty"`
    Version     int       `json:"version"` // schema version for future changes
}

func IsOnboarded(waveDir string) bool
func MarkOnboarded(waveDir string) error
func ClearOnboarding(waveDir string) error
func ReadState(waveDir string) (*State, error)
```

The state is stored as a JSON file at `.wave/.onboarded` for debuggability (vs a bare marker).

### 2. Wizard Steps (`internal/onboarding/steps.go`)

Each step implements a common interface:

```go
type StepResult struct {
    Skipped bool
    Data    map[string]interface{}
}

type WizardStep interface {
    Name() string
    Run(ctx WizardContext) (*StepResult, error)
}
```

Steps:
- **DependencyStep**: Uses `exec.LookPath` for tools (adapter binary, `gh`). Reports missing tools with install URLs. No auto-install in MVP.
- **TestConfigStep**: Calls existing `detectProject()` from `init.go`, presents results in `huh.Input` fields for confirm/override.
- **PipelineSelectionStep**: Discovers pipelines via `tui.DiscoverPipelines()`, presents `huh.MultiSelect` grouped by category. Respects `metadata.release` for default selection.
- **AdapterConfigStep**: Presents `huh.Select` for adapter choice. Runs a verification command (`claude --version` or equivalent) to confirm the binary exists and is authenticated.
- **ModelSelectionStep**: Presents `huh.Select` with known model options for the chosen adapter.

### 3. Wizard Orchestrator (`internal/onboarding/onboarding.go`)

```go
type WizardConfig struct {
    WaveDir     string
    Interactive bool // false when --yes or no TTY
    Reconfigure bool
    Existing    *manifest.Manifest // non-nil when reconfiguring
}

func RunWizard(cfg WizardConfig) (*WizardResult, error)
```

The orchestrator:
1. Prints the Wave logo
2. Runs each step sequentially
3. Collects results into a `WizardResult` struct
4. Builds the manifest from results + existing config (if reconfiguring)
5. Marks onboarding complete

### 4. Pipeline Gating (`cmd/wave/commands/helpers.go`)

```go
func checkOnboarding() error {
    if !onboarding.IsOnboarded(".wave") {
        return fmt.Errorf("onboarding not complete\n\nRun 'wave init' to complete setup before running pipelines")
    }
    return nil
}
```

Called at the top of `runRun()` and `runDo()`. Skipped when `--force` is set (existing flag on `run`).

### 5. Pipeline Category Metadata

Add `Category string` to `PipelineMetadata` in `internal/pipeline/types.go`:

```go
type PipelineMetadata struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description,omitempty"`
    Release     bool   `yaml:"release,omitempty"`
    Category    string `yaml:"category,omitempty"` // "stable", "experimental", "contrib"
    Disabled    bool   `yaml:"disabled,omitempty"`
}
```

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Adapter authentication verification is adapter-specific and may break with new adapters | Medium | Define a simple `Adapter.Check()` interface; start with Claude (`claude --version`). Others return "skipped" |
| Breaking existing `wave init --yes` flow | High | Ensure `--yes` path bypasses all interactive prompts and uses identical defaults to current behavior |
| Test isolation for interactive TUI components | Medium | Use `huh.WithInput()` for testing, or mock the `WizardStep` interface |
| Gating may confuse users upgrading from older Wave versions | Medium | Only gate when no `wave.yaml` exists. If `wave.yaml` exists but no `.onboarded` marker, skip gating (grandfathered) |

## Testing Strategy

### Unit Tests
- `internal/onboarding/state_test.go`: Test `IsOnboarded`, `MarkOnboarded`, `ClearOnboarding` with temp directories
- `internal/onboarding/steps_test.go`: Test each step with mocked dependencies (mock `exec.LookPath`, mock filesystem)
- `internal/onboarding/onboarding_test.go`: Test wizard orchestration in non-interactive mode

### Integration Tests
- `cmd/wave/commands/init_test.go`: Test full init flow with `--yes` flag (non-interactive), verify manifest output
- `cmd/wave/commands/run_test.go`: Test onboarding gating blocks execution, test `--force` bypasses gating

### Manual Testing
- Run `wave init` interactively in a fresh directory
- Run `wave init --reconfigure` on an existing project
- Run `wave run <pipeline>` without onboarding, verify gating
- Run `wave init --yes` in CI-like environment
