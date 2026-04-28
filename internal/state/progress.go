package state

import (
	"database/sql"
	"fmt"
	"time"
)

// =============================================================================
// Progress Tracking Methods (spec 018 - Enhanced Progress Visualization)
// =============================================================================

// SaveProgressSnapshot records a point-in-time progress snapshot.
func (s *stateStore) SaveProgressSnapshot(runID string, stepID string, progress int, action string, etaMs int64, validationPhase string, compactionStats string) error {
	now := time.Now().Unix()

	query := `INSERT INTO progress_snapshot (
	              run_id, step_id, timestamp, progress, current_action,
	              estimated_time_ms, validation_phase, compaction_stats
	          ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, runID, stepID, now, progress, action, etaMs, validationPhase, compactionStats)
	if err != nil {
		return fmt.Errorf("failed to save progress snapshot: %w", err)
	}

	return nil
}

// GetProgressSnapshots retrieves progress snapshots for a run/step.
func (s *stateStore) GetProgressSnapshots(runID string, stepID string, limit int) ([]ProgressSnapshotRecord, error) {
	query := `SELECT id, run_id, step_id, timestamp, progress, current_action,
	                 estimated_time_ms, validation_phase, compaction_stats
	          FROM progress_snapshot
	          WHERE run_id = ?`
	args := []any{runID}

	if stepID != "" {
		query += " AND step_id = ?"
		args = append(args, stepID)
	}

	query += " ORDER BY timestamp DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query progress snapshots: %w", err)
	}
	defer rows.Close()

	var records []ProgressSnapshotRecord
	for rows.Next() {
		var record ProgressSnapshotRecord
		var timestamp int64
		var currentAction, validationPhase, compactionStats sql.NullString
		var estimatedTimeMs sql.NullInt64

		err := rows.Scan(
			&record.ID,
			&record.RunID,
			&record.StepID,
			&timestamp,
			&record.Progress,
			&currentAction,
			&estimatedTimeMs,
			&validationPhase,
			&compactionStats,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan progress snapshot: %w", err)
		}

		record.Timestamp = time.Unix(timestamp, 0)
		if currentAction.Valid {
			record.CurrentAction = currentAction.String
		}
		if estimatedTimeMs.Valid {
			record.EstimatedTimeMs = estimatedTimeMs.Int64
		}
		if validationPhase.Valid {
			record.ValidationPhase = validationPhase.String
		}
		if compactionStats.Valid {
			record.CompactionStats = compactionStats.String
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating progress snapshots: %w", err)
	}

	return records, nil
}

// UpdateStepProgress updates or creates a step progress record.
func (s *stateStore) UpdateStepProgress(runID string, stepID string, persona string, state string, progress int, action string, message string, etaMs int64, tokens int) error {
	now := time.Now().Unix()

	query := `INSERT INTO step_progress (
	              step_id, run_id, persona, state, progress, current_action,
	              message, started_at, updated_at, estimated_completion_ms, tokens_used
	          ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(step_id) DO UPDATE SET
	              persona = excluded.persona,
	              state = excluded.state,
	              progress = excluded.progress,
	              current_action = excluded.current_action,
	              message = excluded.message,
	              updated_at = excluded.updated_at,
	              estimated_completion_ms = excluded.estimated_completion_ms,
	              tokens_used = excluded.tokens_used`

	_, err := s.db.Exec(query, stepID, runID, persona, state, progress, action, message, now, now, etaMs, tokens)
	if err != nil {
		return fmt.Errorf("failed to update step progress: %w", err)
	}

	return nil
}

// GetStepProgress retrieves the current progress for a specific step.
func (s *stateStore) GetStepProgress(stepID string) (*StepProgressRecord, error) {
	query := `SELECT step_id, run_id, persona, state, progress, current_action,
	                 message, started_at, updated_at, estimated_completion_ms, tokens_used
	          FROM step_progress
	          WHERE step_id = ?`

	var record StepProgressRecord
	var persona, currentAction, message sql.NullString
	var startedAt, updatedAt int64
	var estimatedCompletionMs sql.NullInt64

	err := s.db.QueryRow(query, stepID).Scan(
		&record.StepID,
		&record.RunID,
		&persona,
		&record.State,
		&record.Progress,
		&currentAction,
		&message,
		&startedAt,
		&updatedAt,
		&estimatedCompletionMs,
		&record.TokensUsed,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("step progress not found: %s", stepID)
		}
		return nil, fmt.Errorf("failed to get step progress: %w", err)
	}

	if persona.Valid {
		record.Persona = persona.String
	}
	if currentAction.Valid {
		record.CurrentAction = currentAction.String
	}
	if message.Valid {
		record.Message = message.String
	}
	if startedAt > 0 {
		t := time.Unix(startedAt, 0)
		record.StartedAt = &t
	}
	record.UpdatedAt = time.Unix(updatedAt, 0)
	if estimatedCompletionMs.Valid {
		record.EstimatedCompletionMs = estimatedCompletionMs.Int64
	}

	return &record, nil
}

// GetAllStepProgress retrieves progress for all steps in a run.
func (s *stateStore) GetAllStepProgress(runID string) ([]StepProgressRecord, error) {
	query := `SELECT step_id, run_id, persona, state, progress, current_action,
	                 message, started_at, updated_at, estimated_completion_ms, tokens_used
	          FROM step_progress
	          WHERE run_id = ?
	          ORDER BY updated_at ASC`

	rows, err := s.db.Query(query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to query step progress: %w", err)
	}
	defer rows.Close()

	var records []StepProgressRecord
	for rows.Next() {
		var record StepProgressRecord
		var persona, currentAction, message sql.NullString
		var startedAt, updatedAt int64
		var estimatedCompletionMs sql.NullInt64

		err := rows.Scan(
			&record.StepID,
			&record.RunID,
			&persona,
			&record.State,
			&record.Progress,
			&currentAction,
			&message,
			&startedAt,
			&updatedAt,
			&estimatedCompletionMs,
			&record.TokensUsed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan step progress: %w", err)
		}

		if persona.Valid {
			record.Persona = persona.String
		}
		if currentAction.Valid {
			record.CurrentAction = currentAction.String
		}
		if message.Valid {
			record.Message = message.String
		}
		if startedAt > 0 {
			t := time.Unix(startedAt, 0)
			record.StartedAt = &t
		}
		record.UpdatedAt = time.Unix(updatedAt, 0)
		if estimatedCompletionMs.Valid {
			record.EstimatedCompletionMs = estimatedCompletionMs.Int64
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating step progress: %w", err)
	}

	return records, nil
}

// UpdatePipelineProgress updates pipeline-level progress aggregation.
func (s *stateStore) UpdatePipelineProgress(runID string, totalSteps int, completedSteps int, currentStepIndex int, overallProgress int, etaMs int64) error {
	now := time.Now().Unix()

	query := `INSERT INTO pipeline_progress (
	              run_id, total_steps, completed_steps, current_step_index,
	              overall_progress, estimated_completion_ms, updated_at
	          ) VALUES (?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(run_id) DO UPDATE SET
	              total_steps = excluded.total_steps,
	              completed_steps = excluded.completed_steps,
	              current_step_index = excluded.current_step_index,
	              overall_progress = excluded.overall_progress,
	              estimated_completion_ms = excluded.estimated_completion_ms,
	              updated_at = excluded.updated_at`

	_, err := s.db.Exec(query, runID, totalSteps, completedSteps, currentStepIndex, overallProgress, etaMs, now)
	if err != nil {
		return fmt.Errorf("failed to update pipeline progress: %w", err)
	}

	return nil
}

// GetPipelineProgress retrieves pipeline-level progress.
func (s *stateStore) GetPipelineProgress(runID string) (*PipelineProgressRecord, error) {
	query := `SELECT run_id, total_steps, completed_steps, current_step_index,
	                 overall_progress, estimated_completion_ms, updated_at
	          FROM pipeline_progress
	          WHERE run_id = ?`

	var record PipelineProgressRecord
	var updatedAt int64
	var estimatedCompletionMs sql.NullInt64

	err := s.db.QueryRow(query, runID).Scan(
		&record.RunID,
		&record.TotalSteps,
		&record.CompletedSteps,
		&record.CurrentStepIndex,
		&record.OverallProgress,
		&estimatedCompletionMs,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pipeline progress not found: %s", runID)
		}
		return nil, fmt.Errorf("failed to get pipeline progress: %w", err)
	}

	record.UpdatedAt = time.Unix(updatedAt, 0)
	if estimatedCompletionMs.Valid {
		record.EstimatedCompletionMs = estimatedCompletionMs.Int64
	}

	return &record, nil
}
