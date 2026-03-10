package tui

// SuggestProposal contains a prioritized list of pipeline proposals for the TUI.
type SuggestProposal struct {
	Pipelines []SuggestProposedPipeline
	Rationale string
}

// SuggestProposedPipeline is a single pipeline recommendation in the TUI.
type SuggestProposedPipeline struct {
	Name     string
	Reason   string
	Input    string
	Priority int
}

// SuggestDataMsg carries proposal data from the provider.
type SuggestDataMsg struct {
	Proposal *SuggestProposal
	Err      error
}

// SuggestSelectedMsg is sent when a suggestion is selected in the list.
type SuggestSelectedMsg struct {
	Pipeline SuggestProposedPipeline
}

// SuggestLaunchMsg is sent when user wants to launch the selected suggestion.
type SuggestLaunchMsg struct {
	Pipeline SuggestProposedPipeline
}
