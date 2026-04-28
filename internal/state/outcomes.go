package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// RecordOutcome persists a pipeline outcome (PR URL, issue URL, etc.) in the state DB.
// This survives worktree cleanup, unlike artifact files.
func (s *stateStore) RecordOutcome(runID, stepID, outcomeType, label, value, description string, metadata map[string]any) error {
	metadataJSON := ""
	if len(metadata) > 0 {
		b, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal outcome metadata: %w", err)
		}
		metadataJSON = string(b)
	}
	_, err := s.db.Exec(
		"INSERT INTO pipeline_outcome (run_id, step_id, type, label, value, description, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		runID, stepID, outcomeType, label, value, description, metadataJSON, time.Now().Unix(),
	)
	return err
}

// GetOutcomes returns all outcomes for a run.
func (s *stateStore) GetOutcomes(runID string) ([]OutcomeRecord, error) {
	rows, err := s.db.Query(
		"SELECT id, run_id, step_id, type, label, value, description, metadata, created_at FROM pipeline_outcome WHERE run_id = ? ORDER BY created_at",
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOutcomeRows(rows)
}

// GetOutcomesByValue finds runs that produced a specific outcome value (e.g., a PR URL).
func (s *stateStore) GetOutcomesByValue(outcomeType, value string) ([]OutcomeRecord, error) {
	rows, err := s.db.Query(
		"SELECT id, run_id, step_id, type, label, value, description, metadata, created_at FROM pipeline_outcome WHERE type = ? AND value LIKE ? ORDER BY created_at DESC",
		outcomeType, "%"+value+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOutcomeRows(rows)
}

func scanOutcomeRows(rows *sql.Rows) ([]OutcomeRecord, error) {
	var records []OutcomeRecord
	for rows.Next() {
		var r OutcomeRecord
		var createdAt int64
		var typeStr, description, metadataJSON string
		if err := rows.Scan(&r.ID, &r.RunID, &r.StepID, &typeStr, &r.Label, &r.Value, &description, &metadataJSON, &createdAt); err != nil {
			return nil, err
		}
		r.Type = OutcomeType(typeStr)
		r.Description = description
		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &r.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal outcome metadata: %w", err)
			}
		}
		r.CreatedAt = time.Unix(createdAt, 0)
		records = append(records, r)
	}
	return records, rows.Err()
}
