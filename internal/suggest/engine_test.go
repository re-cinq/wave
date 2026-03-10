package suggest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/doctor"
	"github.com/recinq/wave/internal/forge"
)

func setupPipelineDir(t *testing.T, names []string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "pipelines")
	os.MkdirAll(dir, 0755)
	for _, name := range names {
		os.WriteFile(filepath.Join(dir, name+".yaml"), []byte("kind: Pipeline\nmetadata:\n  name: "+name+"\nsteps:\n  - id: s1\n    persona: nav\n"), 0644)
	}
	return dir
}

func TestSuggest_CIFailing(t *testing.T) {
	dir := setupPipelineDir(t, []string{"debug", "improve"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Report: &doctor.Report{
			Codebase: &doctor.CodebaseHealth{
				CI: doctor.CIStatus{Status: "failing", Failures: 2},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(proposal.Pipelines) == 0 {
		t.Fatal("expected at least one proposal")
	}
	if proposal.Pipelines[0].Name != "debug" {
		t.Errorf("expected first proposal to be 'debug', got %q", proposal.Pipelines[0].Name)
	}
	if proposal.Pipelines[0].Priority != 1 {
		t.Errorf("expected priority 1, got %d", proposal.Pipelines[0].Priority)
	}
}

func TestSuggest_PrefixedPipelines(t *testing.T) {
	dir := setupPipelineDir(t, []string{"gh-debug", "gh-implement", "debug"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Report: &doctor.Report{
			ForgeInfo: &forge.ForgeInfo{
				Type:           forge.ForgeGitHub,
				PipelinePrefix: "gh",
			},
			Codebase: &doctor.CodebaseHealth{
				CI: doctor.CIStatus{Status: "failing", Failures: 1},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(proposal.Pipelines) == 0 {
		t.Fatal("expected at least one proposal")
	}
	// Should prefer gh-debug over debug
	if proposal.Pipelines[0].Name != "gh-debug" {
		t.Errorf("expected prefixed pipeline 'gh-debug', got %q", proposal.Pipelines[0].Name)
	}
}

func TestSuggest_CleanState(t *testing.T) {
	dir := setupPipelineDir(t, []string{"improve", "refactor"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Report: &doctor.Report{
			Codebase: &doctor.CodebaseHealth{
				CI: doctor.CIStatus{Status: "passing"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(proposal.Pipelines) != 2 {
		t.Fatalf("expected 2 proposals, got %d", len(proposal.Pipelines))
	}

	names := make(map[string]bool)
	for _, p := range proposal.Pipelines {
		names[p.Name] = true
	}
	if !names["improve"] {
		t.Error("expected 'improve' in proposals")
	}
	if !names["refactor"] {
		t.Error("expected 'refactor' in proposals")
	}
}

func TestSuggest_NilCodebase(t *testing.T) {
	dir := setupPipelineDir(t, []string{"improve"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Report:       &doctor.Report{},
	})
	if err != nil {
		t.Fatal(err)
	}

	// With nil codebase, should fall through to clean-state proposals
	if len(proposal.Pipelines) == 0 {
		t.Fatal("expected fallback proposals")
	}
}

func TestSuggest_NilReport(t *testing.T) {
	_, err := Suggest(EngineOptions{})
	if err == nil {
		t.Error("expected error with nil report")
	}
}

func TestSuggest_Limit(t *testing.T) {
	dir := setupPipelineDir(t, []string{"debug", "implement", "pr-review", "improve", "refactor"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Limit:        2,
		Report: &doctor.Report{
			Codebase: &doctor.CodebaseHealth{
				CI:     doctor.CIStatus{Status: "failing", Failures: 1},
				Issues: doctor.IssueSummary{Open: 5},
				PRs:    doctor.PRSummary{NeedsReview: 3},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(proposal.Pipelines) > 2 {
		t.Errorf("expected at most 2 proposals, got %d", len(proposal.Pipelines))
	}
}

func TestSuggest_PoorQualityIssues(t *testing.T) {
	dir := setupPipelineDir(t, []string{"rewrite"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Report: &doctor.Report{
			Codebase: &doctor.CodebaseHealth{
				Issues: doctor.IssueSummary{PoorQuality: 3},
				CI:     doctor.CIStatus{Status: "passing"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(proposal.Pipelines) == 0 {
		t.Fatal("expected rewrite proposal")
	}
	if proposal.Pipelines[0].Name != "rewrite" {
		t.Errorf("expected 'rewrite', got %q", proposal.Pipelines[0].Name)
	}
}

func TestSuggest_StalePRs(t *testing.T) {
	dir := setupPipelineDir(t, []string{"refresh"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Report: &doctor.Report{
			Codebase: &doctor.CodebaseHealth{
				PRs: doctor.PRSummary{Stale: 2},
				CI:  doctor.CIStatus{Status: "passing"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(proposal.Pipelines) == 0 {
		t.Fatal("expected refresh proposal")
	}
	if proposal.Pipelines[0].Name != "refresh" {
		t.Errorf("expected 'refresh', got %q", proposal.Pipelines[0].Name)
	}
}

func TestSuggest_NoPipelinesAvailable(t *testing.T) {
	dir := setupPipelineDir(t, []string{})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Report: &doctor.Report{
			Codebase: &doctor.CodebaseHealth{
				CI: doctor.CIStatus{Status: "failing", Failures: 1},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(proposal.Pipelines) != 0 {
		t.Errorf("expected 0 proposals when no pipelines exist, got %d", len(proposal.Pipelines))
	}
}

func TestResolvePipeline(t *testing.T) {
	catalog := []string{"gh-debug", "debug", "improve"}

	tests := []struct {
		prefix string
		base   string
		want   string
	}{
		{"gh", "debug", "gh-debug"},
		{"gl", "debug", "debug"},
		{"gh", "improve", "improve"},
		{"gh", "nonexistent", ""},
		{"", "debug", "debug"},
	}

	for _, tt := range tests {
		got := resolvePipeline(catalog, tt.prefix, tt.base)
		if got != tt.want {
			t.Errorf("resolvePipeline(%q, %q) = %q, want %q", tt.prefix, tt.base, got, tt.want)
		}
	}
}
