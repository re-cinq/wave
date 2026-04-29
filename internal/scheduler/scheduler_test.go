package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStore is an in-memory ScheduleStore used to drive the tick loop
// without spinning up SQLite. Only the methods Tick + fireOne touch are
// implemented; the rest panic so a future test that depends on them
// surfaces immediately.
type fakeStore struct {
	mu      sync.Mutex
	rows    []state.ScheduleRecord
	updates []updateCall
}

type updateCall struct {
	id        int64
	nextFire  time.Time
	lastRunID string
}

func (f *fakeStore) ListDueSchedules(now time.Time) ([]state.ScheduleRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var due []state.ScheduleRecord
	for _, r := range f.rows {
		if !r.Active || r.NextFireAt == nil {
			continue
		}
		if !r.NextFireAt.After(now) {
			due = append(due, r)
		}
	}
	return due, nil
}

func (f *fakeStore) UpdateScheduleNextFire(id int64, nextFire time.Time, lastRunID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates = append(f.updates, updateCall{id: id, nextFire: nextFire, lastRunID: lastRunID})
	for i := range f.rows {
		if f.rows[i].ID == id {
			f.rows[i].NextFireAt = &nextFire
			f.rows[i].LastRunID = lastRunID
			return nil
		}
	}
	return errors.New("not found")
}

func (f *fakeStore) CreateSchedule(state.ScheduleRecord) (int64, error) { panic("unused") }
func (f *fakeStore) DeactivateSchedule(int64) error                     { panic("unused") }
func (f *fakeStore) GetSchedule(int64) (*state.ScheduleRecord, error)   { panic("unused") }
func (f *fakeStore) ListSchedules() ([]state.ScheduleRecord, error)     { panic("unused") }

var _ state.ScheduleStore = (*fakeStore)(nil)

func TestTick_FiresDueSchedule(t *testing.T) {
	now := time.Date(2026, 4, 29, 14, 30, 0, 0, time.UTC)
	past := now.Add(-1 * time.Minute)

	store := &fakeStore{
		rows: []state.ScheduleRecord{
			{ID: 1, PipelineName: "p", CronExpr: "*/15 * * * *", Active: true, NextFireAt: &past},
		},
	}

	var dispatched []int64
	disp := DispatcherFunc(func(_ context.Context, s state.ScheduleRecord) (string, error) {
		dispatched = append(dispatched, s.ID)
		return "run-abc", nil
	})

	sched := New(store, disp, Options{Now: func() time.Time { return now }})
	sched.Tick(context.Background())

	assert.Equal(t, []int64{1}, dispatched)
	require.Len(t, store.updates, 1)
	// next_fire_at must advance to a future time
	assert.True(t, store.updates[0].nextFire.After(now))
	assert.Equal(t, "run-abc", store.updates[0].lastRunID)
}

func TestTick_SkipsFutureSchedules(t *testing.T) {
	now := time.Date(2026, 4, 29, 14, 30, 0, 0, time.UTC)
	future := now.Add(1 * time.Hour)

	store := &fakeStore{
		rows: []state.ScheduleRecord{
			{ID: 1, PipelineName: "p", CronExpr: "*/15 * * * *", Active: true, NextFireAt: &future},
		},
	}
	dispatched := 0
	disp := DispatcherFunc(func(context.Context, state.ScheduleRecord) (string, error) {
		dispatched++
		return "", nil
	})

	sched := New(store, disp, Options{Now: func() time.Time { return now }})
	sched.Tick(context.Background())

	assert.Equal(t, 0, dispatched)
	assert.Empty(t, store.updates)
}

func TestTick_BadCronSkipsButAdvances(t *testing.T) {
	now := time.Date(2026, 4, 29, 14, 30, 0, 0, time.UTC)
	past := now.Add(-1 * time.Minute)

	store := &fakeStore{
		rows: []state.ScheduleRecord{
			{ID: 1, PipelineName: "p", CronExpr: "not a cron", Active: true, NextFireAt: &past},
		},
	}
	dispatched := 0
	disp := DispatcherFunc(func(context.Context, state.ScheduleRecord) (string, error) {
		dispatched++
		return "", nil
	})
	sched := New(store, disp, Options{Now: func() time.Time { return now }})
	sched.Tick(context.Background())

	assert.Equal(t, 0, dispatched, "bad cron must not dispatch")
	require.Len(t, store.updates, 1, "bad cron must still advance to prevent churn")
	assert.True(t, store.updates[0].nextFire.After(now))
}

func TestTick_DispatchErrStillAdvances(t *testing.T) {
	now := time.Date(2026, 4, 29, 14, 30, 0, 0, time.UTC)
	past := now.Add(-1 * time.Minute)

	store := &fakeStore{
		rows: []state.ScheduleRecord{
			{ID: 1, PipelineName: "p", CronExpr: "* * * * *", Active: true, NextFireAt: &past},
		},
	}
	disp := DispatcherFunc(func(context.Context, state.ScheduleRecord) (string, error) {
		return "", errors.New("boom")
	})
	sched := New(store, disp, Options{Now: func() time.Time { return now }})
	sched.Tick(context.Background())

	require.Len(t, store.updates, 1, "failed dispatch must still advance next_fire to avoid hot loops")
}

func TestStartStop_TickFiresImmediately(t *testing.T) {
	now := time.Date(2026, 4, 29, 14, 30, 0, 0, time.UTC)
	past := now.Add(-1 * time.Minute)

	store := &fakeStore{
		rows: []state.ScheduleRecord{
			{ID: 1, PipelineName: "p", CronExpr: "* * * * *", Active: true, NextFireAt: &past},
		},
	}
	fired := make(chan struct{}, 1)
	disp := DispatcherFunc(func(context.Context, state.ScheduleRecord) (string, error) {
		select {
		case fired <- struct{}{}:
		default:
		}
		return "", nil
	})

	sched := New(store, disp, Options{
		Tick: 50 * time.Millisecond,
		Now:  func() time.Time { return now },
	})
	sched.Start(context.Background())
	defer sched.Stop()

	select {
	case <-fired:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not fire on initial tick")
	}
}
