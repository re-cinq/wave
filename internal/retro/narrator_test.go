package retro

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/adapter"
)

// mockAdapterRunner implements adapter.AdapterRunner for testing.
type mockAdapterRunner struct {
	resultContent string
	exitCode      int
	err           error
}

func (m *mockAdapterRunner) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &adapter.AdapterResult{
		ExitCode:      m.exitCode,
		ResultContent: m.resultContent,
		Stdout:        strings.NewReader(m.resultContent),
	}, nil
}

func TestNarrator_Narrate(t *testing.T) {
	validJSON := `{
		"smoothness": "smooth",
		"intent": "Implement feature X",
		"outcome": "Completed successfully",
		"friction_points": [{"type": "retry", "step": "implement", "detail": "Wrong import path"}],
		"learnings": [{"category": "code", "detail": "Uses custom middleware"}],
		"open_items": [{"type": "test_gap", "detail": "No integration test"}],
		"recommendations": ["Add integration test coverage"]
	}`

	tests := []struct {
		name       string
		mock       *mockAdapterRunner
		wantErr    bool
		wantSmooth Smoothness
	}{
		{
			name: "successful narration",
			mock: &mockAdapterRunner{
				resultContent: validJSON,
			},
			wantSmooth: SmoothnessSmooth,
		},
		{
			name: "json wrapped in markdown",
			mock: &mockAdapterRunner{
				resultContent: "```json\n" + validJSON + "\n```",
			},
			wantSmooth: SmoothnessSmooth,
		},
		{
			name: "adapter error",
			mock: &mockAdapterRunner{
				err: fmt.Errorf("adapter unavailable"),
			},
			wantErr: true,
		},
		{
			name: "non-zero exit code",
			mock: &mockAdapterRunner{
				exitCode: 1,
			},
			wantErr: true,
		},
		{
			name: "malformed response uses fallback",
			mock: &mockAdapterRunner{
				resultContent: "this is not json",
			},
			wantSmooth: SmoothnessSmooth, // fallback for 0 retries, 0 failures
		},
	}

	quant := &QuantitativeData{
		TotalDurationMs: 120000,
		TotalSteps:      2,
		SuccessCount:    2,
		Steps: []StepMetrics{
			{Name: "plan", DurationMs: 30000, Status: "success"},
			{Name: "implement", DurationMs: 90000, Status: "success"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewNarrator(tt.mock, "test-model")
			narrative, err := n.Narrate(context.Background(), "test-run", "impl-issue", quant)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if narrative.Smoothness != tt.wantSmooth {
				t.Errorf("smoothness: got %s, want %s", narrative.Smoothness, tt.wantSmooth)
			}
		})
	}
}

func TestNarrator_BuildPrompt(t *testing.T) {
	n := NewNarrator(nil, "test-model")
	quant := &QuantitativeData{
		TotalDurationMs: 60000,
		TotalSteps:      1,
		SuccessCount:    1,
		TotalRetries:    2,
		TotalTokens:     5000,
		Steps: []StepMetrics{
			{Name: "implement", DurationMs: 60000, Status: "success", Retries: 2, TokensUsed: 5000},
		},
	}

	prompt := n.buildPrompt("run-1", "impl-issue", quant)

	if !strings.Contains(prompt, "run-1") {
		t.Error("prompt should contain run ID")
	}
	if !strings.Contains(prompt, "impl-issue") {
		t.Error("prompt should contain pipeline name")
	}
	if !strings.Contains(prompt, "implement") {
		t.Error("prompt should contain step name")
	}
	if !strings.Contains(prompt, "smoothness") {
		t.Error("prompt should contain JSON schema instruction")
	}
}

func TestParseNarrativeResponse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantSmooth Smoothness
	}{
		{
			name:       "valid JSON",
			input:      `{"smoothness": "bumpy", "intent": "test", "outcome": "done"}`,
			wantSmooth: SmoothnessBumpy,
		},
		{
			name:       "invalid smoothness defaults to bumpy",
			input:      `{"smoothness": "invalid", "intent": "test", "outcome": "done"}`,
			wantSmooth: SmoothnessBumpy,
		},
		{
			name:    "not JSON",
			input:   "this is not json at all",
			wantErr: true,
		},
		{
			name:       "JSON with surrounding text",
			input:      "Here is the analysis:\n{\"smoothness\": \"effortless\", \"intent\": \"test\", \"outcome\": \"done\"}\nEnd.",
			wantSmooth: SmoothnessEffortless,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseNarrativeResponse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if n.Smoothness != tt.wantSmooth {
				t.Errorf("smoothness: got %s, want %s", n.Smoothness, tt.wantSmooth)
			}
		})
	}
}

func TestFallbackNarrative(t *testing.T) {
	n := NewNarrator(nil, "test")

	tests := []struct {
		name       string
		quant      *QuantitativeData
		wantSmooth Smoothness
	}{
		{
			name:       "clean run",
			quant:      &QuantitativeData{SuccessCount: 2, TotalSteps: 2},
			wantSmooth: SmoothnessSmooth,
		},
		{
			name:       "with retries",
			quant:      &QuantitativeData{SuccessCount: 2, TotalSteps: 2, TotalRetries: 1},
			wantSmooth: SmoothnessBumpy,
		},
		{
			name:       "many retries",
			quant:      &QuantitativeData{SuccessCount: 2, TotalSteps: 2, TotalRetries: 3},
			wantSmooth: SmoothnessStruggled,
		},
		{
			name:       "with failures",
			quant:      &QuantitativeData{SuccessCount: 1, FailureCount: 1, TotalSteps: 2},
			wantSmooth: SmoothnessFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := n.fallbackNarrative(tt.quant)
			if result.Smoothness != tt.wantSmooth {
				t.Errorf("smoothness: got %s, want %s", result.Smoothness, tt.wantSmooth)
			}
		})
	}
}
