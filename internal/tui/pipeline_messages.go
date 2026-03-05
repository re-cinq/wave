package tui

// PipelineDataMsg carries refreshed pipeline data from the provider.
type PipelineDataMsg struct {
	Running   []RunningPipeline
	Finished  []FinishedPipeline
	Available []PipelineInfo
	Err       error
}

// PipelineRefreshTickMsg triggers periodic data refresh.
type PipelineRefreshTickMsg struct{}
