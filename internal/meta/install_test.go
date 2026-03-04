package meta

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockCommandRunner implements CommandRunner for testing.
type mockCommandRunner struct {
	errors map[string]error
	calls  []string
}

func newMockCommandRunner() *mockCommandRunner {
	return &mockCommandRunner{
		errors: make(map[string]error),
	}
}

func (m *mockCommandRunner) withError(command string, err error) *mockCommandRunner {
	m.errors[command] = err
	return m
}

func (m *mockCommandRunner) RunCommand(_ context.Context, command string) error {
	m.calls = append(m.calls, command)
	if err, ok := m.errors[command]; ok {
		return err
	}
	return nil
}

// --- GetInstallable tests ---

func TestGetInstallable_MixedDependencies(t *testing.T) {
	report := DependencyReport{
		Tools: []DependencyStatus{
			{Name: "git", Kind: "tool", Available: true, AutoInstallable: false},
			{Name: "claude", Kind: "tool", Available: false, AutoInstallable: true},
			{Name: "gpt-cli", Kind: "tool", Available: false, AutoInstallable: false},
		},
		Skills: []DependencyStatus{
			{Name: "speckit", Kind: "skill", Available: true, AutoInstallable: true},
			{Name: "test-skill", Kind: "skill", Available: false, AutoInstallable: true},
			{Name: "manual-skill", Kind: "skill", Available: false, AutoInstallable: false},
		},
	}

	installable := GetInstallable(report)

	if len(installable) != 2 {
		t.Fatalf("expected 2 installable deps, got %d", len(installable))
	}

	names := make(map[string]bool)
	for _, dep := range installable {
		names[dep.Name] = true
	}

	if !names["claude"] {
		t.Error("expected 'claude' to be installable (unavailable + auto-installable)")
	}
	if !names["test-skill"] {
		t.Error("expected 'test-skill' to be installable (unavailable + auto-installable)")
	}
	if names["git"] {
		t.Error("'git' should not be installable (already available)")
	}
	if names["gpt-cli"] {
		t.Error("'gpt-cli' should not be installable (not auto-installable)")
	}
	if names["speckit"] {
		t.Error("'speckit' should not be installable (already available)")
	}
	if names["manual-skill"] {
		t.Error("'manual-skill' should not be installable (not auto-installable)")
	}
}

func TestGetInstallable_EmptyReport(t *testing.T) {
	report := DependencyReport{
		Tools:  []DependencyStatus{},
		Skills: []DependencyStatus{},
	}

	installable := GetInstallable(report)

	if len(installable) != 0 {
		t.Errorf("expected 0 installable deps for empty report, got %d", len(installable))
	}
}

func TestGetInstallable_AllAvailable(t *testing.T) {
	report := DependencyReport{
		Tools: []DependencyStatus{
			{Name: "git", Kind: "tool", Available: true, AutoInstallable: true},
			{Name: "claude", Kind: "tool", Available: true, AutoInstallable: true},
		},
		Skills: []DependencyStatus{
			{Name: "speckit", Kind: "skill", Available: true, AutoInstallable: true},
		},
	}

	installable := GetInstallable(report)

	if len(installable) != 0 {
		t.Errorf("expected 0 installable deps when all are available, got %d", len(installable))
	}
}

// --- Install tests ---

func TestInstall_SuccessfulInstall(t *testing.T) {
	runner := newMockCommandRunner()
	installer := NewInstaller(WithCommandRunner(runner))

	deps := []DependencyStatus{
		{Name: "speckit", Kind: "skill", Available: false, AutoInstallable: true},
	}
	commands := map[string]string{
		"speckit": "pip install speckit",
	}

	results := installer.Install(context.Background(), deps, commands)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Errorf("expected successful install, got failure: %s", results[0].Message)
	}
	if results[0].Name != "speckit" {
		t.Errorf("expected name 'speckit', got %q", results[0].Name)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 command call, got %d", len(runner.calls))
	}
	if runner.calls[0] != "pip install speckit" {
		t.Errorf("expected command 'pip install speckit', got %q", runner.calls[0])
	}
}

func TestInstall_FailedInstall(t *testing.T) {
	runner := newMockCommandRunner().
		withError("npm install -g claude", fmt.Errorf("exit status 1: permission denied"))

	installer := NewInstaller(WithCommandRunner(runner))

	deps := []DependencyStatus{
		{Name: "claude", Kind: "tool", Available: false, AutoInstallable: true},
	}
	commands := map[string]string{
		"claude": "npm install -g claude",
	}

	results := installer.Install(context.Background(), deps, commands)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("expected failed install, got success")
	}
	if results[0].Name != "claude" {
		t.Errorf("expected name 'claude', got %q", results[0].Name)
	}
	if results[0].Message == "" {
		t.Error("expected non-empty error message for failed install")
	}
}

func TestInstall_EmptyDeps(t *testing.T) {
	runner := newMockCommandRunner()
	installer := NewInstaller(WithCommandRunner(runner))

	results := installer.Install(context.Background(), nil, map[string]string{
		"speckit": "pip install speckit",
	})

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty deps, got %d", len(results))
	}
	if len(runner.calls) != 0 {
		t.Errorf("expected 0 command calls for empty deps, got %d", len(runner.calls))
	}
}

func TestInstall_MissingInstallCommand(t *testing.T) {
	runner := newMockCommandRunner()
	installer := NewInstaller(WithCommandRunner(runner))

	deps := []DependencyStatus{
		{Name: "unknown-tool", Kind: "tool", Available: false, AutoInstallable: true},
	}
	// Empty install commands map — no command for "unknown-tool".
	commands := map[string]string{}

	results := installer.Install(context.Background(), deps, commands)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("expected failure when install command is missing")
	}
	if results[0].Name != "unknown-tool" {
		t.Errorf("expected name 'unknown-tool', got %q", results[0].Name)
	}
	if results[0].Message == "" {
		t.Error("expected non-empty message explaining missing command")
	}
	if len(runner.calls) != 0 {
		t.Errorf("expected 0 command calls when command is missing, got %d", len(runner.calls))
	}
}

func TestInstall_MultipleDepsMixedResults(t *testing.T) {
	runner := newMockCommandRunner().
		withError("install-b", fmt.Errorf("failed"))

	installer := NewInstaller(WithCommandRunner(runner))

	deps := []DependencyStatus{
		{Name: "dep-a", Kind: "tool", Available: false, AutoInstallable: true},
		{Name: "dep-b", Kind: "skill", Available: false, AutoInstallable: true},
		{Name: "dep-c", Kind: "tool", Available: false, AutoInstallable: true},
	}
	commands := map[string]string{
		"dep-a": "install-a",
		"dep-b": "install-b",
		"dep-c": "install-c",
	}

	results := installer.Install(context.Background(), deps, commands)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// dep-a should succeed.
	if !results[0].Success {
		t.Errorf("expected dep-a to succeed, got: %s", results[0].Message)
	}
	// dep-b should fail.
	if results[1].Success {
		t.Error("expected dep-b to fail")
	}
	// dep-c should succeed.
	if !results[2].Success {
		t.Errorf("expected dep-c to succeed, got: %s", results[2].Message)
	}
}

// --- NewInstaller tests ---

func TestNewInstaller_Defaults(t *testing.T) {
	installer := NewInstaller()

	if installer.runner == nil {
		t.Error("expected non-nil default runner")
	}
	if installer.timeout != 60*time.Second {
		t.Errorf("expected 60s default timeout, got %v", installer.timeout)
	}
}

func TestNewInstaller_WithOptions(t *testing.T) {
	runner := newMockCommandRunner()
	installer := NewInstaller(
		WithCommandRunner(runner),
		WithInstallTimeout(30*time.Second),
	)

	if installer.timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", installer.timeout)
	}
}
