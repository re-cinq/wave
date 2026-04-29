package retro

import (
	"fmt"
	"testing"
	"time"

	"github.com/recinq/wave/internal/metrics"
	"github.com/recinq/wave/internal/state"
)

// mockStateQuerier implements StateQuerier for testing.
type mockStateQuerier struct {
	run      *state.RunRecord
	runErr   error
	perfMetrics []metrics.PerformanceMetricRecord
	metErr   error
	attempts map[string][]state.StepAttemptRecord
	attErr   error
}

func (m *mockStateQuerier) GetRun(runID string) (*state.RunRecord, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	return m.run, nil
}

func (m *mockStateQuerier) GetPerformanceMetrics(runID, stepID string) ([]metrics.PerformanceMetricRecord, error) {
	if m.metErr != nil {
		return nil, m.metErr
	}
	return m.perfMetrics, nil
}

func (m *mockStateQuerier) GetStepAttempts(runID, stepID string) ([]state.StepAttemptRecord, error) {
	if m.attErr != nil {
		return nil, m.attErr
	}
	return m.attempts[stepID], nil
}

func TestCollector_Collect(t *testing.T) {
	now := time.Now()
	completed := now.Add(2 * time.Minute)

	tests := []struct {
		name        string
		mock        *mockStateQuerier
		wantErr     bool
		wantSteps   int
		wantSuccess int
		wantFailure int
		wantRetries int
	}{
		{
			name: "successful run with two steps",
			mock: &mockStateQuerier{
				run: &state.RunRecord{
					RunID:        "test-run-1",
					PipelineName: "impl-issue",
					Status:       "completed",
					TotalTokens:  5000,
					StartedAt:    now,
					CompletedAt:  &completed,
				},
				perfMetrics: []metrics.PerformanceMetricRecord{
					{StepID: "plan", DurationMs: 30000, TokensUsed: 2000, FilesModified: 1, Success: true, Persona: "navigator"},
					{StepID: "implement", DurationMs: 90000, TokensUsed: 3000, FilesModified: 5, Success: true, Persona: "craftsman"},
				},
				attempts: map[string][]state.StepAttemptRecord{
					"plan":      {{Attempt: 1, State: "succeeded"}},
					"implement": {{Attempt: 1, State: "succeeded"}},
				},
			},
			wantSteps:   2,
			wantSuccess: 2,
			wantFailure: 0,
			wantRetries: 0,
		},
		{
			name: "run with retries",
			mock: &mockStateQuerier{
				run: &state.RunRecord{
					RunID:       "test-run-2",
					Status:      "completed",
					StartedAt:   now,
					CompletedAt: &completed,
				},
				perfMetrics: []metrics.PerformanceMetricRecord{
					{StepID: "implement", DurationMs: 60000, TokensUsed: 2000, Success: false, Persona: "craftsman"},
					{StepID: "implement", DurationMs: 60000, TokensUsed: 2000, Success: true, Persona: "craftsman"},
				},
				attempts: map[string][]state.StepAttemptRecord{
					"implement": {
						{Attempt: 1, State: "failed"},
						{Attempt: 2, State: "succeeded"},
					},
				},
			},
			wantSteps:   1,
			wantSuccess: 1,
			wantRetries: 1,
		},
		{
			name: "failed run",
			mock: &mockStateQuerier{
				run: &state.RunRecord{
					RunID:       "test-run-3",
					Status:      "failed",
					StartedAt:   now,
					CompletedAt: &completed,
				},
				perfMetrics: []metrics.PerformanceMetricRecord{
					{StepID: "plan", DurationMs: 30000, TokensUsed: 2000, Success: true},
					{StepID: "implement", DurationMs: 60000, TokensUsed: 3000, Success: false},
				},
				attempts: map[string][]state.StepAttemptRecord{},
			},
			wantSteps:   2,
			wantSuccess: 1,
			wantFailure: 1,
		},
		{
			name: "run not found",
			mock: &mockStateQuerier{
				runErr: fmt.Errorf("not found"),
			},
			wantErr: true,
		},
		{
			name: "empty run no metrics",
			mock: &mockStateQuerier{
				run: &state.RunRecord{
					RunID:       "test-run-4",
					Status:      "completed",
					TotalTokens: 1000,
					StartedAt:   now,
					CompletedAt: &completed,
				},
				perfMetrics: nil,
				attempts: map[string][]state.StepAttemptRecord{},
			},
			wantSteps: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCollector(tt.mock)
			result, err := c.Collect("test-run")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Steps) != tt.wantSteps {
				t.Errorf("steps: got %d, want %d", len(result.Steps), tt.wantSteps)
			}
			if result.SuccessCount != tt.wantSuccess {
				t.Errorf("success count: got %d, want %d", result.SuccessCount, tt.wantSuccess)
			}
			if result.FailureCount != tt.wantFailure {
				t.Errorf("failure count: got %d, want %d", result.FailureCount, tt.wantFailure)
			}
			if result.TotalRetries != tt.wantRetries {
				t.Errorf("total retries: got %d, want %d", result.TotalRetries, tt.wantRetries)
			}
		})
	}
}
