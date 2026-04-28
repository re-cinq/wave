package state

import (
	"database/sql"
	"fmt"
	"time"
)

func (s *stateStore) RecordPerformanceMetric(metric *PerformanceMetricRecord) error {
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

	// Set the ID on the metric
	if id, err := result.LastInsertId(); err == nil {
		metric.ID = id
	}

	return nil
}

// GetPerformanceMetrics retrieves performance metrics for a run, optionally filtered by step.
func (s *stateStore) GetPerformanceMetrics(runID string, stepID string) ([]PerformanceMetricRecord, error) {
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
			return nil, fmt.Errorf("failed to scan performance metric: %w", err)
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

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating performance metrics: %w", err)
	}

	return metrics, nil
}

// GetStepPerformanceStats retrieves aggregated performance statistics for a step.
func (s *stateStore) GetStepPerformanceStats(pipelineName string, stepID string, since time.Time) (*StepPerformanceStats, error) {
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
			// No metrics found - return empty stats
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

	// Calculate token burn rate (tokens per second)
	if stats.AvgDurationMs > 0 && stats.AvgTokensUsed > 0 {
		stats.TokenBurnRate = float64(stats.AvgTokensUsed) / (float64(stats.AvgDurationMs) / 1000.0)
	}

	return &stats, nil
}

// GetRecentPerformanceHistory retrieves recent performance metrics with optional filters.
func (s *stateStore) GetRecentPerformanceHistory(opts PerformanceQueryOptions) ([]PerformanceMetricRecord, error) {
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
			return nil, fmt.Errorf("failed to scan performance metric: %w", err)
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

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating performance history: %w", err)
	}

	return metrics, nil
}

// CleanupOldPerformanceMetrics removes performance metrics older than the specified duration.
// Returns the number of metrics deleted.
func (s *stateStore) CleanupOldPerformanceMetrics(olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan).Unix()

	query := `DELETE FROM performance_metric WHERE started_at < ?`

	result, err := s.db.Exec(query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old performance metrics: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rows), nil
}
