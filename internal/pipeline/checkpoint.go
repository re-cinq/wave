package pipeline

import (
	"encoding/json"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/recinq/wave/internal/state"
)

// CheckpointRecorder captures cumulative artifact snapshots and workspace git
// commit SHAs after each step completes. Checkpoint data is persisted via the
// StateStore and used by fork/rewind operations to restore pipeline state.
type CheckpointRecorder struct {
	store state.RunStore
}

// Record saves a checkpoint for the given step. It captures the current
// artifact paths as a JSON snapshot and attempts to resolve the workspace's
// git HEAD SHA. Recording is best-effort: errors are silently ignored so
// that checkpoint failures never block pipeline execution.
func (r *CheckpointRecorder) Record(execution *PipelineExecution, step *Step, stepIndex int) {
	if r.store == nil {
		return
	}

	// Read shared state under the execution mutex.
	execution.mu.Lock()
	artifactPaths := make(map[string]string, len(execution.ArtifactPaths))
	for k, v := range execution.ArtifactPaths {
		artifactPaths[k] = v
	}
	workspacePath := execution.WorkspacePaths[step.ID]
	pipelineID := execution.Status.ID
	execution.mu.Unlock()

	// Marshal the cumulative artifact snapshot.
	artifactJSON, err := json.Marshal(artifactPaths)
	if err != nil {
		// Best-effort — skip checkpoint on marshal failure.
		return
	}

	// Attempt to capture the workspace git commit SHA. This only succeeds
	// for worktree-backed workspaces that have a .git directory.
	var commitSHA string
	if workspacePath != "" {
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = workspacePath
		out, err := cmd.Output()
		if err == nil {
			commitSHA = strings.TrimSpace(string(out))
		}
	}

	record := &state.CheckpointRecord{
		RunID:              pipelineID,
		StepID:             step.ID,
		StepIndex:          stepIndex,
		WorkspacePath:      workspacePath,
		WorkspaceCommitSHA: commitSHA,
		ArtifactSnapshot:   string(artifactJSON),
		CreatedAt:          time.Now(),
	}

	// Best-effort save — log failures so fork/rewind diagnostics are possible.
	if err := r.store.SaveCheckpoint(record); err != nil {
		log.Printf("Warning: failed to save checkpoint for step %q in run %s: %v", step.ID, pipelineID, err)
	}
}
