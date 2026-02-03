package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RollbackManager handles state management and rollback for failed pipelines
type RollbackManager struct {
	stateDir string
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(stateDir string) *RollbackManager {
	return &RollbackManager{
		stateDir: stateDir,
	}
}

// CheckpointState represents a saved state at a specific point
type CheckpointState struct {
	PipelineID    string                 `json:"pipeline_id"`
	StepID        string                 `json:"step_id"`
	Timestamp     time.Time              `json:"timestamp"`
	WorkspacePath string                 `json:"workspace_path"`
	Artifacts     map[string]string      `json:"artifacts"` // artifact name -> file path
	Metadata      map[string]interface{} `json:"metadata"`
	CanRollback   bool                   `json:"can_rollback"`
}

// RollbackOperation defines an operation that can be undone
type RollbackOperation struct {
	Type        string                 `json:"type"` // "file_created", "file_modified", "file_deleted", "git_commit", etc.
	Target      string                 `json:"target"`
	Backup      string                 `json:"backup,omitempty"` // Backup location for rollback
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CanRevert   bool                   `json:"can_revert"`
	RevertSteps []string               `json:"revert_steps,omitempty"` // Human-readable revert instructions
}

// RollbackLog tracks all operations that can be undone
type RollbackLog struct {
	PipelineID string              `json:"pipeline_id"`
	StartTime  time.Time           `json:"start_time"`
	Operations []RollbackOperation `json:"operations"`
	Checkpoints []CheckpointState  `json:"checkpoints"`
}

// CreateCheckpoint saves the current state for potential rollback
func (m *RollbackManager) CreateCheckpoint(pipelineID, stepID, workspacePath string, artifacts map[string]string) (*CheckpointState, error) {
	checkpoint := &CheckpointState{
		PipelineID:    pipelineID,
		StepID:        stepID,
		Timestamp:     time.Now(),
		WorkspacePath: workspacePath,
		Artifacts:     artifacts,
		Metadata:      make(map[string]interface{}),
		CanRollback:   true,
	}

	// Save checkpoint to disk
	checkpointPath := m.getCheckpointPath(pipelineID, stepID)
	if err := os.MkdirAll(filepath.Dir(checkpointPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	if err := os.WriteFile(checkpointPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write checkpoint: %w", err)
	}

	return checkpoint, nil
}

// LoadCheckpoint loads a previously saved checkpoint
func (m *RollbackManager) LoadCheckpoint(pipelineID, stepID string) (*CheckpointState, error) {
	checkpointPath := m.getCheckpointPath(pipelineID, stepID)
	data, err := os.ReadFile(checkpointPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint: %w", err)
	}

	var checkpoint CheckpointState
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	return &checkpoint, nil
}

// InitRollbackLog initializes a new rollback log for a pipeline
func (m *RollbackManager) InitRollbackLog(pipelineID string) (*RollbackLog, error) {
	log := &RollbackLog{
		PipelineID:  pipelineID,
		StartTime:   time.Now(),
		Operations:  []RollbackOperation{},
		Checkpoints: []CheckpointState{},
	}

	if err := m.saveRollbackLog(log); err != nil {
		return nil, err
	}

	return log, nil
}

// LogOperation records an operation that may need to be rolled back
func (m *RollbackManager) LogOperation(pipelineID string, op RollbackOperation) error {
	log, err := m.loadRollbackLog(pipelineID)
	if err != nil {
		// If log doesn't exist, initialize it
		log, err = m.InitRollbackLog(pipelineID)
		if err != nil {
			return err
		}
	}

	op.Timestamp = time.Now()
	log.Operations = append(log.Operations, op)

	return m.saveRollbackLog(log)
}

// Rollback reverts all operations for a failed pipeline
func (m *RollbackManager) Rollback(pipelineID string, toCheckpoint *CheckpointState) error {
	log, err := m.loadRollbackLog(pipelineID)
	if err != nil {
		return fmt.Errorf("failed to load rollback log: %w", err)
	}

	// Reverse operations in reverse order
	errors := []string{}
	for i := len(log.Operations) - 1; i >= 0; i-- {
		op := log.Operations[i]

		// If rolling back to a checkpoint, only revert operations after it
		if toCheckpoint != nil && op.Timestamp.Before(toCheckpoint.Timestamp) {
			break
		}

		if err := m.revertOperation(op); err != nil {
			errors = append(errors, fmt.Sprintf("failed to revert %s: %v", op.Type, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("rollback completed with errors: %v", errors)
	}

	return nil
}

// revertOperation reverses a single operation
func (m *RollbackManager) revertOperation(op RollbackOperation) error {
	if !op.CanRevert {
		return fmt.Errorf("operation cannot be reverted automatically")
	}

	switch op.Type {
	case "file_created":
		// Delete the created file
		if err := os.Remove(op.Target); err != nil && !os.IsNotExist(err) {
			return err
		}

	case "file_modified":
		// Restore from backup
		if op.Backup != "" {
			data, err := os.ReadFile(op.Backup)
			if err != nil {
				return err
			}
			if err := os.WriteFile(op.Target, data, 0644); err != nil {
				return err
			}
		}

	case "file_deleted":
		// Restore from backup
		if op.Backup != "" {
			data, err := os.ReadFile(op.Backup)
			if err != nil {
				return err
			}
			if err := os.WriteFile(op.Target, data, 0644); err != nil {
				return err
			}
		}

	case "git_commit":
		// Cannot auto-revert git commits - provide instructions
		return fmt.Errorf("git commits must be reverted manually: %s", op.RevertSteps)

	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}

	return nil
}

// GetRollbackPlan generates a plan for rolling back a failed pipeline
func (m *RollbackManager) GetRollbackPlan(pipelineID string) (string, error) {
	log, err := m.loadRollbackLog(pipelineID)
	if err != nil {
		return "", fmt.Errorf("failed to load rollback log: %w", err)
	}

	plan := fmt.Sprintf("Rollback Plan for Pipeline: %s\n", pipelineID)
	plan += fmt.Sprintf("Started: %s\n\n", log.StartTime.Format(time.RFC3339))
	plan += "Operations to Revert (in reverse order):\n\n"

	// Show operations in reverse order
	for i := len(log.Operations) - 1; i >= 0; i-- {
		op := log.Operations[i]
		plan += fmt.Sprintf("%d. [%s] %s\n", len(log.Operations)-i, op.Type, op.Target)

		if !op.CanRevert {
			plan += "   ⚠️  Requires manual intervention\n"
			if len(op.RevertSteps) > 0 {
				plan += "   Steps:\n"
				for _, step := range op.RevertSteps {
					plan += fmt.Sprintf("   - %s\n", step)
				}
			}
		} else {
			plan += "   ✓ Can be automatically reverted\n"
		}
		plan += "\n"
	}

	if len(log.Checkpoints) > 0 {
		plan += "Available Checkpoints:\n"
		for i, cp := range log.Checkpoints {
			plan += fmt.Sprintf("%d. Step: %s (at %s)\n", i+1, cp.StepID, cp.Timestamp.Format(time.RFC3339))
		}
	}

	return plan, nil
}

// CleanupCheckpoints removes old checkpoint data
func (m *RollbackManager) CleanupCheckpoints(pipelineID string) error {
	pipelineDir := filepath.Join(m.stateDir, pipelineID)
	return os.RemoveAll(pipelineDir)
}

// Helper methods

func (m *RollbackManager) getCheckpointPath(pipelineID, stepID string) string {
	return filepath.Join(m.stateDir, pipelineID, "checkpoints", fmt.Sprintf("%s.json", stepID))
}

func (m *RollbackManager) getRollbackLogPath(pipelineID string) string {
	return filepath.Join(m.stateDir, pipelineID, "rollback.json")
}

func (m *RollbackManager) saveRollbackLog(log *RollbackLog) error {
	logPath := m.getRollbackLogPath(log.PipelineID)
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(logPath, data, 0644)
}

func (m *RollbackManager) loadRollbackLog(pipelineID string) (*RollbackLog, error) {
	logPath := m.getRollbackLogPath(pipelineID)
	data, err := os.ReadFile(logPath)
	if err != nil {
		return nil, err
	}

	var log RollbackLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, err
	}

	return &log, nil
}

// CreateBackup creates a backup of a file before modification
func (m *RollbackManager) CreateBackup(pipelineID, filePath string) (string, error) {
	// Read original file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Create backup path
	backupDir := filepath.Join(m.stateDir, pipelineID, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s.backup", filepath.Base(filePath)))
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", err
	}

	return backupPath, nil
}
