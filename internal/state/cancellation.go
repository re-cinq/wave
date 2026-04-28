package state

import (
	"database/sql"
	"fmt"
	"time"
)

func (s *stateStore) RequestCancellation(runID string, force bool) error {
	now := time.Now().Unix()

	query := `INSERT INTO cancellation (run_id, requested_at, force)
	          VALUES (?, ?, ?)
	          ON CONFLICT(run_id) DO UPDATE SET
	              requested_at = excluded.requested_at,
	              force = excluded.force`

	_, err := s.db.Exec(query, runID, now, force)
	if err != nil {
		return fmt.Errorf("failed to request cancellation: %w", err)
	}

	// Force cancel: directly mark the run and all its running steps as cancelled.
	// This handles orphaned runs whose process is no longer running.
	if force {
		if err := s.UpdateRunStatus(runID, "cancelled", "", 0); err != nil {
			return fmt.Errorf("failed to force-cancel run: %w", err)
		}
		// Also cancel all running/pending steps
		_, _ = s.db.Exec(`UPDATE step_state SET state = 'cancelled', completed_at = ? WHERE pipeline_id = ? AND state IN ('running', 'pending', 'started')`, now, runID)
	}

	return nil
}

// CheckCancellation checks if a cancellation has been requested for a run.
// Returns nil if no cancellation is pending.
func (s *stateStore) CheckCancellation(runID string) (*CancellationRecord, error) {
	query := `SELECT run_id, requested_at, force
	          FROM cancellation
	          WHERE run_id = ?`

	var record CancellationRecord
	var requestedAt int64

	err := s.db.QueryRow(query, runID).Scan(
		&record.RunID,
		&requestedAt,
		&record.Force,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to check cancellation: %w", err)
	}

	record.RequestedAt = time.Unix(requestedAt, 0)

	return &record, nil
}

// ClearCancellation removes the cancellation flag for a run.
func (s *stateStore) ClearCancellation(runID string) error {
	query := `DELETE FROM cancellation WHERE run_id = ?`

	_, err := s.db.Exec(query, runID)
	if err != nil {
		return fmt.Errorf("failed to clear cancellation: %w", err)
	}

	return nil
}
