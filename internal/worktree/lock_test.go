package worktree

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRepoLock_BasicAcquireRelease(t *testing.T) {
	rl := newRepoLock()

	ctx := context.Background()
	if err := rl.LockWithContext(ctx); err != nil {
		t.Fatalf("unexpected error acquiring lock: %v", err)
	}

	rl.Unlock()
}

func TestRepoLock_ConcurrentAcquisition(t *testing.T) {
	rl := newRepoLock()
	const goroutines = 10

	var wg sync.WaitGroup
	var counter int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			if err := rl.LockWithContext(ctx); err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			defer rl.Unlock()

			// Critical section: increment and verify atomicity
			val := atomic.AddInt64(&counter, 1)
			time.Sleep(time.Millisecond) // Hold lock briefly
			current := atomic.LoadInt64(&counter)
			if current != val {
				t.Errorf("race detected: expected %d, got %d", val, current)
			}
		}()
	}

	wg.Wait()

	if counter != goroutines {
		t.Errorf("expected counter=%d, got %d", goroutines, counter)
	}
}

func TestRepoLock_Timeout(t *testing.T) {
	rl := newRepoLock()

	// Acquire the lock
	ctx := context.Background()
	if err := rl.LockWithContext(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try to acquire with short timeout — should fail
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := rl.LockWithContext(timeoutCtx)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !contains(err.Error(), "timed out") {
		t.Errorf("expected timeout error message, got: %v", err)
	}

	rl.Unlock()
}

func TestRepoLock_ContextCancellation(t *testing.T) {
	rl := newRepoLock()

	// Acquire the lock
	ctx := context.Background()
	if err := rl.LockWithContext(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try to acquire with cancelled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := rl.LockWithContext(cancelCtx)
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}

	rl.Unlock()
}

func TestRepoLock_CrossRepoParallelism(t *testing.T) {
	// Two different repos should be lockable simultaneously
	lock1 := getRepoLock("/repo/one")
	lock2 := getRepoLock("/repo/two")

	ctx := context.Background()

	// Acquire both locks
	if err := lock1.LockWithContext(ctx); err != nil {
		t.Fatalf("failed to lock repo1: %v", err)
	}
	if err := lock2.LockWithContext(ctx); err != nil {
		t.Fatalf("failed to lock repo2: %v", err)
	}

	// Both acquired — unlock both
	lock1.Unlock()
	lock2.Unlock()
}

func TestRepoLock_SameRepoSameLock(t *testing.T) {
	// Same canonical path should return the same lock
	lock1 := getRepoLock("/same/repo")
	lock2 := getRepoLock("/same/repo")

	if lock1 != lock2 {
		t.Error("expected same lock instance for same repo path")
	}
}

func TestRepoLock_DeferUnlock(t *testing.T) {
	rl := newRepoLock()

	// Simulate defer-based unlock even on panic recovery
	func() {
		ctx := context.Background()
		if err := rl.LockWithContext(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer rl.Unlock()
		// Simulating work that might panic — the defer ensures unlock
	}()

	// Lock should be available again
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := rl.LockWithContext(ctx); err != nil {
		t.Fatalf("lock should be available after defer unlock: %v", err)
	}
	rl.Unlock()
}

func TestCanonicalPath(t *testing.T) {
	// canonicalPath should return absolute path
	path, err := canonicalPath(".")
	if err != nil {
		t.Fatalf("canonicalPath failed: %v", err)
	}

	if path == "" {
		t.Error("expected non-empty canonical path")
	}

	if path == "." {
		t.Error("expected absolute path, got relative")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
