package bench

import (
	"context"
	"fmt"
	"testing"
)

// mockRunner implements pipelineRunner for testing.
type mockRunner struct {
	results map[string]*BenchResult
	err     error
}

func (m *mockRunner) RunTask(_ context.Context, task BenchTask, cfg RunConfig) (*BenchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if r, ok := m.results[task.ID]; ok {
		return r, nil
	}
	return &BenchResult{
		TaskID:   task.ID,
		Pipeline: cfg.Pipeline,
		Status:   StatusPass,
	}, nil
}

func TestRunBenchmark(t *testing.T) {
	tasks := []BenchTask{
		{ID: "task-1", Problem: "Fix bug 1"},
		{ID: "task-2", Problem: "Fix bug 2"},
		{ID: "task-3", Problem: "Fix bug 3"},
	}

	t.Run("all pass", func(t *testing.T) {
		runner := &mockRunner{}
		cfg := RunConfig{Pipeline: "impl-issue"}

		report, err := RunBenchmark(context.Background(), tasks, cfg, runner)
		if err != nil {
			t.Fatalf("RunBenchmark() error = %v", err)
		}
		if report.Total != 3 {
			t.Errorf("Total = %d, want 3", report.Total)
		}
		if report.Passed != 3 {
			t.Errorf("Passed = %d, want 3", report.Passed)
		}
		if report.PassRate != 1.0 {
			t.Errorf("PassRate = %f, want 1.0", report.PassRate)
		}
	})

	t.Run("mixed results", func(t *testing.T) {
		runner := &mockRunner{
			results: map[string]*BenchResult{
				"task-1": {TaskID: "task-1", Status: StatusPass},
				"task-2": {TaskID: "task-2", Status: StatusFail, Error: "test failed"},
				"task-3": {TaskID: "task-3", Status: StatusError, Error: "timeout"},
			},
		}
		cfg := RunConfig{Pipeline: "impl-issue"}

		report, err := RunBenchmark(context.Background(), tasks, cfg, runner)
		if err != nil {
			t.Fatalf("RunBenchmark() error = %v", err)
		}
		if report.Passed != 1 {
			t.Errorf("Passed = %d, want 1", report.Passed)
		}
		if report.Failed != 1 {
			t.Errorf("Failed = %d, want 1", report.Failed)
		}
		if report.Errors != 1 {
			t.Errorf("Errors = %d, want 1", report.Errors)
		}
	})

	t.Run("with limit", func(t *testing.T) {
		runner := &mockRunner{}
		cfg := RunConfig{Pipeline: "impl-issue", Limit: 2}

		report, err := RunBenchmark(context.Background(), tasks, cfg, runner)
		if err != nil {
			t.Fatalf("RunBenchmark() error = %v", err)
		}
		if report.Total != 2 {
			t.Errorf("Total = %d, want 2", report.Total)
		}
	})

	t.Run("runner error", func(t *testing.T) {
		runner := &mockRunner{err: fmt.Errorf("connection refused")}
		cfg := RunConfig{Pipeline: "impl-issue"}

		report, err := RunBenchmark(context.Background(), tasks, cfg, runner)
		if err != nil {
			t.Fatalf("RunBenchmark() error = %v", err)
		}
		// Runner errors are recorded as StatusError in results
		if report.Errors != 3 {
			t.Errorf("Errors = %d, want 3", report.Errors)
		}
	})

	t.Run("missing pipeline wave mode", func(t *testing.T) {
		runner := &mockRunner{}
		cfg := RunConfig{Pipeline: ""}

		_, err := RunBenchmark(context.Background(), tasks, cfg, runner)
		if err == nil {
			t.Fatal("expected error for empty pipeline name in wave mode")
		}
	})

	t.Run("missing pipeline claude mode ok", func(t *testing.T) {
		runner := &mockRunner{}
		cfg := RunConfig{Pipeline: "", Mode: ModeClaude}

		report, err := RunBenchmark(context.Background(), tasks, cfg, runner)
		if err != nil {
			t.Fatalf("RunBenchmark() error = %v", err)
		}
		if report.Mode != ModeClaude {
			t.Errorf("Mode = %q, want %q", report.Mode, ModeClaude)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		runner := &mockRunner{}
		cfg := RunConfig{Pipeline: "impl-issue"}

		report, err := RunBenchmark(ctx, tasks, cfg, runner)
		if err == nil {
			t.Fatal("expected context error")
		}
		// Should still return a partial report
		if report == nil {
			t.Fatal("expected partial report even on cancellation")
		}
	})

	t.Run("mode and label propagation", func(t *testing.T) {
		runner := &mockRunner{}
		cfg := RunConfig{Pipeline: "bench-solve", Mode: ModeWave, RunLabel: "test-v1", Limit: 1}

		report, err := RunBenchmark(context.Background(), tasks, cfg, runner)
		if err != nil {
			t.Fatalf("RunBenchmark() error = %v", err)
		}
		if report.Mode != ModeWave {
			t.Errorf("Mode = %q, want %q", report.Mode, ModeWave)
		}
		if report.RunLabel != "test-v1" {
			t.Errorf("RunLabel = %q, want %q", report.RunLabel, "test-v1")
		}
	})

	t.Run("report timestamps", func(t *testing.T) {
		runner := &mockRunner{}
		cfg := RunConfig{Pipeline: "impl-issue", Limit: 1}

		report, err := RunBenchmark(context.Background(), tasks, cfg, runner)
		if err != nil {
			t.Fatalf("RunBenchmark() error = %v", err)
		}
		if report.StartedAt.IsZero() {
			t.Error("StartedAt should be set")
		}
		if report.CompletedAt.IsZero() {
			t.Error("CompletedAt should be set")
		}
		if report.CompletedAt.Before(report.StartedAt) {
			t.Error("CompletedAt should be after StartedAt")
		}
	})
}
