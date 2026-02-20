package pipeline

import (
	"strings"
	"testing"

	"github.com/recinq/wave/internal/adapter"
)

// =============================================================================
// T021: Test denied tool pattern `Bash(sudo *)` blocks execution
// Verifies that a persona with deny pattern for sudo commands correctly blocks
// execution of sudo commands while allowing other bash operations.
// =============================================================================

func TestPermissionDenial_BashSudoBlocked(t *testing.T) {
	// Create a permission checker with sudo blocked
	checker := adapter.NewPermissionChecker(
		"craftsman",
		[]string{"Read", "Write", "Edit", "Bash"}, // Allow basic tools including Bash
		[]string{"Bash(sudo *)", "Bash(sudo)"},    // But deny sudo commands
	)

	testCases := []struct {
		name        string
		tool        string
		argument    string
		shouldBlock bool
		blockReason string
	}{
		// sudo commands should be blocked
		{
			name:        "sudo apt install blocked",
			tool:        "Bash",
			argument:    "sudo apt install package-name",
			shouldBlock: true,
			blockReason: "blocked by deny pattern",
		},
		{
			name:        "sudo rm blocked",
			tool:        "Bash",
			argument:    "sudo rm -rf /var/log",
			shouldBlock: true,
			blockReason: "blocked by deny pattern",
		},
		{
			name:        "sudo systemctl blocked",
			tool:        "Bash",
			argument:    "sudo systemctl restart nginx",
			shouldBlock: true,
			blockReason: "blocked by deny pattern",
		},
		{
			name:        "sudo chmod blocked",
			tool:        "Bash",
			argument:    "sudo chmod 755 /etc/file",
			shouldBlock: true,
			blockReason: "blocked by deny pattern",
		},
		// Non-sudo commands should be allowed
		{
			name:        "ls command allowed",
			tool:        "Bash",
			argument:    "ls -la",
			shouldBlock: false,
		},
		{
			name:        "git command allowed",
			tool:        "Bash",
			argument:    "git status",
			shouldBlock: false,
		},
		{
			name:        "go test allowed",
			tool:        "Bash",
			argument:    "go test ./...",
			shouldBlock: false,
		},
		{
			name:        "echo with sudo in string allowed",
			tool:        "Bash",
			argument:    "echo 'run sudo to install'",
			shouldBlock: false,
		},
		// Other tools should work normally
		{
			name:        "Read tool allowed",
			tool:        "Read",
			argument:    "/etc/passwd",
			shouldBlock: false,
		},
		{
			name:        "Write tool allowed",
			tool:        "Write",
			argument:    "output.txt",
			shouldBlock: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := checker.CheckPermission(tc.tool, tc.argument)

			if tc.shouldBlock {
				if err == nil {
					t.Fatalf("expected %s with %q to be blocked, but it was allowed",
						tc.tool, tc.argument)
				}

				permErr, ok := err.(*adapter.PermissionError)
				if !ok {
					t.Fatalf("expected *adapter.PermissionError, got %T: %v", err, err)
				}

				if tc.blockReason != "" && !strings.Contains(permErr.Reason, tc.blockReason) {
					t.Errorf("expected reason to contain %q, got: %s",
						tc.blockReason, permErr.Reason)
				}

				if permErr.Tool != tc.tool {
					t.Errorf("expected tool %q in error, got: %s", tc.tool, permErr.Tool)
				}

				if permErr.PersonaName != "craftsman" {
					t.Errorf("expected persona 'craftsman' in error, got: %s", permErr.PersonaName)
				}
			} else {
				if err != nil {
					t.Fatalf("expected %s with %q to be allowed, but got error: %v",
						tc.tool, tc.argument, err)
				}
			}
		})
	}
}

// TestPermissionDenial_SudoVariants tests various sudo command patterns
// to ensure comprehensive blocking of privileged escalation attempts.
func TestPermissionDenial_SudoVariants(t *testing.T) {
	checker := adapter.NewPermissionChecker(
		"reviewer",
		[]string{"Bash"},
		[]string{"Bash(sudo *)", "Bash(sudo)"},
	)

	blockedCommands := []string{
		"sudo ls",
		"sudo -u root command",
		"sudo -i",
		"sudo -s",
		"sudo bash",
		"sudo su",
		"sudo cat /etc/shadow",
	}

	for _, cmd := range blockedCommands {
		t.Run("blocked_"+cmd, func(t *testing.T) {
			err := checker.CheckPermission("Bash", cmd)
			if err == nil {
				t.Errorf("expected %q to be blocked, but it was allowed", cmd)
			}
		})
	}
}

// =============================================================================
// T022: Test path restriction blocks writes outside allowed directories
// Verifies that Write permissions can be scoped to specific directories and
// that writes outside those directories are blocked.
// =============================================================================

func TestPermissionDenial_PathRestriction(t *testing.T) {
	// Create a checker that only allows writing to .wave/ directory
	checker := adapter.NewPermissionChecker(
		"scoped-writer",
		[]string{"Read", "Write(.wave/*)", "Write(.wave/specs/*)", "Write(.wave/artifacts/*)"},
		[]string{},
	)

	testCases := []struct {
		name        string
		tool        string
		argument    string
		shouldAllow bool
	}{
		// Writes to .wave/ should be allowed
		{
			name:        "write to .wave/config.yaml",
			tool:        "Write",
			argument:    ".wave/config.yaml",
			shouldAllow: true,
		},
		{
			name:        "write to .wave/specs/feature.yaml",
			tool:        "Write",
			argument:    ".wave/specs/feature.yaml",
			shouldAllow: true,
		},
		{
			name:        "write to .wave/artifacts/output.json",
			tool:        "Write",
			argument:    ".wave/artifacts/output.json",
			shouldAllow: true,
		},
		// Writes outside .wave/ should be blocked
		{
			name:        "write to src/main.go blocked",
			tool:        "Write",
			argument:    "src/main.go",
			shouldAllow: false,
		},
		{
			name:        "write to /etc/passwd blocked",
			tool:        "Write",
			argument:    "/etc/passwd",
			shouldAllow: false,
		},
		{
			name:        "write to internal/pipeline/executor.go blocked",
			tool:        "Write",
			argument:    "internal/pipeline/executor.go",
			shouldAllow: false,
		},
		{
			name:        "write to ../.wave/sneaky blocked",
			tool:        "Write",
			argument:    "../.wave/sneaky",
			shouldAllow: false,
		},
		// Read should work anywhere (no path restriction)
		{
			name:        "read from src/main.go allowed",
			tool:        "Read",
			argument:    "src/main.go",
			shouldAllow: true,
		},
		{
			name:        "read from /etc/hosts allowed",
			tool:        "Read",
			argument:    "/etc/hosts",
			shouldAllow: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := checker.CheckPermission(tc.tool, tc.argument)

			if tc.shouldAllow {
				if err != nil {
					t.Fatalf("expected %s to %q to be allowed, but got error: %v",
						tc.tool, tc.argument, err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected %s to %q to be blocked, but it was allowed",
						tc.tool, tc.argument)
				}

				permErr, ok := err.(*adapter.PermissionError)
				if !ok {
					t.Fatalf("expected *adapter.PermissionError, got %T", err)
				}

				if permErr.Tool != tc.tool {
					t.Errorf("expected tool %q, got: %s", tc.tool, permErr.Tool)
				}
			}
		})
	}
}

// TestPermissionDenial_DeepPathRestriction tests that double-star patterns
// correctly match deep directory structures.
func TestPermissionDenial_DeepPathRestriction(t *testing.T) {
	// Allow writes only to specs directory and its subdirectories
	checker := adapter.NewPermissionChecker(
		"spec-writer",
		[]string{"Read", "Write(specs/**/*.yaml)", "Write(specs/**/*.md)"},
		[]string{},
	)

	testCases := []struct {
		name        string
		argument    string
		shouldAllow bool
	}{
		// Allowed paths
		{
			name:        "direct child yaml",
			argument:    "specs/feature.yaml",
			shouldAllow: true,
		},
		{
			name:        "nested yaml",
			argument:    "specs/deep/nested/spec.yaml",
			shouldAllow: true,
		},
		{
			name:        "direct child md",
			argument:    "specs/README.md",
			shouldAllow: true,
		},
		{
			name:        "nested md",
			argument:    "specs/features/auth/spec.md",
			shouldAllow: true,
		},
		// Blocked paths
		{
			name:        "wrong extension",
			argument:    "specs/config.json",
			shouldAllow: false,
		},
		{
			name:        "outside specs",
			argument:    "src/main.yaml",
			shouldAllow: false,
		},
		{
			name:        "root level",
			argument:    "feature.yaml",
			shouldAllow: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := checker.CheckPermission("Write", tc.argument)

			if tc.shouldAllow && err != nil {
				t.Errorf("expected Write to %q to be allowed, got: %v", tc.argument, err)
			}
			if !tc.shouldAllow && err == nil {
				t.Errorf("expected Write to %q to be blocked, but was allowed", tc.argument)
			}
		})
	}
}

// =============================================================================
// T023: Test deny rule takes precedence over allow pattern
// Verifies the fail-secure principle: when both allow and deny patterns match,
// deny always wins.
// =============================================================================

func TestPermissionDenial_DenyTakesPrecedence(t *testing.T) {
	// Create a checker with overlapping allow and deny patterns
	// Allow: Write to any path
	// Deny: Write to sensitive paths
	checker := adapter.NewPermissionChecker(
		"sensitive-aware",
		[]string{"Read", "Write", "Edit", "Bash"},
		[]string{
			"Write(*.env)",
			"Write(*.pem)",
			"Write(*.key)",
			"Write(*credentials*)",
			"Write(*secret*)",
			"Write(.git/*)",
			"Bash(rm -rf*)",
			"Bash(dd if=*)",
		},
	)

	testCases := []struct {
		name        string
		tool        string
		argument    string
		shouldBlock bool
		reason      string
	}{
		// Sensitive writes should be blocked (deny takes precedence)
		{
			name:        "env file blocked",
			tool:        "Write",
			argument:    ".env",
			shouldBlock: true,
			reason:      "blocked by deny pattern",
		},
		{
			name:        "production env blocked",
			tool:        "Write",
			argument:    "production.env",
			shouldBlock: true,
			reason:      "blocked by deny pattern",
		},
		{
			name:        "pem file blocked",
			tool:        "Write",
			argument:    "server.pem",
			shouldBlock: true,
			reason:      "blocked by deny pattern",
		},
		{
			name:        "private key blocked",
			tool:        "Write",
			argument:    "private.key",
			shouldBlock: true,
			reason:      "blocked by deny pattern",
		},
		{
			name:        "credentials file blocked",
			tool:        "Write",
			argument:    "aws_credentials.json",
			shouldBlock: true,
			reason:      "blocked by deny pattern",
		},
		{
			name:        "secret file blocked",
			tool:        "Write",
			argument:    "app_secret.txt",
			shouldBlock: true,
			reason:      "blocked by deny pattern",
		},
		{
			name:        "git internal blocked",
			tool:        "Write",
			argument:    ".git/config",
			shouldBlock: true,
			reason:      "blocked by deny pattern",
		},
		// Dangerous bash commands blocked
		{
			name:        "rm -rf blocked",
			tool:        "Bash",
			argument:    "rm -rf /",
			shouldBlock: true,
			reason:      "blocked by deny pattern",
		},
		{
			name:        "dd dangerous blocked",
			tool:        "Bash",
			argument:    "dd if=/dev/zero of=/dev/sda",
			shouldBlock: true,
			reason:      "blocked by deny pattern",
		},
		// Non-sensitive writes should be allowed
		{
			name:        "go file allowed",
			tool:        "Write",
			argument:    "main.go",
			shouldBlock: false,
		},
		{
			name:        "config yaml allowed",
			tool:        "Write",
			argument:    "config.yaml",
			shouldBlock: false,
		},
		{
			name:        "readme allowed",
			tool:        "Write",
			argument:    "README.md",
			shouldBlock: false,
		},
		// Safe bash commands allowed
		{
			name:        "ls allowed",
			tool:        "Bash",
			argument:    "ls -la",
			shouldBlock: false,
		},
		{
			name:        "go test allowed",
			tool:        "Bash",
			argument:    "go test ./...",
			shouldBlock: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := checker.CheckPermission(tc.tool, tc.argument)

			if tc.shouldBlock {
				if err == nil {
					t.Fatalf("deny should take precedence: expected %s(%s) to be blocked",
						tc.tool, tc.argument)
				}

				permErr, ok := err.(*adapter.PermissionError)
				if !ok {
					t.Fatalf("expected *adapter.PermissionError, got %T", err)
				}

				if tc.reason != "" && !strings.Contains(permErr.Reason, tc.reason) {
					t.Errorf("expected reason to contain %q, got: %s",
						tc.reason, permErr.Reason)
				}
			} else {
				if err != nil {
					t.Fatalf("expected %s(%s) to be allowed, got: %v",
						tc.tool, tc.argument, err)
				}
			}
		})
	}
}

// TestPermissionDenial_DenyPrecedence_WildcardConflict tests that deny
// takes precedence even when both use wildcards.
func TestPermissionDenial_DenyPrecedence_WildcardConflict(t *testing.T) {
	// Allow all writes, but deny specific patterns
	checker := adapter.NewPermissionChecker(
		"wildcard-test",
		[]string{"Write(*)"},       // Allow writing to any path
		[]string{"Write(*.secret)"}, // But deny .secret files
	)

	// Regular writes allowed
	err := checker.CheckPermission("Write", "test.txt")
	if err != nil {
		t.Errorf("expected Write(test.txt) allowed with Write(*), got: %v", err)
	}

	// .secret files blocked despite Write(*)
	err = checker.CheckPermission("Write", "config.secret")
	if err == nil {
		t.Error("expected Write(config.secret) blocked by deny pattern, but was allowed")
	}

	permErr, ok := err.(*adapter.PermissionError)
	if !ok {
		t.Fatalf("expected *adapter.PermissionError, got %T", err)
	}

	if !strings.Contains(permErr.Reason, "blocked by deny pattern") {
		t.Errorf("expected 'blocked by deny pattern' in reason, got: %s", permErr.Reason)
	}
}

// TestPermissionDenial_DenyPrecedence_ExactOverlap tests that deny wins
// when allow and deny patterns match exactly the same tool/argument.
func TestPermissionDenial_DenyPrecedence_ExactOverlap(t *testing.T) {
	// Both allow and deny patterns match exactly
	checker := adapter.NewPermissionChecker(
		"exact-overlap",
		[]string{"Write(config.yaml)"},
		[]string{"Write(config.yaml)"},
	)

	err := checker.CheckPermission("Write", "config.yaml")
	if err == nil {
		t.Fatal("deny must take precedence when both allow and deny match exactly")
	}

	permErr, ok := err.(*adapter.PermissionError)
	if !ok {
		t.Fatalf("expected *adapter.PermissionError, got %T", err)
	}

	if !strings.Contains(permErr.Reason, "blocked by deny pattern") {
		t.Errorf("deny pattern should be cited in error, got: %s", permErr.Reason)
	}
}

// TestPermissionDenial_FailSecure tests that when permissions are misconfigured,
// the system defaults to secure behavior.
func TestPermissionDenial_FailSecure(t *testing.T) {
	// Test case: allow list defined but tool not in it
	checker := adapter.NewPermissionChecker(
		"restricted",
		[]string{"Read", "Grep"}, // Only allow Read and Grep
		[]string{},
	)

	// Write not in allow list - should be denied
	err := checker.CheckPermission("Write", "anything.txt")
	if err == nil {
		t.Fatal("Write should be denied when not in allow list (fail-secure)")
	}

	permErr, ok := err.(*adapter.PermissionError)
	if !ok {
		t.Fatalf("expected *adapter.PermissionError, got %T", err)
	}

	if !strings.Contains(permErr.Reason, "not in allowed tools list") {
		t.Errorf("expected 'not in allowed tools list' in reason, got: %s", permErr.Reason)
	}

	// Edit not in allow list - should be denied
	err = checker.CheckPermission("Edit", "file.go")
	if err == nil {
		t.Fatal("Edit should be denied when not in allow list (fail-secure)")
	}

	// Bash not in allow list - should be denied
	err = checker.CheckPermission("Bash", "ls")
	if err == nil {
		t.Fatal("Bash should be denied when not in allow list (fail-secure)")
	}
}

// =============================================================================
// Pipeline-Level Permission Integration Tests
// These tests verify that permissions are correctly applied during pipeline
// step configuration.
// =============================================================================

// TestPipelinePermissions_PersonaConfiguration tests that personas with
// permission configurations correctly create permission checkers.
func TestPipelinePermissions_PersonaConfiguration(t *testing.T) {
	// This test verifies the integration between persona configuration
	// and permission checking at the pipeline level.

	testCases := []struct {
		name         string
		personaName  string
		allowedTools []string
		denyTools    []string
		operations   []struct {
			tool      string
			argument  string
			expectErr bool
		}
	}{
		{
			name:         "navigator - read-only persona",
			personaName:  "navigator",
			allowedTools: []string{"Read", "Grep", "Glob", "WebFetch"},
			denyTools:    []string{"Write(*)", "Edit(*)", "Bash(rm*)", "Bash(sudo*)"},
			operations: []struct {
				tool      string
				argument  string
				expectErr bool
			}{
				{"Read", "src/main.go", false},
				{"Grep", "pattern", false},
				{"Write", "output.txt", true},
				{"Edit", "file.go", true},
				{"Bash", "rm -rf /", true},
			},
		},
		{
			name:         "craftsman - write-allowed persona",
			personaName:  "craftsman",
			allowedTools: []string{"Read", "Write", "Edit", "Bash", "Grep", "Glob"},
			denyTools:    []string{"Bash(sudo *)", "Bash(rm -rf /*)"},
			operations: []struct {
				tool      string
				argument  string
				expectErr bool
			}{
				{"Read", "src/main.go", false},
				{"Write", "src/new_file.go", false},
				{"Edit", "src/main.go", false},
				{"Bash", "go test ./...", false},
				{"Bash", "sudo apt install", true},
				{"Bash", "rm -rf /home", true},
			},
		},
		{
			name:         "auditor - review-only persona",
			personaName:  "auditor",
			allowedTools: []string{"Read", "Grep", "Glob"},
			denyTools:    []string{},
			operations: []struct {
				tool      string
				argument  string
				expectErr bool
			}{
				{"Read", "internal/security/audit.go", false},
				{"Grep", "security vulnerability", false},
				{"Write", "report.md", true}, // Not in allow list
				{"Bash", "go test", true},    // Not in allow list
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checker := adapter.NewPermissionChecker(
				tc.personaName,
				tc.allowedTools,
				tc.denyTools,
			)

			for _, op := range tc.operations {
				err := checker.CheckPermission(op.tool, op.argument)

				if op.expectErr && err == nil {
					t.Errorf("persona %s: expected %s(%s) to be blocked",
						tc.personaName, op.tool, op.argument)
				}
				if !op.expectErr && err != nil {
					t.Errorf("persona %s: expected %s(%s) to be allowed, got: %v",
						tc.personaName, op.tool, op.argument, err)
				}
			}
		})
	}
}

// TestPermissionError_Format verifies that permission errors contain
// actionable information for debugging and logging.
func TestPermissionError_Format(t *testing.T) {
	checker := adapter.NewPermissionChecker(
		"test-persona",
		[]string{"Read"},
		[]string{"Write(*.secret)"},
	)

	// Test deny pattern error format
	err := checker.CheckPermission("Write", "config.secret")
	if err == nil {
		t.Fatal("expected error")
	}

	errMsg := err.Error()
	expectedParts := []string{
		"permission denied",
		"test-persona",
		"Write",
		"config.secret",
		"blocked by deny pattern",
	}

	for _, part := range expectedParts {
		if !strings.Contains(errMsg, part) {
			t.Errorf("error message should contain %q, got: %s", part, errMsg)
		}
	}

	// Test not-allowed error format
	err = checker.CheckPermission("Edit", "file.go")
	if err == nil {
		t.Fatal("expected error")
	}

	errMsg = err.Error()
	if !strings.Contains(errMsg, "not in allowed tools list") {
		t.Errorf("error message should explain why blocked, got: %s", errMsg)
	}
}
