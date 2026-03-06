package tui

// GitStateMsg carries the result of an async git state fetch.
type GitStateMsg struct {
	State GitState
	Err   error
}

// ManifestInfoMsg carries the result of an async manifest info fetch.
type ManifestInfoMsg struct {
	Info ManifestInfo
	Err  error
}

// GitHubInfoMsg carries the result of an async GitHub info fetch.
type GitHubInfoMsg struct {
	Info GitHubInfo
	Err  error
}

// PipelineHealthMsg carries the result of an async pipeline health fetch.
type PipelineHealthMsg struct {
	Health HealthStatus
	Err    error
}

// RunningCountMsg signals a change in the number of running pipelines.
type RunningCountMsg struct {
	Count int
}

// PipelineSelectedMsg signals that a pipeline (running, finished, or available) was selected in the UI.
// For available pipelines, RunID is empty and Kind is itemKindAvailable.
type PipelineSelectedMsg struct {
	RunID         string
	Name          string   // Pipeline or run name
	BranchName    string   // Empty means no finished pipeline selected
	BranchDeleted bool     // True if the branch no longer exists
	Kind          itemKind // itemKindRunning, itemKindFinished, or itemKindAvailable
}

// FocusPane identifies which pane has keyboard focus.
type FocusPane int

const (
	FocusPaneLeft  FocusPane = iota // Default: focus on left pane (pipeline list)
	FocusPaneRight                  // Focus on right pane (pipeline detail)
)

// FocusChangedMsg signals that the focus pane has changed.
type FocusChangedMsg struct {
	Pane FocusPane
}

// DetailDataMsg carries fetched pipeline detail data.
type DetailDataMsg struct {
	AvailableDetail *AvailableDetail
	FinishedDetail  *FinishedDetail
	Err             error
}

// LogoTickMsg is an internal timer message for logo animation.
type LogoTickMsg struct{}

// GitRefreshTickMsg is an internal timer message for periodic git state refresh.
type GitRefreshTickMsg struct{}
