package pipeline

import "testing"

func TestConvergenceTracker_NotStalledWithFewScores(t *testing.T) {
	ct := NewConvergenceTracker(3, 0.05)
	ct.RecordScore(0.5)
	if ct.IsStalled() {
		t.Error("should not be stalled with only 1 score")
	}
	ct.RecordScore(0.5)
	if ct.IsStalled() {
		t.Error("should not be stalled with only 2 scores (window=3)")
	}
}

func TestConvergenceTracker_StalledWhenFlat(t *testing.T) {
	ct := NewConvergenceTracker(3, 0.05)
	ct.RecordScore(0.5)
	ct.RecordScore(0.5)
	ct.RecordScore(0.51) // improvement < 0.05
	if !ct.IsStalled() {
		t.Error("should be stalled: 0.51 - 0.5 = 0.01 < 0.05")
	}
}

func TestConvergenceTracker_NotStalledWhenImproving(t *testing.T) {
	ct := NewConvergenceTracker(3, 0.05)
	ct.RecordScore(0.5)
	ct.RecordScore(0.55)
	ct.RecordScore(0.6)
	if ct.IsStalled() {
		t.Error("should not be stalled: 0.6 - 0.5 = 0.1 >= 0.05")
	}
}

func TestConvergenceTracker_StalledAfterPlateau(t *testing.T) {
	ct := NewConvergenceTracker(3, 0.1)
	// Initial improvement
	ct.RecordScore(0.3)
	ct.RecordScore(0.5)
	ct.RecordScore(0.7)
	if ct.IsStalled() {
		t.Error("should not be stalled during improvement phase")
	}
	// Plateau
	ct.RecordScore(0.71)
	ct.RecordScore(0.72)
	if !ct.IsStalled() {
		t.Error("should be stalled: last 3 scores [0.7, 0.71, 0.72] delta = 0.02 < 0.1")
	}
}

func TestConvergenceTracker_LastScore(t *testing.T) {
	ct := NewConvergenceTracker(3, 0.05)
	if ct.LastScore() != 0 {
		t.Error("empty tracker should return 0")
	}
	ct.RecordScore(0.75)
	if ct.LastScore() != 0.75 {
		t.Errorf("LastScore() = %f, want 0.75", ct.LastScore())
	}
}

func TestConvergenceTracker_Summary(t *testing.T) {
	ct := NewConvergenceTracker(3, 0.05)
	if ct.Summary() != "no scores recorded" {
		t.Errorf("empty summary = %q", ct.Summary())
	}
	ct.RecordScore(0.75)
	want := "75% after 1 round(s)"
	if ct.Summary() != want {
		t.Errorf("Summary() = %q, want %q", ct.Summary(), want)
	}
}

func TestExtractScoreFromError(t *testing.T) {
	tests := []struct {
		msg       string
		wantScore float64
		wantOK    bool
	}{
		{"Score: 75% (threshold: 100%)", 0.75, true},
		{"Score: 0% (threshold: 100%)", 0, true},
		{"Score: 100% (threshold: 100%)", 1.0, true},
		{"no score here", 0, false},
		{"LLM judge score 50% is below threshold", 0, false}, // different format
	}

	for _, tt := range tests {
		score, ok := ExtractScoreFromError(tt.msg)
		if ok != tt.wantOK {
			t.Errorf("ExtractScoreFromError(%q) ok = %v, want %v", tt.msg, ok, tt.wantOK)
		}
		if score != tt.wantScore {
			t.Errorf("ExtractScoreFromError(%q) = %f, want %f", tt.msg, score, tt.wantScore)
		}
	}
}

func TestNewConvergenceTracker_Defaults(t *testing.T) {
	ct := NewConvergenceTracker(0, 0)
	if ct.window != 2 {
		t.Errorf("window = %d, want 2 (minimum)", ct.window)
	}
	if ct.minImprovement != 0.05 {
		t.Errorf("minImprovement = %f, want 0.05 (default)", ct.minImprovement)
	}
}
