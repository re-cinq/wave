package metrics

import (
	"database/sql"
	"fmt"
	"time"
)

// DB is the minimal database surface the metrics Store needs. *sql.DB and
// *sql.Tx satisfy it implicitly. Defining it here keeps the metrics package
// free of any dependency on internal/state.
type DB interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Store provides query/write access to the performance_metric and
// retrospective tables. Schema migrations for those tables are owned by
// internal/state's migration runner; this Store is a query layer only.
type Store struct {
	db DB
}

// NewStore constructs a Store backed by the given DB handle.
func NewStore(db DB) *Store {
	return &Store{db: db}
}

// RecordPerformanceMetric persists a single performance metric row and sets
// the metric.ID field on success.
func (s *Store) RecordPerformanceMetric(metric *PerformanceMetricRecord) error {
	startedAt := metric.StartedAt.Unix()
	var completedAt *int64
	if metric.CompletedAt != nil {
		ca := metric.CompletedAt.Unix()
		completedAt = &ca
	}

	query := `INSERT INTO performance_metric (
	              run_id, step_id, pipeline_name, persona, started_at, completed_at,
	              duration_ms, tokens_used, files_modified, artifacts_generated,
	              memory_bytes, success, error_message
	          ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := s.db.Exec(
		query,
		metric.RunID,
		metric.StepID,
		metric.PipelineName,
		metric.Persona,
		startedAt,
		completedAt,
		metric.DurationMs,
		metric.TokensUsed,
		metric.FilesModified,
		metric.ArtifactsGenerated,
		metric.MemoryBytes,
		metric.Success,
		metric.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to record performance metric: %w", err)
	}

	if id, err := result.LastInsertId(); err == nil {
		metric.ID = id
	}

	return nil
}

// GetPerformanceMetrics retrieves performance metrics for a run, optionally filtered by step.
func (s *Store) GetPerformanceMetrics(runID string, stepID string) ([]PerformanceMetricRecord, error) {
	query := `SELECT id, run_id, step_id, pipeline_name, persona, started_at, completed_at,
	                 duration_ms, tokens_used, files_modified, artifacts_generated,
	                 memory_bytes, success, error_message
	          FROM performance_metric
	          WHERE run_id = ?`
	args := []any{runID}

	if stepID != "" {
		query += " AND step_id = ?"
		args = append(args, stepID)
	}

	query += " ORDER BY started_at ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query performance metrics: %w", err)
	}
	defer rows.Close()

	var metrics []PerformanceMetricRecord
	for rows.Next() {
		metric, err := scanMetricRow(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating performance metrics: %w", err)
	}

	return metrics, nil
}

// GetStepPerformanceStats retrieves aggregated performance statistics for a step.
func (s *Store) GetStepPerformanceStats(pipelineName string, stepID string, since time.Time) (*StepPerformanceStats, error) {
	query := `SELECT
	              COUNT(*) as total_runs,
	              SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful_runs,
	              SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failed_runs,
	              AVG(duration_ms) as avg_duration,
	              MIN(duration_ms) as min_duration,
	              MAX(duration_ms) as max_duration,
	              AVG(tokens_used) as avg_tokens,
	              SUM(tokens_used) as total_tokens,
	              AVG(files_modified) as avg_files,
	              AVG(artifacts_generated) as avg_artifacts,
	              MAX(started_at) as last_run,
	              persona
	          FROM performance_metric
	          WHERE pipeline_name = ? AND step_id = ? AND started_at >= ?
	          GROUP BY step_id, persona`

	var stats StepPerformanceStats
	var lastRun int64
	var avgDuration, avgTokens, avgFiles, avgArtifacts sql.NullFloat64
	var minDuration, maxDuration, totalTokens sql.NullInt64
	var persona sql.NullString

	err := s.db.QueryRow(query, pipelineName, stepID, since.Unix()).Scan(
		&stats.TotalRuns,
		&stats.SuccessfulRuns,
		&stats.FailedRuns,
		&avgDuration,
		&minDuration,
		&maxDuration,
		&avgTokens,
		&totalTokens,
		&avgFiles,
		&avgArtifacts,
		&lastRun,
		&persona,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &StepPerformanceStats{
				StepID: stepID,
			}, nil
		}
		return nil, fmt.Errorf("failed to get step performance stats: %w", err)
	}

	stats.StepID = stepID
	if persona.Valid {
		stats.Persona = persona.String
	}
	if avgDuration.Valid {
		stats.AvgDurationMs = int64(avgDuration.Float64)
	}
	if minDuration.Valid {
		stats.MinDurationMs = minDuration.Int64
	}
	if maxDuration.Valid {
		stats.MaxDurationMs = maxDuration.Int64
	}
	if avgTokens.Valid {
		stats.AvgTokensUsed = int(avgTokens.Float64)
	}
	if totalTokens.Valid {
		stats.TotalTokensUsed = int(totalTokens.Int64)
	}
	if avgFiles.Valid {
		stats.AvgFilesModified = int(avgFiles.Float64)
	}
	if avgArtifacts.Valid {
		stats.AvgArtifacts = int(avgArtifacts.Float64)
	}
	stats.LastRunAt = time.Unix(lastRun, 0)

	if stats.AvgDurationMs > 0 && stats.AvgTokensUsed > 0 {
		stats.TokenBurnRate = float64(stats.AvgTokensUsed) / (float64(stats.AvgDurationMs) / 1000.0)
	}

	return &stats, nil
}

// GetRecentPerformanceHistory retrieves recent performance metrics with optional filters.
func (s *Store) GetRecentPerformanceHistory(opts PerformanceQueryOptions) ([]PerformanceMetricRecord, error) {
	query := `SELECT id, run_id, step_id, pipeline_name, persona, started_at, completed_at,
	                 duration_ms, tokens_used, files_modified, artifacts_generated,
	                 memory_bytes, success, error_message
	          FROM performance_metric
	          WHERE 1=1`
	args := []any{}

	if opts.PipelineName != "" {
		query += " AND pipeline_name = ?"
		args = append(args, opts.PipelineName)
	}
	if opts.StepID != "" {
		query += " AND step_id = ?"
		args = append(args, opts.StepID)
	}
	if opts.Persona != "" {
		query += " AND persona = ?"
		args = append(args, opts.Persona)
	}
	if !opts.Since.IsZero() {
		query += " AND started_at >= ?"
		args = append(args, opts.Since.Unix())
	}

	query += " ORDER BY started_at DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query performance history: %w", err)
	}
	defer rows.Close()

	var metrics []PerformanceMetricRecord
	for rows.Next() {
		metric, err := scanMetricRow(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating performance history: %w", err)
	}

	return metrics, nil
}

// CleanupOldPerformanceMetrics removes performance metrics older than the
// specified duration. Returns the number of rows deleted.
func (s *Store) CleanupOldPerformanceMetrics(olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan).Unix()

	result, err := s.db.Exec(`DELETE FROM performance_metric WHERE started_at < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old performance metrics: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rows), nil
}

// scanMetricRow decodes one performance_metric row into a record.
func scanMetricRow(rows *sql.Rows) (PerformanceMetricRecord, error) {
	var metric PerformanceMetricRecord
	var startedAt int64
	var completedAt sql.NullInt64
	var persona, errorMessage sql.NullString
	var tokensUsed, filesModified, artifactsGenerated sql.NullInt64
	var memoryBytes, durationMs sql.NullInt64

	err := rows.Scan(
		&metric.ID,
		&metric.RunID,
		&metric.StepID,
		&metric.PipelineName,
		&persona,
		&startedAt,
		&completedAt,
		&durationMs,
		&tokensUsed,
		&filesModified,
		&artifactsGenerated,
		&memoryBytes,
		&metric.Success,
		&errorMessage,
	)
	if err != nil {
		return metric, fmt.Errorf("failed to scan performance metric: %w", err)
	}

	metric.StartedAt = time.Unix(startedAt, 0)
	if completedAt.Valid {
		t := time.Unix(completedAt.Int64, 0)
		metric.CompletedAt = &t
	}
	if persona.Valid {
		metric.Persona = persona.String
	}
	if durationMs.Valid {
		metric.DurationMs = durationMs.Int64
	}
	if tokensUsed.Valid {
		metric.TokensUsed = int(tokensUsed.Int64)
	}
	if filesModified.Valid {
		metric.FilesModified = int(filesModified.Int64)
	}
	if artifactsGenerated.Valid {
		metric.ArtifactsGenerated = int(artifactsGenerated.Int64)
	}
	if memoryBytes.Valid {
		metric.MemoryBytes = memoryBytes.Int64
	}
	if errorMessage.Valid {
		metric.ErrorMessage = errorMessage.String
	}

	return metric, nil
}

// SaveRetrospective saves a retrospective index record. Updates the row in
// place if one already exists for the run.
func (s *Store) SaveRetrospective(record *RetrospectiveRecord) error {
	var exists int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM retrospective WHERE run_id = ?", record.RunID).Scan(&exists)
	if exists > 0 {
		_, err := s.db.Exec(`
			UPDATE retrospective SET smoothness = ?, status = ?, file_path = ?
			WHERE run_id = ?
		`, record.Smoothness, record.Status, record.FilePath, record.RunID)
		return err
	}
	_, err := s.db.Exec(`
		INSERT INTO retrospective (run_id, pipeline_name, smoothness, status, file_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, record.RunID, record.PipelineName, record.Smoothness, record.Status, record.FilePath, record.CreatedAt.Unix())
	return err
}

// GetRetrospective retrieves a retrospective record by run ID.
func (s *Store) GetRetrospective(runID string) (*RetrospectiveRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, run_id, pipeline_name, smoothness, status, file_path, created_at
		FROM retrospective WHERE run_id = ?
	`, runID)

	var r RetrospectiveRecord
	var createdAt int64
	err := row.Scan(&r.ID, &r.RunID, &r.PipelineName, &r.Smoothness, &r.Status, &r.FilePath, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("retrospective not found for run %s: %w", runID, err)
	}
	r.CreatedAt = time.Unix(createdAt, 0)
	return &r, nil
}

// ListRetrospectives returns retrospectives matching the given filters.
func (s *Store) ListRetrospectives(opts ListRetrosOptions) ([]RetrospectiveRecord, error) {
	query := "SELECT id, run_id, pipeline_name, smoothness, status, file_path, created_at FROM retrospective WHERE 1=1"
	var args []any

	if opts.PipelineName != "" {
		query += " AND pipeline_name = ?"
		args = append(args, opts.PipelineName)
	}
	if opts.SinceUnix > 0 {
		query += " AND created_at >= ?"
		args = append(args, opts.SinceUnix)
	}
	query += " ORDER BY created_at DESC"
	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list retrospectives: %w", err)
	}
	defer rows.Close()

	var records []RetrospectiveRecord
	for rows.Next() {
		var r RetrospectiveRecord
		var createdAt int64
		if err := rows.Scan(&r.ID, &r.RunID, &r.PipelineName, &r.Smoothness, &r.Status, &r.FilePath, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan retrospective: %w", err)
		}
		r.CreatedAt = time.Unix(createdAt, 0)
		records = append(records, r)
	}
	return records, nil
}

// DeleteRetrospective removes a retrospective record by run ID.
func (s *Store) DeleteRetrospective(runID string) error {
	_, err := s.db.Exec("DELETE FROM retrospective WHERE run_id = ?", runID)
	return err
}

// UpdateRetrospectiveSmoothness updates the smoothness rating for a retrospective.
func (s *Store) UpdateRetrospectiveSmoothness(runID string, smoothness string) error {
	_, err := s.db.Exec("UPDATE retrospective SET smoothness = ? WHERE run_id = ?", smoothness, runID)
	return err
}

// UpdateRetrospectiveStatus updates the status for a retrospective.
func (s *Store) UpdateRetrospectiveStatus(runID string, status string) error {
	_, err := s.db.Exec("UPDATE retrospective SET status = ? WHERE run_id = ?", status, runID)
	return err
}
