package health_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemaPath returns the absolute path to the health-analysis contract schema.
// The test runs from internal/health/ so the schema is two levels up.
func schemaPath(t *testing.T) string {
	t.Helper()
	// Resolve relative to the test file's package directory.
	abs, err := filepath.Abs(filepath.Join("..", "..", ".wave", "contracts", "health-analysis.schema.json"))
	require.NoError(t, err, "failed to resolve schema path")
	_, err = os.Stat(abs)
	require.NoError(t, err, "schema file does not exist at %s", abs)
	return abs
}

// writeArtifact creates the workspace directory structure expected by the
// json_schema validator and writes data as .wave/artifact.json.
func writeArtifact(t *testing.T, workspace string, data []byte) {
	t.Helper()
	artifactDir := filepath.Join(workspace, ".wave")
	require.NoError(t, os.MkdirAll(artifactDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(artifactDir, "artifact.json"), data, 0o644))
}

func TestHealthReportSchemaCompliance(t *testing.T) {
	now := time.Now().UTC()
	lastRunAt := now.Add(-2 * time.Hour)

	report := health.HealthReport{
		ForgeType:  health.GitHub,
		Repository: "re-cinq/wave",
		AnalyzedAt: now,
		Commits: &health.CommitAnalysis{
			TotalCount: 42,
			WindowDays: 30,
			Authors: []health.AuthorActivity{
				{Name: "alice", CommitCount: 25},
				{Name: "bob", CommitCount: 17},
			},
			AreasOfActivity: []string{"pipeline", "health"},
			FrequencyPerDay: 1.4,
		},
		PullRequests: &health.PRSummary{
			TotalOpen: 3,
			ByReviewState: map[string]int{
				"APPROVED":        1,
				"REVIEW_REQUIRED": 2,
			},
			Stale: []health.StalePR{
				{
					Number:          101,
					Title:           "old PR",
					Author:          "charlie",
					DaysSinceUpdate: 21,
				},
			},
			RecentActivity: 2,
		},
		Issues: &health.IssueSummary{
			TotalOpen: 5,
			ByCategory: map[string]int{
				"bug":         2,
				"enhancement": 3,
			},
			ByPriority: map[string]int{
				"high": 1,
				"low":  4,
			},
			Actionable: []health.ActionableIssue{
				{
					Number:   200,
					Title:    "fix the thing",
					Labels:   []string{"bug", "priority: high"},
					Priority: "high",
				},
			},
		},
		CIStatus: &health.CIStatus{
			RecentRuns:    10,
			PassRate:      90.0,
			LastRunStatus: "success",
			LastRunAt:     &lastRunAt,
		},
	}

	data, err := json.Marshal(report)
	require.NoError(t, err, "failed to marshal HealthReport to JSON")

	workspace := t.TempDir()
	writeArtifact(t, workspace, data)

	cfg := contract.ContractConfig{
		Type:                    "json_schema",
		SchemaPath:              schemaPath(t),
		DisableWrapperDetection: true,
	}

	err = contract.Validate(cfg, workspace)
	assert.NoError(t, err, "valid HealthReport should pass schema validation")
}

func TestHealthReportSchemaComplianceMalformed(t *testing.T) {
	// Construct a report that is missing required fields: forge_type is empty,
	// repository is empty, commits/pull_requests/issues are nil.  The schema
	// requires non-empty forge_type (enum), non-empty repository (minLength: 1),
	// and requires commits, pull_requests, and issues objects.
	report := health.HealthReport{
		ForgeType:    "",          // violates enum constraint
		Repository:   "",          // violates minLength: 1
		AnalyzedAt:   time.Time{}, // zero-value still serializes to a date-time string
		Commits:      nil,         // required field missing (marshals as null)
		PullRequests: nil,         // required field missing (marshals as null)
		Issues:       nil,         // required field missing (marshals as null)
	}

	data, err := json.Marshal(report)
	require.NoError(t, err, "failed to marshal malformed HealthReport to JSON")

	workspace := t.TempDir()
	writeArtifact(t, workspace, data)

	cfg := contract.ContractConfig{
		Type:                    "json_schema",
		SchemaPath:              schemaPath(t),
		DisableWrapperDetection: true,
	}

	err = contract.Validate(cfg, workspace)
	assert.Error(t, err, "malformed HealthReport should fail schema validation")
}
