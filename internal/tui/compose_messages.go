package tui

// ComposeActiveMsg signals the status bar to switch to compose mode hints.
type ComposeActiveMsg struct {
	Active bool
}

// ComposeSequenceChangedMsg signals that the sequence was modified.
type ComposeSequenceChangedMsg struct {
	Sequence   Sequence
	Validation CompatibilityResult
	Parallel   bool
	Stages     [][]int
}

// ComposeStartMsg signals that the user wants to start the sequence.
type ComposeStartMsg struct {
	Sequence Sequence
	Parallel bool    // When true, launch via `wave compose --parallel`
	Stages   [][]int // Stage boundaries — each group of indices runs in parallel
}

// ComposeCancelMsg signals that compose mode should close.
type ComposeCancelMsg struct{}
