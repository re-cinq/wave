package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchedule_CreateGetList(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	next := time.Now().Add(1 * time.Hour)
	id, err := store.CreateSchedule(ScheduleRecord{
		PipelineName: "ops-bootstrap",
		CronExpr:     "0 * * * *",
		InputRef:     "{}",
		Active:       true,
		NextFireAt:   &next,
	})
	require.NoError(t, err)
	require.NotZero(t, id)

	got, err := store.GetSchedule(id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "ops-bootstrap", got.PipelineName)
	require.NotNil(t, got.NextFireAt)

	all, err := store.ListSchedules()
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestSchedule_ListDueSchedules(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	due := now.Add(-1 * time.Minute)
	future := now.Add(1 * time.Hour)

	for _, s := range []ScheduleRecord{
		{PipelineName: "due-active", CronExpr: "* * * * *", Active: true, NextFireAt: &due},
		{PipelineName: "due-inactive", CronExpr: "* * * * *", Active: false, NextFireAt: &due},
		{PipelineName: "future-active", CronExpr: "* * * * *", Active: true, NextFireAt: &future},
		{PipelineName: "no-fire", CronExpr: "* * * * *", Active: true},
	} {
		_, err := store.CreateSchedule(s)
		require.NoError(t, err)
	}

	dueRows, err := store.ListDueSchedules(now)
	require.NoError(t, err)
	require.Len(t, dueRows, 1, "only the active+past row should be returned")
	assert.Equal(t, "due-active", dueRows[0].PipelineName)
}

func TestSchedule_UpdateNextFireAndDeactivate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	id, err := store.CreateSchedule(ScheduleRecord{
		PipelineName: "p", CronExpr: "0 0 * * *", Active: true,
	})
	require.NoError(t, err)

	next := time.Now().Add(24 * time.Hour)
	require.NoError(t, store.UpdateScheduleNextFire(id, next, "run-abc"))

	got, err := store.GetSchedule(id)
	require.NoError(t, err)
	require.NotNil(t, got.NextFireAt)
	assert.Equal(t, "run-abc", got.LastRunID)

	require.NoError(t, store.DeactivateSchedule(id))
	got, err = store.GetSchedule(id)
	require.NoError(t, err)
	assert.False(t, got.Active)
}

func TestSchedule_UpdateMissing(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	err := store.UpdateScheduleNextFire(999, time.Now(), "x")
	assert.Error(t, err)
}
