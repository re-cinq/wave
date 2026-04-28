package state

import (
	"fmt"
	"time"
)

// RecordOntologyUsage inserts an ontology usage record for decision lineage tracking.
func (s *stateStore) RecordOntologyUsage(runID, stepID, contextName string, invariantCount int, status string, contractPassed *bool) error {
	var cp *int
	if contractPassed != nil {
		v := 0
		if *contractPassed {
			v = 1
		}
		cp = &v
	}
	_, err := s.db.Exec(
		`INSERT INTO ontology_usage (run_id, step_id, context_name, invariant_count, step_status, contract_passed) VALUES (?, ?, ?, ?, ?, ?)`,
		runID, stepID, contextName, invariantCount, status, cp,
	)
	if err != nil {
		return fmt.Errorf("failed to record ontology usage: %w", err)
	}
	return nil
}

// GetOntologyStats returns aggregated statistics for a single ontology context.
func (s *stateStore) GetOntologyStats(contextName string) (*OntologyStats, error) {
	row := s.db.QueryRow(
		`SELECT context_name,
		        COUNT(*) as total_runs,
		        SUM(CASE WHEN step_status = 'success' THEN 1 ELSE 0 END) as successes,
		        SUM(CASE WHEN step_status = 'failed' THEN 1 ELSE 0 END) as failures,
		        ROUND(100.0 * SUM(CASE WHEN step_status = 'success' THEN 1 ELSE 0 END) / COUNT(*), 1) as success_rate,
		        MAX(created_at) as last_used
		 FROM ontology_usage
		 WHERE context_name = ?
		 GROUP BY context_name`,
		contextName,
	)

	var stats OntologyStats
	var lastUsed int64
	err := row.Scan(&stats.ContextName, &stats.TotalRuns, &stats.Successes, &stats.Failures, &stats.SuccessRate, &lastUsed)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return &OntologyStats{ContextName: contextName}, nil
		}
		return nil, fmt.Errorf("failed to get ontology stats for %s: %w", contextName, err)
	}
	stats.LastUsed = time.Unix(lastUsed, 0)
	return &stats, nil
}

// GetOntologyStatsAll returns aggregated statistics for all ontology contexts, sorted by total_runs DESC.
func (s *stateStore) GetOntologyStatsAll() ([]OntologyStats, error) {
	rows, err := s.db.Query(
		`SELECT context_name,
		        COUNT(*) as total_runs,
		        SUM(CASE WHEN step_status = 'success' THEN 1 ELSE 0 END) as successes,
		        SUM(CASE WHEN step_status = 'failed' THEN 1 ELSE 0 END) as failures,
		        ROUND(100.0 * SUM(CASE WHEN step_status = 'success' THEN 1 ELSE 0 END) / COUNT(*), 1) as success_rate,
		        MAX(created_at) as last_used
		 FROM ontology_usage
		 GROUP BY context_name
		 ORDER BY total_runs DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get all ontology stats: %w", err)
	}
	defer rows.Close()

	var allStats []OntologyStats
	for rows.Next() {
		var stats OntologyStats
		var lastUsed int64
		err := rows.Scan(&stats.ContextName, &stats.TotalRuns, &stats.Successes, &stats.Failures, &stats.SuccessRate, &lastUsed)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ontology stats: %w", err)
		}
		stats.LastUsed = time.Unix(lastUsed, 0)
		allStats = append(allStats, stats)
	}
	return allStats, nil
}
