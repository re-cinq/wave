package state

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

func (s *stateStore) CreateRun(pipelineName string, input string) (string, error) {
	return s.CreateRunWithLimit(pipelineName, input, 0)
}

// CreateRunWithLimit creates a new run, atomically enforcing a concurrency limit.
// If maxConcurrent > 0, the INSERT is rejected when the limit is reached.
// Returns ErrConcurrencyLimit when the limit is hit.
func (s *stateStore) CreateRunWithLimit(pipelineName string, input string, maxConcurrent int) (string, error) {
	now := s.now()
	randBytes := make([]byte, 2)
	if _, err := rand.Read(randBytes); err != nil {
		randBytes = []byte{byte(now.Nanosecond() >> 8), byte(now.Nanosecond())}
	}
	suffix := hex.EncodeToString(randBytes)
	runID := fmt.Sprintf("%s-%s-%s", pipelineName, now.Format("20060102-150405"), suffix)

	if maxConcurrent > 0 {
		// Atomic check-and-insert within a transaction
		tx, err := s.db.Begin()
		if err != nil {
			return "", fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() { _ = tx.Rollback() }()

		var count int
		err = tx.QueryRow(`SELECT COUNT(*) FROM pipeline_run WHERE status IN ('running', 'pending') AND started_at > unixepoch() - 300`).Scan(&count)
		if err != nil {
			return "", fmt.Errorf("failed to count running runs: %w", err)
		}
		if count >= maxConcurrent {
			return "", ErrConcurrencyLimit
		}

		_, err = tx.Exec(`INSERT INTO pipeline_run (run_id, pipeline_name, status, input, started_at)
		                   VALUES (?, ?, 'pending', ?, ?)`, runID, pipelineName, input, now.Unix())
		if err != nil {
			return "", fmt.Errorf("failed to create run: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return "", fmt.Errorf("failed to commit run: %w", err)
		}
		return runID, nil
	}

	// No limit — simple insert
	_, err := s.db.Exec(`INSERT INTO pipeline_run (run_id, pipeline_name, status, input, started_at)
	                      VALUES (?, ?, 'pending', ?, ?)`, runID, pipelineName, input, now.Unix())
	if err != nil {
		return "", fmt.Errorf("failed to create run: %w", err)
	}
	return runID, nil
}

// ErrConcurrencyLimit is returned when max_concurrent_workers is reached.
var ErrConcurrencyLimit = fmt.Errorf("concurrency limit reached")

// UpdateRunStatus updates the status, current step, and token count for a run.
// Sets completed_at if status is completed, failed, or cancelled.
func (s *stateStore) UpdateRunStatus(runID string, status string, currentStep string, tokens int) error {
	now := time.Now().Unix()

	var completedAt *int64
	var cancelledAt *int64
	if status == "completed" || status == "failed" {
		completedAt = &now
	}
	if status == "cancelled" {
		cancelledAt = &now
		completedAt = &now
	}

	query := `UPDATE pipeline_run
	          SET status = ?, current_step = ?, total_tokens = ?, completed_at = ?, cancelled_at = ?
	          WHERE run_id = ?`

	result, err := s.db.Exec(query, status, currentStep, tokens, completedAt, cancelledAt, runID)
	if err != nil {
		return fmt.Errorf("failed to update run status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("run not found: %s", runID)
	}

	return nil
}

// UpdateRunBranch updates the branch_name for a pipeline run.
func (s *stateStore) UpdateRunBranch(runID string, branch string) error {
	query := `UPDATE pipeline_run SET branch_name = ? WHERE run_id = ?`
	result, err := s.db.Exec(query, branch, runID)
	if err != nil {
		return fmt.Errorf("failed to update run branch: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("run not found: %s", runID)
	}
	return nil
}

// UpdateRunPID sets the OS process ID for a detached pipeline run.
func (s *stateStore) UpdateRunPID(runID string, pid int) error {
	query := `UPDATE pipeline_run SET pid = ? WHERE run_id = ?`
	_, err := s.db.Exec(query, pid, runID)
	if err != nil {
		return fmt.Errorf("failed to update run PID: %w", err)
	}
	return nil
}

// UpdateRunHeartbeat refreshes the last_heartbeat timestamp for a running
// pipeline. The reconciler reads this column to distinguish runs whose owning
// process is still alive from runs whose process died without updating the DB.
func (s *stateStore) UpdateRunHeartbeat(runID string) error {
	query := `UPDATE pipeline_run SET last_heartbeat = ? WHERE run_id = ?`
	_, err := s.db.Exec(query, s.now().Unix(), runID)
	if err != nil {
		return fmt.Errorf("failed to update run heartbeat: %w", err)
	}
	return nil
}

// ReapOrphans marks every "running" pipeline whose last_heartbeat is older
// than staleAfter (or has never reported a heartbeat AND started more than
// staleAfter ago) as failed with reason "orphaned (no heartbeat)". Returns
// the number of rows transitioned. Issue #1467 — fixes the dead-process /
// stale-DB-row leak where host sleep / sandbox cycle / SIGKILL skipped the
// deferred UpdateRunStatus and left max_concurrent_workers wedged.
func (s *stateStore) ReapOrphans(staleAfter time.Duration) (int, error) {
	now := s.now().Unix()
	cutoff := now - int64(staleAfter.Seconds())

	query := `UPDATE pipeline_run
	          SET status = 'failed',
	              error_message = COALESCE(NULLIF(error_message, ''), 'orphaned (no heartbeat for ' || ? || 's)'),
	              completed_at = ?
	          WHERE status = 'running'
	            AND started_at < ?
	            AND (
	                  (last_heartbeat IS NOT NULL AND last_heartbeat > 0 AND last_heartbeat < ?)
	               OR (last_heartbeat IS NULL OR last_heartbeat = 0)
	            )`

	result, err := s.db.Exec(query, int64(staleAfter.Seconds()), now, cutoff, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to reap orphans: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return int(rows), nil
}

// GetRun retrieves a single run record by ID.
func (s *stateStore) GetRun(runID string) (*RunRecord, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message, tags_json, branch_name, pid,
	                 parent_run_id, parent_step_id, forked_from_run_id, last_heartbeat,
	                 iterate_index, iterate_total, iterate_mode, run_kind, sub_pipeline_ref
	          FROM pipeline_run
	          WHERE run_id = ?`

	var record RunRecord
	var startedAt int64
	var completedAt, cancelledAt sql.NullInt64
	var input, currentStep, errorMessage, tagsJSON, branchName sql.NullString
	var pid sql.NullInt64
	var parentRunID, parentStepID, forkedFromRunID sql.NullString
	var lastHeartbeat int64
	var iterateIndex, iterateTotal sql.NullInt64
	var iterateMode, runKind, subPipelineRef sql.NullString

	err := s.db.QueryRow(query, runID).Scan(
		&record.RunID,
		&record.PipelineName,
		&record.Status,
		&input,
		&currentStep,
		&record.TotalTokens,
		&startedAt,
		&completedAt,
		&cancelledAt,
		&errorMessage,
		&tagsJSON,
		&branchName,
		&pid,
		&parentRunID,
		&parentStepID,
		&forkedFromRunID,
		&lastHeartbeat,
		&iterateIndex,
		&iterateTotal,
		&iterateMode,
		&runKind,
		&subPipelineRef,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("run not found: %s", runID)
		}
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	record.StartedAt = time.Unix(startedAt, 0)
	if input.Valid {
		record.Input = input.String
	}
	if currentStep.Valid {
		record.CurrentStep = currentStep.String
	}
	if completedAt.Valid {
		t := time.Unix(completedAt.Int64, 0)
		record.CompletedAt = &t
	}
	if cancelledAt.Valid {
		t := time.Unix(cancelledAt.Int64, 0)
		record.CancelledAt = &t
	}
	if errorMessage.Valid {
		record.ErrorMessage = errorMessage.String
	}
	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &record.Tags); err != nil {
			// If JSON parsing fails, treat as empty tags
			record.Tags = []string{}
		}
	}
	if branchName.Valid {
		record.BranchName = branchName.String
	}
	if pid.Valid {
		record.PID = int(pid.Int64)
	}
	if parentRunID.Valid {
		record.ParentRunID = parentRunID.String
	}
	if parentStepID.Valid {
		record.ParentStepID = parentStepID.String
	}
	if forkedFromRunID.Valid {
		record.ForkedFromRunID = forkedFromRunID.String
	}
	if lastHeartbeat > 0 {
		record.LastHeartbeat = time.Unix(lastHeartbeat, 0)
	}
	if iterateIndex.Valid {
		v := int(iterateIndex.Int64)
		record.IterateIndex = &v
	}
	if iterateTotal.Valid {
		v := int(iterateTotal.Int64)
		record.IterateTotal = &v
	}
	if iterateMode.Valid {
		record.IterateMode = iterateMode.String
	}
	if runKind.Valid {
		record.RunKind = runKind.String
	}
	if subPipelineRef.Valid {
		record.SubPipelineRef = subPipelineRef.String
	}

	return &record, nil
}

// GetRunningRuns returns all runs with status 'running'.
func (s *stateStore) GetRunningRuns() ([]RunRecord, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message, tags_json, branch_name, pid,
	                 parent_run_id, parent_step_id, forked_from_run_id, last_heartbeat,
	                 iterate_index, iterate_total, iterate_mode, run_kind, sub_pipeline_ref
	          FROM pipeline_run
	          WHERE (status = 'running' OR (status = 'pending' AND started_at > unixepoch() - 300))
	          ORDER BY started_at DESC`

	return s.queryRuns(query)
}

// ListRuns returns runs matching the specified options.
func (s *stateStore) ListRuns(opts ListRunsOptions) ([]RunRecord, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message, tags_json, branch_name, pid,
	                 parent_run_id, parent_step_id, forked_from_run_id, last_heartbeat,
	                 iterate_index, iterate_total, iterate_mode, run_kind, sub_pipeline_ref
	          FROM pipeline_run
	          WHERE 1=1`
	args := []any{}

	if opts.PipelineName != "" {
		query += " AND pipeline_name = ?"
		args = append(args, opts.PipelineName)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}
	if opts.OlderThan > 0 {
		cutoff := s.now().Add(-opts.OlderThan).Unix()
		query += " AND started_at < ?"
		args = append(args, cutoff)
	}
	// Filter by tags - run must have at least one of the specified tags
	if len(opts.Tags) > 0 {
		// Use SQLite's json_each to search within tags_json array
		query += " AND ("
		for i, tag := range opts.Tags {
			if i > 0 {
				query += " OR "
			}
			query += "EXISTS (SELECT 1 FROM json_each(tags_json) WHERE json_each.value = ?)"
			args = append(args, tag)
		}
		query += ")"
	}

	if opts.SinceUnix > 0 {
		query += " AND started_at >= ?"
		args = append(args, opts.SinceUnix)
	}

	if opts.TopLevelOnly {
		query += " AND (parent_run_id IS NULL OR parent_run_id = '')"
	}

	// Cursor-based pagination: return runs before the cursor position
	if opts.BeforeUnix > 0 {
		if opts.BeforeRunID != "" {
			query += " AND (started_at < ? OR (started_at = ? AND run_id < ?))"
			args = append(args, opts.BeforeUnix, opts.BeforeUnix, opts.BeforeRunID)
		} else {
			query += " AND started_at < ?"
			args = append(args, opts.BeforeUnix)
		}
	}

	query += " ORDER BY started_at DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	return s.queryRunsWithArgs(query, args...)
}

// GetMostRecentRunID returns the run_id with the most recent started_at.
// Returns ("", nil) when no runs exist so callers can switch on empty string
// without depending on database/sql sentinel errors.
func (s *stateStore) GetMostRecentRunID() (string, error) {
	var runID string
	err := s.db.QueryRow(
		`SELECT run_id FROM pipeline_run ORDER BY started_at DESC, run_id DESC LIMIT 1`,
	).Scan(&runID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to query most recent run: %w", err)
	}
	return runID, nil
}

// RunExists reports whether a run with the given ID exists.
func (s *stateStore) RunExists(runID string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM pipeline_run WHERE run_id = ?`, runID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check run existence: %w", err)
	}
	return count > 0, nil
}

// GetRunStatus returns the status of a run.
// Returns ("", nil) when the run does not exist.
func (s *stateStore) GetRunStatus(runID string) (string, error) {
	var status string
	err := s.db.QueryRow(
		`SELECT status FROM pipeline_run WHERE run_id = ?`, runID,
	).Scan(&status)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to query run status: %w", err)
	}
	return status, nil
}

// ListPipelineNamesByStatus returns distinct pipeline names whose status matches
// the given status (case-insensitive). Falls back to pipeline_state if pipeline_run
// query fails (legacy schema compatibility).
func (s *stateStore) ListPipelineNamesByStatus(status string) ([]string, error) {
	names, err := s.listDistinctPipelineNames(
		`SELECT DISTINCT pipeline_name FROM pipeline_run WHERE LOWER(status) = LOWER(?)`,
		status,
	)
	if err == nil {
		return names, nil
	}
	// Fallback for legacy/partial schemas
	return s.listDistinctPipelineNames(
		`SELECT DISTINCT pipeline_name FROM pipeline_state WHERE LOWER(status) = LOWER(?)`,
		status,
	)
}

func (s *stateStore) listDistinctPipelineNames(query, status string) ([]string, error) {
	rows, err := s.db.Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query pipeline names by status: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pipeline names: %w", err)
	}
	return names, nil
}

// BackfillRunTokens updates pipeline_run.total_tokens from event_log for
// finalized runs that still have 0 tokens. Idempotent — re-running yields 0
// affected rows once all runs have been backfilled.
func (s *stateStore) BackfillRunTokens() (int64, error) {
	result, err := s.db.Exec(`
		UPDATE pipeline_run SET total_tokens = (
			SELECT COALESCE(SUM(el.tokens_used), 0)
			FROM event_log el
			WHERE el.run_id = pipeline_run.run_id AND el.tokens_used > 0
		)
		WHERE total_tokens = 0
		AND status IN ('completed', 'failed', 'cancelled')
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to backfill run tokens: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n, nil
}

// DeleteRun removes a run and its associated events, artifacts, and cancellation records.
func (s *stateStore) DeleteRun(runID string) error {
	// Due to foreign key ON DELETE CASCADE, deleting from pipeline_run
	// will automatically delete related event_log, artifact, and cancellation records.
	query := `DELETE FROM pipeline_run WHERE run_id = ?`

	result, err := s.db.Exec(query, runID)
	if err != nil {
		return fmt.Errorf("failed to delete run: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("run not found: %s", runID)
	}

	return nil
}

// LogEvent records an event in the event_log table.

func (s *stateStore) queryRuns(query string) ([]RunRecord, error) {
	return s.queryRunsWithArgs(query)
}

func (s *stateStore) queryRunsWithArgs(query string, args ...any) ([]RunRecord, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query runs: %w", err)
	}
	defer rows.Close()

	var records []RunRecord
	for rows.Next() {
		var record RunRecord
		var startedAt int64
		var completedAt, cancelledAt sql.NullInt64
		var input, currentStep, errorMessage, tagsJSON, branchName sql.NullString
		var pid sql.NullInt64
		var parentRunID, parentStepID, forkedFromRunID sql.NullString
		var lastHeartbeat int64
		var iterateIndex, iterateTotal sql.NullInt64
		var iterateMode, runKind, subPipelineRef sql.NullString

		err := rows.Scan(
			&record.RunID,
			&record.PipelineName,
			&record.Status,
			&input,
			&currentStep,
			&record.TotalTokens,
			&startedAt,
			&completedAt,
			&cancelledAt,
			&errorMessage,
			&tagsJSON,
			&branchName,
			&pid,
			&parentRunID,
			&parentStepID,
			&forkedFromRunID,
			&lastHeartbeat,
			&iterateIndex,
			&iterateTotal,
			&iterateMode,
			&runKind,
			&subPipelineRef,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan run: %w", err)
		}

		record.StartedAt = time.Unix(startedAt, 0)
		if input.Valid {
			record.Input = input.String
		}
		if currentStep.Valid {
			record.CurrentStep = currentStep.String
		}
		if completedAt.Valid {
			t := time.Unix(completedAt.Int64, 0)
			record.CompletedAt = &t
		}
		if cancelledAt.Valid {
			t := time.Unix(cancelledAt.Int64, 0)
			record.CancelledAt = &t
		}
		if errorMessage.Valid {
			record.ErrorMessage = errorMessage.String
		}
		if tagsJSON.Valid && tagsJSON.String != "" {
			if err := json.Unmarshal([]byte(tagsJSON.String), &record.Tags); err != nil {
				// If JSON parsing fails, treat as empty tags
				record.Tags = []string{}
			}
		}
		if branchName.Valid {
			record.BranchName = branchName.String
		}
		if pid.Valid {
			record.PID = int(pid.Int64)
		}
		if parentRunID.Valid {
			record.ParentRunID = parentRunID.String
		}
		if parentStepID.Valid {
			record.ParentStepID = parentStepID.String
		}
		if forkedFromRunID.Valid {
			record.ForkedFromRunID = forkedFromRunID.String
		}
		if lastHeartbeat > 0 {
			record.LastHeartbeat = time.Unix(lastHeartbeat, 0)
		}
		if iterateIndex.Valid {
			v := int(iterateIndex.Int64)
			record.IterateIndex = &v
		}
		if iterateTotal.Valid {
			v := int(iterateTotal.Int64)
			record.IterateTotal = &v
		}
		if iterateMode.Valid {
			record.IterateMode = iterateMode.String
		}
		if runKind.Valid {
			record.RunKind = runKind.String
		}
		if subPipelineRef.Valid {
			record.SubPipelineRef = subPipelineRef.String
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating runs: %w", err)
	}

	return records, nil
}

func (s *stateStore) SetRunTags(runID string, tags []string) error {
	// Ensure tags is not nil for JSON encoding
	if tags == nil {
		tags = []string{}
	}

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `UPDATE pipeline_run SET tags_json = ? WHERE run_id = ?`

	result, err := s.db.Exec(query, string(tagsJSON), runID)
	if err != nil {
		return fmt.Errorf("failed to set run tags: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("run not found: %s", runID)
	}

	return nil
}

// GetRunTags retrieves the tags for a pipeline run.
func (s *stateStore) GetRunTags(runID string) ([]string, error) {
	query := `SELECT tags_json FROM pipeline_run WHERE run_id = ?`

	var tagsJSON sql.NullString
	err := s.db.QueryRow(query, runID).Scan(&tagsJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("run not found: %s", runID)
		}
		return nil, fmt.Errorf("failed to get run tags: %w", err)
	}

	if !tagsJSON.Valid || tagsJSON.String == "" {
		return []string{}, nil
	}

	var tags []string
	if err := json.Unmarshal([]byte(tagsJSON.String), &tags); err != nil {
		return []string{}, nil
	}

	return tags, nil
}

// AddRunTag adds a tag to a pipeline run if it doesn't already exist.
func (s *stateStore) AddRunTag(runID string, tag string) error {
	// Get current tags
	tags, err := s.GetRunTags(runID)
	if err != nil {
		return err
	}

	// Check if tag already exists
	for _, existingTag := range tags {
		if existingTag == tag {
			return nil // Tag already exists, nothing to do
		}
	}

	// Add the new tag
	tags = append(tags, tag)

	return s.SetRunTags(runID, tags)
}

// RemoveRunTag removes a tag from a pipeline run.
func (s *stateStore) RemoveRunTag(runID string, tag string) error {
	// Get current tags
	tags, err := s.GetRunTags(runID)
	if err != nil {
		return err
	}

	// Filter out the tag to remove
	newTags := make([]string, 0, len(tags))
	for _, existingTag := range tags {
		if existingTag != tag {
			newTags = append(newTags, existingTag)
		}
	}

	return s.SetRunTags(runID, newTags)
}

// RecordStepAttempt inserts a step attempt record into the step_attempt table.
func (s *stateStore) RecordStepAttempt(record *StepAttemptRecord) error {
	var completedAt *int64
	if record.CompletedAt != nil {
		t := record.CompletedAt.Unix()
		completedAt = &t
	}
	_, err := s.db.Exec(
		`INSERT INTO step_attempt (run_id, step_id, attempt, state, error_message, failure_class, stdout_tail, tokens_used, duration_ms, started_at, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.RunID, record.StepID, record.Attempt, record.State, record.ErrorMessage, record.FailureClass, record.StdoutTail, record.TokensUsed, record.DurationMs, record.StartedAt.Unix(), completedAt,
	)
	return err
}

// GetStepAttempts retrieves all attempt records for a step, ordered by attempt number.
func (s *stateStore) GetStepAttempts(runID string, stepID string) ([]StepAttemptRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, step_id, attempt, state, error_message, failure_class, stdout_tail, tokens_used, duration_ms, started_at, completed_at FROM step_attempt WHERE run_id = ? AND step_id = ? ORDER BY attempt ASC`,
		runID, stepID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []StepAttemptRecord
	for rows.Next() {
		var r StepAttemptRecord
		var startedAt int64
		var completedAtNull *int64
		err := rows.Scan(&r.ID, &r.RunID, &r.StepID, &r.Attempt, &r.State, &r.ErrorMessage, &r.FailureClass, &r.StdoutTail, &r.TokensUsed, &r.DurationMs, &startedAt, &completedAtNull)
		if err != nil {
			return nil, err
		}
		r.StartedAt = time.Unix(startedAt, 0)
		if completedAtNull != nil {
			t := time.Unix(*completedAtNull, 0)
			r.CompletedAt = &t
		}
		records = append(records, r)
	}
	return records, nil
}
