package tui

// PRDataMsg carries fetched pull request data from the provider.
type PRDataMsg struct {
	PRs []PRData
	Err error
}

// PRSelectedMsg signals that a pull request was selected in the list.
type PRSelectedMsg struct {
	Number int
	Title  string
	Index  int
}
