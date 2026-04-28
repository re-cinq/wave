package state

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (s *stateStore) LogEvent(runID string, stepID string, state string, persona string, message string, tokens int, durationMs int64, model string, configuredModel string, adapter string) error {
	now := s.now().Unix()

	query := `INSERT INTO event_log (run_id, timestamp, step_id, state, persona, message, tokens_used, duration_ms, model, configured_model, adapter)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, runID, now, stepID, state, persona, message, tokens, durationMs, model, configuredModel, adapter)
	if err != nil {
		return fmt.Errorf("failed to log event: %w", err)
	}

	return nil
}

// GetEvents retrieves events for a run with optional filtering.
//
// Ordering rules:
//   - TailLimit > 0: query runs in DESC order with LIMIT, results are reversed
//     before return so callers always see ASC order. Other ordering flags are
//     ignored in this mode.
//   - OrderDesc: timestamp DESC, id DESC.
//   - Default: timestamp ASC.
func (s *stateStore) GetEvents(runID string, opts EventQueryOptions) ([]LogRecord, error) {
	query := `SELECT id, run_id, timestamp, step_id, state, persona, message, tokens_used, duration_ms, model, configured_model, adapter
	          FROM event_log
	          WHERE run_id = ?`
	args := []any{runID}

	if opts.AfterID > 0 {
		query += " AND id > ?"
		args = append(args, opts.AfterID)
	}
	if opts.StepID != "" {
		query += " AND step_id = ?"
		args = append(args, opts.StepID)
	}
	if opts.ErrorsOnly {
		query += " AND state = 'failed'"
	}
	if opts.SinceUnix > 0 {
		query += " AND timestamp >= ?"
		args = append(args, opts.SinceUnix)
	}

	tailMode := opts.TailLimit > 0
	switch {
	case tailMode:
		query += " ORDER BY timestamp DESC, id DESC LIMIT ?"
		args = append(args, opts.TailLimit)
	case opts.OrderDesc:
		query += " ORDER BY timestamp DESC, id DESC"
	default:
		query += " ORDER BY timestamp ASC, id ASC"
	}

	if !tailMode && opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
		if opts.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, opts.Offset)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var records []LogRecord
	for rows.Next() {
		var record LogRecord
		var timestamp int64
		var stepID, persona, message, model, configuredModel, adapter sql.NullString
		var tokensUsed, durationMs sql.NullInt64

		err := rows.Scan(
			&record.ID,
			&record.RunID,
			&timestamp,
			&stepID,
			&record.State,
			&persona,
			&message,
			&tokensUsed,
			&durationMs,
			&model,
			&configuredModel,
			&adapter,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		record.Timestamp = time.Unix(timestamp, 0)
		if stepID.Valid {
			record.StepID = stepID.String
		}
		if persona.Valid {
			record.Persona = persona.String
		}
		if message.Valid {
			record.Message = message.String
		}
		if tokensUsed.Valid {
			record.TokensUsed = int(tokensUsed.Int64)
		}
		if durationMs.Valid {
			record.DurationMs = durationMs.Int64
		}
		if model.Valid {
			record.Model = model.String
		}
		if configuredModel.Valid {
			record.ConfiguredModel = configuredModel.String
		}
		if adapter.Valid {
			record.Adapter = adapter.String
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	// Tail mode queries DESC for SQL-side LIMIT correctness; flip to ASC for callers.
	if tailMode && len(records) > 1 {
		for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
			records[i], records[j] = records[j], records[i]
		}
	}

	return records, nil
}

// GetEventAggregateStats returns aggregate metrics over event_log entries in
// terminal states (completed, failed) for the given run.
func (s *stateStore) GetEventAggregateStats(runID string) (*EventAggregateStats, error) {
	var stats EventAggregateStats
	var avg, minD, maxD sql.NullFloat64
	err := s.db.QueryRow(`
		SELECT
		    COUNT(*),
		    COALESCE(SUM(COALESCE(tokens_used, 0)), 0),
		    AVG(COALESCE(duration_ms, 0)),
		    MIN(COALESCE(duration_ms, 0)),
		    MAX(COALESCE(duration_ms, 0))
		FROM event_log
		WHERE run_id = ? AND state IN ('completed', 'failed')
	`, runID).Scan(&stats.TotalEvents, &stats.TotalTokens, &avg, &minD, &maxD)
	if err != nil {
		return nil, fmt.Errorf("failed to query event aggregate stats: %w", err)
	}
	if avg.Valid {
		stats.AvgDurationMs = avg.Float64
	}
	if minD.Valid {
		stats.MinDurationMs = minD.Float64
	}
	if maxD.Valid {
		stats.MaxDurationMs = maxD.Float64
	}
	return &stats, nil
}

// GetAuditEvents retrieves events across all runs, filtered by state types,
// ordered by timestamp descending. Used by the admin audit log viewer.
func (s *stateStore) GetAuditEvents(states []string, limit, offset int) ([]LogRecord, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT e.id, e.run_id, e.timestamp, e.step_id, e.state, e.persona, e.message, e.tokens_used, e.duration_ms
	          FROM event_log e`

	var args []any
	if len(states) > 0 {
		placeholders := make([]string, len(states))
		for i, st := range states {
			placeholders[i] = "?"
			args = append(args, st)
		}
		query += " WHERE e.state IN (" + strings.Join(placeholders, ",") + ")"
	}

	query += " ORDER BY e.timestamp DESC, e.id DESC"
	query += " LIMIT ?"
	args = append(args, limit)
	if offset > 0 {
		query += " OFFSET ?"
		args = append(args, offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit events: %w", err)
	}
	defer rows.Close()

	var records []LogRecord
	for rows.Next() {
		var record LogRecord
		var timestamp int64
		var stepID, persona, message sql.NullString
		var tokensUsed, durationMs sql.NullInt64

		err := rows.Scan(
			&record.ID,
			&record.RunID,
			&timestamp,
			&stepID,
			&record.State,
			&persona,
			&message,
			&tokensUsed,
			&durationMs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit event: %w", err)
		}

		record.Timestamp = time.Unix(timestamp, 0)
		if stepID.Valid {
			record.StepID = stepID.String
		}
		if persona.Valid {
			record.Persona = persona.String
		}
		if message.Valid {
			record.Message = message.String
		}
		if tokensUsed.Valid {
			record.TokensUsed = int(tokensUsed.Int64)
		}
		if durationMs.Valid {
			record.DurationMs = durationMs.Int64
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit events: %w", err)
	}

	return records, nil
}
