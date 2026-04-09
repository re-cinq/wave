package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/bench"
)

func TestNewBenchCmd_Structure(t *testing.T) {
	cmd := NewBenchCmd()

	if cmd.Use != "bench" {
		t.Errorf("Use = %q, want %q", cmd.Use, "bench")
	}

	subs := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subs[sub.Use] = true
	}

	for _, want := range []string{"run", "report", "list", "compare"} {
		if !subs[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

func TestBenchRunCmd_Validation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing dataset",
			args:    []string{"run", "--pipeline", "bench-solve"},
			wantErr: "--dataset is required",
		},
		{
			name:    "missing pipeline in wave mode",
			args:    []string{"run", "--dataset", "test.jsonl"},
			wantErr: "--pipeline is required",
		},
		{
			name:    "invalid mode",
			args:    []string{"run", "--dataset", "test.jsonl", "--mode", "invalid"},
			wantErr: "--mode must be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewBenchCmd()
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != "" {
				if got := err.Error(); !strings.Contains(got, tt.wantErr) {
					t.Errorf("error = %q, want to contain %q", got, tt.wantErr)
				}
			}
		})
	}
}

func TestBenchRunCmd_ClaudeModeNoRequirePipeline(t *testing.T) {
	// Claude mode should not require --pipeline, but will fail on dataset load
	cmd := NewBenchCmd()
	cmd.SetArgs([]string{"run", "--dataset", "/nonexistent.jsonl", "--mode", "claude"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error (dataset not found)")
	}
	// The error should be about loading the dataset, not missing pipeline
	if strings.Contains(err.Error(), "--pipeline is required") {
		t.Errorf("claude mode should not require --pipeline, got: %v", err)
	}
}

func TestBenchCompareCmd_Validation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing base",
			args:    []string{"compare", "--compare", "b.json"},
			wantErr: "--base is required",
		},
		{
			name:    "missing compare",
			args:    []string{"compare", "--base", "a.json"},
			wantErr: "--compare is required",
		},
		{
			name:    "nonexistent base file",
			args:    []string{"compare", "--base", "/nonexistent/a.json", "--compare", "/nonexistent/b.json"},
			wantErr: "load base report",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewBenchCmd()
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != "" {
				if got := err.Error(); !strings.Contains(got, tt.wantErr) {
					t.Errorf("error = %q, want to contain %q", got, tt.wantErr)
				}
			}
		})
	}
}

func TestBenchCompareCmd_WithFiles(t *testing.T) {
	dir := t.TempDir()

	baseReport := bench.BenchReport{
		Pipeline: "baseline",
		Results: []bench.BenchResult{
			{TaskID: "t1", Status: bench.StatusPass},
			{TaskID: "t2", Status: bench.StatusFail},
		},
	}
	baseReport.Tally()

	compReport := bench.BenchReport{
		Pipeline: "wave",
		Results: []bench.BenchResult{
			{TaskID: "t1", Status: bench.StatusPass},
			{TaskID: "t2", Status: bench.StatusPass},
		},
	}
	compReport.Tally()

	basePath := filepath.Join(dir, "base.json")
	compPath := filepath.Join(dir, "comp.json")

	writeJSON(t, basePath, baseReport)
	writeJSON(t, compPath, compReport)

	cmd := NewBenchCmd()
	cmd.SetArgs([]string{"compare", "--base", basePath, "--compare", compPath})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("compare failed: %v", err)
	}
}

func TestBenchListCmd_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	cmd := NewBenchCmd()
	cmd.SetArgs([]string{"list", "--datasets-dir", dir})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestBenchListCmd_MissingDir(t *testing.T) {
	cmd := NewBenchCmd()
	cmd.SetArgs([]string{"list", "--datasets-dir", "/nonexistent/dir"})
	err := cmd.Execute()
	// Should not error — just prints "no datasets" message
	if err != nil {
		t.Fatalf("list should handle missing dir gracefully: %v", err)
	}
}

func TestBenchReportCmd_Validation(t *testing.T) {
	cmd := NewBenchCmd()
	cmd.SetArgs([]string{"report"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing --results")
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
