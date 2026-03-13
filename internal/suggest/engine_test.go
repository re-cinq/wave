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
	// Should prefer unified (bare) pipeline over forge-prefixed
	if proposal.Pipelines[0].Name != "debug" {
		t.Errorf("expected unified pipeline 'debug', got %q", proposal.Pipelines[0].Name)
	}
}

func TestSuggest_PrefixedFallback(t *testing.T) {
	// When only prefixed pipeline exists, it should be used
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
	// Should fall back to gh-debug when bare debug doesn't exist
	if proposal.Pipelines[0].Name != "gh-debug" {
		t.Errorf("expected fallback to 'gh-debug', got %q", proposal.Pipelines[0].Name)
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

func TestSuggest_SequenceChain(t *testing.T) {
	dir := setupPipelineDir(t, []string{"research", "implement"})

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
	if seqProposal.Sequence[0] != "research" || seqProposal.Sequence[1] != "implement" {
		t.Errorf("expected [research, implement], got %v", seqProposal.Sequence)
	}
}

func TestSuggest_ParallelGroup(t *testing.T) {
	dir := setupPipelineDir(t, []string{"implement", "pr-review"})

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
		{"gh-implement", "implement"},
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
	catalog := []string{"gh-debug", "debug", "improve"}

	tests := []struct {
		prefix string
		base   string
		want   string
	}{
		{"gh", "debug", "debug"},       // bare name preferred over prefixed
		{"gl", "debug", "debug"},       // bare name found, no gl-debug needed
		{"gh", "improve", "improve"},   // bare name only
		{"gh", "nonexistent", ""},      // neither exists
		{"", "debug", "debug"},         // no prefix, bare name found
	}

	for _, tt := range tests {
		got := resolvePipeline(catalog, tt.prefix, tt.base)
		if got != tt.want {
			t.Errorf("resolvePipeline(%q, %q) = %q, want %q", tt.prefix, tt.base, got, tt.want)
		}
	}
}

func TestResolvePipeline_FallbackToPrefixed(t *testing.T) {
	// When bare name doesn't exist, fall back to forge-prefixed
	catalog := []string{"gh-debug"}

	got := resolvePipeline(catalog, "gh", "debug")
	if got != "gh-debug" {
		t.Errorf("resolvePipeline should fall back to prefixed, got %q", got)
	}
}
