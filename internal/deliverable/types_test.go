package deliverable

import (
	"os"
	"testing"
)

func TestNewBranchDeliverable(t *testing.T) {
	d := NewBranchDeliverable("step-1", "feat/my-branch", "/workspace/path", "Feature branch")

	if d.Type != TypeBranch {
		t.Errorf("expected Type=%q, got %q", TypeBranch, d.Type)
	}
	if d.Name != "feat/my-branch" {
		t.Errorf("expected Name=%q, got %q", "feat/my-branch", d.Name)
	}
	if d.Path != "/workspace/path" {
		t.Errorf("expected Path=%q, got %q", "/workspace/path", d.Path)
	}
	if d.Description != "Feature branch" {
		t.Errorf("expected Description=%q, got %q", "Feature branch", d.Description)
	}
	if d.StepID != "step-1" {
		t.Errorf("expected StepID=%q, got %q", "step-1", d.StepID)
	}
	if d.Metadata == nil {
		t.Fatal("expected Metadata to be non-nil")
	}
	pushed, ok := d.Metadata["pushed"].(bool)
	if !ok || pushed {
		t.Errorf("expected Metadata[\"pushed\"]=false, got %v", d.Metadata["pushed"])
	}
	if d.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestNewIssueDeliverable(t *testing.T) {
	d := NewIssueDeliverable("step-2", "Issue #42", "https://github.com/org/repo/issues/42", "Bug fix issue")

	if d.Type != TypeIssue {
		t.Errorf("expected Type=%q, got %q", TypeIssue, d.Type)
	}
	if d.Name != "Issue #42" {
		t.Errorf("expected Name=%q, got %q", "Issue #42", d.Name)
	}
	if d.Path != "https://github.com/org/repo/issues/42" {
		t.Errorf("expected Path to be issue URL, got %q", d.Path)
	}
	if d.Description != "Bug fix issue" {
		t.Errorf("expected Description=%q, got %q", "Bug fix issue", d.Description)
	}
	if d.StepID != "step-2" {
		t.Errorf("expected StepID=%q, got %q", "step-2", d.StepID)
	}
	if d.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestBranchDeliverableString(t *testing.T) {
	d := NewBranchDeliverable("step-1", "feat/branch", "/ws/path", "desc")

	// Test ASCII mode (default, unless nerd font env set)
	os.Unsetenv("NERD_FONT")
	result := d.String()
	if result == "" {
		t.Error("expected non-empty String() output")
	}
	// The output should contain the path
	if !contains(result, "/ws/path") {
		t.Errorf("expected String() to contain path, got %q", result)
	}
}

func TestIssueDeliverableString(t *testing.T) {
	d := NewIssueDeliverable("step-2", "Issue", "https://github.com/org/repo/issues/1", "desc")

	os.Unsetenv("NERD_FONT")
	result := d.String()
	if result == "" {
		t.Error("expected non-empty String() output")
	}
	if !contains(result, "https://github.com/org/repo/issues/1") {
		t.Errorf("expected String() to contain URL, got %q", result)
	}
}

func TestTrackerAddBranch(t *testing.T) {
	tracker := NewTracker("test-pipeline")
	tracker.AddBranch("step-1", "feat/test", "/ws/test", "Test branch")

	branches := tracker.GetByType(TypeBranch)
	if len(branches) != 1 {
		t.Fatalf("expected 1 branch deliverable, got %d", len(branches))
	}
	if branches[0].Name != "feat/test" {
		t.Errorf("expected branch name %q, got %q", "feat/test", branches[0].Name)
	}
}

func TestTrackerAddIssue(t *testing.T) {
	tracker := NewTracker("test-pipeline")
	tracker.AddIssue("step-1", "Issue #1", "https://github.com/org/repo/issues/1", "Test issue")

	issues := tracker.GetByType(TypeIssue)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue deliverable, got %d", len(issues))
	}
	if issues[0].Name != "Issue #1" {
		t.Errorf("expected issue name %q, got %q", "Issue #1", issues[0].Name)
	}
}

func TestTrackerUpdateMetadata(t *testing.T) {
	tracker := NewTracker("test-pipeline")
	tracker.AddBranch("step-1", "feat/push-test", "/ws/test", "Test branch")

	// Update push status
	tracker.UpdateMetadata(TypeBranch, "feat/push-test", "pushed", true)
	tracker.UpdateMetadata(TypeBranch, "feat/push-test", "remote_ref", "origin/feat/push-test")

	branches := tracker.GetByType(TypeBranch)
	if len(branches) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(branches))
	}

	pushed, ok := branches[0].Metadata["pushed"].(bool)
	if !ok || !pushed {
		t.Errorf("expected pushed=true, got %v", branches[0].Metadata["pushed"])
	}

	ref, ok := branches[0].Metadata["remote_ref"].(string)
	if !ok || ref != "origin/feat/push-test" {
		t.Errorf("expected remote_ref=%q, got %v", "origin/feat/push-test", branches[0].Metadata["remote_ref"])
	}
}

func TestTrackerUpdateMetadataNonExistent(t *testing.T) {
	tracker := NewTracker("test-pipeline")
	tracker.AddBranch("step-1", "feat/existing", "/ws/test", "Test")

	// Update a non-existent deliverable — should be a no-op
	tracker.UpdateMetadata(TypeBranch, "feat/nonexistent", "pushed", true)

	// Original should be unchanged
	branches := tracker.GetByType(TypeBranch)
	if len(branches) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(branches))
	}
	pushed, ok := branches[0].Metadata["pushed"].(bool)
	if !ok || pushed {
		t.Errorf("expected pushed=false (unchanged), got %v", branches[0].Metadata["pushed"])
	}
}

func TestTrackerUpdateMetadataNilMetadata(t *testing.T) {
	tracker := NewTracker("test-pipeline")
	// Manually add a deliverable with nil metadata
	tracker.Add(&Deliverable{
		Type:   TypeBranch,
		Name:   "feat/nil-meta",
		Path:   "/ws/test",
		StepID: "step-1",
	})

	// Should not panic and should create the metadata map
	tracker.UpdateMetadata(TypeBranch, "feat/nil-meta", "pushed", true)

	branches := tracker.GetByType(TypeBranch)
	if len(branches) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(branches))
	}
	if branches[0].Metadata == nil {
		t.Fatal("expected Metadata to be initialized")
	}
	pushed, ok := branches[0].Metadata["pushed"].(bool)
	if !ok || !pushed {
		t.Errorf("expected pushed=true, got %v", branches[0].Metadata["pushed"])
	}
}

func TestGetByTypeBranchAndIssue(t *testing.T) {
	tracker := NewTracker("test-pipeline")
	tracker.AddBranch("step-1", "feat/a", "/ws/a", "Branch A")
	tracker.AddBranch("step-2", "feat/b", "/ws/b", "Branch B")
	tracker.AddIssue("step-1", "Issue #1", "https://github.com/org/repo/issues/1", "Issue 1")
	tracker.AddPR("step-1", "PR #10", "https://github.com/org/repo/pulls/10", "PR 10")

	branches := tracker.GetByType(TypeBranch)
	if len(branches) != 2 {
		t.Errorf("expected 2 branches, got %d", len(branches))
	}

	issues := tracker.GetByType(TypeIssue)
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}

	prs := tracker.GetByType(TypePR)
	if len(prs) != 1 {
		t.Errorf("expected 1 PR, got %d", len(prs))
	}

	all := tracker.GetAll()
	if len(all) != 4 {
		t.Errorf("expected 4 total deliverables, got %d", len(all))
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
