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
	_ = os.MkdirAll(dir, 0755)
	for _, name := range names {
		_ = os.WriteFile(filepath.Join(dir, name+".yaml"), []byte("kind: Pipeline\nmetadata:\n  name: "+name+"\nsteps:\n  - id: s1\n    persona: nav\n"), 0644)
	}
	return dir
}

func TestSuggest_CIFailing(t *testing.T) {
	dir := setupPipelineDir(t, []string{"ops-debug", "impl-improve"})

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
	if proposal.Pipelines[0].Name != "ops-debug" {
		t.Errorf("expected first proposal to be 'ops-debug', got %q", proposal.Pipelines[0].Name)
	}
	if proposal.Pipelines[0].Priority != 1 {
		t.Errorf("expected priority 1, got %d", proposal.Pipelines[0].Priority)
	}
}

func TestSuggest_CIFailing_BareNameFallback(t *testing.T) {
	// Old-style bare name should still work
	dir := setupPipelineDir(t, []string{"debug"})

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

	if len(proposal.Pipelines) == 0 {
		t.Fatal("expected at least one proposal")
	}
	if proposal.Pipelines[0].Name != "debug" {
		t.Errorf("expected 'debug' (bare fallback), got %q", proposal.Pipelines[0].Name)
	}
}

func TestSuggest_PrefixedPipelines(t *testing.T) {
	dir := setupPipelineDir(t, []string{"gh-debug", "gh-implement", "ops-debug"})

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
	// Should prefer taxonomy-prefixed over forge-prefixed
	if proposal.Pipelines[0].Name != "ops-debug" {
		t.Errorf("expected taxonomy-prefixed 'ops-debug', got %q", proposal.Pipelines[0].Name)
	}
}

func TestSuggest_PrefixedFallback(t *testing.T) {
	// When only forge-prefixed pipeline exists, it should be used
	dir := setupPipelineDir(t, []string{"gh-debug"})

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
	// Should fall back to gh-debug when bare and taxonomy-prefixed don't exist
	if proposal.Pipelines[0].Name != "gh-debug" {
		t.Errorf("expected fallback to 'gh-debug', got %q", proposal.Pipelines[0].Name)
	}
}

func TestSuggest_CleanState(t *testing.T) {
	dir := setupPipelineDir(t, []string{"impl-improve", "impl-refactor"})

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
	if !names["impl-improve"] {
		t.Error("expected 'impl-improve' in proposals")
	}
	if !names["impl-refactor"] {
		t.Error("expected 'impl-refactor' in proposals")
	}
}

func TestSuggest_NilCodebase(t *testing.T) {
	dir := setupPipelineDir(t, []string{"impl-improve"})

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
	dir := setupPipelineDir(t, []string{"ops-debug", "impl-issue", "ops-pr-review", "impl-improve", "impl-refactor"})

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
	dir := setupPipelineDir(t, []string{"ops-rewrite"})

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
	if proposal.Pipelines[0].Name != "ops-rewrite" {
		t.Errorf("expected 'ops-rewrite', got %q", proposal.Pipelines[0].Name)
	}
}

func TestSuggest_StalePRs(t *testing.T) {
	dir := setupPipelineDir(t, []string{"ops-refresh"})

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
	if proposal.Pipelines[0].Name != "ops-refresh" {
		t.Errorf("expected 'ops-refresh', got %q", proposal.Pipelines[0].Name)
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

func TestSuggest_SequenceChain(t *testing.T) {
	dir := setupPipelineDir(t, []string{"plan-research", "impl-issue"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Limit:        10,
		Report: &doctor.Report{
			Codebase: &doctor.CodebaseHealth{
				Issues: doctor.IssueSummary{Open: 5},
				CI:     doctor.CIStatus{Status: "passing"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should have at least the sequence proposal
	var seqProposal *ProposedPipeline
	for i, p := range proposal.Pipelines {
		if p.Type == "sequence" {
			seqProposal = &proposal.Pipelines[i]
			break
		}
	}

	if seqProposal == nil {
		t.Fatal("expected a sequence proposal")
	}
	if len(seqProposal.Sequence) != 2 {
		t.Errorf("expected sequence of 2, got %d", len(seqProposal.Sequence))
	}
	if seqProposal.Sequence[0] != "plan-research" || seqProposal.Sequence[1] != "impl-issue" {
		t.Errorf("expected [plan-research, impl-issue], got %v", seqProposal.Sequence)
	}
}

func TestSuggest_ParallelGroup(t *testing.T) {
	dir := setupPipelineDir(t, []string{"impl-issue", "ops-pr-review"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Limit:        10,
		Report: &doctor.Report{
			Codebase: &doctor.CodebaseHealth{
				Issues: doctor.IssueSummary{Open: 5},
				PRs:    doctor.PRSummary{NeedsReview: 3},
				CI:     doctor.CIStatus{Status: "passing"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var parProposal *ProposedPipeline
	for i, p := range proposal.Pipelines {
		if p.Type == "parallel" {
			parProposal = &proposal.Pipelines[i]
			break
		}
	}

	if parProposal == nil {
		t.Fatal("expected a parallel proposal")
	}
	if len(parProposal.Sequence) != 2 {
		t.Errorf("expected parallel group of 2, got %d", len(parProposal.Sequence))
	}
}

func TestStripForgePrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"gh-implement", "implement"}, // stripForgePrefix strips gh- prefix
		{"gl-debug", "debug"},
		{"implement", "implement"},
		{"bb-review", "review"},
	}
	for _, tt := range tests {
		got := stripForgePrefix(tt.input)
		if got != tt.want {
			t.Errorf("stripForgePrefix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolvePipeline(t *testing.T) {
	catalog := []string{"gh-debug", "ops-debug", "impl-improve"}

	tests := []struct {
		prefix string
		base   string
		want   string
	}{
		{"gh", "debug", "ops-debug"},      // taxonomy-prefixed preferred over forge-prefixed
		{"gl", "debug", "ops-debug"},      // taxonomy-prefixed found, no gl-debug needed
		{"gh", "improve", "impl-improve"}, // taxonomy-prefixed resolved
		{"gh", "nonexistent", ""},         // neither exists
		{"", "debug", "ops-debug"},        // no forge prefix, taxonomy resolved
	}

	for _, tt := range tests {
		got := resolvePipeline(catalog, tt.prefix, tt.base)
		if got != tt.want {
			t.Errorf("resolvePipeline(%q, %q) = %q, want %q", tt.prefix, tt.base, got, tt.want)
		}
	}
}

func TestResolvePipeline_BareNamePreferred(t *testing.T) {
	// When bare name exists, prefer it over taxonomy-prefixed
	catalog := []string{"debug", "ops-debug"}

	got := resolvePipeline(catalog, "", "debug")
	if got != "debug" {
		t.Errorf("resolvePipeline should prefer bare name, got %q", got)
	}
}

func TestResolvePipeline_FallbackToForgePrefixed(t *testing.T) {
	// When only forge-prefixed exists, fall back to it
	catalog := []string{"gh-debug"}

	got := resolvePipeline(catalog, "gh", "debug")
	if got != "gh-debug" {
		t.Errorf("resolvePipeline should fall back to forge-prefixed, got %q", got)
	}
}
