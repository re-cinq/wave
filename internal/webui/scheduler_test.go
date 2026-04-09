package webui

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduler_BasicSubmit(t *testing.T) {
	s := NewScheduler(5)

	var executed atomic.Bool
	done := make(chan struct{})

	err := s.Submit(context.Background(), func() {
		executed.Store(true)
		close(done)
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	<-done
	if !executed.Load() {
		t.Error("function was not executed")
	}
}

func TestScheduler_DefaultConcurrency(t *testing.T) {
	s := NewScheduler(0)
	if cap(s.sem) != 5 {
		t.Errorf("expected default capacity 5, got %d", cap(s.sem))
	}

	s2 := NewScheduler(-1)
	if cap(s2.sem) != 5 {
		t.Errorf("expected default capacity 5 for -1, got %d", cap(s2.sem))
	}
}

func TestScheduler_MaxConcurrency(t *testing.T) {
	s := NewScheduler(2)

	var maxConcurrent atomic.Int32
	var current atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 6; i++ {
		wg.Add(1)
		err := s.Submit(context.Background(), func() {
			defer wg.Done()
			c := current.Add(1)
			// Track max concurrent
			for {
				old := maxConcurrent.Load()
				if c <= old || maxConcurrent.CompareAndSwap(old, c) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			current.Add(-1)
		})
		if err != nil {
			t.Fatalf("Submit failed: %v", err)
		}
	}

	wg.Wait()

	if max := maxConcurrent.Load(); max > 2 {
		t.Errorf("max concurrent exceeded limit: got %d, want <= 2", max)
	}
}

func TestScheduler_CancelledContext(t *testing.T) {
	s := NewScheduler(1)

	// Fill the single slot
	blocker := make(chan struct{})
	err := s.Submit(context.Background(), func() {
		<-blocker
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Try to submit with already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = s.Submit(ctx, func() {
		t.Error("should not execute")
	})
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	close(blocker)
	_ = s.Shutdown(context.Background())
}

func TestScheduler_ActiveCount(t *testing.T) {
	s := NewScheduler(5)

	if s.ActiveCount() != 0 {
		t.Errorf("expected 0 active, got %d", s.ActiveCount())
	}

	blocker := make(chan struct{})
	for i := 0; i < 3; i++ {
		_ = s.Submit(context.Background(), func() {
			<-blocker
		})
	}

	// Give goroutines time to start
	time.Sleep(50 * time.Millisecond)

	if s.ActiveCount() != 3 {
		t.Errorf("expected 3 active, got %d", s.ActiveCount())
	}

	close(blocker)
	_ = s.Shutdown(context.Background())

	if s.ActiveCount() != 0 {
		t.Errorf("expected 0 active after shutdown, got %d", s.ActiveCount())
	}
}

func TestScheduler_Shutdown(t *testing.T) {
	s := NewScheduler(5)

	var completed atomic.Int32
	for i := 0; i < 5; i++ {
		_ = s.Submit(context.Background(), func() {
			time.Sleep(50 * time.Millisecond)
			completed.Add(1)
		})
	}

	err := s.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if completed.Load() != 5 {
		t.Errorf("expected 5 completed, got %d", completed.Load())
	}
}

func TestScheduler_ShutdownTimeout(t *testing.T) {
	s := NewScheduler(1)

	blocker := make(chan struct{})
	_ = s.Submit(context.Background(), func() {
		<-blocker
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := s.Shutdown(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}

	close(blocker)
	// Clean up
	_ = s.Shutdown(context.Background())
}

func TestScheduler_SlotRelease(t *testing.T) {
	s := NewScheduler(1)

	// Submit and complete 3 tasks sequentially (capacity 1)
	for i := 0; i < 3; i++ {
		done := make(chan struct{})
		err := s.Submit(context.Background(), func() {
			close(done)
		})
		if err != nil {
			t.Fatalf("Submit %d failed: %v", i, err)
		}
		<-done
		// Wait for slot to be released
		time.Sleep(10 * time.Millisecond)
	}
}
