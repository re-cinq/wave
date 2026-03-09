package tui

// IssueDataMsg carries fetched issue data from the provider.
type IssueDataMsg struct {
	Issues []IssueData
	Err    error
}

// IssueSelectedMsg signals that an issue was selected in the list.
type IssueSelectedMsg struct {
	Number int
	Title  string
	Index  int
}

// IssueLaunchMsg signals that the user wants to launch a pipeline against an issue.
type IssueLaunchMsg struct {
	PipelineName string
	IssueURL     string
}
