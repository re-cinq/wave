package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/state"
)

func TestPrepareChatWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()
	completed := now.Add(5 * time.Minute)

	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:        "test-run-001",
			PipelineName: "test-pipeline",
			Status:       "completed",
			Input:        "test input",
			TotalTokens:  5000,
			StartedAt:    now,
			CompletedAt:  &completed,
		},
		Steps: []ChatStepContext{
			{
				StepID:     "analyze",
				Persona:    "navigator",
				State:      "completed",
				Duration:   2 * time.Minute,
				TokensUsed: 3000,
			},
			{
				StepID:        "implement",
				Persona:       "craftsman",
				State:         "completed",
				Duration:      3 * time.Minute,
				TokensUsed:    2000,
				WorkspacePath: "/tmp/fake-workspace",
			},
		},
		Artifacts: []state.ArtifactRecord{
			{StepID: "analyze", Name: "plan.json", Path: ".wave/output/plan.json", Type: "json", SizeBytes: 512},
		},
		ProjectRoot: tmpDir,
	}

	wsPath, err := PrepareChatWorkspace(ctx, ChatWorkspaceOptions{Model: "sonnet"})
	if err != nil {
		t.Fatalf("PrepareChatWorkspace failed: %v", err)
	}

	// Verify workspace directory was created
	expectedDir := filepath.Join(tmpDir, ".wave", "chat", "test-run-001")
	if wsPath != expectedDir {
		t.Errorf("expected workspace path %s, got %s", expectedDir, wsPath)
	}

	// Verify CLAUDE.md exists and contains expected content
	claudeMd, err := os.ReadFile(filepath.Join(wsPath, adapter.InstructionFilename("claude")))
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}
	claudeMdStr := string(claudeMd)

	// Check key sections
	if !strings.Contains(claudeMdStr, "# Wave Pipeline Analysis") {
		t.Error("CLAUDE.md missing header")
	}
	if !strings.Contains(claudeMdStr, "test-run-001") {
		t.Error("CLAUDE.md missing run ID")
	}
	if !strings.Contains(claudeMdStr, "test-pipeline") {
		t.Error("CLAUDE.md missing pipeline name")
	}
	if !strings.Contains(claudeMdStr, "completed") {
		t.Error("CLAUDE.md missing status")
	}
	if !strings.Contains(claudeMdStr, "analyze") {
		t.Error("CLAUDE.md missing step name")
	}
	if !strings.Contains(claudeMdStr, "navigator") {
		t.Error("CLAUDE.md missing persona name")
	}
	if !strings.Contains(claudeMdStr, "plan.json") {
		t.Error("CLAUDE.md missing artifact name")
	}

	// Verify settings.json exists
	settingsPath := filepath.Join(wsPath, ".claude", "settings.json")
	settingsData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(settingsData, &settings); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v", err)
	}

	if model, ok := settings["model"].(string); !ok || model != "sonnet" {
		t.Errorf("expected model 'sonnet', got %v", settings["model"])
	}

	// Verify permissions include read-only tools
	perms, ok := settings["permissions"].(map[string]interface{})
	if !ok {
		t.Fatal("settings.json missing permissions")
	}
	allow, ok := perms["allow"].([]interface{})
	if !ok {
		t.Fatal("settings.json missing allow list")
	}

	allowStrs := make([]string, len(allow))
	for i, a := range allow {
		allowStrs[i] = a.(string)
	}
	if !containsStr(allowStrs, "Read") {
		t.Error("allow list missing Read")
	}
	if !containsStr(allowStrs, "Glob") {
		t.Error("allow list missing Glob")
	}
	if !containsStr(allowStrs, "Grep") {
		t.Error("allow list missing Grep")
	}

	// Verify deny list
	deny, ok := perms["deny"].([]interface{})
	if !ok {
		t.Fatal("settings.json missing deny list")
	}
	denyStrs := make([]string, len(deny))
	for i, d := range deny {
		denyStrs[i] = d.(string)
	}
	if !containsStr(denyStrs, "Write") {
		t.Error("deny list missing Write")
	}
	if !containsStr(denyStrs, "Edit") {
		t.Error("deny list missing Edit")
	}
}

func TestPrepareChatWorkspace_DefaultModel(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()

	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:        "model-test",
			PipelineName: "test",
			Status:       "completed",
			StartedAt:    now,
		},
		ProjectRoot: tmpDir,
	}

	wsPath, err := PrepareChatWorkspace(ctx, ChatWorkspaceOptions{})
	if err != nil {
		t.Fatalf("PrepareChatWorkspace failed: %v", err)
	}

	settingsData, err := os.ReadFile(filepath.Join(wsPath, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}

	var settings map[string]interface{}
	_ = json.Unmarshal(settingsData, &settings)

	// When no model is specified, the model field should be omitted from settings.json
	// so the adapter uses its own default rather than a hardcoded model name.
	if model, ok := settings["model"]; ok && model != "" {
		t.Errorf("expected model field to be absent or empty when no model specified, got %v", model)
	}
}

func TestPrepareChatWorkspace_WithFailures(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()

	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:        "fail-test",
			PipelineName: "test",
			Status:       "failed",
			StartedAt:    now,
			ErrorMessage: "pipeline failed",
		},
		Steps: []ChatStepContext{
			{
				StepID:       "build",
				Persona:      "craftsman",
				State:        "failed",
				ErrorMessage: "compilation error: missing import",
			},
		},
		ProjectRoot: tmpDir,
	}

	wsPath, err := PrepareChatWorkspace(ctx, ChatWorkspaceOptions{})
	if err != nil {
		t.Fatalf("PrepareChatWorkspace failed: %v", err)
	}

	claudeMd, err := os.ReadFile(filepath.Join(wsPath, adapter.InstructionFilename("claude")))
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}

	if !strings.Contains(string(claudeMd), "Failures") {
		t.Error("CLAUDE.md missing Failures section for failed run")
	}
	if !strings.Contains(string(claudeMd), "compilation error: missing import") {
		t.Error("CLAUDE.md missing error message")
	}
}

func TestChatFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "-"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{5 * time.Minute, "5m0s"},
		{65 * time.Minute, "1h5m"},
		{2*time.Hour + 30*time.Minute, "2h30m"},
	}

	for _, tt := range tests {
		got := chatFormatDuration(tt.d)
		if got != tt.want {
			t.Errorf("chatFormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestChatFormatTokens(t *testing.T) {
	tests := []struct {
		tokens int
		want   string
	}{
		{0, "-"},
		{500, "500"},
		{1500, "1k"},
		{45000, "45k"},
		{1500000, "1.5M"},
	}

	for _, tt := range tests {
		got := chatFormatTokens(tt.tokens)
		if got != tt.want {
			t.Errorf("chatFormatTokens(%d) = %q, want %q", tt.tokens, got, tt.want)
		}
	}
}

func TestChatFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "-"},
		{512, "512 B"},
		{2048, "2.0 KB"},
		{1572864, "1.5 MB"},
	}

	for _, tt := range tests {
		got := chatFormatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("chatFormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func containsStr(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}

func TestBuildChatClaudeMd_WithArtifactContent(t *testing.T) {
	now := time.Now()
	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:        "art-test",
			PipelineName: "gh-implement",
			Status:       "completed",
			StartedAt:    now,
		},
		Steps: []ChatStepContext{
			{StepID: "implement", Persona: "craftsman", State: "completed"},
		},
		Pipeline: &Pipeline{
			Metadata: PipelineMetadata{Name: "gh-implement"},
		},
		Artifacts: []state.ArtifactRecord{
			{StepID: "implement", Name: "pr-result.json", Path: ".wave/output/pr-result.json"},
		},
		ArtifactContents: map[string]string{
			"pr-result.json": `{"pr_url":"https://github.com/test/repo/pull/1"}`,
		},
		ChatConfig: &ChatContextConfig{
			SuggestedQuestions: []string{"Would you like to review the changes?", "Are there failing tests?"},
			FocusAreas:         []string{"code changes", "test coverage"},
		},
		ProjectRoot: "/tmp/test",
	}

	md := buildChatClaudeMd(ctx, ChatModeAnalysis, "", "")

	// Check artifact content section
	if !strings.Contains(md, "## Key Artifact Content") {
		t.Error("missing Key Artifact Content section")
	}
	if !strings.Contains(md, "pr-result.json") {
		t.Error("missing artifact name in content section")
	}
	if !strings.Contains(md, "pr_url") {
		t.Error("missing artifact content")
	}

	// Check suggested questions section
	if !strings.Contains(md, "## Suggested Questions") {
		t.Error("missing Suggested Questions section")
	}
	if !strings.Contains(md, "Would you like to review the changes?") {
		t.Error("missing suggested question")
	}

	// Check focus areas section
	if !strings.Contains(md, "## Focus Areas") {
		t.Error("missing Focus Areas section")
	}
	if !strings.Contains(md, "code changes") {
		t.Error("missing focus area")
	}

	// Check post-mortem questions (should be PR-related since we have pr-result artifact)
	if !strings.Contains(md, "## Post-Mortem Questions") {
		t.Error("missing Post-Mortem Questions section")
	}
}

func TestBuildChatClaudeMd_NoChatConfig(t *testing.T) {
	now := time.Now()
	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:        "no-cfg-test",
			PipelineName: "test",
			Status:       "completed",
			StartedAt:    now,
		},
		ProjectRoot: "/tmp/test",
		// No ChatConfig, no ArtifactContents
	}

	md := buildChatClaudeMd(ctx, ChatModeAnalysis, "", "")

	// Should NOT have the new sections
	if strings.Contains(md, "## Key Artifact Content") {
		t.Error("should not have Key Artifact Content section without ChatConfig")
	}
	if strings.Contains(md, "## Suggested Questions") {
		t.Error("should not have Suggested Questions section without ChatConfig")
	}
	if strings.Contains(md, "## Focus Areas") {
		t.Error("should not have Focus Areas section without ChatConfig")
	}

	// Should still have standard sections
	if !strings.Contains(md, "## Run Summary") {
		t.Error("missing Run Summary section")
	}
	if !strings.Contains(md, "## Wave Infrastructure") {
		t.Error("missing Wave Infrastructure section")
	}
}

func TestGeneratePostMortemQuestions_FailedRun(t *testing.T) {
	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:  "fail-run",
			Status: "failed",
		},
		Steps: []ChatStepContext{
			{StepID: "build", State: "failed"},
		},
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "test"}},
	}

	questions := generatePostMortemQuestions(ctx)
	if len(questions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(questions))
	}

	if !strings.Contains(questions[0], "build") {
		t.Errorf("expected failed step name in first question, got: %s", questions[0])
	}
	if !strings.Contains(questions[1], "retry") {
		t.Errorf("expected retry suggestion, got: %s", questions[1])
	}
}

func TestGeneratePostMortemQuestions_PRPipeline(t *testing.T) {
	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:  "pr-run",
			Status: "completed",
		},
		Artifacts: []state.ArtifactRecord{
			{Name: "pr-result.json"},
		},
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "gh-implement"}},
	}

	questions := generatePostMortemQuestions(ctx)
	if len(questions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(questions))
	}

	if !strings.Contains(questions[0], "review") {
		t.Errorf("expected review question for PR pipeline, got: %s", questions[0])
	}
}

func TestGeneratePostMortemQuestions_ReviewPipeline(t *testing.T) {
	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:  "rev-run",
			Status: "completed",
		},
		Artifacts: []state.ArtifactRecord{
			{Name: "review-verdict.json"},
		},
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "gh-pr-review"}},
	}

	questions := generatePostMortemQuestions(ctx)
	if len(questions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(questions))
	}

	if !strings.Contains(questions[0], "critical findings") {
		t.Errorf("expected findings question for review pipeline, got: %s", questions[0])
	}
}

func TestGeneratePostMortemQuestions_GenericPipeline(t *testing.T) {
	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:  "gen-run",
			Status: "completed",
		},
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "custom-pipeline"}},
	}

	questions := generatePostMortemQuestions(ctx)
	if len(questions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(questions))
	}

	if !strings.Contains(questions[0], "key outputs") {
		t.Errorf("expected generic output question, got: %s", questions[0])
	}
}

func TestBuildChatClaudeMd_StepFilter(t *testing.T) {
	now := time.Now()
	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:        "step-filter-test",
			PipelineName: "test-pipeline",
			Status:       "completed",
			StartedAt:    now,
		},
		Steps: []ChatStepContext{
			{StepID: "analyze", Persona: "navigator", State: "completed", Duration: 2 * time.Minute, TokensUsed: 3000},
			{StepID: "implement", Persona: "craftsman", State: "completed", Duration: 3 * time.Minute, TokensUsed: 2000, WorkspacePath: "/tmp/ws-impl"},
		},
		Artifacts: []state.ArtifactRecord{
			{StepID: "analyze", Name: "plan.json", Path: ".wave/output/plan.json", Type: "json"},
			{StepID: "implement", Name: "pr-result.json", Path: ".wave/output/pr-result.json", Type: "json"},
		},
		ProjectRoot: "/tmp/test",
	}

	md := buildChatClaudeMd(ctx, ChatModeAnalysis, "implement", "")

	// Header should mention the step
	if !strings.Contains(md, "# Wave Step Analysis: implement") {
		t.Error("missing step-scoped header")
	}

	// Should contain implement step data
	if !strings.Contains(md, "implement") {
		t.Error("missing implement step in results")
	}
	if !strings.Contains(md, "craftsman") {
		t.Error("missing craftsman persona in results")
	}

	// Step results table should NOT contain the other step
	// The "analyze" step should not be in the step results table rows
	lines := strings.Split(md, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "| 1 |") && strings.Contains(line, "analyze") {
			t.Error("step results should not contain 'analyze' step when filtered to 'implement'")
		}
	}

	// Artifacts should only show implement step's artifacts
	if !strings.Contains(md, "pr-result.json") {
		t.Error("missing implement step artifact")
	}
	// plan.json from analyze step should not appear in the filtered artifacts table
	artifactsSection := extractSection(md, "## Artifacts")
	if artifactsSection != "" && strings.Contains(artifactsSection, "plan.json") {
		t.Error("artifacts section should not contain analyze step's plan.json when filtered to implement")
	}

	// Workspaces should only show implement step
	if !strings.Contains(md, "/tmp/ws-impl") {
		t.Error("missing implement workspace path")
	}
}

func TestBuildChatClaudeMd_ArtifactFocus(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()

	// Create a test artifact file
	artDir := filepath.Join(tmpDir, ".wave", "output")
	if err := os.MkdirAll(artDir, 0755); err != nil {
		t.Fatal(err)
	}
	artContent := `{"status": "merged", "pr_url": "https://github.com/test/repo/pull/42"}`
	artPath := filepath.Join(artDir, "pr-result.json")
	if err := os.WriteFile(artPath, []byte(artContent), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:        "art-focus-test",
			PipelineName: "test-pipeline",
			Status:       "completed",
			StartedAt:    now,
		},
		Steps: []ChatStepContext{
			{StepID: "implement", Persona: "craftsman", State: "completed"},
		},
		Artifacts: []state.ArtifactRecord{
			{StepID: "implement", Name: "pr-result.json", Path: ".wave/output/pr-result.json", Type: "json", SizeBytes: int64(len(artContent))},
			{StepID: "implement", Name: "log.txt", Path: ".wave/output/log.txt", Type: "text"},
		},
		ProjectRoot: tmpDir,
	}

	md := buildChatClaudeMd(ctx, ChatModeAnalysis, "", "pr-result.json")

	// Header should mention the artifact
	if !strings.Contains(md, "# Wave Artifact Analysis: pr-result.json") {
		t.Error("missing artifact-focused header")
	}

	// Should contain focused artifact content section
	if !strings.Contains(md, "## Focused Artifact Content") {
		t.Error("missing Focused Artifact Content section")
	}
	if !strings.Contains(md, "pr_url") {
		t.Error("missing artifact content in focused section")
	}

	// Artifacts table should only show the focused artifact
	artifactsSection := extractSection(md, "## Artifacts")
	if artifactsSection != "" && strings.Contains(artifactsSection, "log.txt") {
		t.Error("artifacts table should not contain log.txt when focused on pr-result.json")
	}
}

func TestPrepareChatWorkspace_StepFilter(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()

	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:        "step-ws-test",
			PipelineName: "test",
			Status:       "completed",
			StartedAt:    now,
		},
		Steps: []ChatStepContext{
			{StepID: "analyze", Persona: "navigator", State: "completed"},
			{StepID: "implement", Persona: "craftsman", State: "completed"},
		},
		ProjectRoot: tmpDir,
	}

	wsPath, err := PrepareChatWorkspace(ctx, ChatWorkspaceOptions{
		Model:      "sonnet",
		StepFilter: "implement",
	})
	if err != nil {
		t.Fatalf("PrepareChatWorkspace failed: %v", err)
	}

	claudeMd, err := os.ReadFile(filepath.Join(wsPath, adapter.InstructionFilename("claude")))
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}
	md := string(claudeMd)

	if !strings.Contains(md, "# Wave Step Analysis: implement") {
		t.Error("CLAUDE.md missing step-scoped header")
	}
}

func TestPrepareChatWorkspace_ArtifactFocus(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()

	// Create artifact file
	artDir := filepath.Join(tmpDir, ".wave", "output")
	if err := os.MkdirAll(artDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "plan.md"), []byte("# Plan\nDo things"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := &ChatContext{
		Run: &state.RunRecord{
			RunID:        "art-ws-test",
			PipelineName: "test",
			Status:       "completed",
			StartedAt:    now,
		},
		Steps: []ChatStepContext{
			{StepID: "plan", Persona: "navigator", State: "completed"},
		},
		Artifacts: []state.ArtifactRecord{
			{StepID: "plan", Name: "plan.md", Path: ".wave/output/plan.md", Type: "markdown"},
		},
		ProjectRoot: tmpDir,
	}

	wsPath, err := PrepareChatWorkspace(ctx, ChatWorkspaceOptions{
		ArtifactName: "plan.md",
	})
	if err != nil {
		t.Fatalf("PrepareChatWorkspace failed: %v", err)
	}

	claudeMd, err := os.ReadFile(filepath.Join(wsPath, adapter.InstructionFilename("claude")))
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}
	md := string(claudeMd)

	if !strings.Contains(md, "# Wave Artifact Analysis: plan.md") {
		t.Error("CLAUDE.md missing artifact-focused header")
	}
	if !strings.Contains(md, "## Focused Artifact Content") {
		t.Error("CLAUDE.md missing focused artifact content section")
	}
	if !strings.Contains(md, "Do things") {
		t.Error("CLAUDE.md missing artifact content")
	}
}

// extractSection extracts the text between a section header and the next section or end of string.
func extractSection(md, header string) string {
	idx := strings.Index(md, header)
	if idx < 0 {
		return ""
	}
	rest := md[idx+len(header):]
	nextSection := strings.Index(rest, "\n## ")
	if nextSection < 0 {
		return rest
	}
	return rest[:nextSection]
}
