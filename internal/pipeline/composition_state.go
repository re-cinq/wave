package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// IterationState tracks per-item progress for resumable iteration.
type IterationState struct {
	StepID         string               `json:"step_id"`
	TotalItems     int                  `json:"total_items"`
	CompletedItems int                  `json:"completed_items"`
	Items          []IterationItemState `json:"items"`
}

// IterationItemState tracks a single iteration item.
type IterationItemState struct {
	Index         int    `json:"index"`
	Status        string `json:"status"` // "pending", "completed", "failed", "skipped"
	PipelineRunID string `json:"pipeline_run_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

// SaveIterationState persists iteration progress to disk.
func SaveIterationState(wsRoot, pipelineID, stepID string, state *IterationState) error {
	dir := filepath.Join(wsRoot, pipelineID, ".wave", "composition")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create composition state dir: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal iteration state: %w", err)
	}

	path := filepath.Join(dir, stepID+"-iteration.json")
	return os.WriteFile(path, data, 0644)
}

// LoadIterationState loads iteration progress from disk.
func LoadIterationState(wsRoot, pipelineID, stepID string) (*IterationState, error) {
	path := filepath.Join(wsRoot, pipelineID, ".wave", "composition", stepID+"-iteration.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state IterationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal iteration state: %w", err)
	}
	return &state, nil
}

// GateState tracks gate resolution for resumability.
type GateState struct {
	StepID     string `json:"step_id"`
	GateType   string `json:"gate_type"`
	Status     string `json:"status"` // "waiting", "resolved", "timed_out"
	ResolvedBy string `json:"resolved_by,omitempty"`
}

// SaveGateState persists gate resolution to disk.
func SaveGateState(wsRoot, pipelineID, stepID string, state *GateState) error {
	dir := filepath.Join(wsRoot, pipelineID, ".wave", "composition")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create composition state dir: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal gate state: %w", err)
	}

	path := filepath.Join(dir, stepID+"-gate.json")
	return os.WriteFile(path, data, 0644)
}

// LoadGateState loads gate resolution from disk.
func LoadGateState(wsRoot, pipelineID, stepID string) (*GateState, error) {
	path := filepath.Join(wsRoot, pipelineID, ".wave", "composition", stepID+"-gate.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state GateState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gate state: %w", err)
	}
	return &state, nil
}
