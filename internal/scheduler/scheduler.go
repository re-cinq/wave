package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/recinq/wave/internal/state"
)

// Dispatcher is what the scheduler calls when a schedule fires. It is
// supplied by the boot path (webui server in Phase 1.5+, CLI scheduler
// daemon in a later phase) so this package stays independent of the
// runner / executor wiring.
//
// Implementations must be non-blocking: spawn the run on a goroutine and
// return promptly so a long-running pipeline does not stall the tick
// loop's other due jobs. The returned runID is stored in
// schedule.last_run_id; an empty string is fine when the dispatcher has
// not minted one yet.
type Dispatcher interface {
	Dispatch(ctx context.Context, schedule state.ScheduleRecord) (runID string, err error)
}

// DispatcherFunc adapts a plain function to the Dispatcher interface.
type DispatcherFunc func(ctx context.Context, schedule state.ScheduleRecord) (string, error)

func (f DispatcherFunc) Dispatch(ctx context.Context, s state.ScheduleRecord) (string, error) {
	return f(ctx, s)
}

// Scheduler reads state.ScheduleStore on every tick, fires every schedule
// whose next_fire_at has passed, and advances next_fire_at to the next
// cron match. Concurrent ticks are guarded so a slow Dispatch on tick N
// does not double-fire on tick N+1.
type Scheduler struct {
	store      state.ScheduleStore
	dispatcher Dispatcher
	tick       time.Duration
	now        func() time.Time

	mu       sync.Mutex
	stopCh   chan struct{}
	doneCh   chan struct{}
	running  bool
}

// Options configures a Scheduler. Zero values yield sensible defaults:
// 30-second tick, time.Now clock.
type Options struct {
	Tick time.Duration
	Now  func() time.Time
}

// New returns a Scheduler bound to the supplied store and dispatcher.
// The scheduler does not start ticking until Start is called.
func New(store state.ScheduleStore, dispatcher Dispatcher, opts Options) *Scheduler {
	if opts.Tick <= 0 {
		opts.Tick = 30 * time.Second
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Scheduler{
		store:      store,
		dispatcher: dispatcher,
		tick:       opts.Tick,
		now:        opts.Now,
	}
}

// Start spawns the tick loop. Returns immediately. Calling Start twice
// without Stop is a no-op.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.mu.Unlock()

	go s.loop(ctx)
}

// Stop halts the tick loop and waits up to 5 seconds for the in-flight
// tick to finish. Safe to call multiple times.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	close(s.stopCh)
	doneCh := s.doneCh
	s.running = false
	s.mu.Unlock()

	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		log.Printf("scheduler: stop timed out waiting for tick loop")
	}
}

func (s *Scheduler) loop(ctx context.Context) {
	defer close(s.doneCh)

	// Fire one tick immediately so a freshly-started server doesn't wait
	// the full Tick interval before processing overdue schedules.
	s.Tick(ctx)

	t := time.NewTicker(s.tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-t.C:
			s.Tick(ctx)
		}
	}
}

// Tick processes all due schedules once. Exposed so callers (tests, manual
// triggers from a webui button) can drive the loop deterministically.
func (s *Scheduler) Tick(ctx context.Context) {
	now := s.now()
	due, err := s.store.ListDueSchedules(now)
	if err != nil {
		log.Printf("scheduler: list due schedules: %v", err)
		return
	}
	for _, sched := range due {
		if err := ctx.Err(); err != nil {
			return
		}
		s.fireOne(ctx, sched, now)
	}
}

func (s *Scheduler) fireOne(ctx context.Context, sched state.ScheduleRecord, now time.Time) {
	expr, err := Parse(sched.CronExpr)
	if err != nil {
		log.Printf("scheduler: parse cron %q for schedule %d: %v", sched.CronExpr, sched.ID, err)
		// Advance next_fire_at by a day so we don't churn the bad row
		// on every tick. Operators can fix the expression and the next
		// scheduled fire will pick up the corrected value.
		_ = s.store.UpdateScheduleNextFire(sched.ID, now.Add(24*time.Hour), sched.LastRunID)
		return
	}

	runID, dispatchErr := s.dispatcher.Dispatch(ctx, sched)
	if dispatchErr != nil {
		log.Printf("scheduler: dispatch schedule %d (%s): %v", sched.ID, sched.PipelineName, dispatchErr)
		// Still advance so a chronically-broken dispatcher doesn't
		// re-fire continuously.
	}

	next, err := expr.NextFire(now)
	if err != nil {
		log.Printf("scheduler: next-fire for schedule %d: %v", sched.ID, err)
		return
	}
	lastRunID := sched.LastRunID
	if runID != "" {
		lastRunID = runID
	}
	if err := s.store.UpdateScheduleNextFire(sched.ID, next, lastRunID); err != nil {
		log.Printf("scheduler: update next-fire schedule %d: %v", sched.ID, err)
	}
}
