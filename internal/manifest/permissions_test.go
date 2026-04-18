package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/adapter"
)

// =============================================================================
// Persona Permission Model Tests
//
// These tests verify the permission model for personas defined in wave.yaml,
// focusing on artifact creation permissions and deny pattern enforcement.
// =============================================================================

// TestPersonaPermission_ImplementerCanCreateArtifact verifies that the implementer
// persona has Write permission for creating artifact.json files.
func TestPersonaPermission_ImplementerCanCreateArtifact(t *testing.T) {
	// Load manifest to verify implementer persona configuration
	m := createTestManifestWithPersonas(t)

	implementer := m.GetPersona("implementer")
	if implementer == nil {
		t.Fatal("implementer persona not found in manifest")
	}

	// Create permission checker for implementer
	checker := adapter.NewPermissionChecker(
		"implementer",
		implementer.Permissions.AllowedTools,
		implementer.Permissions.Deny,
	)

	// Test 1: Implementer should be able to write artifact.json
	err := checker.CheckPermission("Write", ".agents/artifact.json")
	if err != nil {
		t.Errorf("implementer should be able to create artifact.json, got error: %v", err)
	}

	// Test 2: Implementer should be able to write to artifacts directory
	err = checker.CheckPermission("Write", ".agents/artifacts/result.json")
	if err != nil {
		t.Errorf("implementer should be able to write to .agents/artifacts/, got error: %v", err)
	}

	// Test 3: Implementer should have Write in allowed tools
	hasWrite := false
	for _, tool := range implementer.Permissions.AllowedTools {
		if strings.HasPrefix(tool, "Write") {
			hasWrite = true
			break
		}
	}
	if !hasWrite {
		t.Error("implementer persona should have Write in allowed_tools")
	}

	// Test 4: Implementer should be able to write source files
	err = checker.CheckPermission("Write", "src/main.go")
	if err != nil {
		t.Errorf("implementer should be able to write source files, got error: %v", err)
	}
}

// TestPersonaPermission_ReviewerCanCreateArtifact verifies that the reviewer
// persona has limited Write permission specifically for artifact.json.
func TestPersonaPermission_ReviewerCanCreateArtifact(t *testing.T) {
	m := createTestManifestWithPersonas(t)

	reviewer := m.GetPersona("reviewer")
	if reviewer == nil {
		t.Fatal("reviewer persona not found in manifest")
	}

	checker := adapter.NewPermissionChecker(
		"reviewer",
		reviewer.Permissions.AllowedTools,
		reviewer.Permissions.Deny,
	)

	// Test 1: Reviewer should be able to write artifact.json
	err := checker.CheckPermission("Write", ".agents/artifact.json")
	if err != nil {
		t.Errorf("reviewer should be able to create artifact.json, got error: %v", err)
	}

	// Test 2: Reviewer should be able to write to artifacts directory
	err = checker.CheckPermission("Write", ".agents/artifacts/review-result.json")
	if err != nil {
		t.Errorf("reviewer should be able to write to .agents/artifacts/, got error: %v", err)
	}

	// Test 3: Verify reviewer has specific Write patterns in allowed tools
	hasArtifactWrite := false
	for _, tool := range reviewer.Permissions.AllowedTools {
		if tool == "Write(.agents/artifact.json)" || tool == "Write(.agents/artifacts/*)" {
			hasArtifactWrite = true
			break
		}
	}
	if !hasArtifactWrite {
		t.Errorf("reviewer should have Write(.agents/artifact.json) or Write(.agents/artifacts/*) in allowed_tools, got: %v", reviewer.Permissions.AllowedTools)
	}
}

// TestPersonaPermission_ReviewerCannotWriteSourceFiles verifies that the reviewer
// persona cannot write source code files (.go, .ts).
func TestPersonaPermission_ReviewerCannotWriteSourceFiles(t *testing.T) {
	m := createTestManifestWithPersonas(t)

	reviewer := m.GetPersona("reviewer")
	if reviewer == nil {
		t.Fatal("reviewer persona not found in manifest")
	}

	checker := adapter.NewPermissionChecker(
		"reviewer",
		reviewer.Permissions.AllowedTools,
		reviewer.Permissions.Deny,
	)

	// Test 1: Reviewer should NOT be able to write .go files
	err := checker.CheckPermission("Write", "src/main.go")
	if err == nil {
		t.Error("reviewer should NOT be able to write .go files")
	}

	// Test 2: Reviewer should NOT be able to write .ts files
	err = checker.CheckPermission("Write", "src/app.ts")
	if err == nil {
		t.Error("reviewer should NOT be able to write .ts files")
	}

	// Test 3: Reviewer should NOT be able to write .py files
	err = checker.CheckPermission("Write", "scripts/tool.py")
	if err == nil {
		t.Error("reviewer should NOT be able to write .py files")
	}

	// Test 4: Reviewer should NOT be able to write .rs files
	err = checker.CheckPermission("Write", "src/lib.rs")
	if err == nil {
		t.Error("reviewer should NOT be able to write .rs files")
	}

	// Test 5: Verify deny patterns include source file restrictions
	hasDenyGo := false
	hasDenyTs := false
	hasDenyPy := false
	hasDenyRs := false
	for _, deny := range reviewer.Permissions.Deny {
		switch deny {
		case "Write(*.go)":
			hasDenyGo = true
		case "Write(*.ts)":
			hasDenyTs = true
		case "Write(*.py)":
			hasDenyPy = true
		case "Write(*.rs)":
			hasDenyRs = true
		}
	}
	if !hasDenyGo {
		t.Errorf("reviewer should have Write(*.go) in deny patterns, got: %v", reviewer.Permissions.Deny)
	}
	if !hasDenyTs {
		t.Errorf("reviewer should have Write(*.ts) in deny patterns, got: %v", reviewer.Permissions.Deny)
	}
	if !hasDenyPy {
		t.Errorf("reviewer should have Write(*.py) in deny patterns, got: %v", reviewer.Permissions.Deny)
	}
	if !hasDenyRs {
		t.Errorf("reviewer should have Write(*.rs) in deny patterns, got: %v", reviewer.Permissions.Deny)
	}
}

// TestPersonaPermission_ReviewerCannotRunDestructiveCommands verifies that the
// reviewer persona cannot run destructive bash commands (rm, git push, git commit).
func TestPersonaPermission_ReviewerCannotRunDestructiveCommands(t *testing.T) {
	m := createTestManifestWithPersonas(t)

	reviewer := m.GetPersona("reviewer")
	if reviewer == nil {
		t.Fatal("reviewer persona not found in manifest")
	}

	checker := adapter.NewPermissionChecker(
		"reviewer",
		reviewer.Permissions.AllowedTools,
		reviewer.Permissions.Deny,
	)

	// Destructive commands that should be denied
	deniedCommands := []struct {
		command string
		reason  string
	}{
		{"rm foo.txt", "reviewer should not be able to delete files"},
		{"rm -rf /tmp", "reviewer should not be able to recursively delete"},
		{"rm -f important.go", "reviewer should not be able to force-delete files"},
		{"git push origin main", "reviewer should not be able to push to remote"},
		{"git push --force", "reviewer should not be able to force push"},
		{"git commit -m \"msg\"", "reviewer should not be able to commit"},
		{"git commit --amend", "reviewer should not be able to amend commits"},
	}

	for _, dc := range deniedCommands {
		t.Run(dc.command, func(t *testing.T) {
			err := checker.CheckPermission("Bash", dc.command)
			if err == nil {
				t.Errorf("%s, but command was allowed: %s", dc.reason, dc.command)
			}
		})
	}

	// Safe commands that should still be allowed
	allowedCommands := []struct {
		command string
		reason  string
	}{
		{"go test ./...", "reviewer should be able to run tests"},
		{"npm test", "reviewer should be able to run npm tests"},
	}

	for _, ac := range allowedCommands {
		t.Run(ac.command, func(t *testing.T) {
			err := checker.CheckPermission("Bash", ac.command)
			if err != nil {
				t.Errorf("%s, got error: %v", ac.reason, err)
			}
		})
	}
}

// TestPersonaPermission_NavigatorCannotWrite verifies that the navigator
// persona has no Write permission (read-only).
func TestPersonaPermission_NavigatorCannotWrite(t *testing.T) {
	m := createTestManifestWithPersonas(t)

	navigator := m.GetPersona("navigator")
	if navigator == nil {
		t.Fatal("navigator persona not found in manifest")
	}

	checker := adapter.NewPermissionChecker(
		"navigator",
		navigator.Permissions.AllowedTools,
		navigator.Permissions.Deny,
	)

	// Test 1: Navigator should NOT be able to write any files
	err := checker.CheckPermission("Write", ".agents/artifact.json")
	if err == nil {
		t.Error("navigator should NOT be able to write artifact.json")
	}

	// Test 2: Navigator should NOT be able to write source files
	err = checker.CheckPermission("Write", "src/main.go")
	if err == nil {
		t.Error("navigator should NOT be able to write source files")
	}

	// Test 3: Verify Write(*) is in deny patterns
	hasDenyWrite := false
	for _, deny := range navigator.Permissions.Deny {
		if deny == "Write(*)" {
			hasDenyWrite = true
			break
		}
	}
	if !hasDenyWrite {
		t.Errorf("navigator should have Write(*) in deny patterns, got: %v", navigator.Permissions.Deny)
	}

	// Test 4: Navigator should still be able to Read
	err = checker.CheckPermission("Read", "src/main.go")
	if err != nil {
		t.Errorf("navigator should be able to Read files, got error: %v", err)
	}
}

// TestPersonaPermission_AuditorCannotWrite verifies that the auditor
// persona has no Write or Edit permission.
func TestPersonaPermission_AuditorCannotWrite(t *testing.T) {
	m := createTestManifestWithPersonas(t)

	auditor := m.GetPersona("auditor")
	if auditor == nil {
		t.Fatal("auditor persona not found in manifest")
	}

	checker := adapter.NewPermissionChecker(
		"auditor",
		auditor.Permissions.AllowedTools,
		auditor.Permissions.Deny,
	)

	// Test 1: Auditor should NOT be able to write any files
	err := checker.CheckPermission("Write", ".agents/artifact.json")
	if err == nil {
		t.Error("auditor should NOT be able to write artifact.json")
	}

	// Test 2: Auditor should NOT be able to edit any files
	err = checker.CheckPermission("Edit", "src/main.go")
	if err == nil {
		t.Error("auditor should NOT be able to edit source files")
	}

	// Test 3: Verify deny patterns
	hasDenyWrite := false
	hasDenyEdit := false
	for _, deny := range auditor.Permissions.Deny {
		if deny == "Write(*)" {
			hasDenyWrite = true
		}
		if deny == "Edit(*)" {
			hasDenyEdit = true
		}
	}
	if !hasDenyWrite {
		t.Errorf("auditor should have Write(*) in deny patterns, got: %v", auditor.Permissions.Deny)
	}
	if !hasDenyEdit {
		t.Errorf("auditor should have Edit(*) in deny patterns, got: %v", auditor.Permissions.Deny)
	}

	// Test 4: Auditor should still be able to Read and Grep
	err = checker.CheckPermission("Read", "src/main.go")
	if err != nil {
		t.Errorf("auditor should be able to Read files, got error: %v", err)
	}
}

// TestPersonaPermission_DenyPatternTakesPrecedence verifies that deny patterns
// always take precedence over allow patterns.
func TestPersonaPermission_DenyPatternTakesPrecedence(t *testing.T) {
	testCases := []struct {
		name         string
		allowedTools []string
		denyTools    []string
		tool         string
		argument     string
		expectDeny   bool
		reason       string
	}{
		{
			name:         "deny wildcard blocks all writes",
			allowedTools: []string{"Read", "Write"},
			denyTools:    []string{"Write(*)"},
			tool:         "Write",
			argument:     "any-file.txt",
			expectDeny:   true,
			reason:       "deny(*) should block even when Write is allowed",
		},
		{
			name:         "specific deny blocks matching writes",
			allowedTools: []string{"Read", "Write"},
			denyTools:    []string{"Write(*.go)"},
			tool:         "Write",
			argument:     "main.go",
			expectDeny:   true,
			reason:       "deny(*.go) should block .go file writes",
		},
		{
			name:         "specific deny does not block non-matching",
			allowedTools: []string{"Read", "Write"},
			denyTools:    []string{"Write(*.go)"},
			tool:         "Write",
			argument:     "config.yaml",
			expectDeny:   false,
			reason:       "deny(*.go) should not block .yaml file writes",
		},
		{
			name:         "deny bash dangerous commands",
			allowedTools: []string{"Bash"},
			denyTools:    []string{"Bash(rm -rf /*)"},
			tool:         "Bash",
			argument:     "rm -rf /home",
			expectDeny:   true,
			reason:       "deny(rm -rf /*) should block dangerous rm commands",
		},
		{
			name:         "allow bash safe commands",
			allowedTools: []string{"Bash"},
			denyTools:    []string{"Bash(rm -rf /*)"},
			tool:         "Bash",
			argument:     "ls -la",
			expectDeny:   false,
			reason:       "deny(rm -rf /*) should not block ls command",
		},
		{
			name:         "deny sudo commands",
			allowedTools: []string{"Bash"},
			denyTools:    []string{"Bash(sudo *)"},
			tool:         "Bash",
			argument:     "sudo apt install foo",
			expectDeny:   true,
			reason:       "deny(sudo *) should block sudo commands",
		},
		{
			name:         "deny rm commands",
			allowedTools: []string{"Bash"},
			denyTools:    []string{"Bash(rm *)"},
			tool:         "Bash",
			argument:     "rm important-file.txt",
			expectDeny:   true,
			reason:       "deny(rm *) should block rm commands",
		},
		{
			name:         "deny git push commands",
			allowedTools: []string{"Bash"},
			denyTools:    []string{"Bash(git push*)"},
			tool:         "Bash",
			argument:     "git push origin main",
			expectDeny:   true,
			reason:       "deny(git push*) should block git push commands",
		},
		{
			name:         "deny git commit commands",
			allowedTools: []string{"Bash"},
			denyTools:    []string{"Bash(git commit*)"},
			tool:         "Bash",
			argument:     "git commit -m \"test\"",
			expectDeny:   true,
			reason:       "deny(git commit*) should block git commit commands",
		},
		{
			name:         "deny rm does not block other commands",
			allowedTools: []string{"Bash"},
			denyTools:    []string{"Bash(rm *)"},
			tool:         "Bash",
			argument:     "ls -la",
			expectDeny:   false,
			reason:       "deny(rm *) should not block ls command",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checker := adapter.NewPermissionChecker("test-persona", tc.allowedTools, tc.denyTools)
			err := checker.CheckPermission(tc.tool, tc.argument)

			if tc.expectDeny {
				if err == nil {
					t.Errorf("%s: expected operation to be denied", tc.reason)
				} else if !adapter.IsPermissionError(err) {
					t.Errorf("expected PermissionError, got: %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("%s: expected operation to be allowed, got error: %v", tc.reason, err)
				}
			}
		})
	}
}

// TestPersonaPermission_ArtifactCreationScenarios tests various artifact creation
// scenarios across different personas.
func TestPersonaPermission_ArtifactCreationScenarios(t *testing.T) {
	m := createTestManifestWithPersonas(t)

	testCases := []struct {
		personaName   string
		artifactPath  string
		shouldSucceed bool
		description   string
	}{
		// Implementer scenarios
		{"implementer", ".agents/artifact.json", true, "implementer can create artifact.json"},
		{"implementer", ".agents/artifacts/step-result.json", true, "implementer can create files in .agents/artifacts/"},
		{"implementer", "src/generated.go", true, "implementer can create source files"},

		// Reviewer scenarios
		{"reviewer", ".agents/artifact.json", true, "reviewer can create artifact.json"},
		{"reviewer", ".agents/artifacts/review.json", true, "reviewer can create files in .agents/artifacts/"},
		{"reviewer", "src/main.go", false, "reviewer cannot create .go source files"},
		{"reviewer", "src/app.ts", false, "reviewer cannot create .ts source files"},
		{"reviewer", "scripts/tool.py", false, "reviewer cannot create .py source files"},
		{"reviewer", "src/lib.rs", false, "reviewer cannot create .rs source files"},

		// Navigator scenarios (read-only)
		{"navigator", ".agents/artifact.json", false, "navigator cannot create artifact.json"},
		{"navigator", "any-file.txt", false, "navigator cannot create any files"},

		// Auditor scenarios (read-only)
		{"auditor", ".agents/artifact.json", false, "auditor cannot create artifact.json"},
		{"auditor", "report.json", false, "auditor cannot create any files"},

		// Planner scenarios (read-only)
		{"planner", ".agents/artifact.json", false, "planner cannot create artifact.json"},
		{"planner", "plan.md", false, "planner cannot create any files"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			persona := m.GetPersona(tc.personaName)
			if persona == nil {
				t.Skipf("persona %s not found in test manifest", tc.personaName)
				return
			}

			checker := adapter.NewPermissionChecker(
				tc.personaName,
				persona.Permissions.AllowedTools,
				persona.Permissions.Deny,
			)

			err := checker.CheckPermission("Write", tc.artifactPath)

			if tc.shouldSucceed {
				if err != nil {
					t.Errorf("expected success, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("expected permission denied, but operation was allowed")
				}
			}
		})
	}
}

// TestPersonaPermission_ToolPatternParsing tests that permission patterns are
// correctly parsed from the manifest format.
func TestPersonaPermission_ToolPatternParsing(t *testing.T) {
	m := createTestManifestWithPersonas(t)

	reviewer := m.GetPersona("reviewer")
	if reviewer == nil {
		t.Fatal("reviewer persona not found")
	}

	// Verify the pattern structure in reviewer permissions
	expectedAllowPatterns := map[string]bool{
		"Read":                       true,
		"Glob":                       true,
		"Grep":                       true,
		"Write(.agents/artifact.json)": true,
		"Write(.agents/artifacts/*)":   true,
		"Bash(go test*)":             true,
		"Bash(npm test*)":            true,
	}

	for _, tool := range reviewer.Permissions.AllowedTools {
		if !expectedAllowPatterns[tool] {
			// Not an error, just log unexpected patterns
			t.Logf("found additional allowed tool: %s", tool)
		}
	}

	// Check key patterns exist
	foundArtifactJson := false
	foundArtifactsDir := false
	for _, tool := range reviewer.Permissions.AllowedTools {
		if tool == "Write(.agents/artifact.json)" {
			foundArtifactJson = true
		}
		if tool == "Write(.agents/artifacts/*)" {
			foundArtifactsDir = true
		}
	}

	if !foundArtifactJson {
		t.Error("reviewer should have Write(artifact.json) pattern")
	}
	if !foundArtifactsDir {
		t.Error("reviewer should have Write(.agents/artifacts/*) pattern")
	}
}

// TestPersonaPermission_CraftsmanFullAccess verifies that the craftsman persona
// has full implementation access.
func TestPersonaPermission_CraftsmanFullAccess(t *testing.T) {
	m := createTestManifestWithPersonas(t)

	craftsman := m.GetPersona("craftsman")
	if craftsman == nil {
		t.Fatal("craftsman persona not found")
	}

	checker := adapter.NewPermissionChecker(
		"craftsman",
		craftsman.Permissions.AllowedTools,
		craftsman.Permissions.Deny,
	)

	// Craftsman should have broad access
	allowedOperations := []struct {
		tool     string
		argument string
	}{
		{"Read", "src/main.go"},
		{"Write", "src/main.go"},
		{"Edit", "src/main.go"},
		{"Bash", "go test ./..."},
		{"Write", ".agents/artifact.json"},
		{"Write", ".agents/artifacts/result.json"},
	}

	for _, op := range allowedOperations {
		err := checker.CheckPermission(op.tool, op.argument)
		if err != nil {
			t.Errorf("craftsman should be allowed %s(%s), got error: %v", op.tool, op.argument, err)
		}
	}

	// But craftsman should be denied dangerous operations
	err := checker.CheckPermission("Bash", "rm -rf /home")
	if err == nil {
		t.Error("craftsman should be denied dangerous rm -rf commands")
	}
}

// TestPersonaPermission_PhilosopherLimitedWrite verifies that the philosopher
// persona can only write to .agents/specs/.
func TestPersonaPermission_PhilosopherLimitedWrite(t *testing.T) {
	m := createTestManifestWithPersonas(t)

	philosopher := m.GetPersona("philosopher")
	if philosopher == nil {
		t.Fatal("philosopher persona not found")
	}

	checker := adapter.NewPermissionChecker(
		"philosopher",
		philosopher.Permissions.AllowedTools,
		philosopher.Permissions.Deny,
	)

	// Philosopher should be able to write to .agents/specs/
	err := checker.CheckPermission("Write", ".agents/specs/feature.yaml")
	if err != nil {
		t.Errorf("philosopher should be able to write to .agents/specs/, got error: %v", err)
	}

	// Philosopher should NOT be able to run Bash commands
	err = checker.CheckPermission("Bash", "echo hello")
	if err == nil {
		t.Error("philosopher should not be able to run Bash commands")
	}

	// Verify Bash(*) is in deny patterns
	hasDenyBash := false
	for _, deny := range philosopher.Permissions.Deny {
		if deny == "Bash(*)" {
			hasDenyBash = true
			break
		}
	}
	if !hasDenyBash {
		t.Errorf("philosopher should have Bash(*) in deny patterns, got: %v", philosopher.Permissions.Deny)
	}
}

// TestLoadWaveYAML_PersonaPermissions loads the actual wave.yaml and verifies
// persona permissions are correctly defined.
func TestLoadWaveYAML_PersonaPermissions(t *testing.T) {
	// Try to find wave.yaml relative to the test
	waveYAMLPath := findWaveYAML(t)
	if waveYAMLPath == "" {
		t.Skip("wave.yaml not found, skipping integration test")
		return
	}

	manifest, err := Load(waveYAMLPath)
	if err != nil {
		t.Fatalf("failed to load wave.yaml: %v", err)
	}

	// Verify expected personas exist
	expectedPersonas := []string{
		"implementer",
		"reviewer",
		"navigator",
		"auditor",
		"craftsman",
		"philosopher",
		"planner",
	}

	for _, name := range expectedPersonas {
		persona := manifest.GetPersona(name)
		if persona == nil {
			t.Errorf("expected persona '%s' not found in wave.yaml", name)
		}
	}

	// Verify implementer has Write permission
	implementer := manifest.GetPersona("implementer")
	if implementer != nil {
		hasWrite := false
		for _, tool := range implementer.Permissions.AllowedTools {
			if strings.HasPrefix(tool, "Write") {
				hasWrite = true
				break
			}
		}
		if !hasWrite {
			t.Error("implementer in wave.yaml should have Write permission")
		}
	}

	// Verify reviewer has limited Write permission
	reviewer := manifest.GetPersona("reviewer")
	if reviewer != nil {
		hasArtifactWrite := false
		for _, tool := range reviewer.Permissions.AllowedTools {
			if tool == "Write(.agents/artifact.json)" || tool == "Write(.agents/artifacts/*)" {
				hasArtifactWrite = true
				break
			}
		}
		if !hasArtifactWrite {
			t.Error("reviewer in wave.yaml should have Write(.agents/artifact.json) or Write(.agents/artifacts/*) permission")
		}

		// Deny rules are intentionally empty — security is enforced via bubblewrap
		// sandbox and CLAUDE.md behavioral restrictions (issue #282).
		if len(reviewer.Permissions.Deny) != 0 {
			t.Errorf("reviewer in wave.yaml should have empty deny list (security via sandbox), got: %v", reviewer.Permissions.Deny)
		}
	}

	// Verify navigator has empty deny list — security via sandbox, not deny rules
	navigator := manifest.GetPersona("navigator")
	if navigator != nil {
		// Deny rules are intentionally empty — security is enforced via bubblewrap
		// sandbox and CLAUDE.md behavioral restrictions (issue #282).
		if len(navigator.Permissions.Deny) != 0 {
			t.Errorf("navigator in wave.yaml should have empty deny list (security via sandbox), got: %v", navigator.Permissions.Deny)
		}
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// createTestManifestWithPersonas creates a test manifest with personas matching
// the expected wave.yaml configuration.
func createTestManifestWithPersonas(t *testing.T) *Manifest {
	t.Helper()

	return &Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata: Metadata{
			Name:        "test-wave",
			Description: "Test manifest for permission tests",
		},
		Adapters: map[string]Adapter{
			"claude": {
				Binary: "claude",
				Mode:   "headless",
			},
		},
		Personas: map[string]Persona{
			"implementer": {
				Adapter:     "claude",
				Description: "Code execution and artifact generation for pipeline steps",
				Permissions: Permissions{
					AllowedTools: []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
					Deny:         []string{"Bash(rm -rf /*)", "Bash(sudo *)"},
				},
			},
			"reviewer": {
				Adapter:     "claude",
				Description: "Quality review, validation, and assessment",
				Permissions: Permissions{
					AllowedTools: []string{
						"Read",
						"Glob",
						"Grep",
						"Write(.agents/artifact.json)",
						"Write(.agents/artifacts/*)",
						"Bash(go test*)",
						"Bash(npm test*)",
					},
					Deny: []string{"Write(*.go)", "Write(*.ts)", "Write(*.py)", "Write(*.rs)", "Edit(*)", "Bash(rm *)", "Bash(git push*)", "Bash(git commit*)"},
				},
			},
			"navigator": {
				Adapter:     "claude",
				Description: "Read-only codebase exploration and analysis",
				Permissions: Permissions{
					AllowedTools: []string{
						"Read",
						"Glob",
						"Grep",
						"Bash(git log*)",
						"Bash(git status*)",
					},
					Deny: []string{"Write(*)", "Edit(*)", "Bash(git commit*)", "Bash(git push*)"},
				},
			},
			"auditor": {
				Adapter:     "claude",
				Description: "Security review and quality assurance",
				Permissions: Permissions{
					AllowedTools: []string{
						"Read",
						"Grep",
						"Bash(go vet*)",
						"Bash(npm audit*)",
					},
					Deny: []string{"Write(*)", "Edit(*)"},
				},
			},
			"craftsman": {
				Adapter:     "claude",
				Description: "Code implementation and testing",
				Permissions: Permissions{
					AllowedTools: []string{"Read", "Write", "Edit", "Bash"},
					Deny:         []string{"Bash(rm -rf /*)"},
				},
			},
			"philosopher": {
				Adapter:     "claude",
				Description: "Architecture design and specification",
				Permissions: Permissions{
					AllowedTools: []string{"Read", "Write(.agents/specs/*)"},
					Deny:         []string{"Bash(*)"},
				},
			},
			"planner": {
				Adapter:     "claude",
				Description: "Task breakdown and project planning",
				Permissions: Permissions{
					AllowedTools: []string{"Read", "Glob", "Grep"},
					Deny:         []string{"Write(*)", "Edit(*)", "Bash(*)"},
				},
			},
		},
		Runtime: Runtime{
			WorkspaceRoot: ".agents/workspaces",
		},
	}
}

// findWaveYAML attempts to find the wave.yaml file relative to the test.
func findWaveYAML(t *testing.T) string {
	t.Helper()

	// Try common relative paths from test execution location
	candidates := []string{
		"../../wave.yaml",
		"../../../wave.yaml",
		"wave.yaml",
	}

	// Also try from current working directory
	if cwd, err := os.Getwd(); err == nil {
		// Walk up from cwd looking for wave.yaml
		dir := cwd
		for i := 0; i < 5; i++ {
			candidate := filepath.Join(dir, "wave.yaml")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
			dir = filepath.Dir(dir)
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}
