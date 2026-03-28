package webui

import (
	"context"
	"sync"
	"sync/atomic"
)

// Scheduler manages concurrent pipeline run execution with a FIFO queue
// and configurable concurrency limit.
type Scheduler struct {
	sem    chan struct{}
	active atomic.Int32
	wg     sync.WaitGroup
}

// NewScheduler creates a scheduler with the given max concurrency.
// If maxConcurrent <= 0, defaults to 5.
func NewScheduler(maxConcurrent int) *Scheduler {
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	return &Scheduler{
		sem: make(chan struct{}, maxConcurrent),
	}
}

// Submit queues a function for execution. It blocks until a concurrency slot
// is available or ctx is cancelled. The function runs in a background goroutine.
func (s *Scheduler) Submit(ctx context.Context, fn func()) error {
	select {
	case s.sem <- struct{}{}:
		// Got a slot
	case <-ctx.Done():
		return ctx.Err()
	}

	s.wg.Add(1)
	s.active.Add(1)
	go func() {
		defer func() {
			s.active.Add(-1)
			s.wg.Done()
			<-s.sem // release slot
		}()
		fn()
	}()

	return nil
}

// ActiveCount returns the number of currently executing runs.
func (s *Scheduler) ActiveCount() int {
	return int(s.active.Load())
}

// MaxConcurrency returns the maximum number of concurrent runs allowed.
func (s *Scheduler) MaxConcurrency() int {
	return cap(s.sem)
}

// Shutdown waits for all in-flight runs to complete or ctx to expire.
func (s *Scheduler) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
