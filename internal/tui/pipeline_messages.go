package tui

import (
	"context"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
)

// PipelineDataMsg carries refreshed pipeline data from the provider.
type PipelineDataMsg struct {
	Running   []RunningPipeline
	Finished  []FinishedPipeline
	Available []PipelineInfo
	Err       error
}

// PipelineRefreshTickMsg triggers periodic data refresh.
type PipelineRefreshTickMsg struct{}

// DetailPaneState tracks the right pane's current rendering mode.
type DetailPaneState int

const (
	stateEmpty           DetailPaneState = iota // No selection
	stateLoading                               // Fetching data
	stateAvailableDetail                       // Available pipeline config
	stateFinishedDetail                        // Finished pipeline results
	stateRunningInfo                           // Running pipeline info
	stateRunningLive                           // Live output for running pipeline
	stateConfiguring                           // Argument form active
	stateLaunching                             // Brief "Starting..." indicator
	stateError                                 // Launch error display
)

// LaunchDependencies holds the dependencies needed to launch pipelines from the TUI.
// Passed at TUI construction time; executor infrastructure is created on demand.
type LaunchDependencies struct {
	Manifest     *manifest.Manifest
	Store        state.StateStore
	PipelinesDir string
}

// LaunchConfig holds the user's pipeline launch configuration from the argument form.
type LaunchConfig struct {
	PipelineName  string
	Input         string
	ModelOverride string
	Flags         []string // e.g., "--verbose", "--debug", "--dry-run"
	DryRun        bool     // Extracted from Flags for convenience
}

// LaunchRequestMsg is emitted when the argument form is submitted.
type LaunchRequestMsg struct {
	Config LaunchConfig
}

// PipelineLaunchedMsg signals that a pipeline launch was initiated.
type PipelineLaunchedMsg struct {
	RunID        string
	PipelineName string
	CancelFunc   context.CancelFunc
}

// PipelineLaunchResultMsg signals that a launched pipeline has finished execution.
type PipelineLaunchResultMsg struct {
	RunID string
	Err   error
}

// LaunchErrorMsg signals a pre-execution failure (adapter resolution, manifest loading, etc.).
type LaunchErrorMsg struct {
	PipelineName string
	Err          error
}

// ConfigureFormMsg tells the detail pane to create and show the argument form.
type ConfigureFormMsg struct {
	PipelineName string
	InputExample string
}

// FormActiveMsg signals the status bar whether a form is active for hint switching.
type FormActiveMsg struct {
	Active bool
}

// PipelineEventMsg carries an executor event for a specific pipeline run.
// Delivered via program.Send() from the progress emitter callback.
type PipelineEventMsg struct {
	RunID string
	Event event.Event
}

// ElapsedTickMsg drives elapsed time updates for running pipelines in the left pane.
type ElapsedTickMsg struct{}

// TransitionTimerMsg signals that the completion transition delay has elapsed.
type TransitionTimerMsg struct {
	RunID string
}

// LiveOutputActiveMsg signals the status bar to switch to live output hints.
type LiveOutputActiveMsg struct {
	Active bool
}
