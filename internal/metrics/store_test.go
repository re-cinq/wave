package metrics

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupTestStore creates a metrics Store backed by a fresh in-memory SQLite
// database. The schema is bootstrapped with just the two tables this package
// owns — production migrations are run by internal/state, but for unit tests
// we keep the dependency at zero.
func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err, "failed to open test database")

	createPerformanceMetric := `
		CREATE TABLE performance_metric (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL,
			step_id TEXT NOT NULL,
			pipeline_name TEXT NOT NULL,
			persona TEXT,
			started_at INTEGER NOT NULL,
			completed_at INTEGER,
			duration_ms INTEGER,
			tokens_used INTEGER,
			files_modified INTEGER,
			artifacts_generated INTEGER,
			memory_bytes INTEGER,
			success INTEGER NOT NULL,
			error_message TEXT
		)`
	_, err = db.Exec(createPerformanceMetric)
	require.NoError(t, err)

	createRetrospective := `
		CREATE TABLE retrospective (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL UNIQUE,
			pipeline_name TEXT NOT NULL,
			smoothness TEXT,
			status TEXT NOT NULL,
			file_path TEXT,
			created_at INTEGER NOT NULL
		)`
	_, err = db.Exec(createRetrospective)
	require.NoError(t, err)

	store := NewStore(db)
	cleanup := func() { _ = db.Close() }
	return store, cleanup
}

// TestRecordPerformanceMetric covers round-trip persistence + the stepID filter.
func TestRecordPerformanceMetric(t *testing.T) {
	t.Run("round-trip record and retrieve", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		startedAt := time.Now().Truncate(time.Second)
		completedAt := startedAt.Add(5 * time.Second)

		metric := &PerformanceMetricRecord{
			RunID:              "run-1",
			StepID:             "step-1",
			PipelineName:       "test-pipeline",
			Persona:            "craftsman",
			StartedAt:          startedAt,
			CompletedAt:        &completedAt,
			DurationMs:         5000,
			TokensUsed:         1200,
			FilesModified:      3,
			ArtifactsGenerated: 2,
			MemoryBytes:        1024000,
			Success:            true,
		}

		err := store.RecordPerformanceMetric(metric)
		require.NoError(t, err)
		assert.NotZero(t, metric.ID, "ID should be set after insert")

		got, err := store.GetPerformanceMetrics("run-1", "")
		require.NoError(t, err)
		require.Len(t, got, 1)

		m := got[0]
		assert.Equal(t, "run-1", m.RunID)
		assert.Equal(t, "step-1", m.StepID)
		assert.Equal(t, "test-pipeline", m.PipelineName)
		assert.Equal(t, "craftsman", m.Persona)
		assert.Equal(t, startedAt.Unix(), m.StartedAt.Unix())
		require.NotNil(t, m.CompletedAt)
		assert.Equal(t, completedAt.Unix(), m.CompletedAt.Unix())
		assert.Equal(t, int64(5000), m.DurationMs)
		assert.Equal(t, 1200, m.TokensUsed)
		assert.Equal(t, 3, m.FilesModified)
		assert.Equal(t, 2, m.ArtifactsGenerated)
		assert.Equal(t, int64(1024000), m.MemoryBytes)
		assert.True(t, m.Success)
		assert.Empty(t, m.ErrorMessage)
	})

	t.Run("filter by stepID", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		for _, stepID := range []string{"step-a", "step-a", "step-b"} {
			require.NoError(t, store.RecordPerformanceMetric(&PerformanceMetricRecord{
				RunID:        "run-1",
				StepID:       stepID,
				PipelineName: "test-pipeline",
				StartedAt:    now,
				Success:      true,
			}))
		}

		all, err := store.GetPerformanceMetrics("run-1", "")
		require.NoError(t, err)
		assert.Len(t, all, 3)

		stepA, err := store.GetPerformanceMetrics("run-1", "step-a")
		require.NoError(t, err)
		assert.Len(t, stepA, 2)
		for _, m := range stepA {
			assert.Equal(t, "step-a", m.StepID)
		}

		stepB, err := store.GetPerformanceMetrics("run-1", "step-b")
		require.NoError(t, err)
		assert.Len(t, stepB, 1)
		assert.Equal(t, "step-b", stepB[0].StepID)
	})

	t.Run("nil CompletedAt handling", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		err := store.RecordPerformanceMetric(&PerformanceMetricRecord{
			RunID:        "run-1",
			StepID:       "step-1",
			PipelineName: "test-pipeline",
			StartedAt:    now,
			CompletedAt:  nil,
			Success:      false,
			ErrorMessage: "timed out",
		})
		require.NoError(t, err)

		metrics, err := store.GetPerformanceMetrics("run-1", "step-1")
		require.NoError(t, err)
		require.Len(t, metrics, 1)
		assert.Nil(t, metrics[0].CompletedAt)
		assert.False(t, metrics[0].Success)
		assert.Equal(t, "timed out", metrics[0].ErrorMessage)
	})
}

// TestGetStepPerformanceStats covers the aggregation query with success/failure mix.
func TestGetStepPerformanceStats(t *testing.T) {
	t.Run("aggregation across multiple metrics", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		durations := []int64{1000, 2000, 3000}
		tokens := []int{100, 200, 300}
		successes := []bool{true, true, false}

		for i := 0; i < 3; i++ {
			require.NoError(t, store.RecordPerformanceMetric(&PerformanceMetricRecord{
				RunID:              "run-x",
				StepID:             "build",
				PipelineName:       "my-pipeline",
				Persona:            "craftsman",
				StartedAt:          now.Add(time.Duration(i) * time.Second),
				DurationMs:         durations[i],
				TokensUsed:         tokens[i],
				FilesModified:      i + 1,
				ArtifactsGenerated: 1,
				Success:            successes[i],
			}))
		}

		stats, err := store.GetStepPerformanceStats("my-pipeline", "build", now.Add(-1*time.Hour))
		require.NoError(t, err)
		require.NotNil(t, stats)

		assert.Equal(t, "build", stats.StepID)
		assert.Equal(t, "craftsman", stats.Persona)
		assert.Equal(t, 3, stats.TotalRuns)
		assert.Equal(t, 2, stats.SuccessfulRuns)
		assert.Equal(t, 1, stats.FailedRuns)
		assert.Equal(t, int64(2000), stats.AvgDurationMs)
		assert.Equal(t, int64(1000), stats.MinDurationMs)
		assert.Equal(t, int64(3000), stats.MaxDurationMs)
		assert.Equal(t, 200, stats.AvgTokensUsed)
		assert.Equal(t, 600, stats.TotalTokensUsed)
		assert.True(t, stats.TokenBurnRate > 0)
	})

	t.Run("empty result for non-existent step", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		stats, err := store.GetStepPerformanceStats("no-pipeline", "no-step", time.Now().Add(-1*time.Hour))
		require.NoError(t, err)
		require.NotNil(t, stats)
		assert.Equal(t, 0, stats.TotalRuns)
		assert.Equal(t, "no-step", stats.StepID)
	})

	t.Run("since filter excludes old metrics", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		oldTime := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
		recentTime := time.Now().Truncate(time.Second)
		cutoff := time.Now().Add(-1 * time.Hour)

		require.NoError(t, store.RecordPerformanceMetric(&PerformanceMetricRecord{
			RunID:        "old-run",
			StepID:       "step-x",
			PipelineName: "pipe",
			StartedAt:    oldTime,
			DurationMs:   9999,
			TokensUsed:   9999,
			Success:      true,
		}))
		require.NoError(t, store.RecordPerformanceMetric(&PerformanceMetricRecord{
			RunID:        "new-run",
			StepID:       "step-x",
			PipelineName: "pipe",
			StartedAt:    recentTime,
			DurationMs:   500,
			TokensUsed:   100,
			Success:      true,
		}))

		stats, err := store.GetStepPerformanceStats("pipe", "step-x", cutoff)
		require.NoError(t, err)
		require.NotNil(t, stats)
		assert.Equal(t, 1, stats.TotalRuns)
		assert.Equal(t, int64(500), stats.AvgDurationMs)
		assert.Equal(t, 100, stats.AvgTokensUsed)
	})
}

// TestGetRecentPerformanceHistory covers the filtered listing query.
func TestGetRecentPerformanceHistory(t *testing.T) {
	t.Run("limit enforcement", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		for i := 0; i < 5; i++ {
			require.NoError(t, store.RecordPerformanceMetric(&PerformanceMetricRecord{
				RunID:        "r",
				StepID:       "step-1",
				PipelineName: "pipe",
				StartedAt:    now.Add(time.Duration(i) * time.Second),
				Success:      true,
			}))
		}

		got, err := store.GetRecentPerformanceHistory(PerformanceQueryOptions{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("filter by pipeline name", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		for _, pname := range []string{"alpha", "alpha", "beta"} {
			require.NoError(t, store.RecordPerformanceMetric(&PerformanceMetricRecord{
				RunID:        "r-" + pname,
				StepID:       "s1",
				PipelineName: pname,
				StartedAt:    now,
				Success:      true,
			}))
		}

		got, err := store.GetRecentPerformanceHistory(PerformanceQueryOptions{PipelineName: "alpha"})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		for _, m := range got {
			assert.Equal(t, "alpha", m.PipelineName)
		}
	})

	t.Run("filter by step ID", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		for _, sid := range []string{"build", "build", "test"} {
			require.NoError(t, store.RecordPerformanceMetric(&PerformanceMetricRecord{
				RunID:        "r",
				StepID:       sid,
				PipelineName: "pipe",
				StartedAt:    now,
				Success:      true,
			}))
		}

		got, err := store.GetRecentPerformanceHistory(PerformanceQueryOptions{StepID: "build"})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("filter by persona", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		for _, persona := range []string{"navigator", "craftsman", "navigator"} {
			require.NoError(t, store.RecordPerformanceMetric(&PerformanceMetricRecord{
				RunID:        "r",
				StepID:       "s1",
				PipelineName: "pipe",
				Persona:      persona,
				StartedAt:    now,
				Success:      true,
			}))
		}

		got, err := store.GetRecentPerformanceHistory(PerformanceQueryOptions{Persona: "navigator"})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		for _, m := range got {
			assert.Equal(t, "navigator", m.Persona)
		}
	})
}

// TestCleanupOldPerformanceMetrics covers retention pruning.
func TestCleanupOldPerformanceMetrics(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	require.NoError(t, store.RecordPerformanceMetric(&PerformanceMetricRecord{
		RunID:      "run-1",
		StepID:     "step-1",
		Persona:    "navigator",
		StartedAt:  now,
		DurationMs: 100,
		Success:    true,
	}))

	deleted, err := store.CleanupOldPerformanceMetrics(24 * time.Hour * 365 * 100)
	require.NoError(t, err)
	assert.Equal(t, 0, deleted, "nothing should be deleted with 100-year window")

	// Cleanup with zero duration prunes anything older than now — outcome
	// is timing-sensitive on a single insert, so we just assert no error.
	_, err = store.CleanupOldPerformanceMetrics(0)
	require.NoError(t, err)
}

// TestRetrospective covers save / get / list / delete / status updates.
func TestRetrospective(t *testing.T) {
	t.Run("save then get round-trip", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		rec := &RetrospectiveRecord{
			RunID:        "run-42",
			PipelineName: "rebuild",
			Smoothness:   "smooth",
			Status:       "complete",
			FilePath:     "/tmp/run-42.json",
			CreatedAt:    now,
		}

		require.NoError(t, store.SaveRetrospective(rec))

		got, err := store.GetRetrospective("run-42")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "run-42", got.RunID)
		assert.Equal(t, "rebuild", got.PipelineName)
		assert.Equal(t, "smooth", got.Smoothness)
		assert.Equal(t, "complete", got.Status)
		assert.Equal(t, "/tmp/run-42.json", got.FilePath)
		assert.Equal(t, now.Unix(), got.CreatedAt.Unix())
	})

	t.Run("save updates existing row in place", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		require.NoError(t, store.SaveRetrospective(&RetrospectiveRecord{
			RunID:        "run-x",
			PipelineName: "pipe",
			Smoothness:   "",
			Status:       "quantitative",
			FilePath:     "/tmp/run-x.json",
			CreatedAt:    now,
		}))
		require.NoError(t, store.SaveRetrospective(&RetrospectiveRecord{
			RunID:        "run-x",
			PipelineName: "pipe",
			Smoothness:   "effortless",
			Status:       "complete",
			FilePath:     "/tmp/run-x.json",
			CreatedAt:    now,
		}))

		got, err := store.GetRetrospective("run-x")
		require.NoError(t, err)
		assert.Equal(t, "effortless", got.Smoothness)
		assert.Equal(t, "complete", got.Status)
	})

	t.Run("list with filters", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		require.NoError(t, store.SaveRetrospective(&RetrospectiveRecord{RunID: "a", PipelineName: "alpha", Status: "complete", CreatedAt: now}))
		require.NoError(t, store.SaveRetrospective(&RetrospectiveRecord{RunID: "b", PipelineName: "alpha", Status: "complete", CreatedAt: now.Add(time.Second)}))
		require.NoError(t, store.SaveRetrospective(&RetrospectiveRecord{RunID: "c", PipelineName: "beta", Status: "complete", CreatedAt: now.Add(2 * time.Second)}))

		got, err := store.ListRetrospectives(ListRetrosOptions{PipelineName: "alpha"})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		for _, r := range got {
			assert.Equal(t, "alpha", r.PipelineName)
		}

		got, err = store.ListRetrospectives(ListRetrosOptions{Limit: 1})
		require.NoError(t, err)
		assert.Len(t, got, 1)
		// Newest first
		assert.Equal(t, "c", got[0].RunID)
	})

	t.Run("update smoothness and status", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		require.NoError(t, store.SaveRetrospective(&RetrospectiveRecord{
			RunID:        "run-q",
			PipelineName: "pipe",
			Status:       "quantitative",
			CreatedAt:    now,
		}))

		require.NoError(t, store.UpdateRetrospectiveSmoothness("run-q", "bumpy"))
		require.NoError(t, store.UpdateRetrospectiveStatus("run-q", "complete"))

		got, err := store.GetRetrospective("run-q")
		require.NoError(t, err)
		assert.Equal(t, "bumpy", got.Smoothness)
		assert.Equal(t, "complete", got.Status)
	})

	t.Run("delete", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		require.NoError(t, store.SaveRetrospective(&RetrospectiveRecord{
			RunID:        "run-del",
			PipelineName: "pipe",
			Status:       "complete",
			CreatedAt:    now,
		}))

		require.NoError(t, store.DeleteRetrospective("run-del"))

		_, err := store.GetRetrospective("run-del")
		require.Error(t, err)
	})
}
