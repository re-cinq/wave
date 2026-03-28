package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRenderRetro_Nil(t *testing.T) {
	assert.Equal(t, "", RenderRetro(nil))
}

func TestRenderRetro_Minimal(t *testing.T) {
	retro := &RetroViewModel{
		TotalSteps:   3,
		SuccessCount: 3,
		Smoothness:   "smooth",
	}

	view := detailStripAnsi(RenderRetro(retro))

	assert.Contains(t, view, "Retrospective:")
	assert.Contains(t, view, "Smoothness:")
	assert.Contains(t, view, "smooth")
	assert.Contains(t, view, "Steps:")
	assert.Contains(t, view, "3/3 succeeded")
	assert.NotContains(t, view, "failed")
}

func TestRenderRetro_Full(t *testing.T) {
	retro := &RetroViewModel{
		RunID:        "run-xyz",
		Pipeline:     "impl-issue",
		Duration:     5*time.Minute + 30*time.Second,
		TotalSteps:   4,
		SuccessCount: 3,
		FailureCount: 1,
		TotalRetries: 2,
		TotalTokens:  125000,
		Smoothness:   "bumpy",
		Intent:       "Implement feature X",
		Outcome:      "Partial success, one step failed",
		FrictionPoints: []RetroFrictionPoint{
			{Type: "retry", Step: "implement", Detail: "Contract validation timeout"},
			{Type: "contract_failure", Step: "test", Detail: "Tests failed on first attempt"},
		},
		Learnings: []RetroLearning{
			{Category: "performance", Detail: "Contract tests should use shorter timeout"},
			{Category: "reliability", Detail: "Retry logic handled the failure well"},
		},
		Recommendations: []string{
			"Consider splitting the implement step",
			"Add pre-validation checks",
		},
	}

	view := detailStripAnsi(RenderRetro(retro))

	assert.Contains(t, view, "Retrospective:")
	assert.Contains(t, view, "bumpy")
	assert.Contains(t, view, "3/4 succeeded")
	assert.Contains(t, view, "1 failed")
	assert.Contains(t, view, "Retries:")
	assert.Contains(t, view, "2")
	assert.Contains(t, view, "Tokens:")
	assert.Contains(t, view, "125.0k")
	assert.Contains(t, view, "Intent:")
	assert.Contains(t, view, "Implement feature X")
	assert.Contains(t, view, "Outcome:")
	assert.Contains(t, view, "Partial success")
	assert.Contains(t, view, "Friction:")
	assert.Contains(t, view, "implement (retry)")
	assert.Contains(t, view, "Contract validation timeout")
	assert.Contains(t, view, "Learnings:")
	assert.Contains(t, view, "[performance]")
	assert.Contains(t, view, "Recommendations:")
	assert.Contains(t, view, "Consider splitting the implement step")
}

func TestRenderRetro_SmoothnessColors(t *testing.T) {
	tests := []struct {
		smoothness string
	}{
		{"effortless"},
		{"smooth"},
		{"bumpy"},
		{"struggled"},
		{"failed"},
		{"unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.smoothness, func(t *testing.T) {
			retro := &RetroViewModel{
				TotalSteps:   1,
				SuccessCount: 1,
				Smoothness:   tt.smoothness,
			}
			view := detailStripAnsi(RenderRetro(retro))
			assert.Contains(t, view, tt.smoothness)
		})
	}
}

func TestRenderRetro_NoFailures_HidesFailureCount(t *testing.T) {
	retro := &RetroViewModel{
		TotalSteps:   5,
		SuccessCount: 5,
		FailureCount: 0,
	}

	view := detailStripAnsi(RenderRetro(retro))

	assert.Contains(t, view, "5/5 succeeded")
	assert.NotContains(t, view, "failed")
}

func TestRenderRetro_ZeroRetries_HidesRetries(t *testing.T) {
	retro := &RetroViewModel{
		TotalSteps:   1,
		SuccessCount: 1,
		TotalRetries: 0,
	}

	view := detailStripAnsi(RenderRetro(retro))

	assert.NotContains(t, view, "Retries:")
}

func TestRenderRetro_ZeroTokens_HidesTokens(t *testing.T) {
	retro := &RetroViewModel{
		TotalSteps:   1,
		SuccessCount: 1,
		TotalTokens:  0,
	}

	view := detailStripAnsi(RenderRetro(retro))

	assert.NotContains(t, view, "Tokens:")
}

func TestRenderRetro_FrictionPointWithoutStep(t *testing.T) {
	retro := &RetroViewModel{
		TotalSteps:   1,
		SuccessCount: 1,
		FrictionPoints: []RetroFrictionPoint{
			{Type: "timeout", Detail: "API rate limit hit"},
		},
	}

	view := detailStripAnsi(RenderRetro(retro))

	assert.Contains(t, view, "timeout: API rate limit hit")
	assert.NotContains(t, view, "(timeout)")
}

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		tokens   int
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{1000, "1.0k"},
		{12345, "12.3k"},
		{125000, "125.0k"},
		{1000000, "1.0M"},
		{2500000, "2.5M"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatTokenCount(tt.tokens))
		})
	}
}

func TestStepTypeBadge(t *testing.T) {
	tests := []struct {
		stepType string
		expected string
	}{
		{"conditional", "[cond]"},
		{"command", "[cmd]"},
		{"gate", "[gate]"},
		{"pipeline", "[sub]"},
		{"", ""},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.stepType, func(t *testing.T) {
			assert.Equal(t, tt.expected, stepTypeBadge(tt.stepType))
		})
	}
}

func TestRenderFinishedDetail_WithStepTypeBadges(t *testing.T) {
	detail := &FinishedDetail{
		RunID:    "run-badges",
		Name:     "test-pipeline",
		Status:   "completed",
		Duration: 2 * time.Minute,
		Steps: []StepResult{
			{ID: "approve-plan", Status: "completed", Duration: 30 * time.Second, StepType: "gate", Persona: "navigator"},
			{ID: "run-tests", Status: "completed", Duration: 5 * time.Second, StepType: "command"},
			{ID: "check-result", Status: "completed", Duration: time.Second, StepType: "conditional"},
			{ID: "implement", Status: "completed", Duration: 1 * time.Minute, Persona: "craftsman"},
		},
	}

	view := detailStripAnsi(renderFinishedDetail(detail, 80, false, "", nil))

	assert.Contains(t, view, "[gate] approve-plan")
	assert.Contains(t, view, "[cmd] run-tests")
	assert.Contains(t, view, "[cond] check-result")
	// Regular agent step should NOT have a type badge prefix
	assert.NotContains(t, view, "[sub] implement")
	assert.Contains(t, view, "implement")
}

func TestRenderFinishedDetail_WithRetro(t *testing.T) {
	detail := &FinishedDetail{
		RunID:    "run-retro",
		Name:     "test-pipeline",
		Status:   "completed",
		Duration: 5 * time.Minute,
		Steps: []StepResult{
			{ID: "step1", Status: "completed", Duration: 2 * time.Minute, Persona: "navigator"},
		},
		Retro: &RetroViewModel{
			TotalSteps:   1,
			SuccessCount: 1,
			Smoothness:   "smooth",
			Intent:       "Test the retro integration",
			Outcome:      "Success",
		},
	}

	view := detailStripAnsi(renderFinishedDetail(detail, 80, false, "", nil))

	assert.Contains(t, view, "Retrospective:")
	assert.Contains(t, view, "smooth")
	assert.Contains(t, view, "Test the retro integration")
	assert.Contains(t, view, "Success")
}

func TestRenderFinishedDetail_WithoutRetro(t *testing.T) {
	detail := fullFinishedDetail("completed")
	detail.Retro = nil

	view := detailStripAnsi(renderFinishedDetail(detail, 80, false, "", nil))

	assert.NotContains(t, view, "Retrospective:")
}
