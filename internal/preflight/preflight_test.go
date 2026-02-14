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

// T011: Test nil emitter doesn't panic
func TestNilEmitter_NoPanic(t *testing.T) {
	skills := map[string]manifest.SkillConfig{
		"myskill": {Check: "true"},
	}

	// No WithEmitter option — emitter is nil
	c := NewChecker(skills)

	// CheckTools should work without panic
	results, err := c.CheckTools([]string{"sh"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(results) != 1 || !results[0].OK {
		t.Error("expected tool 'sh' to be found")
	}

	// CheckSkills should work without panic
	results, err = c.CheckSkills([]string{"myskill"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(results) != 1 || !results[0].OK {
		t.Error("expected skill to be installed")
	}
}

// T012: Test emitter callback is called during tool checks
func TestEmitter_ToolChecks(t *testing.T) {
	type emitRecord struct {
		name, kind, message string
	}
	var emitted []emitRecord

	c := NewChecker(nil, WithEmitter(func(name, kind, message string) {
		emitted = append(emitted, emitRecord{name, kind, message})
	}))

	// Check a tool that exists
	results, err := c.CheckTools([]string{"sh"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !results[0].OK {
		t.Error("expected tool 'sh' to be found")
	}

	// Should have 2 emitter calls: "checking" and "found"
	if len(emitted) != 2 {
		t.Fatalf("expected 2 emitter calls, got %d: %+v", len(emitted), emitted)
	}

	// First call: checking
	if emitted[0].kind != "tool" {
		t.Errorf("expected kind 'tool', got %q", emitted[0].kind)
	}
	if !contains(emitted[0].message, "checking") {
		t.Errorf("expected 'checking' in message, got %q", emitted[0].message)
	}

	// Second call: found
	if !contains(emitted[1].message, "found") {
		t.Errorf("expected 'found' in message, got %q", emitted[1].message)
	}
}

// T013: Test emitter callback is called during skill checks (already installed)
func TestEmitter_SkillAlreadyInstalled(t *testing.T) {
	type emitRecord struct {
		name, kind, message string
	}
	var emitted []emitRecord

	skills := map[string]manifest.SkillConfig{
		"myskill": {Check: "true"},
	}

	c := NewChecker(skills,
		WithEmitter(func(name, kind, message string) {
			emitted = append(emitted, emitRecord{name, kind, message})
		}),
	)

	results, err := c.CheckSkills([]string{"myskill"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !results[0].OK {
		t.Error("expected skill to be installed")
	}

	// Should have emitter calls with kind="skill"
	if len(emitted) < 2 {
		t.Fatalf("expected at least 2 emitter calls, got %d: %+v", len(emitted), emitted)
	}

	// Verify kind is "skill" for all calls
	for _, e := range emitted {
		if e.kind != "skill" {
			t.Errorf("expected kind 'skill', got %q", e.kind)
		}
	}

	// Should have "checking" and "installed" messages
	foundChecking := false
	foundInstalled := false
	for _, e := range emitted {
		if contains(e.message, "checking") {
			foundChecking = true
		}
		if contains(e.message, "installed") {
			foundInstalled = true
		}
	}
	if !foundChecking {
		t.Error("expected 'checking' message from emitter")
	}
	if !foundInstalled {
		t.Error("expected 'installed' message from emitter")
	}
}

// T014: Test emitter callback for install+init sequence
func TestEmitter_InstallInitSequence(t *testing.T) {
	type emitRecord struct {
		name, kind, message string
	}
	var emitted []emitRecord

	skills := map[string]manifest.SkillConfig{
		"myskill": {
			Check:   "check-cmd",
			Install: "install-cmd",
			Init:    "init-cmd",
		},
	}

	callNum := 0
	c := NewChecker(skills,
		WithEmitter(func(name, kind, message string) {
			emitted = append(emitted, emitRecord{name, kind, message})
		}),
		WithRunCmd(func(name string, args ...string) error {
			callNum++
			// First call is check (fail), rest succeed
			if callNum == 1 {
				return fmt.Errorf("not installed")
			}
			return nil
		}),
	)

	results, err := c.CheckSkills([]string{"myskill"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !results[0].OK {
		t.Error("expected skill to be installed after install+init")
	}

	// Verify emitter message sequence: checking → installing → initializing → installed
	var messages []string
	for _, e := range emitted {
		messages = append(messages, e.message)
	}

	expectedSequence := []string{"checking", "installing", "initializing", "installed successfully"}
	for _, expected := range expectedSequence {
		found := false
		for _, msg := range messages {
			if contains(msg, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected message containing %q in sequence, got: %v", expected, messages)
		}
	}
}

// T015: Test edge case: install succeeds but re-check fails
func TestEmitter_InstallSucceeds_RecheckFails(t *testing.T) {
	type emitRecord struct {
		name, kind, message string
	}
	var emitted []emitRecord

	skills := map[string]manifest.SkillConfig{
		"myskill": {
			Check:   "check-cmd",
			Install: "install-cmd",
		},
	}

	c := NewChecker(skills,
		WithEmitter(func(name, kind, message string) {
			emitted = append(emitted, emitRecord{name, kind, message})
		}),
		WithRunCmd(func(name string, args ...string) error {
			// check always fails, install succeeds
			cmd := args[len(args)-1]
			if cmd == "check-cmd" {
				return fmt.Errorf("not installed")
			}
			return nil // install succeeds
		}),
	)

	results, err := c.CheckSkills([]string{"myskill"})
	if err == nil {
		t.Fatal("expected error for skill that fails re-check")
	}
	if results[0].OK {
		t.Error("expected skill to not be OK")
	}
	if !contains(results[0].Message, "still not detected after install") {
		t.Errorf("expected 'still not detected' message, got %q", results[0].Message)
	}

	// Verify emitter captured the "still not detected" message
	foundNotDetected := false
	for _, e := range emitted {
		if contains(e.message, "still not detected") {
			foundNotDetected = true
			break
		}
	}
	if !foundNotDetected {
		t.Error("expected emitter to capture 'still not detected' message")
	}
}

// T016: Test edge case: init fails after successful install
func TestEmitter_InitFailsAfterInstall(t *testing.T) {
	type emitRecord struct {
		name, kind, message string
	}
	var emitted []emitRecord

	skills := map[string]manifest.SkillConfig{
		"myskill": {
			Check:   "check-cmd",
			Install: "install-cmd",
			Init:    "init-cmd",
		},
	}

	c := NewChecker(skills,
		WithEmitter(func(name, kind, message string) {
			emitted = append(emitted, emitRecord{name, kind, message})
		}),
		WithRunCmd(func(name string, args ...string) error {
			cmd := args[len(args)-1]
			if cmd == "check-cmd" {
				return fmt.Errorf("not installed")
			}
			if cmd == "init-cmd" {
				return fmt.Errorf("init failed")
			}
			return nil // install succeeds
		}),
	)

	results, err := c.CheckSkills([]string{"myskill"})
	if err == nil {
		t.Fatal("expected error for failed init")
	}
	if results[0].OK {
		t.Error("expected skill to not be OK after init failure")
	}
	if !contains(results[0].Message, "init failed") {
		t.Errorf("expected 'init failed' in message, got %q", results[0].Message)
	}

	// Verify emitter captured init failure
	foundInitFailed := false
	for _, e := range emitted {
		if contains(e.message, "init failed") {
			foundInitFailed = true
			break
		}
	}
	if !foundInitFailed {
		t.Error("expected emitter to capture 'init failed' message")
	}
}

// T017: Test edge case: skill declared in requires but not in manifest (with emitter)
func TestEmitter_UndeclaredSkill(t *testing.T) {
	type emitRecord struct {
		name, kind, message string
	}
	var emitted []emitRecord

	// Empty skills map — no skills declared
	c := NewChecker(nil, WithEmitter(func(name, kind, message string) {
		emitted = append(emitted, emitRecord{name, kind, message})
	}))

	results, err := c.CheckSkills([]string{"undeclared"})
	if err == nil {
		t.Fatal("expected error for undeclared skill")
	}
	if results[0].OK {
		t.Error("expected undeclared skill to fail")
	}
	if !contains(results[0].Message, "not declared") {
		t.Errorf("expected 'not declared' in message, got %q", results[0].Message)
	}

	// Verify emitter fired for undeclared skill
	if len(emitted) == 0 {
		t.Fatal("expected emitter to fire for undeclared skill")
	}
	foundNotDeclared := false
	for _, e := range emitted {
		if contains(e.message, "not declared") {
			foundNotDeclared = true
			break
		}
	}
	if !foundNotDeclared {
		t.Error("expected emitter to capture 'not declared' message")
	}
}

// T018: Test WithRunCmd option works for test injection
func TestWithRunCmd_Option(t *testing.T) {
	var called bool
	skills := map[string]manifest.SkillConfig{
		"myskill": {Check: "check-cmd"},
	}

	c := NewChecker(skills, WithRunCmd(func(name string, args ...string) error {
		called = true
		return nil // check succeeds
	}))

	results, err := c.CheckSkills([]string{"myskill"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !results[0].OK {
		t.Error("expected skill to be installed via mock")
	}
	if !called {
		t.Error("expected WithRunCmd mock to be called")
	}
}

// contains checks if substr is in s (helper for tests).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
