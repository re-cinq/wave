package state

import (
	"fmt"
	"time"
)

// SaveRetrospective saves a retrospective index record.
func (s *stateStore) SaveRetrospective(record *RetrospectiveRecord) error {
	// Check if exists first
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
func (s *stateStore) GetRetrospective(runID string) (*RetrospectiveRecord, error) {
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
func (s *stateStore) ListRetrospectives(opts ListRetrosOptions) ([]RetrospectiveRecord, error) {
	query := "SELECT id, run_id, pipeline_name, smoothness, status, file_path, created_at FROM retrospective WHERE 1=1"
	var args []interface{}

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
func (s *stateStore) DeleteRetrospective(runID string) error {
	_, err := s.db.Exec("DELETE FROM retrospective WHERE run_id = ?", runID)
	return err
}

// UpdateRetrospectiveSmoothness updates the smoothness rating for a retrospective.
func (s *stateStore) UpdateRetrospectiveSmoothness(runID string, smoothness string) error {
	_, err := s.db.Exec("UPDATE retrospective SET smoothness = ? WHERE run_id = ?", smoothness, runID)
	return err
}

// UpdateRetrospectiveStatus updates the status for a retrospective.
func (s *stateStore) UpdateRetrospectiveStatus(runID string, status string) error {
	_, err := s.db.Exec("UPDATE retrospective SET status = ? WHERE run_id = ?", status, runID)
	return err
}
