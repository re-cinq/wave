package tui

// HealthAllCompleteMsg signals that all infrastructure health checks have resolved.
type HealthAllCompleteMsg struct {
	HasErrors bool // true if any check returned HealthCheckErr
}

// HealthTransitionMsg triggers the auto-transition from health to proposals.
type HealthTransitionMsg struct{}

// HealthContinueMsg signals the user chose to continue despite health errors.
type HealthContinueMsg struct{}

// SuggestModifyMsg requests input modification for a proposal before launch.
type SuggestModifyMsg struct {
	Pipeline SuggestProposedPipeline
}
