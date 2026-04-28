package state

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// --- Checkpoint tracking (fork/rewind) ---

func (s *stateStore) SaveCheckpoint(record *CheckpointRecord) error {
	now := time.Now().Unix()

	// Upsert: replace existing checkpoint for same run+step
	query := `INSERT INTO checkpoint (run_id, step_id, step_index, workspace_path, workspace_commit_sha, artifact_snapshot, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(run_id, step_id) DO UPDATE SET
	              step_index = excluded.step_index,
	              workspace_path = excluded.workspace_path,
	              workspace_commit_sha = excluded.workspace_commit_sha,
	              artifact_snapshot = excluded.artifact_snapshot,
	              created_at = excluded.created_at`

	_, err := s.db.Exec(query, record.RunID, record.StepID, record.StepIndex, record.WorkspacePath, record.WorkspaceCommitSHA, record.ArtifactSnapshot, now)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}
	return nil
}

func (s *stateStore) GetCheckpoint(runID, stepID string) (*CheckpointRecord, error) {
	query := `SELECT id, run_id, step_id, step_index, workspace_path, workspace_commit_sha, artifact_snapshot, created_at
	          FROM checkpoint
	          WHERE run_id = ? AND step_id = ?`

	var record CheckpointRecord
	var createdAt int64
	var sha sql.NullString

	err := s.db.QueryRow(query, runID, stepID).Scan(
		&record.ID, &record.RunID, &record.StepID, &record.StepIndex,
		&record.WorkspacePath, &sha, &record.ArtifactSnapshot, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("checkpoint not found for run %s step %s", runID, stepID)
		}
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}

	if sha.Valid {
		record.WorkspaceCommitSHA = sha.String
	}
	record.CreatedAt = time.Unix(createdAt, 0)
	return &record, nil
}

func (s *stateStore) GetCheckpoints(runID string) ([]CheckpointRecord, error) {
	query := `SELECT id, run_id, step_id, step_index, workspace_path, workspace_commit_sha, artifact_snapshot, created_at
	          FROM checkpoint
	          WHERE run_id = ?
	          ORDER BY step_index ASC`

	rows, err := s.db.Query(query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to query checkpoints: %w", err)
	}
	defer rows.Close()

	var records []CheckpointRecord
	for rows.Next() {
		var record CheckpointRecord
		var createdAt int64
		var sha sql.NullString

		err := rows.Scan(
			&record.ID, &record.RunID, &record.StepID, &record.StepIndex,
			&record.WorkspacePath, &sha, &record.ArtifactSnapshot, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checkpoint: %w", err)
		}

		if sha.Valid {
			record.WorkspaceCommitSHA = sha.String
		}
		record.CreatedAt = time.Unix(createdAt, 0)
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating checkpoints: %w", err)
	}
	return records, nil
}

func (s *stateStore) DeleteCheckpointsAfterStep(runID string, stepIndex int) error {
	query := `DELETE FROM checkpoint WHERE run_id = ? AND step_index > ?`
	_, err := s.db.Exec(query, runID, stepIndex)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoints after step index %d: %w", stepIndex, err)
	}
	return nil
}

func (s *stateStore) CreateRunWithFork(pipelineName, input, forkedFromRunID string) (string, error) {
	now := time.Now()
	randBytes := make([]byte, 2)
	if _, err := rand.Read(randBytes); err != nil {
		randBytes = []byte{byte(now.Nanosecond() >> 8), byte(now.Nanosecond())}
	}
	suffix := hex.EncodeToString(randBytes)
	runID := fmt.Sprintf("%s-%s-%s", pipelineName, now.Format("20060102-150405"), suffix)

	query := `INSERT INTO pipeline_run (run_id, pipeline_name, status, input, started_at, forked_from_run_id)
	          VALUES (?, ?, 'pending', ?, ?, ?)`

	_, err := s.db.Exec(query, runID, pipelineName, input, now.Unix(), forkedFromRunID)
	if err != nil {
		return "", fmt.Errorf("failed to create forked run: %w", err)
	}
	return runID, nil
}

// =============================================================================
// Parent-Child Run Linkage
// =============================================================================

// SetParentRun sets the parent run ID and step ID on a child run record.
func (s *stateStore) SetParentRun(childRunID, parentRunID, stepID string) error {
	query := `UPDATE pipeline_run SET parent_run_id = ?, parent_step_id = ? WHERE run_id = ?`

	result, err := s.db.Exec(query, parentRunID, stepID, childRunID)
	if err != nil {
		return fmt.Errorf("failed to set parent run: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("run not found: %s", childRunID)
	}

	return nil
}

// SetRunComposition records the composition metadata for a child run —
// run kind, sub-pipeline reference, and per-iterate-item index/total/mode.
// Issue #1450 — used by the WebUI to render parent → step → item-N
// breadcrumbs and run-kind chips without re-deriving from event_log.
//
// Pass nil for iterateIndex / iterateTotal when the launch was not an
// iterate item. Empty strings for iterateMode / subPipelineRef are valid
// for non-iterate / non-sub-pipeline launches respectively.
func (s *stateStore) SetRunComposition(childRunID, runKind, subPipelineRef, iterateMode string, iterateIndex, iterateTotal *int) error {
	query := `UPDATE pipeline_run
	          SET run_kind = ?, sub_pipeline_ref = ?, iterate_mode = ?,
	              iterate_index = ?, iterate_total = ?
	          WHERE run_id = ?`

	var idxArg, totalArg any
	if iterateIndex != nil {
		idxArg = *iterateIndex
	}
	if iterateTotal != nil {
		totalArg = *iterateTotal
	}

	result, err := s.db.Exec(query, runKind, subPipelineRef, iterateMode, idxArg, totalArg, childRunID)
	if err != nil {
		return fmt.Errorf("failed to set run composition metadata: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("run not found: %s", childRunID)
	}

	return nil
}

// GetSubtreeTokens walks parent_run_id recursively from rootRunID and
// sums total_tokens across the root + every descendant. Used by the
// WebUI to display subtree-rolled-up cost on parent runs (issue #1450).
//
// Returns 0 with no error when the run has no children — a single-row
// rollup equals the root's own total_tokens.
func (s *stateStore) GetSubtreeTokens(rootRunID string) (int64, error) {
	query := `WITH RECURSIVE subtree(run_id) AS (
	    SELECT run_id FROM pipeline_run WHERE run_id = ?
	  UNION ALL
	    SELECT pr.run_id FROM pipeline_run pr
	    JOIN subtree s ON pr.parent_run_id = s.run_id
	)
	SELECT COALESCE(SUM(pr.total_tokens), 0)
	FROM pipeline_run pr
	JOIN subtree s ON pr.run_id = s.run_id`

	var total sql.NullInt64
	if err := s.db.QueryRow(query, rootRunID).Scan(&total); err != nil {
		return 0, fmt.Errorf("failed to compute subtree tokens: %w", err)
	}
	if total.Valid {
		return total.Int64, nil
	}
	return 0, nil
}

// GetChildRuns returns all runs that are children of the specified parent run,
// ordered by started_at.
func (s *stateStore) GetChildRuns(parentRunID string) ([]RunRecord, error) {
	// Sort by iterate_index (NULLS LAST handled by COALESCE), then started_at,
	// so iterate-children render in their YAML-defined order rather than
	// goroutine-launch order. Issue #1450.
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message, tags_json, branch_name, pid,
	                 parent_run_id, parent_step_id, forked_from_run_id, last_heartbeat,
	                 iterate_index, iterate_total, iterate_mode, run_kind, sub_pipeline_ref
	          FROM pipeline_run
	          WHERE parent_run_id = ?
	          ORDER BY COALESCE(iterate_index, 1<<30) ASC, started_at ASC`

	return s.queryRunsWithArgs(query, parentRunID)
}
