package suggest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/doctor"
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

func TestSuggest_OpenIssues(t *testing.T) {
	dir := setupPipelineDir(t, []string{"impl-issue"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Report: &doctor.Report{
			Codebase: &doctor.CodebaseHealth{
				Issues: doctor.IssueSummary{Open: 3},
				CI:     doctor.CIStatus{Status: "passing"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(proposal.Pipelines) == 0 {
		t.Fatal("expected at least one proposal")
	}
	if proposal.Pipelines[0].Name != "impl-issue" {
		t.Errorf("expected first proposal to be 'impl-issue', got %q", proposal.Pipelines[0].Name)
	}
	if proposal.Pipelines[0].Priority != 3 {
		t.Errorf("expected priority 3, got %d", proposal.Pipelines[0].Priority)
	}
}

func TestSuggest_PRsNeedingReview(t *testing.T) {
	dir := setupPipelineDir(t, []string{"ops-pr-review"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Report: &doctor.Report{
			Codebase: &doctor.CodebaseHealth{
				PRs: doctor.PRSummary{NeedsReview: 2},
				CI:  doctor.CIStatus{Status: "passing"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(proposal.Pipelines) == 0 {
		t.Fatal("expected at least one proposal")
	}
	if proposal.Pipelines[0].Name != "ops-pr-review" {
		t.Errorf("expected 'ops-pr-review', got %q", proposal.Pipelines[0].Name)
	}
}

func TestSuggest_NilCodebase(t *testing.T) {
	dir := setupPipelineDir(t, []string{"impl-issue"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Report:       &doctor.Report{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(proposal.Pipelines) != 0 {
		t.Errorf("expected 0 proposals with nil codebase, got %d", len(proposal.Pipelines))
	}
}

func TestSuggest_NilReport(t *testing.T) {
	_, err := Suggest(EngineOptions{})
	if err == nil {
		t.Error("expected error with nil report")
	}
}

func TestSuggest_Limit(t *testing.T) {
	dir := setupPipelineDir(t, []string{"impl-issue", "plan-research", "ops-pr-review"})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Limit:        2,
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

	if len(proposal.Pipelines) > 2 {
		t.Errorf("expected at most 2 proposals, got %d", len(proposal.Pipelines))
	}
}

func TestSuggest_NoPipelinesAvailable(t *testing.T) {
	dir := setupPipelineDir(t, []string{})

	proposal, err := Suggest(EngineOptions{
		PipelinesDir: dir,
		Report: &doctor.Report{
			Codebase: &doctor.CodebaseHealth{
				Issues: doctor.IssueSummary{Open: 1},
				CI:     doctor.CIStatus{Status: "passing"},
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
	catalog := []string{"ops-pr-review", "impl-issue", "plan-research"}

	tests := []struct {
		prefix string
		base   string
		want   string
	}{
		{"", "impl-issue", "impl-issue"},       // exact bare name
		{"", "plan-research", "plan-research"}, // exact bare name
		{"", "ops-pr-review", "ops-pr-review"}, // exact bare name
		{"", "nonexistent", ""},                // not in catalog
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
	catalog := []string{"impl-issue"}

	got := resolvePipeline(catalog, "", "impl-issue")
	if got != "impl-issue" {
		t.Errorf("resolvePipeline should match bare name, got %q", got)
	}
}

func TestResolvePipeline_FallbackToForgePrefixed(t *testing.T) {
	// When only forge-prefixed exists, fall back to it
	catalog := []string{"gh-impl-issue"}

	got := resolvePipeline(catalog, "gh", "impl-issue")
	if got != "gh-impl-issue" {
		t.Errorf("resolvePipeline should fall back to forge-prefixed, got %q", got)
	}
}
