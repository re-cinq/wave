package tui

// ViewType identifies the active content view in the TUI.
type ViewType int

const (
	ViewPipelines ViewType = iota
	ViewPersonas
	ViewContracts
	ViewSkills
	ViewHealth
	ViewIssues
	ViewPullRequests
	ViewSuggest
)

// String returns the display name for the view (used as status bar label).
func (v ViewType) String() string {
	switch v {
	case ViewPipelines:
		return "Pipelines"
	case ViewPersonas:
		return "Personas"
	case ViewContracts:
		return "Contracts"
	case ViewSkills:
		return "Skills"
	case ViewHealth:
		return "Health"
	case ViewIssues:
		return "Issues"
	case ViewPullRequests:
		return "Pull Requests"
	case ViewSuggest:
		return "Suggest"
	default:
		return "Unknown"
	}
}

// ViewChangedMsg is emitted when the user switches views via Tab.
type ViewChangedMsg struct {
	View ViewType
}

// PipelineStepRef identifies a pipeline/step pair for cross-references.
type PipelineStepRef struct {
	PipelineName string
	StepID       string
}
