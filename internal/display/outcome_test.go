package display

import (
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/deliverable"
	"github.com/recinq/wave/internal/event"
)

func TestBuildOutcome_BranchAndPR(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")
	tracker.AddBranch("step-1", "feat/my-feature", "/ws/test", "Feature branch")
	tracker.UpdateMetadata(deliverable.TypeBranch, "feat/my-feature", "pushed", true)
	tracker.UpdateMetadata(deliverable.TypeBranch, "feat/my-feature", "remote_ref", "origin/feat/my-feature")
	tracker.AddPR("step-2", "Pull Request", "https://github.com/org/repo/pull/42", "New PR")
	tracker.AddFile("step-1", "output.json", "/ws/test/output.json", "Output")
	tracker.AddFile("step-1", "result.md", "/ws/test/result.md", "Result")

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 30*time.Second, 5000, "/ws/test", nil)

	if outcome.Branch != "feat/my-feature" {
		t.Errorf("expected Branch=%q, got %q", "feat/my-feature", outcome.Branch)
	}
	if !outcome.Pushed {
		t.Error("expected Pushed=true")
	}
	if outcome.RemoteRef != "origin/feat/my-feature" {
		t.Errorf("expected RemoteRef=%q, got %q", "origin/feat/my-feature", outcome.RemoteRef)
	}
	if len(outcome.PullRequests) != 1 {
		t.Fatalf("expected 1 PR, got %d", len(outcome.PullRequests))
	}
	if outcome.PullRequests[0].URL != "https://github.com/org/repo/pull/42" {
		t.Errorf("expected PR URL, got %q", outcome.PullRequests[0].URL)
	}
	if outcome.ArtifactCount != 2 {
		t.Errorf("expected ArtifactCount=2, got %d", outcome.ArtifactCount)
	}
	if !outcome.Success {
		t.Error("expected Success=true")
	}
}

func TestBuildOutcome_Empty(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 10*time.Second, 1000, "", nil)

	if outcome.Branch != "" {
		t.Errorf("expected empty Branch, got %q", outcome.Branch)
	}
	if len(outcome.PullRequests) != 0 {
		t.Errorf("expected 0 PRs, got %d", len(outcome.PullRequests))
	}
	if len(outcome.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(outcome.Issues))
	}
	if outcome.ArtifactCount != 0 {
		t.Errorf("expected ArtifactCount=0, got %d", outcome.ArtifactCount)
	}
}

func TestBuildOutcome_OnlyArtifacts(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")
	for i := 0; i < 10; i++ {
		tracker.AddFile("step-1", "file", "/ws/test/file", "desc")
	}

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 5*time.Second, 500, "", nil)

	if outcome.Branch != "" {
		t.Errorf("expected no branch, got %q", outcome.Branch)
	}
	if len(outcome.PullRequests) != 0 {
		t.Errorf("expected 0 PRs, got %d", len(outcome.PullRequests))
	}
	// Note: dedup in tracker means only 1 unique file since path+stepID match
	if outcome.ArtifactCount < 1 {
		t.Errorf("expected at least 1 artifact, got %d", outcome.ArtifactCount)
	}
}

func TestBuildOutcome_PushFailure(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")
	tracker.AddBranch("step-1", "feat/push-fail", "/ws/test", "Feature branch")
	tracker.UpdateMetadata(deliverable.TypeBranch, "feat/push-fail", "push_error", "authentication failed")

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 5*time.Second, 500, "", nil)

	if outcome.PushError != "authentication failed" {
		t.Errorf("expected PushError=%q, got %q", "authentication failed", outcome.PushError)
	}
}

func TestBuildOutcome_LargeDeliverableCount(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")
	tracker.AddBranch("step-1", "feat/large", "/ws/test", "Branch")
	tracker.AddPR("step-2", "PR", "https://github.com/org/repo/pull/1", "PR")
	// Add 50+ unique files
	for i := 0; i < 55; i++ {
		tracker.AddFile("step-1", "file", "/ws/test/file"+string(rune('A'+i%26))+string(rune('0'+i/26)), "desc")
	}

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 60*time.Second, 10000, "", nil)

	if outcome.ArtifactCount < 50 {
		t.Errorf("expected at least 50 artifacts, got %d", outcome.ArtifactCount)
	}
}

func TestBuildOutcome_NilTracker(t *testing.T) {
	outcome := BuildOutcome(nil, "test-pipeline", "run-123", true, 5*time.Second, 500, "", nil)

	if outcome == nil {
		t.Fatal("expected non-nil outcome even with nil tracker")
	}
	if outcome.PipelineName != "test-pipeline" {
		t.Errorf("expected PipelineName=%q, got %q", "test-pipeline", outcome.PipelineName)
	}
}

func TestBuildOutcome_WithIssues(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")
	tracker.AddIssue("step-1", "Bug Fix", "https://github.com/org/repo/issues/10", "Bug")

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 5*time.Second, 500, "", nil)

	if len(outcome.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(outcome.Issues))
	}
	if outcome.Issues[0].Label != "Bug Fix" {
		t.Errorf("expected issue label %q, got %q", "Bug Fix", outcome.Issues[0].Label)
	}
}

func TestGenerateNextSteps_PRExists(t *testing.T) {
	outcome := &PipelineOutcome{
		PullRequests: []OutcomeLink{{Label: "Pull Request", URL: "https://github.com/org/repo/pull/42"}},
	}

	steps := GenerateNextSteps(outcome)

	if len(steps) == 0 {
		t.Fatal("expected at least one next step")
	}

	found := false
	for _, s := range steps {
		if strings.Contains(s.Label, "Review the pull request") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected next step about reviewing PR")
	}
}

func TestGenerateNextSteps_BranchPushed(t *testing.T) {
	outcome := &PipelineOutcome{
		Branch:    "feat/test",
		Pushed:    true,
		RemoteRef: "origin/feat/test",
	}

	steps := GenerateNextSteps(outcome)

	found := false
	for _, s := range steps {
		if strings.Contains(s.Label, "View changes on remote") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected next step about viewing remote changes")
	}
}

func TestGenerateNextSteps_WorkspacePath(t *testing.T) {
	outcome := &PipelineOutcome{
		WorkspacePath: "/ws/test/workspace",
	}

	steps := GenerateNextSteps(outcome)

	found := false
	for _, s := range steps {
		if strings.Contains(s.Label, "Inspect workspace") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected next step about inspecting workspace")
	}
}

func TestGenerateNextSteps_NoOutcomes(t *testing.T) {
	outcome := &PipelineOutcome{}

	steps := GenerateNextSteps(outcome)

	if len(steps) != 0 {
		t.Errorf("expected 0 next steps for empty outcome, got %d", len(steps))
	}
}

func TestRenderOutcomeSummary_DefaultMode(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")
	tracker.AddBranch("step-1", "feat/render-test", "/ws/test", "Branch")
	tracker.AddPR("step-2", "Pull Request", "https://github.com/org/repo/pull/5", "PR")
	tracker.AddFile("step-1", "output.json", "/ws/test/output.json", "Output")

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 30*time.Second, 5000, "", nil)

	// Use ANSI-disabled formatter for predictable output
	formatter := NewFormatterWithConfig("off", true)
	result := RenderOutcomeSummary(outcome, false, formatter)

	if !strings.Contains(result, "Outcomes") {
		t.Error("expected 'Outcomes' header in output")
	}
	if !strings.Contains(result, "feat/render-test") {
		t.Error("expected branch name in output")
	}
	if !strings.Contains(result, "https://github.com/org/repo/pull/5") {
		t.Error("expected PR URL in output")
	}
	if !strings.Contains(result, "1 artifacts produced") {
		t.Errorf("expected artifact count in output, got:\n%s", result)
	}
}

func TestRenderOutcomeSummary_VerboseMode(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")
	tracker.AddBranch("step-1", "feat/verbose", "/ws/test", "Branch")
	tracker.AddFile("step-1", "output.json", "/ws/test/output.json", "Output")
	tracker.AddFile("step-1", "log.txt", "/ws/test/log.txt", "Log")

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 30*time.Second, 5000, "", nil)

	formatter := NewFormatterWithConfig("off", true)
	result := RenderOutcomeSummary(outcome, true, formatter)

	if !strings.Contains(result, "artifacts produced") {
		t.Error("expected artifacts section in verbose output")
	}
	if !strings.Contains(result, "output.json") {
		t.Error("expected artifact paths listed in verbose output")
	}
}

func TestRenderOutcomeSummary_EmptyOutcome(t *testing.T) {
	outcome := &PipelineOutcome{
		PipelineName: "test",
		RunID:        "run-1",
		Success:      true,
	}

	formatter := NewFormatterWithConfig("off", true)
	result := RenderOutcomeSummary(outcome, false, formatter)

	// FR-008: empty outcome categories should not appear
	if strings.Contains(result, "Outcomes") {
		t.Error("expected no 'Outcomes' header when no outcomes exist")
	}
	if strings.Contains(result, "None") {
		t.Error("expected no 'None' text in output")
	}
}

func TestRenderOutcomeSummary_Nil(t *testing.T) {
	result := RenderOutcomeSummary(nil, false, NewFormatterWithConfig("off", true))
	if result != "" {
		t.Errorf("expected empty string for nil outcome, got %q", result)
	}
}

func TestRenderOutcomeSummary_PushError(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")
	tracker.AddBranch("step-1", "feat/push-err", "/ws/test", "Branch")
	tracker.UpdateMetadata(deliverable.TypeBranch, "feat/push-err", "push_error", "auth failed")

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 5*time.Second, 500, "", nil)

	formatter := NewFormatterWithConfig("off", true)
	result := RenderOutcomeSummary(outcome, false, formatter)

	if !strings.Contains(result, "push failed") {
		t.Errorf("expected push failure warning in output, got:\n%s", result)
	}
}

func TestRenderOutcomeSummary_NonTTYNoANSI(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")
	tracker.AddBranch("step-1", "feat/no-ansi", "/ws/test", "Branch")
	tracker.AddPR("step-2", "PR", "https://github.com/org/repo/pull/1", "PR")

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 5*time.Second, 500, "", nil)

	// Create a formatter with ANSI disabled
	formatter := NewFormatterWithConfig("off", true)
	result := RenderOutcomeSummary(outcome, false, formatter)

	// SC-006: output should have no ANSI escape codes
	if strings.Contains(result, "\033[") {
		t.Errorf("expected no ANSI escape codes in non-TTY output, got:\n%q", result)
	}
}

func TestRenderOutcomeSummary_NextSteps(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")
	tracker.AddPR("step-1", "Pull Request", "https://github.com/org/repo/pull/5", "PR")

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 5*time.Second, 500, "/ws/test", nil)

	formatter := NewFormatterWithConfig("off", true)
	result := RenderOutcomeSummary(outcome, false, formatter)

	if !strings.Contains(result, "Next Steps") {
		t.Errorf("expected 'Next Steps' section in output, got:\n%s", result)
	}
	if !strings.Contains(result, "Review the pull request") {
		t.Errorf("expected PR review suggestion, got:\n%s", result)
	}
}

func TestToOutcomesJSON(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")
	tracker.AddBranch("step-1", "feat/json-test", "/ws/test", "Branch")
	tracker.UpdateMetadata(deliverable.TypeBranch, "feat/json-test", "pushed", true)
	tracker.AddPR("step-2", "Pull Request", "https://github.com/org/repo/pull/1", "PR")
	tracker.AddIssue("step-3", "Issue #5", "https://github.com/org/repo/issues/5", "Issue")
	tracker.AddFile("step-1", "output", "/ws/test/output.json", "Output file")

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 30*time.Second, 5000, "", nil)

	outJSON := outcome.ToOutcomesJSON()

	if outJSON == nil {
		t.Fatal("expected non-nil OutcomesJSON")
	}
	if outJSON.Branch != "feat/json-test" {
		t.Errorf("expected Branch=%q, got %q", "feat/json-test", outJSON.Branch)
	}
	if !outJSON.Pushed {
		t.Error("expected Pushed=true")
	}
	if len(outJSON.PullRequests) != 1 {
		t.Errorf("expected 1 PR, got %d", len(outJSON.PullRequests))
	}
	if len(outJSON.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(outJSON.Issues))
	}
	if len(outJSON.Deliverables) < 3 {
		t.Errorf("expected at least 3 deliverables, got %d", len(outJSON.Deliverables))
	}
}

func TestToOutcomesJSON_Nil(t *testing.T) {
	var outcome *PipelineOutcome
	outJSON := outcome.ToOutcomesJSON()
	if outJSON != nil {
		t.Error("expected nil OutcomesJSON for nil PipelineOutcome")
	}
}

func TestToOutcomesJSON_EmptySlices(t *testing.T) {
	outcome := &PipelineOutcome{
		PipelineName: "test",
	}
	outJSON := outcome.ToOutcomesJSON()

	// Empty arrays, not nil
	if outJSON.PullRequests == nil {
		t.Error("expected non-nil PullRequests slice")
	}
	if outJSON.Issues == nil {
		t.Error("expected non-nil Issues slice")
	}
	if outJSON.Deployments == nil {
		t.Error("expected non-nil Deployments slice")
	}
	if outJSON.Deliverables == nil {
		t.Error("expected non-nil Deliverables slice")
	}
}

func TestFilterArtifacts_DeduplicatesByPath(t *testing.T) {
	tracker := deliverable.NewTracker("test-pipeline")
	// Simulate 4 steps in a shared worktree all producing artifact.json
	tracker.AddFile("step-1", "issue_analysis", "/ws/shared/artifact.json", "json")
	tracker.AddFile("step-2", "enhancement_plan", "/ws/shared/artifact.json", "json")
	tracker.AddFile("step-3", "enhancement_results", "/ws/shared/artifact.json", "json")
	tracker.AddFile("step-4", "verification_report", "/ws/shared/artifact.json", "json")

	outcome := BuildOutcome(tracker, "test-pipeline", "run-123", true, 30*time.Second, 5000, "", nil)

	if outcome.ArtifactCount != 1 {
		t.Errorf("expected 1 deduplicated artifact, got %d", outcome.ArtifactCount)
	}

	formatter := NewFormatterWithConfig("off", true)
	result := RenderOutcomeSummary(outcome, false, formatter)

	if strings.Contains(result, "4 artifacts produced") {
		t.Errorf("expected deduplicated artifact count, got:\n%s", result)
	}
	// Should show exactly 1
	if !strings.Contains(result, "1 artifacts produced") {
		t.Errorf("expected '1 artifacts produced' in output, got:\n%s", result)
	}
}

// Verify the Event struct has the Outcomes field (compilation test)
func TestEventOutcomesField(t *testing.T) {
	e := event.Event{
		State: "completed",
		Outcomes: &event.OutcomesJSON{
			Branch: "feat/test",
			Pushed: true,
		},
	}
	if e.Outcomes == nil {
		t.Error("expected non-nil Outcomes")
	}
	if e.Outcomes.Branch != "feat/test" {
		t.Errorf("expected Branch=%q, got %q", "feat/test", e.Outcomes.Branch)
	}
}
