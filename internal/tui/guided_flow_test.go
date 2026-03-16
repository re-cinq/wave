package tui

import "testing"

func TestGuidedFlowState_IsGuided(t *testing.T) {
	var nilState *GuidedFlowState
	if nilState.IsGuided() {
		t.Error("nil state should not be guided")
	}

	state := &GuidedFlowState{Phase: GuidedPhaseHealth}
	if !state.IsGuided() {
		t.Error("non-nil state should be guided")
	}
}

func TestGuidedFlowState_Transitions(t *testing.T) {
	tests := []struct {
		name       string
		initial    GuidedFlowPhase
		transition func(s *GuidedFlowState)
		wantPhase  GuidedFlowPhase
	}{
		{
			name:       "Health to Proposals",
			initial:    GuidedPhaseHealth,
			transition: func(s *GuidedFlowState) { s.TransitionToProposals() },
			wantPhase:  GuidedPhaseProposals,
		},
		{
			name:       "Proposals to Fleet",
			initial:    GuidedPhaseProposals,
			transition: func(s *GuidedFlowState) { s.TransitionToFleet() },
			wantPhase:  GuidedPhaseFleet,
		},
		{
			name:       "Fleet to Attached",
			initial:    GuidedPhaseFleet,
			transition: func(s *GuidedFlowState) { s.TransitionToAttached() },
			wantPhase:  GuidedPhaseAttached,
		},
		{
			name:       "Attached to Fleet (detach)",
			initial:    GuidedPhaseAttached,
			transition: func(s *GuidedFlowState) { s.DetachToFleet() },
			wantPhase:  GuidedPhaseFleet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &GuidedFlowState{Phase: tt.initial}
			tt.transition(s)
			if s.Phase != tt.wantPhase {
				t.Errorf("got phase %d, want %d", s.Phase, tt.wantPhase)
			}
		})
	}
}

func TestGuidedFlowState_TransitionIdempotent(t *testing.T) {
	s := &GuidedFlowState{Phase: GuidedPhaseHealth}
	s.TransitionToProposals()
	s.TransitionToProposals()
	if s.Phase != GuidedPhaseProposals {
		t.Errorf("double transition should be idempotent, got %d", s.Phase)
	}
}

func TestGuidedFlowState_TransitionToProposals_SetsHealthComplete(t *testing.T) {
	s := &GuidedFlowState{Phase: GuidedPhaseHealth}
	s.TransitionToProposals()
	if !s.HealthComplete {
		t.Error("TransitionToProposals should set HealthComplete")
	}
}

func TestGuidedFlowState_ViewForPhase(t *testing.T) {
	tests := []struct {
		name     string
		phase    GuidedFlowPhase
		wantView ViewType
	}{
		{
			name:     "Health phase maps to ViewHealth",
			phase:    GuidedPhaseHealth,
			wantView: ViewHealth,
		},
		{
			name:     "Proposals phase maps to ViewSuggest",
			phase:    GuidedPhaseProposals,
			wantView: ViewSuggest,
		},
		{
			name:     "Fleet phase maps to ViewPipelines",
			phase:    GuidedPhaseFleet,
			wantView: ViewPipelines,
		},
		{
			name:     "Attached phase maps to ViewPipelines",
			phase:    GuidedPhaseAttached,
			wantView: ViewPipelines,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &GuidedFlowState{Phase: tt.phase}
			got := s.ViewForPhase()
			if got != tt.wantView {
				t.Errorf("got %d, want %d", got, tt.wantView)
			}
		})
	}
}

func TestGuidedFlowState_TabTarget(t *testing.T) {
	tests := []struct {
		name      string
		phase     GuidedFlowPhase
		wantView  ViewType
		wantBlock bool
	}{
		{
			name:     "Health phase targets Pipelines",
			phase:    GuidedPhaseHealth,
			wantView: ViewPipelines,
		},
		{
			name:     "Proposals phase targets Pipelines",
			phase:    GuidedPhaseProposals,
			wantView: ViewPipelines,
		},
		{
			name:     "Fleet phase targets Suggest",
			phase:    GuidedPhaseFleet,
			wantView: ViewSuggest,
		},
		{
			name:      "Attached phase blocks Tab",
			phase:     GuidedPhaseAttached,
			wantView:  ViewType(-1),
			wantBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &GuidedFlowState{Phase: tt.phase}
			got := s.TabTarget()
			if got != tt.wantView {
				t.Errorf("got %d, want %d", got, tt.wantView)
			}
		})
	}
}
