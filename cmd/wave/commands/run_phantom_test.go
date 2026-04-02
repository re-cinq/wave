//go:build integration

package commands

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPhantomRunRecords_DetachWithFromStep(t *testing.T) {
	// Test for issue #700: --from-step with --detach creates 3 phantom run records
	// This test should be run with a clean state database

	// Clean up any existing state
	cleanupTestState(t)

	// Create a simple test pipeline
	testPipeline := `
apiVersion: wave.recinq.dev/v1
kind: Pipeline
metadata:
  name: test-pipeline
  description: Test pipeline for phantom run records
input:
  source: test input
steps:
  - id: step1
    persona: researcher
    description: First test step
  - id: step2
    persona: researcher
    description: Second test step
`

	// Create the .wave directory and test pipeline
	require.NoError(t, os.MkdirAll(".wave/pipelines", 0755))
	require.NoError(t, os.WriteFile(".wave/pipelines/test-pipeline.yaml", []byte(testPipeline), 0644))

	// Create wave manifest
	manifest := `
apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
  description: Test project for phantom run records
personas:
  researcher:
    adapter: mock
    system_prompt_file: personas/researcher.md
    temperature: 0.1
    permissions:
      allowed_tools:
        - Read
        - Glob
        - Grep
      deny: []
runtime:
  workspace_root: .wave/workspaces
  pipeline_id_hash_length: 4
`
	require.NoError(t, os.WriteFile("wave.yaml", []byte(manifest), 0644))

	// Initialize state database
	stateDB := ".wave/state.db"
	store, err := state.NewStateStore(stateDB)
	require.NoError(t, err, "Failed to create state store")
	defer store.Close()

	// Get initial count of pipeline runs
	initialCount, err := store.ListRuns(state.ListRunsOptions{Limit: 1000})
	require.NoError(t, err, "Failed to get initial pipeline run count")

	// Test the scenario: wave run --detach --from-step step2 test-pipeline
	// We'll simulate this by calling the relevant functions directly since we can't
	// easily test the detached subprocess in integration tests

	// Load the pipeline
	m, err := loadManifest("wave.yaml")
	require.NoError(t, err, "Failed to load manifest")

	p, err := loadPipeline("test-pipeline", m)
	require.NoError(t, err, "Failed to load pipeline")

	// Create a run ID (simulating what --detach does)
	runID, err := store.CreateRun(p.Metadata.Name, "test input")
	require.NoError(t, err, "Failed to create run ID")

	// Create executor with the run ID
	executor := pipeline.NewDefaultPipelineExecutor(
		adapter.NewMockAdapter(),
		pipeline.WithRunID(runID),
		pipeline.WithStateStore(store),
	)

	// Now simulate ResumeFromStep with the run ID
	ctx := context.Background()
	resumeManager := pipeline.NewResumeManager(executor)

	// This should reuse the existing runID, not create a new one
	err = resumeManager.ResumeFromStep(ctx, p, m, "test input", "step2", false, runID)

	// If there's an error about missing workspaces, that's expected in this test
	// The important thing is to check how many run records were created

	// Get final count of pipeline runs
	finalCount, err := store.ListRuns(state.ListRunsOptions{Limit: 1000})
	require.NoError(t, err, "Failed to get final pipeline run count")

	// Should only have 1 more run record than we started with
	// If the bug exists, we'll see 3 extra records
	expectedCount := len(initialCount) + 1
	assert.Equal(t, expectedCount, len(finalCount),
		"Expected only 1 new pipeline run record, but found %d extra records (initial: %d, final: %d)",
		len(finalCount)-len(initialCount), len(initialCount), len(finalCount))

	// Clean up
	cleanupTestState(t)
}

func cleanupTestState(t *testing.T) {
	t.Helper()

	// Remove test files and directories
	paths := []string{
		".wave",
		"wave.yaml",
		"test-pipeline.yaml",
	}

	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			t.Logf("Warning: failed to remove %s: %v", path, err)
		}
	}
}

func loadManifest(path string) (*manifest.Manifest, error) {
	manifestData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var m manifest.Manifest
	if err := yaml.Unmarshal(manifestData, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &m, nil
}
