//go:build integration

package adapter

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// concurrentMockRunner is an inline test double used by
// TestAdapterIntegration_ConcurrentAccess. Living inside the adapter
// package avoids importing the adaptertest helper package — which would
// create an import cycle with this file's `package adapter` declaration.
type concurrentMockRunner struct {
	stdoutJSON string
	tokensUsed int
}

func (m *concurrentMockRunner) Run(_ context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	_ = cfg
	return &AdapterResult{
		ExitCode:      0,
		Stdout:        bytes.NewReader([]byte(m.stdoutJSON)),
		TokensUsed:    m.tokensUsed,
		ResultContent: m.stdoutJSON,
	}, nil
}

// concurrentMockRegistry is a minimal registry that hands out
// concurrentMockRunner instances by name.
type concurrentMockRegistry struct {
	mu       sync.RWMutex
	adapters map[string]*concurrentMockRunner
}

func newConcurrentMockRegistry() *concurrentMockRegistry {
	return &concurrentMockRegistry{adapters: make(map[string]*concurrentMockRunner)}
}

func (r *concurrentMockRegistry) Register(name string, m *concurrentMockRunner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[name] = m
}

func (r *concurrentMockRegistry) CreateRunner(name string) AdapterRunner {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if m, ok := r.adapters[name]; ok {
		return m
	}
	return &concurrentMockRunner{}
}

// TestAdapterIntegration_ConcurrentAccess tests concurrent adapter usage
func TestAdapterIntegration_ConcurrentAccess(t *testing.T) {
	const numWorkers = 10
	const opsPerWorker = 5

	registry := newConcurrentMockRegistry()

	// Register different adapters for each worker
	for i := 0; i < numWorkers; i++ {
		name := fmt.Sprintf("worker-%d", i)
		registry.Register(name, &concurrentMockRunner{
			stdoutJSON: fmt.Sprintf(`{"worker": %d, "timestamp": "%d"}`, i, time.Now().Unix()),
			tokensUsed: 100 + i*10,
		})
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
