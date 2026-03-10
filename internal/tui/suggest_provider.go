package tui

// SuggestDataProvider fetches pipeline suggestions.
type SuggestDataProvider interface {
	FetchSuggestions() (*SuggestProposal, error)
}

// FuncSuggestDataProvider wraps a function as a SuggestDataProvider.
type FuncSuggestDataProvider struct {
	Fn func() (*SuggestProposal, error)
}

// FetchSuggestions calls the wrapped function.
func (p *FuncSuggestDataProvider) FetchSuggestions() (*SuggestProposal, error) {
	return p.Fn()
}
