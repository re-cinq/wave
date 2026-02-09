//go:build integration

package adapter

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Integration Tests for Adapter Package
// =============================================================================

// TestAdapterIntegration_FullWorkflow tests a complete adapter workflow
func TestAdapterIntegration_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	// Test with ProcessGroupRunner
	t.Run("ProcessGroupRunner", func(t *testing.T) {
		runner := NewProcessGroupRunner()
		ctx := context.Background()

		// Step 1: Create a test file
		testFile := filepath.Join(tempDir, "test-input.txt")
		if err := os.WriteFile(testFile, []byte("hello integration test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Step 2: Run cat command to read the file
		cfg := AdapterRunConfig{
			Adapter:       "cat",
			Persona:       "test-reader",
			WorkspacePath: tempDir,
			Prompt:        testFile,
			Timeout:       10 * time.Second,
			Env:           []string{"TEST_MODE=integration"},
		}

		result, err := runner.Run(ctx, cfg)
		if err != nil {
			t.Fatalf("adapter run failed: %v", err)
		}

		// Step 3: Verify results
		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got: %d", result.ExitCode)
		}

		data, err := io.ReadAll(result.Stdout)
		if err != nil {
			t.Fatalf("failed to read output: %v", err)
		}

		if !strings.Contains(string(data), "hello integration test") {
			t.Errorf("output missing expected content: %s", string(data))
		}

		if result.TokensUsed <= 0 {
			t.Errorf("expected positive token count, got: %d", result.TokensUsed)
		}
	})

	// Test with ClaudeAdapter workspace preparation
	t.Run("ClaudeAdapter_WorkspacePreparation", func(t *testing.T) {
		adapter := NewClaudeAdapter()
		workspacePath := filepath.Join(tempDir, "claude-workspace")

		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		cfg := AdapterRunConfig{
			Persona:      "integration-test",
			SystemPrompt: "You are an integration test assistant.",
			Temperature:  0.8,
			AllowedTools: []string{"Read", "Write", "Edit", "Bash"},
			DenyTools:    []string{"Bash(rm -rf*)"},
		}

		err := adapter.prepareWorkspace(workspacePath, cfg)
		if err != nil {
			t.Fatalf("workspace preparation failed: %v", err)
		}

		// Verify .claude directory
		claudeDir := filepath.Join(workspacePath, ".claude")
		if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
			t.Error(".claude directory not created")
		}

		// Verify settings.json
		settingsPath := filepath.Join(claudeDir, "settings.json")
		settingsData, err := os.ReadFile(settingsPath)
		if err != nil {
			t.Errorf("failed to read settings.json: %v", err)
		}

		if !strings.Contains(string(settingsData), `"model"`) {
			t.Error("settings.json missing model configuration")
		}

		// Verify CLAUDE.md
		claudeMdPath := filepath.Join(workspacePath, "CLAUDE.md")
		claudeMdData, err := os.ReadFile(claudeMdPath)
		if err != nil {
			t.Errorf("failed to read CLAUDE.md: %v", err)
		}

		if !strings.Contains(string(claudeMdData), "integration test assistant") {
			t.Error("CLAUDE.md missing system prompt")
		}
	})
}

// TestAdapterIntegration_ConcurrentAccess tests concurrent adapter usage
func TestAdapterIntegration_ConcurrentAccess(t *testing.T) {
	const numWorkers = 10
	const opsPerWorker = 5

	registry := NewMockAdapterRegistry()

	// Register different adapters for each worker
	for i := 0; i < numWorkers; i++ {
		name := fmt.Sprintf("worker-%d", i)
		adapter := NewMockAdapter(
			WithStdoutJSON(fmt.Sprintf(`{"worker": %d, "timestamp": "%d"}`, i, time.Now().Unix())),
			WithTokensUsed(100 + i*10),
		)
		registry.Register(name, adapter)
	}

	var wg sync.WaitGroup
	errors := make(chan error, numWorkers*opsPerWorker)
	results := make(chan *AdapterResult, numWorkers*opsPerWorker)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			adapterName := fmt.Sprintf("worker-%d", workerID)
			runner := registry.CreateRunner(adapterName)

			for j := 0; j < opsPerWorker; j++ {
				ctx := context.Background()
				cfg := AdapterRunConfig{
					Adapter:       adapterName,
					Persona:       fmt.Sprintf("worker-%d", workerID),
					WorkspacePath: fmt.Sprintf("/tmp/worker-%d", workerID),
					Prompt:        fmt.Sprintf("task-%d-%d", workerID, j),
					Timeout:       5 * time.Second,
				}

				result, err := runner.Run(ctx, cfg)
				if err != nil {
					errors <- fmt.Errorf("worker %d, op %d: %w", workerID, j, err)
					continue
				}

				results <- result
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	close(errors)
	close(results)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify results
	resultCount := 0
	for result := range results {
		resultCount++
		if result.ExitCode != 0 {
			t.Errorf("unexpected exit code: %d", result.ExitCode)
		}
		if result.TokensUsed < 100 {
			t.Errorf("unexpected token count: %d", result.TokensUsed)
		}
	}

	expectedResults := numWorkers * opsPerWorker
	if resultCount != expectedResults {
		t.Errorf("expected %d results, got %d", expectedResults, resultCount)
	}
}

// TestAdapterIntegration_PermissionEnforcement tests permission enforcement
func TestAdapterIntegration_PermissionEnforcement(t *testing.T) {
	scenarios := []struct {
		name       string
		persona    string
		allowed    []string
		denied     []string
		operations []struct {
			tool        string
			arg         string
			shouldAllow bool
		}
	}{
		{
			name:    "navigator_read_only_workflow",
			persona: "navigator",
			allowed: []string{"Read", "Glob", "Grep", "Bash(find *)", "Bash(ls *)"},
			denied:  []string{"Write(*)", "Edit(*)", "Bash(rm *)", "Bash(git commit*)"},
			operations: []struct {
				tool        string
				arg         string
				shouldAllow bool
			}{
				{"Read", "src/main.go", true},
				{"Glob", "**/*.go", true},
				{"Grep", "func main", true},
				{"Bash", "find . -name '*.go'", true},
				{"Write", "notes.md", false},
				{"Edit", "config.yaml", false},
				{"Bash", "rm temp.txt", false},
			},
		},
		{
			name:    "craftsman_development_workflow",
			persona: "craftsman",
			allowed: []string{"Read", "Write", "Edit", "Bash"},
			denied:  []string{"Bash(sudo *)", "Write(/etc/*)", "Bash(rm -rf /)"},
			operations: []struct {
				tool        string
				arg         string
				shouldAllow bool
			}{
				{"Read", "requirements.txt", true},
				{"Write", "src/feature.go", true},
				{"Edit", "README.md", true},
				{"Bash", "go build .", true},
				{"Bash", "git add .", true},
				{"Bash", "sudo make install", false},
				{"Write", "/etc/hosts", false},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			checker := NewPermissionChecker(scenario.persona, scenario.allowed, scenario.denied)

			for i, op := range scenario.operations {
				err := checker.CheckPermission(op.tool, op.arg)
				allowed := err == nil

				if allowed != op.shouldAllow {
					t.Errorf("operation %d: %s(%s) - expected allowed=%v, got allowed=%v (error: %v)",
						i+1, op.tool, op.arg, op.shouldAllow, allowed, err)
				}
			}
		})
	}
}

// TestAdapterIntegration_ClaudeAdapterArgs tests Claude adapter argument building
func TestAdapterIntegration_ClaudeAdapterArgs(t *testing.T) {
	adapter := NewClaudeAdapter()

	testCases := []struct {
		name     string
		config   AdapterRunConfig
		verify   func([]string) error
	}{
		{
			name: "basic_prompt",
			config: AdapterRunConfig{
				Prompt: "Hello, Claude!",
			},
			verify: func(args []string) error {
				if !contains(args, "Hello, Claude!") {
					return fmt.Errorf("prompt not found in args: %v", args)
				}
				return nil
			},
		},
		{
			name: "with_allowed_tools",
			config: AdapterRunConfig{
				Prompt:       "Test with tools",
				AllowedTools: []string{"Read", "Write", "Edit"},
			},
			verify: func(args []string) error {
				if !contains(args, "--allowedTools") {
					return fmt.Errorf("--allowedTools flag not found")
				}
				idx := indexOf(args, "--allowedTools")
				if idx == -1 || idx+1 >= len(args) {
					return fmt.Errorf("--allowedTools value not found")
				}
				expectedTools := "Read,Write,Edit"
				if args[idx+1] != expectedTools {
					return fmt.Errorf("expected tools %q, got %q", expectedTools, args[idx+1])
				}
				return nil
			},
		},
		{
			name: "output_format",
			config: AdapterRunConfig{
				Prompt: "Format test",
			},
			verify: func(args []string) error {
				if !contains(args, "--output-format") || !contains(args, "stream-json") {
					return fmt.Errorf("output format not set correctly: %v", args)
				}
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := adapter.buildArgs(tc.config)
			if err := tc.verify(args); err != nil {
				t.Error(err)
			}
		})
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}