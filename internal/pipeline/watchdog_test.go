package pipeline

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStallWatchdog_TimeoutFires(t *testing.T) {
	w := NewStallWatchdog(50 * time.Millisecond)
	ctx := w.Start(context.Background())
	defer w.Stop()

	select {
	case <-ctx.Done():
		assert.Equal(t, context.Canceled, ctx.Err())
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected context to be canceled by stall watchdog within 200ms")
	}
}

func TestStallWatchdog_ActivityResetsTimeout(t *testing.T) {
	w := NewStallWatchdog(100 * time.Millisecond)
	ctx := w.Start(context.Background())
	defer w.Stop()

	// Send activity every 50ms for ~250ms (5 ticks).
	stopActivity := make(chan struct{})
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		count := 0
		for range ticker.C {
			w.NotifyActivity()
			count++
			if count >= 5 {
				close(stopActivity)
				return
			}
		}
	}()

	// At 150ms the context should still be alive because activity keeps
	// resetting the timer.
	select {
	case <-ctx.Done():
		t.Fatal("context canceled too early; activity should have kept it alive")
	case <-time.After(150 * time.Millisecond):
		// good — still alive
	}

	// Wait for the activity goroutine to finish sending.
	<-stopActivity

	// After activity stops, the watchdog should fire within its 100ms
	// timeout. Give a generous 300ms to avoid CI flakes.
	select {
	case <-ctx.Done():
		assert.Equal(t, context.Canceled, ctx.Err())
	case <-time.After(300 * time.Millisecond):
		t.Fatal("expected context to be canceled after activity stopped")
	}
}

func TestStallWatchdog_StopPreventsTimeout(t *testing.T) {
	w := NewStallWatchdog(50 * time.Millisecond)
	_ = w.Start(context.Background())
	w.Stop()

	// The done channel should be closed after Stop returns.
	select {
	case <-w.done:
		// success — goroutine exited
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected done channel to be closed after Stop")
	}
}

func TestStallWatchdog_ParentContextCancellation(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	w := NewStallWatchdog(5 * time.Second) // long timeout — should not fire
	ctx := w.Start(parent)
	defer w.Stop()

	cancelParent()

	select {
	case <-ctx.Done():
		// Parent cancellation propagated.
		require.Error(t, ctx.Err())
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected context to be canceled when parent is canceled")
	}
}

func TestNewStallWatchdog_PanicsOnZero(t *testing.T) {
	assert.Panics(t, func() {
		NewStallWatchdog(0)
	})
}

func TestNewStallWatchdog_PanicsOnNegative(t *testing.T) {
	assert.Panics(t, func() {
		NewStallWatchdog(-1 * time.Second)
	})
}

func TestStallWatchdog_ConcurrentActivity(t *testing.T) {
	w := NewStallWatchdog(5 * time.Second)
	_ = w.Start(context.Background())
	defer w.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				w.NotifyActivity()
			}
		}()
	}

	// If there is a data race or panic, the race detector / test runner
	// will catch it. We just need all goroutines to complete cleanly.
	wg.Wait()
}
