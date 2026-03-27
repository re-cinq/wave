package retro

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
)

// mockRunner implements adapter.AdapterRunner for testing.
type mockRunner struct {
	result *adapter.AdapterResult
	err    error
	config adapter.AdapterRunConfig // captured for assertions
}

func (m *mockRunner) Run(_ context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	m.config = cfg
	return m.result, m.err
}

func narratorSampleRetro() *Retrospective {
	return &Retrospective{
		RunID:    "run-123",
		Pipeline: "impl-issue",
		Timestamp: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC),
		Quantitative: QuantitativeData{
			TotalDurationMs: 45000,
			TotalSteps:      2,
			SuccessCount:    2,
			FailureCount:    0,
			TotalRetries:    1,
			Steps: []StepMetrics{
				{Name: "fetch-assess", DurationMs: 15000, Retries: 0, Status: "success", TokensUsed: 500},
				{Name: "implement", DurationMs: 30000, Retries: 1, Status: "success", TokensUsed: 2000},
			},
		},
	}
}

const validNarrativeJSON = `{
  "smoothness": "smooth",
  "intent": "Implement a new feature based on GitHub issue",
  "outcome": "Feature implemented successfully with one retry",
  "friction_points": [{"type": "retry", "step": "implement", "detail": "First attempt had a compilation error"}],
  "learnings": [{"category": "code", "detail": "Package requires explicit imports"}],
  "open_items": [{"type": "test_gap", "detail": "Edge case for empty input not tested"}]
}`

func TestNarrate_Success(t *testing.T) {
	runner := &mockRunner{
		result: &adapter.AdapterResult{
			ExitCode:      0,
			ResultContent: validNarrativeJSON,
		},
	}

	narrator := NewNarrator(runner, "claude-haiku-4-5")
	narrative, err := narrator.Narrate(context.Background(), narratorSampleRetro(), "Fix bug #42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if narrative.Smoothness != "smooth" {
		t.Errorf("smoothness = %q, want %q", narrative.Smoothness, "smooth")
	}
	if narrative.Intent != "Implement a new feature based on GitHub issue" {
		t.Errorf("intent = %q, want %q", narrative.Intent, "Implement a new feature based on GitHub issue")
	}
	if narrative.Outcome != "Feature implemented successfully with one retry" {
		t.Errorf("outcome = %q, want %q", narrative.Outcome, "Feature implemented successfully with one retry")
	}
	if len(narrative.FrictionPoints) != 1 {
		t.Fatalf("friction_points length = %d, want 1", len(narrative.FrictionPoints))
	}
	if narrative.FrictionPoints[0].Type != "retry" {
		t.Errorf("friction_points[0].type = %q, want %q", narrative.FrictionPoints[0].Type, "retry")
	}
	if narrative.FrictionPoints[0].Step != "implement" {
		t.Errorf("friction_points[0].step = %q, want %q", narrative.FrictionPoints[0].Step, "implement")
	}
	if len(narrative.Learnings) != 1 {
		t.Fatalf("learnings length = %d, want 1", len(narrative.Learnings))
	}
	if narrative.Learnings[0].Category != "code" {
		t.Errorf("learnings[0].category = %q, want %q", narrative.Learnings[0].Category, "code")
	}
	if len(narrative.OpenItems) != 1 {
		t.Fatalf("open_items length = %d, want 1", len(narrative.OpenItems))
	}
	if narrative.OpenItems[0].Type != "test_gap" {
		t.Errorf("open_items[0].type = %q, want %q", narrative.OpenItems[0].Type, "test_gap")
	}
}

func TestNarrate_PromptConstruction(t *testing.T) {
	runner := &mockRunner{
		result: &adapter.AdapterResult{
			ExitCode:      0,
			ResultContent: validNarrativeJSON,
		},
	}

	narrator := NewNarrator(runner, "claude-haiku-4-5")
	_, err := narrator.Narrate(context.Background(), narratorSampleRetro(), "Fix bug #42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prompt := runner.config.Prompt

	// Verify prompt contains pipeline name.
	if !strings.Contains(prompt, "Pipeline: impl-issue") {
		t.Error("prompt missing pipeline name")
	}

	// Verify prompt contains the input.
	if !strings.Contains(prompt, "Input: Fix bug #42") {
		t.Error("prompt missing input")
	}

	// Verify prompt contains total duration.
	if !strings.Contains(prompt, "Total Duration: 45000ms") {
		t.Error("prompt missing total duration")
	}

	// Verify prompt contains step names and details.
	if !strings.Contains(prompt, "fetch-assess: success (15000ms, 0 retries, 500 tokens)") {
		t.Error("prompt missing fetch-assess step details")
	}
	if !strings.Contains(prompt, "implement: success (30000ms, 1 retries, 2000 tokens)") {
		t.Error("prompt missing implement step details")
	}

	// Verify adapter config.
	if runner.config.Adapter != "claude" {
		t.Errorf("adapter = %q, want %q", runner.config.Adapter, "claude")
	}
	if runner.config.Model != "claude-haiku-4-5" {
		t.Errorf("model = %q, want %q", runner.config.Model, "claude-haiku-4-5")
	}
}

func TestNarrate_AdapterFailure(t *testing.T) {
	runner := &mockRunner{
		err: errors.New("adapter connection refused"),
	}

	narrator := NewNarrator(runner, "claude-haiku-4-5")
	_, err := narrator.Narrate(context.Background(), narratorSampleRetro(), "Fix bug #42")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "adapter connection refused") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "adapter connection refused")
	}
	if !strings.Contains(err.Error(), "narrator adapter run failed") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "narrator adapter run failed")
	}
}

func TestNarrate_InvalidJSON(t *testing.T) {
	runner := &mockRunner{
		result: &adapter.AdapterResult{
			ExitCode:      0,
			ResultContent: "This is not JSON at all, just some text response.",
		},
	}

	narrator := NewNarrator(runner, "claude-haiku-4-5")
	_, err := narrator.Narrate(context.Background(), narratorSampleRetro(), "Fix bug #42")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "narrator failed to parse response") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "narrator failed to parse response")
	}
}

func TestNarrate_JSONInMarkdownCodeBlock(t *testing.T) {
	wrapped := "Here is the analysis:\n\n```json\n" + validNarrativeJSON + "\n```\n\nHope that helps!"

	runner := &mockRunner{
		result: &adapter.AdapterResult{
			ExitCode:      0,
			ResultContent: wrapped,
		},
	}

	narrator := NewNarrator(runner, "claude-haiku-4-5")
	narrative, err := narrator.Narrate(context.Background(), narratorSampleRetro(), "Fix bug #42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if narrative.Smoothness != "smooth" {
		t.Errorf("smoothness = %q, want %q", narrative.Smoothness, "smooth")
	}
	if len(narrative.FrictionPoints) != 1 {
		t.Errorf("friction_points length = %d, want 1", len(narrative.FrictionPoints))
	}
}

func TestNarrate_JSONInPlainCodeBlock(t *testing.T) {
	// Code block without language tag.
	wrapped := "```\n" + validNarrativeJSON + "\n```"

	runner := &mockRunner{
		result: &adapter.AdapterResult{
			ExitCode:      0,
			ResultContent: wrapped,
		},
	}

	narrator := NewNarrator(runner, "claude-haiku-4-5")
	narrative, err := narrator.Narrate(context.Background(), narratorSampleRetro(), "Fix bug #42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if narrative.Smoothness != "smooth" {
		t.Errorf("smoothness = %q, want %q", narrative.Smoothness, "smooth")
	}
}
