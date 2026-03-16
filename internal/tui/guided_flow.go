package tui

// GuidedFlowPhase represents the current phase of the guided workflow.
type GuidedFlowPhase int

const (
	GuidedPhaseHealth    GuidedFlowPhase = iota // Infrastructure health checks running
	GuidedPhaseProposals                        // Showing pipeline proposals (ViewSuggest)
	GuidedPhaseFleet                            // Monitoring active runs (ViewPipelines)
	GuidedPhaseAttached                         // Attached to live output of a running pipeline
)

// GuidedFlowState manages the guided workflow lifecycle.
// When non-nil on ContentModel, it overrides startup view and Tab behavior.
type GuidedFlowState struct {
	Phase           GuidedFlowPhase
	HealthComplete  bool // All infrastructure checks finished
	HasErrors       bool // At least one health check returned error
	UserConfirmed   bool // User chose to continue despite health errors
	TransitionTimer bool // Auto-transition timer is running
}

// IsGuided returns true if the guided flow is active.
func (s *GuidedFlowState) IsGuided() bool {
	return s != nil
}

// TransitionToProposals moves to the proposals phase.
func (s *GuidedFlowState) TransitionToProposals() {
	s.Phase = GuidedPhaseProposals
	s.HealthComplete = true
}

// TransitionToFleet moves to the fleet monitoring phase.
func (s *GuidedFlowState) TransitionToFleet() {
	s.Phase = GuidedPhaseFleet
}

// TransitionToAttached moves to the attached live output phase.
func (s *GuidedFlowState) TransitionToAttached() {
	s.Phase = GuidedPhaseAttached
}

// DetachToFleet returns from attached mode to fleet view.
func (s *GuidedFlowState) DetachToFleet() {
	s.Phase = GuidedPhaseFleet
}

// TabTarget returns the toggle destination view for the current phase.
// In guided mode, Tab toggles between Suggest (proposals) and Pipelines (fleet).
func (s *GuidedFlowState) TabTarget() ViewType {
	switch s.Phase {
	case GuidedPhaseHealth, GuidedPhaseProposals:
		return ViewPipelines
	case GuidedPhaseFleet:
		return ViewSuggest
	case GuidedPhaseAttached:
		return ViewType(-1) // Tab blocked during attachment
	default:
		return ViewPipelines
	}
}
