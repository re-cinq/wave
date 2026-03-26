package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordOntologyUsage(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	passed := true
	err = store.RecordOntologyUsage(runID, "analyze", "billing", 3, "success", &passed)
	require.NoError(t, err)

	// Verify via stats query
	stats, err := store.GetOntologyStats("billing")
	require.NoError(t, err)
	assert.Equal(t, "billing", stats.ContextName)
	assert.Equal(t, 1, stats.TotalRuns)
	assert.Equal(t, 1, stats.Successes)
	assert.Equal(t, 0, stats.Failures)
	assert.Equal(t, 100.0, stats.SuccessRate)
}

func TestRecordOntologyUsage_NilContractPassed(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	// Record with nil contract_passed (no contract on the step)
	err = store.RecordOntologyUsage(runID, "implement", "orders", 2, "success", nil)
	require.NoError(t, err)

	stats, err := store.GetOntologyStats("orders")
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRuns)
	assert.Equal(t, 1, stats.Successes)
}

func TestGetOntologyStats_Aggregation(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID1, err := store.CreateRun("pipeline-1", "input1")
	require.NoError(t, err)

	runID2, err := store.CreateRun("pipeline-2", "input2")
	require.NoError(t, err)

	passed := true
	failed := false

	// Record multiple usages for the same context
	err = store.RecordOntologyUsage(runID1, "step-a", "billing", 3, "success", &passed)
	require.NoError(t, err)

	err = store.RecordOntologyUsage(runID1, "step-b", "billing", 3, "failed", &failed)
	require.NoError(t, err)

	err = store.RecordOntologyUsage(runID2, "step-a", "billing", 3, "success", &passed)
	require.NoError(t, err)

	stats, err := store.GetOntologyStats("billing")
	require.NoError(t, err)
	assert.Equal(t, "billing", stats.ContextName)
	assert.Equal(t, 3, stats.TotalRuns)
	assert.Equal(t, 2, stats.Successes)
	assert.Equal(t, 1, stats.Failures)
	// 2/3 = 66.7%
	assert.InDelta(t, 66.7, stats.SuccessRate, 0.1)
}

func TestGetOntologyStats_EmptyTable(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	stats, err := store.GetOntologyStats("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "nonexistent", stats.ContextName)
	assert.Equal(t, 0, stats.TotalRuns)
	assert.Equal(t, 0, stats.Successes)
	assert.Equal(t, 0, stats.Failures)
	assert.Equal(t, float64(0), stats.SuccessRate)
}

func TestGetOntologyStatsAll(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	passed := true

	// Record usage for multiple contexts
	err = store.RecordOntologyUsage(runID, "step-a", "billing", 3, "success", &passed)
	require.NoError(t, err)

	err = store.RecordOntologyUsage(runID, "step-a", "orders", 2, "success", &passed)
	require.NoError(t, err)

	err = store.RecordOntologyUsage(runID, "step-b", "billing", 3, "success", &passed)
	require.NoError(t, err)

	allStats, err := store.GetOntologyStatsAll()
	require.NoError(t, err)
	require.Len(t, allStats, 2)

	// Sorted by total_runs DESC — billing has 2, orders has 1
	assert.Equal(t, "billing", allStats[0].ContextName)
	assert.Equal(t, 2, allStats[0].TotalRuns)
	assert.Equal(t, "orders", allStats[1].ContextName)
	assert.Equal(t, 1, allStats[1].TotalRuns)
}

func TestGetOntologyStatsAll_EmptyTable(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	allStats, err := store.GetOntologyStatsAll()
	require.NoError(t, err)
	assert.Empty(t, allStats)
}

func TestRecordOntologyUsage_SkippedStatus(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	err = store.RecordOntologyUsage(runID, "step-a", "auth", 1, "skipped", nil)
	require.NoError(t, err)

	stats, err := store.GetOntologyStats("auth")
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRuns)
	assert.Equal(t, 0, stats.Successes)
	assert.Equal(t, 0, stats.Failures)
	// Skipped is neither success nor failure
	assert.Equal(t, 0.0, stats.SuccessRate)
}
