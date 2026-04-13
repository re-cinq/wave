package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/recinq/wave/internal/fileutil"
	"github.com/recinq/wave/internal/state"
)

// ForkManager creates new pipeline runs by forking from checkpoints of existing runs.
type ForkManager struct {
	store state.StateStore
}

// NewForkManager creates a new ForkManager.
func NewForkManager(store state.StateStore) *ForkManager {
	return &ForkManager{store: store}
}

// ForkPoint represents an available fork point in a run.
type ForkPoint struct {
	StepID    string `json:"step_id"`
	StepIndex int    `json:"step_index"`
	HasSHA    bool   `json:"has_sha"`
}

// ListForkPoints returns available fork points (completed steps with checkpoints) for a run.
func (fm *ForkManager) ListForkPoints(runID string) ([]ForkPoint, error) {
	checkpoints, err := fm.store.GetCheckpoints(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get checkpoints: %w", err)
	}

	points := make([]ForkPoint, len(checkpoints))
	for i, cp := range checkpoints {
		points[i] = ForkPoint{
			StepID:    cp.StepID,
			StepIndex: cp.StepIndex,
			HasSHA:    cp.WorkspaceCommitSHA != "",
		}
	}
	return points, nil
}

// Fork creates a new run that branches from a completed step of an existing run.
// It copies artifacts from all steps up to the fork point and returns the new run ID.
// By default only completed runs can be forked. Set allowFailed to true to permit
// forking failed or cancelled runs.
func (fm *ForkManager) Fork(sourceRunID, fromStep string, p *Pipeline, allowFailed ...bool) (string, error) {
	// Validate source run exists and is in a forkable state
	run, err := fm.store.GetRun(sourceRunID)
	if err != nil {
		return "", fmt.Errorf("source run %q not found: %w", sourceRunID, err)
	}
	if run.Status == "running" {
		return "", fmt.Errorf("cannot fork a running run %q — wait for it to complete or cancel it", sourceRunID)
	}
	if run.Status != "completed" {
		if len(allowFailed) == 0 || !allowFailed[0] {
			return "", fmt.Errorf("cannot fork %s run %q — use --allow-failed to fork non-completed runs", run.Status, sourceRunID)
		}
	}

	// Resolve step: by name or by index
	stepID := resolveStepID(fromStep, p)
	if stepID == "" {
		return "", fmt.Errorf("step %q not found in pipeline %q", fromStep, p.Metadata.Name)
	}

	// Get checkpoint for the fork point
	checkpoint, err := fm.store.GetCheckpoint(sourceRunID, stepID)
	if err != nil {
		return "", fmt.Errorf("no checkpoint found for step %q in run %q — the step may not have completed successfully: %w", stepID, sourceRunID, err)
	}

	// Create a new run record with fork lineage
	newRunID, err := fm.store.CreateRunWithFork(run.PipelineName, run.Input, sourceRunID)
	if err != nil {
		return "", fmt.Errorf("failed to create forked run: %w", err)
	}

	// Copy artifact files to new run workspace
	if err := fm.copyArtifacts(checkpoint, sourceRunID, newRunID); err != nil {
		return "", fmt.Errorf("failed to copy artifacts for fork: %w", err)
	}

	// Copy checkpoints from source run up to and including the fork point
	if err := fm.copyCheckpoints(sourceRunID, newRunID, checkpoint.StepIndex); err != nil {
		return "", fmt.Errorf("failed to copy checkpoints: %w", err)
	}

	return newRunID, nil
}

// copyArtifacts copies artifact files referenced in the checkpoint snapshot.
func (fm *ForkManager) copyArtifacts(checkpoint *state.CheckpointRecord, sourceRunID, newRunID string) error {
	var artifacts map[string]string
	if err := json.Unmarshal([]byte(checkpoint.ArtifactSnapshot), &artifacts); err != nil {
		return fmt.Errorf("failed to parse artifact snapshot: %w", err)
	}

	for _, srcPath := range artifacts {
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue // skip missing artifacts
		}

		// Build destination path by replacing source run ID with new run ID in path
		dstPath := replaceRunIDInPath(srcPath, sourceRunID, newRunID)
		if dstPath == srcPath {
			// If path doesn't contain run ID, create in .wave/artifacts/<newRunID>/
			dstPath = filepath.Join(".wave", "artifacts", newRunID, filepath.Base(srcPath))
		}

		if err := fileutil.CopyPath(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy artifact %s: %w", srcPath, err)
		}
	}
	return nil
}

// copyCheckpoints copies checkpoint records from source run to the new run up to stepIndex.
func (fm *ForkManager) copyCheckpoints(sourceRunID, newRunID string, maxStepIndex int) error {
	checkpoints, err := fm.store.GetCheckpoints(sourceRunID)
	if err != nil {
		return err
	}

	for _, cp := range checkpoints {
		if cp.StepIndex > maxStepIndex {
			break
		}
		newCP := &state.CheckpointRecord{
			RunID:              newRunID,
			StepID:             cp.StepID,
			StepIndex:          cp.StepIndex,
			WorkspacePath:      cp.WorkspacePath,
			WorkspaceCommitSHA: cp.WorkspaceCommitSHA,
			ArtifactSnapshot:   cp.ArtifactSnapshot,
		}
		if err := fm.store.SaveCheckpoint(newCP); err != nil {
			return fmt.Errorf("failed to copy checkpoint for step %s: %w", cp.StepID, err)
		}
	}
	return nil
}

// resolveStepID resolves a step reference (name or numeric index) to a step ID.
func resolveStepID(ref string, p *Pipeline) string {
	// Try direct match by step ID
	for _, step := range p.Steps {
		if step.ID == ref {
			return step.ID
		}
	}

	// Try numeric index
	var idx int
	if _, err := fmt.Sscanf(ref, "%d", &idx); err == nil {
		if idx >= 0 && idx < len(p.Steps) {
			return p.Steps[idx].ID
		}
	}

	return ""
}

// replaceRunIDInPath replaces the source run ID with the new run ID in a file path.
func replaceRunIDInPath(path, oldRunID, newRunID string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	// Walk path segments and replace matching ones
	parts := splitPath(dir)
	for i, part := range parts {
		if part == oldRunID {
			parts[i] = newRunID
		}
	}
	newDir := filepath.Join(parts...)
	if newDir == dir {
		return path // no replacement found
	}
	return filepath.Join(newDir, base)
}

// splitPath splits a path into its directory components.
func splitPath(path string) []string {
	path = filepath.Clean(path)
	if path == "." || path == "" {
		return nil
	}

	var parts []string
	for path != "" && path != "." {
		dir, file := filepath.Split(path)
		if file != "" {
			parts = append([]string{file}, parts...)
		}
		// Clean removes trailing slashes: "a/b/" -> "a/b"
		cleaned := filepath.Clean(dir)
		if cleaned == path {
			// Stuck at root (e.g., "/")
			if cleaned != "." && cleaned != "" {
				parts = append([]string{cleaned}, parts...)
			}
			break
		}
		path = cleaned
	}
	return parts
}

// copyFile is defined in subpipeline.go (shared by both fork and sub-pipeline composition).
