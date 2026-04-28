package state

import (
	"database/sql"
	"fmt"
	"time"
)

// RecordDecision appends a decision record to the decision log.
func (s *stateStore) RecordDecision(record *DecisionRecord) error {
	ts := record.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	contextJSON := record.Context
	if contextJSON == "" {
		contextJSON = "{}"
	}
	result, err := s.db.Exec(
		`INSERT INTO decision_log (run_id, step_id, timestamp, category, decision, rationale, context_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		record.RunID, record.StepID, ts.Unix(), record.Category, record.Decision, record.Rationale, contextJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to record decision: %w", err)
	}
	id, _ := result.LastInsertId()
	record.ID = id
	return nil
}

// GetDecisions returns all decision records for a run, ordered by timestamp.
func (s *stateStore) GetDecisions(runID string) ([]*DecisionRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, step_id, timestamp, category, decision, rationale, context_json
		FROM decision_log WHERE run_id = ? ORDER BY timestamp ASC, id ASC`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query decisions: %w", err)
	}
	defer rows.Close()
	return scanDecisionRecords(rows)
}

// GetDecisionsByStep returns decision records for a specific run and step.
func (s *stateStore) GetDecisionsByStep(runID, stepID string) ([]*DecisionRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, step_id, timestamp, category, decision, rationale, context_json
		FROM decision_log WHERE run_id = ? AND step_id = ? ORDER BY timestamp ASC, id ASC`,
		runID, stepID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query decisions by step: %w", err)
	}
	defer rows.Close()
	return scanDecisionRecords(rows)
}

// GetDecisionsFiltered returns decision records for a run filtered by step
// and/or category. Empty filter values match all entries on that field.
func (s *stateStore) GetDecisionsFiltered(runID string, opts DecisionQueryOptions) ([]*DecisionRecord, error) {
	query := `SELECT id, run_id, step_id, timestamp, category, decision, rationale, context_json
	          FROM decision_log WHERE run_id = ?`
	args := []any{runID}

	if opts.StepID != "" {
		query += " AND step_id = ?"
		args = append(args, opts.StepID)
	}
	if opts.Category != "" {
		query += " AND category = ?"
		args = append(args, opts.Category)
	}
	query += " ORDER BY timestamp ASC, id ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query decisions: %w", err)
	}
	defer rows.Close()
	return scanDecisionRecords(rows)
}

// scanDecisionRecords scans rows into DecisionRecord slices.
func scanDecisionRecords(rows *sql.Rows) ([]*DecisionRecord, error) {
	var records []*DecisionRecord
	for rows.Next() {
		var r DecisionRecord
		var ts int64
		err := rows.Scan(&r.ID, &r.RunID, &r.StepID, &ts, &r.Category, &r.Decision, &r.Rationale, &r.Context)
		if err != nil {
			return nil, fmt.Errorf("failed to scan decision record: %w", err)
		}
		r.Timestamp = time.Unix(ts, 0)
		records = append(records, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating decision records: %w", err)
	}
	return records, nil
}
