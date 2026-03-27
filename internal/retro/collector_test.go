package retro_test

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/retro"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/testutil"
)

func TestCollector_BasicSuccessfulRun(t *testing.T) {
	now := time.Now()
	completed := now.Add(10 * time.Second)

	store := testutil.NewMockStateStore(
		testutil.WithGetStepStates(func(pipelineID string) ([]state.StepStateRecord, error) {
			return []state.StepStateRecord{
				{StepID: "step-a", PipelineID: pipelineID, State: state.StateCompleted},
				{StepID: "step-b", PipelineID: pipelineID, State: state.StateCompleted},
			}, nil
		}),
		testutil.WithGetStepAttempts(func(runID, stepID string) ([]state.StepAttemptRecord, error) {
			// Single attempt per step — no retries.
			return []state.StepAttemptRecord{
				{RunID: runID, StepID: stepID, Attempt: 1, State: "succeeded", StartedAt: now, CompletedAt: &completed},
			}, nil
		}),
		testutil.WithGetPerformanceMetrics(func(runID, stepID string) ([]state.PerformanceMetricRecord, error) {
			return []state.PerformanceMetricRecord{
				{StepID: "step-a", RunID: runID, DurationMs: 5000, TokensUsed: 100, Persona: "navigator", Success: true, StartedAt: now, CompletedAt: &completed},
				{StepID: "step-b", RunID: runID, DurationMs: 3000, TokensUsed: 80, Persona: "craftsman", Success: true, StartedAt: now, CompletedAt: &completed},
			}, nil
		}),
	)

	c := retro.NewCollector(store)
	r, err := c.Collect("run-123", "impl-issue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.RunID != "run-123" {
		t.Errorf("RunID = %q, want %q", r.RunID, "run-123")
	}
	if r.Pipeline != "impl-issue" {
		t.Errorf("Pipeline = %q, want %q", r.Pipeline, "impl-issue")
	}
	if r.Narrative != nil {
		t.Errorf("Narrative = %v, want nil", r.Narrative)
	}

	q := r.Quantitative
	if q.TotalSteps != 2 {
		t.Errorf("TotalSteps = %d, want 2", q.TotalSteps)
	}
	if q.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2", q.SuccessCount)
	}
	if q.FailureCount != 0 {
		t.Errorf("FailureCount = %d, want 0", q.FailureCount)
	}
	if q.TotalRetries != 0 {
		t.Errorf("TotalRetries = %d, want 0", q.TotalRetries)
	}
	if q.TotalDurationMs != 8000 {
		t.Errorf("TotalDurationMs = %d, want 8000", q.TotalDurationMs)
	}

	if len(q.Steps) != 2 {
		t.Fatalf("len(Steps) = %d, want 2", len(q.Steps))
	}

	stepA := q.Steps[0]
	if stepA.Name != "step-a" {
		t.Errorf("Steps[0].Name = %q, want %q", stepA.Name, "step-a")
	}
	if stepA.DurationMs != 5000 {
		t.Errorf("Steps[0].DurationMs = %d, want 5000", stepA.DurationMs)
	}
	if stepA.TokensUsed != 100 {
		t.Errorf("Steps[0].TokensUsed = %d, want 100", stepA.TokensUsed)
	}
	if stepA.Status != "success" {
		t.Errorf("Steps[0].Status = %q, want %q", stepA.Status, "success")
	}
	if stepA.Retries != 0 {
		t.Errorf("Steps[0].Retries = %d, want 0", stepA.Retries)
	}
	if stepA.ExitCode != 0 {
		t.Errorf("Steps[0].ExitCode = %d, want 0", stepA.ExitCode)
	}

	stepB := q.Steps[1]
	if stepB.Name != "step-b" {
		t.Errorf("Steps[1].Name = %q, want %q", stepB.Name, "step-b")
	}
	if stepB.DurationMs != 3000 {
		t.Errorf("Steps[1].DurationMs = %d, want 3000", stepB.DurationMs)
	}
	if stepB.TokensUsed != 80 {
		t.Errorf("Steps[1].TokensUsed = %d, want 80", stepB.TokensUsed)
	}
}

func TestCollector_RunWithRetries(t *testing.T) {
	now := time.Now()
	completed := now.Add(20 * time.Second)

	store := testutil.NewMockStateStore(
		testutil.WithGetStepStates(func(pipelineID string) ([]state.StepStateRecord, error) {
			return []state.StepStateRecord{
				{StepID: "step-a", PipelineID: pipelineID, State: state.StateCompleted},
				{StepID: "step-b", PipelineID: pipelineID, State: state.StateCompleted},
			}, nil
		}),
		testutil.WithGetStepAttempts(func(runID, stepID string) ([]state.StepAttemptRecord, error) {
			if stepID == "step-a" {
				// 3 attempts = 2 retries.
				return []state.StepAttemptRecord{
					{RunID: runID, StepID: stepID, Attempt: 1, State: "failed", StartedAt: now},
					{RunID: runID, StepID: stepID, Attempt: 2, State: "failed", StartedAt: now},
					{RunID: runID, StepID: stepID, Attempt: 3, State: "succeeded", StartedAt: now, CompletedAt: &completed},
				}, nil
			}
			// step-b: single attempt.
			return []state.StepAttemptRecord{
				{RunID: runID, StepID: stepID, Attempt: 1, State: "succeeded", StartedAt: now, CompletedAt: &completed},
			}, nil
		}),
		testutil.WithGetPerformanceMetrics(func(runID, stepID string) ([]state.PerformanceMetricRecord, error) {
			return []state.PerformanceMetricRecord{
				{StepID: "step-a", RunID: runID, DurationMs: 15000, TokensUsed: 300, Persona: "craftsman", Success: true, StartedAt: now, CompletedAt: &completed},
				{StepID: "step-b", RunID: runID, DurationMs: 2000, TokensUsed: 50, Persona: "navigator", Success: true, StartedAt: now, CompletedAt: &completed},
			}, nil
		}),
	)

	c := retro.NewCollector(store)
	r, err := c.Collect("run-retry", "impl-issue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := r.Quantitative
	if q.TotalRetries != 2 {
		t.Errorf("TotalRetries = %d, want 2", q.TotalRetries)
	}
	if q.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2", q.SuccessCount)
	}
	if q.TotalDurationMs != 17000 {
		t.Errorf("TotalDurationMs = %d, want 17000", q.TotalDurationMs)
	}

	if len(q.Steps) != 2 {
		t.Fatalf("len(Steps) = %d, want 2", len(q.Steps))
	}
	if q.Steps[0].Retries != 2 {
		t.Errorf("Steps[0].Retries = %d, want 2", q.Steps[0].Retries)
	}
	if q.Steps[1].Retries != 0 {
		t.Errorf("Steps[1].Retries = %d, want 0", q.Steps[1].Retries)
	}
}

func TestCollector_AllStepsFailed(t *testing.T) {
	now := time.Now()

	store := testutil.NewMockStateStore(
		testutil.WithGetStepStates(func(pipelineID string) ([]state.StepStateRecord, error) {
			return []state.StepStateRecord{
				{StepID: "step-a", PipelineID: pipelineID, State: state.StateFailed},
				{StepID: "step-b", PipelineID: pipelineID, State: state.StateFailed},
			}, nil
		}),
		testutil.WithGetStepAttempts(func(runID, stepID string) ([]state.StepAttemptRecord, error) {
			return []state.StepAttemptRecord{
				{RunID: runID, StepID: stepID, Attempt: 1, State: "failed", StartedAt: now},
			}, nil
		}),
		testutil.WithGetPerformanceMetrics(func(runID, stepID string) ([]state.PerformanceMetricRecord, error) {
			return []state.PerformanceMetricRecord{
				{StepID: "step-a", RunID: runID, DurationMs: 1000, TokensUsed: 20, Success: false, StartedAt: now, ErrorMessage: "contract failed"},
				{StepID: "step-b", RunID: runID, DurationMs: 500, TokensUsed: 10, Success: false, StartedAt: now, ErrorMessage: "timeout"},
			}, nil
		}),
	)

	c := retro.NewCollector(store)
	r, err := c.Collect("run-fail", "impl-issue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := r.Quantitative
	if q.TotalSteps != 2 {
		t.Errorf("TotalSteps = %d, want 2", q.TotalSteps)
	}
	if q.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0", q.SuccessCount)
	}
	if q.FailureCount != 2 {
		t.Errorf("FailureCount = %d, want 2", q.FailureCount)
	}
	if q.TotalDurationMs != 1500 {
		t.Errorf("TotalDurationMs = %d, want 1500", q.TotalDurationMs)
	}

	for i, step := range q.Steps {
		if step.Status != "failed" {
			t.Errorf("Steps[%d].Status = %q, want %q", i, step.Status, "failed")
		}
		if step.ExitCode != 1 {
			t.Errorf("Steps[%d].ExitCode = %d, want 1", i, step.ExitCode)
		}
	}
}

func TestCollector_EmptyRun(t *testing.T) {
	store := testutil.NewMockStateStore(
		testutil.WithGetStepStates(func(pipelineID string) ([]state.StepStateRecord, error) {
			return nil, nil
		}),
		testutil.WithGetPerformanceMetrics(func(runID, stepID string) ([]state.PerformanceMetricRecord, error) {
			return nil, nil
		}),
	)

	c := retro.NewCollector(store)
	r, err := c.Collect("run-empty", "impl-issue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := r.Quantitative
	if q.TotalSteps != 0 {
		t.Errorf("TotalSteps = %d, want 0", q.TotalSteps)
	}
	if q.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0", q.SuccessCount)
	}
	if q.FailureCount != 0 {
		t.Errorf("FailureCount = %d, want 0", q.FailureCount)
	}
	if q.TotalRetries != 0 {
		t.Errorf("TotalRetries = %d, want 0", q.TotalRetries)
	}
	if q.TotalDurationMs != 0 {
		t.Errorf("TotalDurationMs = %d, want 0", q.TotalDurationMs)
	}
	if q.Steps != nil {
		t.Errorf("Steps = %v, want nil", q.Steps)
	}
	if r.Narrative != nil {
		t.Errorf("Narrative = %v, want nil", r.Narrative)
	}
}
