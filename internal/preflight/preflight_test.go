package preflight

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/skill"
)

func TestCheckTools_Found(t *testing.T) {
	c := NewChecker(nil)

	// "sh" should exist on any system
	results, err := c.CheckTools([]string{"sh"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].OK {
		t.Errorf("expected tool 'sh' to be found")
	}
	if results[0].Kind != "tool" {
		t.Errorf("expected kind 'tool', got %q", results[0].Kind)
	}
}

func TestCheckTools_NotFound(t *testing.T) {
	c := NewChecker(nil)

	results, err := c.CheckTools([]string{"nonexistent-tool-xyz-999"})
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].OK {
		t.Error("expected tool to not be found")
	}
}

func TestCheckTools_Mixed(t *testing.T) {
	c := NewChecker(nil)

	results, err := c.CheckTools([]string{"sh", "nonexistent-tool-xyz-999"})
	if err == nil {
		t.Fatal("expected error for mixed results")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].OK {
		t.Error("expected 'sh' to be found")
	}
	if results[1].OK {
		t.Error("expected nonexistent tool to not be found")
	}
}

func TestCheckTools_Empty(t *testing.T) {
	c := NewChecker(nil)

	results, err := c.CheckTools(nil)
	if err != nil {
		t.Fatalf("expected no error for empty tools, got: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestCheckSkills_Undeclared(t *testing.T) {
	c := NewChecker(nil) // No skills configured

	results, err := c.CheckSkills([]string{"speckit"})
	if err == nil {
		t.Fatal("expected error for undeclared skill")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].OK {
		t.Error("expected undeclared skill to fail")
	}
	if results[0].Kind != "skill" {
		t.Errorf("expected kind 'skill', got %q", results[0].Kind)
	}
}

func TestCheckSkills_InstalledViaCheck(t *testing.T) {
	skills := map[string]skill.SkillConfig{
		"myskill": {
			Check: "true", // always succeeds
		},
	}

	c := NewChecker(skills)

	results, err := c.CheckSkills([]string{"myskill"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].OK {
		t.Error("expected skill to be installed")
	}
}

func TestCheckSkills_CheckFails_NoInstall(t *testing.T) {
	skills := map[string]skill.SkillConfig{
		"myskill": {
			Check: "false", // always fails
		},
	}

	c := NewChecker(skills)

	results, err := c.CheckSkills([]string{"myskill"})
	if err == nil {
		t.Fatal("expected error for missing skill without install")
	}
	if results[0].OK {
		t.Error("expected skill to not be installed")
	}
}

func TestCheckSkills_AutoInstallSuccess(t *testing.T) {
	callCount := 0
	skills := map[string]skill.SkillConfig{
		"myskill": {
			Install: "echo installing",
			Check:   "true",
		},
	}

	c := NewChecker(skills)
	// Override runCmd to track calls
	c.runCmd = func(name string, args ...string) error {
		callCount++
		return nil // All commands succeed
	}

	results, err := c.CheckSkills([]string{"myskill"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !results[0].OK {
		t.Error("expected skill to be installed after auto-install")
	}
}

func TestCheckSkills_AutoInstallFails(t *testing.T) {
	skills := map[string]skill.SkillConfig{
		"myskill": {
			Install: "exit 1",
			Check:   "false",
		},
	}

	c := NewChecker(skills)
	c.runCmd = func(name string, args ...string) error {
		return fmt.Errorf("command failed")
	}

	results, err := c.CheckSkills([]string{"myskill"})
	if err == nil {
		t.Fatal("expected error for failed install")
	}
	if results[0].OK {
		t.Error("expected skill to not be installed after failed install")
	}
}

func TestCheckSkills_WithInit(t *testing.T) {
	var commands []string
	skills := map[string]skill.SkillConfig{
		"myskill": {
			Install: "install-cmd",
			Init:    "init-cmd",
			Check:   "check-cmd",
		},
	}

	c := NewChecker(skills)
	callNum := 0
	c.runCmd = func(name string, args ...string) error {
		cmd := name + " " + fmt.Sprintf("%v", args)
		commands = append(commands, cmd)
		callNum++
		// First call is check (fail), second is install, third is re-check
		if callNum == 1 {
			return fmt.Errorf("not installed")
		}
		return nil
	}

	results, err := c.CheckSkills([]string{"myskill"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !results[0].OK {
		t.Error("expected skill to be installed after install")
	}
	// Should have 3 calls: check, install, re-check (init runs in worktree, not preflight)
	if len(commands) != 3 {
		t.Errorf("expected 3 commands, got %d: %v", len(commands), commands)
	}
}

func TestRun_AllPass(t *testing.T) {
	skills := map[string]skill.SkillConfig{
		"myskill": {Check: "true"},
	}

	c := NewChecker(skills)
	results, err := c.Run([]string{"sh"}, []string{"myskill"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestRun_ToolFails(t *testing.T) {
	c := NewChecker(nil)
	_, err := c.Run([]string{"nonexistent-tool-xyz-999"}, nil)
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
}

func TestRun_Empty(t *testing.T) {
	c := NewChecker(nil)
	results, err := c.Run(nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSkillError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *SkillError
		expected string
	}{
		{
			name: "with underlying error",
			err: &SkillError{
				MissingSkills: []string{"speckit"},
				Err:           fmt.Errorf("missing required skills: speckit"),
			},
			expected: "missing required skills: speckit",
		},
		{
			name: "without underlying error",
			err: &SkillError{
				MissingSkills: []string{"speckit", "testkit"},
				Err:           nil,
			},
			expected: "missing required skills: speckit, testkit",
		},
		{
			name: "single skill",
			err: &SkillError{
				MissingSkills: []string{"speckit"},
			},
			expected: "missing required skills: speckit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("SkillError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSkillError_Unwrap(t *testing.T) {
	underlyingErr := fmt.Errorf("base error")
	skillErr := &SkillError{
		MissingSkills: []string{"speckit"},
		Err:           underlyingErr,
	}

	unwrapped := skillErr.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}
}

func TestToolError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ToolError
		expected string
	}{
		{
			name: "with underlying error",
			err: &ToolError{
				MissingTools: []string{"gh"},
				Err:          fmt.Errorf("missing required tools: gh"),
			},
			expected: "missing required tools: gh",
		},
		{
			name: "without underlying error",
			err: &ToolError{
				MissingTools: []string{"gh", "jq"},
				Err:          nil,
			},
			expected: "missing required tools: gh, jq",
		},
		{
			name: "single tool",
			err: &ToolError{
				MissingTools: []string{"gh"},
			},
			expected: "missing required tools: gh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("ToolError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestToolError_Unwrap(t *testing.T) {
	underlyingErr := fmt.Errorf("base error")
	toolErr := &ToolError{
		MissingTools: []string{"gh"},
		Err:          underlyingErr,
	}

	unwrapped := toolErr.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}
}

func TestCheckSkills_ReturnsSkillError(t *testing.T) {
	skills := map[string]skill.SkillConfig{
		"speckit": {
			Check: "false", // not installed
		},
		"testkit": {
			Check: "false", // not installed
		},
	}

	c := NewChecker(skills)
	_, err := c.CheckSkills([]string{"speckit", "testkit"})

	if err == nil {
		t.Fatal("expected error for missing skills")
	}

	var skillErr *SkillError
	if !errors.As(err, &skillErr) {
		t.Fatalf("expected SkillError, got %T: %v", err, err)
	}

	if len(skillErr.MissingSkills) != 2 {
		t.Errorf("expected 2 missing skills, got %d", len(skillErr.MissingSkills))
	}

	expectedSkills := map[string]bool{"speckit": true, "testkit": true}
	for _, skill := range skillErr.MissingSkills {
		if !expectedSkills[skill] {
			t.Errorf("unexpected missing skill: %s", skill)
		}
	}
}

func TestCheckTools_ReturnsToolError(t *testing.T) {
	c := NewChecker(nil)
	_, err := c.CheckTools([]string{"nonexistent-tool-xyz-999", "another-fake-tool-abc-123"})

	if err == nil {
		t.Fatal("expected error for missing tools")
	}

	var toolErr *ToolError
	if !errors.As(err, &toolErr) {
		t.Fatalf("expected ToolError, got %T: %v", err, err)
	}

	if len(toolErr.MissingTools) != 2 {
		t.Errorf("expected 2 missing tools, got %d", len(toolErr.MissingTools))
	}

	expectedTools := map[string]bool{
		"nonexistent-tool-xyz-999":  true,
		"another-fake-tool-abc-123": true,
	}
	for _, tool := range toolErr.MissingTools {
		if !expectedTools[tool] {
			t.Errorf("unexpected missing tool: %s", tool)
		}
	}
}

func TestRun_PreservesSkillError(t *testing.T) {
	skills := map[string]skill.SkillConfig{
		"speckit": {Check: "false"},
	}

	c := NewChecker(skills)
	_, err := c.Run(nil, []string{"speckit"})

	if err == nil {
		t.Fatal("expected error for missing skill")
	}

	var skillErr *SkillError
	if !errors.As(err, &skillErr) {
		t.Fatalf("expected SkillError to be preserved, got %T: %v", err, err)
	}

	if len(skillErr.MissingSkills) != 1 || skillErr.MissingSkills[0] != "speckit" {
		t.Errorf("expected missing skill 'speckit', got %v", skillErr.MissingSkills)
	}
}

func TestRun_PreservesToolError(t *testing.T) {
	c := NewChecker(nil)
	_, err := c.Run([]string{"nonexistent-tool-xyz-999"}, nil)

	if err == nil {
		t.Fatal("expected error for missing tool")
	}

	var toolErr *ToolError
	if !errors.As(err, &toolErr) {
		t.Fatalf("expected ToolError to be preserved, got %T: %v", err, err)
	}

	if len(toolErr.MissingTools) != 1 || toolErr.MissingTools[0] != "nonexistent-tool-xyz-999" {
		t.Errorf("expected missing tool 'nonexistent-tool-xyz-999', got %v", toolErr.MissingTools)
	}
}

func TestCheckBrowserBinary_NotFound(t *testing.T) {
	c := NewChecker(nil)
	// Override PATH to be empty so no browser is found
	origLookPath := BrowserBinaries
	BrowserBinaries = []string{"nonexistent-browser-xyz-999"}
	defer func() { BrowserBinaries = origLookPath }()

	path, result := c.CheckBrowserBinary()
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}
	if result.OK {
		t.Error("expected browser check to fail")
	}
	if result.Kind != "tool" {
		t.Errorf("expected kind 'tool', got %q", result.Kind)
	}
}

func TestCheckBrowserBinary_Found(t *testing.T) {
	c := NewChecker(nil)
	// "sh" is guaranteed to exist — use it as a stand-in for a browser binary
	origBinaries := BrowserBinaries
	BrowserBinaries = []string{"sh"}
	defer func() { BrowserBinaries = origBinaries }()

	path, result := c.CheckBrowserBinary()
	if path == "" {
		t.Error("expected non-empty path")
	}
	if !result.OK {
		t.Errorf("expected browser check to pass, got: %s", result.Message)
	}
}

func TestCheckDockerDaemon(t *testing.T) {
	c := NewChecker(nil)
	result := c.CheckDockerDaemon()
	// On CI/dev machines, docker may or may not be installed
	// Just verify the result structure is valid
	if result.Name != "docker" {
		t.Errorf("expected name 'docker', got %q", result.Name)
	}
	if result.Kind != "tool" {
		t.Errorf("expected kind 'tool', got %q", result.Kind)
	}
}

func TestCheckDockerDaemon_DaemonNotRunning(t *testing.T) {
	c := NewChecker(nil)
	// Override runCmd to simulate daemon not running
	c.runCmd = func(name string, args ...string) error {
		return fmt.Errorf("cannot connect to docker daemon")
	}

	result := c.CheckDockerDaemon()
	// If docker binary is found but daemon is down, result depends on LookPath
	// We just verify the structure
	if result.Name != "docker" {
		t.Errorf("expected name 'docker', got %q", result.Name)
	}
	if result.Kind != "tool" {
		t.Errorf("expected kind 'tool', got %q", result.Kind)
	}
}

func TestCheckBubblewrap(t *testing.T) {
	c := NewChecker(nil)
	result := c.CheckBubblewrap()
	if result.Name != "bwrap" {
		t.Errorf("expected name 'bwrap', got %q", result.Name)
	}
	if result.Kind != "tool" {
		t.Errorf("expected kind 'tool', got %q", result.Kind)
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"longer than max", 10, "longer tha..."},
		{"  trimmed  ", 20, "trimmed"},
		{"", 10, ""},
	}
	for _, tt := range tests {
		got := truncateOutput(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateOutput(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestCheckSkills_PostInstallPathFallback(t *testing.T) {
	// Simulate: check always fails, install succeeds. The re-check via
	// isSkillInstalledWithToolBin also fails since the binary doesn't exist.
	// Verifies we get a diagnostic error message.
	skills := map[string]skill.SkillConfig{
		"myskill": {
			Check:   "nonexistent-binary-xyz --version",
			Install: "echo installing",
		},
	}

	c := NewChecker(skills)
	c.runCmd = func(name string, args ...string) error {
		cmd := strings.Join(args, " ")
		// Install commands succeed
		if strings.Contains(cmd, "installing") || strings.Contains(cmd, "echo") {
			return nil
		}
		// All check commands fail
		return fmt.Errorf("not found")
	}

	results, err := c.CheckSkills([]string{"myskill"})
	if err == nil {
		t.Fatal("expected error for skill that fails check after install")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].OK {
		t.Error("expected skill to fail (binary doesn't exist)")
	}
	if !strings.Contains(results[0].Message, "still not detected after install") {
		t.Errorf("expected diagnostic message, got: %s", results[0].Message)
	}
}

func TestRun_BothFailReturnsPreflightError(t *testing.T) {
	skills := map[string]skill.SkillConfig{
		"speckit": {Check: "false"},
	}

	c := NewChecker(skills)
	_, err := c.Run([]string{"nonexistent-tool-xyz-999"}, []string{"speckit"})

	if err == nil {
		t.Fatal("expected error")
	}

	// Should get PreflightError wrapping both
	var preflightErr *PreflightError
	if !errors.As(err, &preflightErr) {
		t.Fatalf("expected PreflightError, got %T: %v", err, err)
	}

	// Both typed errors should be extractable via errors.As
	var skillErr *SkillError
	if !errors.As(err, &skillErr) {
		t.Fatal("expected SkillError to be extractable from PreflightError")
	}
	if len(skillErr.MissingSkills) != 1 || skillErr.MissingSkills[0] != "speckit" {
		t.Errorf("expected missing skill 'speckit', got %v", skillErr.MissingSkills)
	}

	var toolErr *ToolError
	if !errors.As(err, &toolErr) {
		t.Fatal("expected ToolError to be extractable from PreflightError")
	}
	if len(toolErr.MissingTools) != 1 || toolErr.MissingTools[0] != "nonexistent-tool-xyz-999" {
		t.Errorf("expected missing tool, got %v", toolErr.MissingTools)
	}
}
