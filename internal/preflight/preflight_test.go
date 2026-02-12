package preflight

import (
	"fmt"
	"testing"

	"github.com/recinq/wave/internal/manifest"
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
	skills := map[string]manifest.SkillConfig{
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
	skills := map[string]manifest.SkillConfig{
		"myskill": {
			Check: "false", // always fails
		},
	}

	c := NewChecker(skills)

	results, err := c.CheckSkills([]string{"myskill"})
	if err == nil {
		t.Fatal("expected error for missing skill without install")
	}
	if !results[0].OK == false {
		t.Error("expected skill to not be installed")
	}
}

func TestCheckSkills_AutoInstallSuccess(t *testing.T) {
	callCount := 0
	skills := map[string]manifest.SkillConfig{
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
	skills := map[string]manifest.SkillConfig{
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
	skills := map[string]manifest.SkillConfig{
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
		// First call is check (fail), second is install, third is init, fourth is re-check
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
		t.Error("expected skill to be installed after install+init")
	}
	// Should have 4 calls: check, install, init, re-check
	if len(commands) != 4 {
		t.Errorf("expected 4 commands, got %d: %v", len(commands), commands)
	}
}

func TestRun_AllPass(t *testing.T) {
	skills := map[string]manifest.SkillConfig{
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
