package pipeline

import (
	"context"
	"testing"
	"time"
)

func newWatchdog(t *testing.T, timeout time.Duration) *StallWatchdog {
	t.Helper()
	wd, err := NewStallWatchdog(timeout)
	if err != nil {
		t.Fatalf("NewStallWatchdog: %v", err)
	}
	return wd
}

func TestStallWatchdog_ActivityPreventsCancel(t *testing.T) {
	wd := newWatchdog(t, 100*time.Millisecond)
	ctx := wd.Start(context.Background())
	defer wd.Stop()

	time.Sleep(50 * time.Millisecond)
	wd.NotifyActivity()
	wd.NotifyProgress()
	time.Sleep(50 * time.Millisecond)
	wd.NotifyActivity()
	wd.NotifyProgress()
	time.Sleep(50 * time.Millisecond)

	select {
	case <-ctx.Done():
		t.Fatal("context cancelled despite activity")
	default:
	}
}

func TestStallWatchdog_SilenceCancels(t *testing.T) {
	wd := newWatchdog(t, 50*time.Millisecond)
	ctx := wd.Start(context.Background())
	defer wd.Stop()

	select {
	case <-ctx.Done():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("context not cancelled after stall timeout")
	}
}

func TestStallWatchdog_ReadOnlyLoopCancels(t *testing.T) {
	wd := newWatchdog(t, 100*time.Millisecond)
	ctx := wd.Start(context.Background())
	defer wd.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
				wd.NotifyActivity() // read-only, no NotifyProgress
			}
		}
	}()

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context not cancelled despite no progress")
	}
	<-done
}

func TestStallWatchdog_WriteProgressPreventsCancel(t *testing.T) {
	wd := newWatchdog(t, 100*time.Millisecond)
	ctx := wd.Start(context.Background())
	defer wd.Stop()

	for i := 0; i < 5; i++ {
		time.Sleep(80 * time.Millisecond)
		wd.NotifyActivity()
		wd.NotifyProgress()
	}

	select {
	case <-ctx.Done():
		t.Fatal("context cancelled despite write progress")
	default:
	}
}

func TestNewStallWatchdog_InvalidTimeout(t *testing.T) {
	for _, tc := range []struct {
		name    string
		timeout time.Duration
	}{
		{"zero", 0},
		{"negative", -time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wd, err := NewStallWatchdog(tc.timeout)
			if err == nil {
				t.Fatal("expected error for invalid timeout")
			}
			if wd != nil {
				t.Fatal("expected nil watchdog when error returned")
			}
		})
	}
}

func TestIsProgressTool(t *testing.T) {
	readOnly := []string{"Read", "Glob", "Grep", "WebSearch", "WebFetch", "ToolSearch", "TaskList", "TaskGet"}
	for _, tool := range readOnly {
		if IsProgressTool(tool) {
			t.Errorf("expected %q to be read-only", tool)
		}
	}
	writeable := []string{"Write", "Edit", "Bash", "NotebookEdit", "Agent"}
	for _, tool := range writeable {
		if !IsProgressTool(tool) {
			t.Errorf("expected %q to be progress tool", tool)
		}
	}
}
