package tui

import (
	"context"
	"sync"
	"testing"

	"github.com/recinq/wave/internal/event"
	"github.com/stretchr/testify/assert"
)

func TestNewPipelineLauncher_InitializesEmptyMaps(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	assert.NotNil(t, launcher.cancelFns)
	assert.Empty(t, launcher.cancelFns)
	assert.NotNil(t, launcher.buffers)
	assert.Empty(t, launcher.buffers)
}

func TestPipelineLauncher_Cancel_UnknownRunID_IsNoOp(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	// Should not panic
	launcher.Cancel("nonexistent-run-id")
}

func TestPipelineLauncher_Cancel_InvokesStoredCancelFunc(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})

	ctx, cancel := context.WithCancel(context.Background())
	launcher.mu.Lock()
	launcher.cancelFns["test-run-1"] = cancel
	launcher.mu.Unlock()

	launcher.Cancel("test-run-1")

	// Context should be cancelled
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.Canceled, ctx.Err())
}

func TestPipelineLauncher_CancelAll_InvokesAllCancelFuncs(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	ctx3, cancel3 := context.WithCancel(context.Background())

	launcher.mu.Lock()
	launcher.cancelFns["run-1"] = cancel1
	launcher.cancelFns["run-2"] = cancel2
	launcher.cancelFns["run-3"] = cancel3
	launcher.mu.Unlock()

	launcher.CancelAll()

	assert.Error(t, ctx1.Err())
	assert.Error(t, ctx2.Err())
	assert.Error(t, ctx3.Err())
	assert.Empty(t, launcher.cancelFns, "map should be cleared after CancelAll")
}

func TestPipelineLauncher_CancelAll_EmptyMap_IsNoOp(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	// Should not panic
	launcher.CancelAll()
	assert.Empty(t, launcher.cancelFns)
}

func TestPipelineLauncher_Cleanup_RemovesCancelAndBuffer(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})

	_, cancel := context.WithCancel(context.Background())
	launcher.mu.Lock()
	launcher.cancelFns["run-to-clean"] = cancel
	launcher.cancelFns["run-to-keep"] = cancel
	launcher.buffers["run-to-clean"] = NewEventBuffer(10)
	launcher.buffers["run-to-keep"] = NewEventBuffer(10)
	launcher.mu.Unlock()

	launcher.Cleanup("run-to-clean")

	launcher.mu.Lock()
	_, cancelExists := launcher.cancelFns["run-to-clean"]
	_, cancelKept := launcher.cancelFns["run-to-keep"]
	_, bufExists := launcher.buffers["run-to-clean"]
	_, bufKept := launcher.buffers["run-to-keep"]
	launcher.mu.Unlock()

	assert.False(t, cancelExists, "cleaned up cancel entry should be gone")
	assert.True(t, cancelKept, "other cancel entries should remain")
	assert.False(t, bufExists, "cleaned up buffer entry should be gone")
	assert.True(t, bufKept, "other buffer entries should remain")
}

func TestPipelineLauncher_Cleanup_NonexistentRunID_IsNoOp(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	// Should not panic
	launcher.Cleanup("nonexistent-run-id")
}

func TestPipelineLauncher_ConcurrentCancelAndCleanup(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})

	// Add several cancel functions
	for i := 0; i < 10; i++ {
		_, cancel := context.WithCancel(context.Background())
		launcher.mu.Lock()
		launcher.cancelFns[string(rune('A'+i))] = cancel
		launcher.mu.Unlock()
	}

	// Concurrently cancel and cleanup
	var wg sync.WaitGroup
	wg.Add(20)
	for i := 0; i < 10; i++ {
		id := string(rune('A' + i))
		go func() {
			defer wg.Done()
			launcher.Cancel(id)
		}()
		go func() {
			defer wg.Done()
			launcher.Cleanup(id)
		}()
	}
	wg.Wait()
}

func TestPipelineLauncher_Launch_MissingPipelineDir_ReturnsError(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{
		PipelinesDir: "/nonexistent/dir",
	})

	cmd := launcher.Launch(LaunchConfig{PipelineName: "nonexistent"})
	assert.NotNil(t, cmd)

	msg := cmd()
	errMsg, ok := msg.(LaunchErrorMsg)
	assert.True(t, ok, "should return LaunchErrorMsg")
	assert.Contains(t, errMsg.Err.Error(), "loading pipeline")
}

func TestPipelineLauncher_SetProgram(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	assert.Nil(t, launcher.program)

	// SetProgram with nil should not panic
	launcher.SetProgram(nil)
	assert.Nil(t, launcher.program)
}

func TestPipelineLauncher_GetBuffer_ReturnsNilForUnknown(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	buf := launcher.GetBuffer("nonexistent")
	assert.Nil(t, buf)
}

func TestPipelineLauncher_GetBuffer_ReturnsBuffer(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	expected := NewEventBuffer(100)
	launcher.mu.Lock()
	launcher.buffers["run-1"] = expected
	launcher.mu.Unlock()

	buf := launcher.GetBuffer("run-1")
	assert.Equal(t, expected, buf)
}

func TestPipelineLauncher_HasBuffer(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	assert.False(t, launcher.HasBuffer("run-1"))

	launcher.mu.Lock()
	launcher.buffers["run-1"] = NewEventBuffer(100)
	launcher.mu.Unlock()

	assert.True(t, launcher.HasBuffer("run-1"))
}

func TestTUIProgressEmitter_EmitProgress_NilProgram(t *testing.T) {
	emitter := &TUIProgressEmitter{program: nil, runID: "run-1"}
	// Should not panic with nil program
	err := emitter.EmitProgress(event.Event{State: event.StateStarted})
	assert.NoError(t, err)
}

func TestPipelineLauncher_DismissRun_ActiveRun_CallsCancel(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})

	ctx, cancel := context.WithCancel(context.Background())
	launcher.mu.Lock()
	launcher.cancelFns["test-run"] = cancel
	launcher.mu.Unlock()

	launcher.DismissRun("test-run")

	assert.Error(t, ctx.Err())
	assert.Equal(t, context.Canceled, ctx.Err())
}

func TestPipelineLauncher_DismissRun_StaleRun_NilStore_IsNoOp(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})

	// No cancel function and no store — should not panic
	assert.NotPanics(t, func() {
		launcher.DismissRun("stale-run")
	})
}

func TestPipelineLauncher_DismissRun_UnknownRun_IsNoOp(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	// Should not panic
	launcher.DismissRun("nonexistent")
}

func TestDBLoggingEmitter_SkipsEmptyHeartbeats(t *testing.T) {
	var emitted []event.Event
	inner := &captureEmitter{events: &emitted}
	d := &dbLoggingEmitter{
		inner: inner,
		store: nil, // nil store — LogEvent won't be called
		runID: "run-1",
	}

	// Empty heartbeat — should still call inner.Emit but skip LogEvent
	d.Emit(event.Event{State: "step_progress"})
	assert.Len(t, emitted, 1, "inner.Emit should still be called")
}

// captureEmitter is a test helper that captures emitted events.
type captureEmitter struct {
	events *[]event.Event
}

func (c *captureEmitter) Emit(ev event.Event) {
	*c.events = append(*c.events, ev)
}
