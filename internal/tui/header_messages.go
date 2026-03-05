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

// PipelineSelectedMsg signals that a finished pipeline was selected in the UI.
type PipelineSelectedMsg struct {
	RunID         string
	BranchName    string // Empty means no finished pipeline selected
	BranchDeleted bool   // True if the branch no longer exists
}

// LogoTickMsg is an internal timer message for logo animation.
type LogoTickMsg struct{}

// GitRefreshTickMsg is an internal timer message for periodic git state refresh.
type GitRefreshTickMsg struct{}
