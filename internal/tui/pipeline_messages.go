package tui

import (
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
	stateLoading                                // Fetching data
	stateAvailableDetail                        // Available pipeline config
	stateFinishedDetail                         // Finished pipeline results
	stateRunningInfo                            // Running pipeline info
	stateRunningLive                            // Live output for running pipeline
	stateConfiguring                            // Argument form active
	stateLaunching                              // Brief "Starting..." indicator
	stateError                                  // Launch error display
	stateComposing                              // Compose mode artifact flow
)

// LaunchDependencies holds the dependencies needed to launch pipelines from the TUI.
// Passed at TUI construction time; executor infrastructure is created on demand.
type LaunchDependencies struct {
	Manifest        *manifest.Manifest
	Store           state.StateStore
	PipelinesDir    string
	SuggestProvider SuggestDataProvider
}

// LaunchConfig holds the user's pipeline launch configuration from the argument form.
type LaunchConfig struct {
	PipelineName  string
	Input         string
	ModelOverride string
	Flags         []string // e.g., "--verbose", "--debug", "--dry-run"
	DryRun        bool     // Extracted from Flags for convenience
	Verbose       bool     // Extracted from Flags for convenience
	Debug         bool     // Extracted from Flags for convenience
}

// LaunchRequestMsg is emitted when the argument form is submitted.
type LaunchRequestMsg struct {
	Config LaunchConfig
}

// PipelineLaunchedMsg signals that a pipeline launch was initiated.
type PipelineLaunchedMsg struct {
	RunID        string
	PipelineName string
	Input        string
	Verbose      bool // Propagate launch flags to live output display
	Debug        bool
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

// ChatSessionEndedMsg signals that an interactive chat session has ended.
// Triggers data refresh (re-fetch finished detail, git state) to reflect changes
// the user may have made during the session.
type ChatSessionEndedMsg struct {
	Err error
}

// BranchCheckoutMsg signals the result of a branch checkout attempt.
type BranchCheckoutMsg struct {
	BranchName string
	Success    bool
	Err        error
}

// DiffViewEndedMsg signals that the diff pager has exited.
type DiffViewEndedMsg struct {
	Err error
}

// FinishedDetailActiveMsg signals the status bar to switch to finished detail hints.
type FinishedDetailActiveMsg struct {
	Active bool
}

// RunEventsMsg carries persisted event log records fetched from the state store.
type RunEventsMsg struct {
	RunID  string
	Events []state.LogRecord
	Err    error
}

// RunningInfoActiveMsg signals the status bar when stateRunningInfo pane is active.
type RunningInfoActiveMsg struct {
	Active bool
}

// DetachedEventPollTickMsg triggers periodic event polling for detached pipeline live output.
type DetachedEventPollTickMsg struct {
	RunID string
}

// DashboardTickMsg drives elapsed time updates in the live output dashboard view.
type DashboardTickMsg struct{}
