package state

import (
	"time"
)

// RecordOrchestrationDecision inserts a new orchestration decision record.
func (s *stateStore) RecordOrchestrationDecision(record *OrchestrationDecision) error {
	_, err := s.db.Exec(
		`INSERT INTO orchestration_decision (run_id, input_text, domain, complexity, pipeline_name, model_tier, reason, outcome, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.RunID, record.InputText, record.Domain, record.Complexity,
		record.PipelineName, record.ModelTier, record.Reason, "pending",
		time.Now().Unix(),
	)
	return err
}

// UpdateOrchestrationOutcome updates the outcome of an orchestration decision after pipeline completion.
func (s *stateStore) UpdateOrchestrationOutcome(runID string, outcome string, tokensUsed int, durationMs int64) error {
	_, err := s.db.Exec(
		`UPDATE orchestration_decision SET outcome = ?, tokens_used = ?, duration_ms = ?, completed_at = ? WHERE run_id = ?`,
		outcome, tokensUsed, durationMs, time.Now().Unix(), runID,
	)
	return err
}

// ListOrchestrationDecisionSummary returns aggregated decision stats grouped by domain, complexity, pipeline.
func (s *stateStore) ListOrchestrationDecisionSummary(limit int) ([]OrchestrationDecisionSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT domain, complexity, pipeline_name,
		        COUNT(*) as total,
		        SUM(CASE WHEN outcome = 'completed' THEN 1 ELSE 0 END) as completed,
		        SUM(CASE WHEN outcome = 'failed' THEN 1 ELSE 0 END) as failed,
		        COALESCE(AVG(CASE WHEN tokens_used > 0 THEN tokens_used END), 0) as avg_tokens,
		        COALESCE(AVG(CASE WHEN duration_ms > 0 THEN duration_ms END), 0) as avg_duration
		 FROM orchestration_decision
		 GROUP BY domain, complexity, pipeline_name
		 ORDER BY total DESC
		 LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []OrchestrationDecisionSummary
	for rows.Next() {
		var s OrchestrationDecisionSummary
		if err := rows.Scan(&s.Domain, &s.Complexity, &s.PipelineName,
			&s.Total, &s.Completed, &s.Failed, &s.AvgTokens, &s.AvgDurationMs); err != nil {
			return nil, err
		}
		if s.Total > 0 {
			s.SuccessRate = float64(s.Completed) / float64(s.Total) * 100
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

// GetOrchestrationStats returns aggregate stats for a pipeline name.
func (s *stateStore) GetOrchestrationStats(pipelineName string) (*OrchestrationStats, error) {
	row := s.db.QueryRow(
		`SELECT pipeline_name,
		        COUNT(*) as total,
		        SUM(CASE WHEN outcome = 'completed' THEN 1 ELSE 0 END) as completed,
		        SUM(CASE WHEN outcome = 'failed' THEN 1 ELSE 0 END) as failed,
		        SUM(CASE WHEN outcome = 'cancelled' THEN 1 ELSE 0 END) as cancelled,
		        COALESCE(AVG(CASE WHEN tokens_used > 0 THEN tokens_used END), 0) as avg_tokens,
		        COALESCE(AVG(CASE WHEN duration_ms > 0 THEN duration_ms END), 0) as avg_duration
		 FROM orchestration_decision
		 WHERE pipeline_name = ?
		 GROUP BY pipeline_name`,
		pipelineName,
	)

	var stats OrchestrationStats
	err := row.Scan(&stats.PipelineName, &stats.TotalRuns, &stats.Completed, &stats.Failed, &stats.Cancelled, &stats.AvgTokens, &stats.AvgDurationMs)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
