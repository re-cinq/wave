package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
				StepID:      "analyze",
				Persona:     "navigator",
				State:       "completed",
				Duration:    2 * time.Minute,
				TokensUsed:  3000,
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
	claudeMd, err := os.ReadFile(filepath.Join(wsPath, "CLAUDE.md"))
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
	json.Unmarshal(settingsData, &settings)

	if model, ok := settings["model"].(string); !ok || model != "sonnet" {
		t.Errorf("expected default model 'sonnet', got %v", settings["model"])
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

	claudeMd, err := os.ReadFile(filepath.Join(wsPath, "CLAUDE.md"))
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
