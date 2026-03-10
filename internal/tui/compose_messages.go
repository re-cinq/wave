package tui

// ComposeActiveMsg signals the status bar to switch to compose mode hints.
type ComposeActiveMsg struct {
	Active bool
}

// ComposeSequenceChangedMsg signals that the sequence was modified.
type ComposeSequenceChangedMsg struct {
	Sequence   Sequence
	Validation CompatibilityResult
}

// ComposeStartMsg signals that the user wants to start the sequence.
type ComposeStartMsg struct {
	Sequence Sequence
}

// ComposeCancelMsg signals that compose mode should close.
type ComposeCancelMsg struct{}

// ComposeFocusDetailMsg signals that Enter was pressed on a boundary to focus the artifact flow detail.
type ComposeFocusDetailMsg struct{}
