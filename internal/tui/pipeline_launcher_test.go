package tui

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPipelineLauncher_InitializesEmptyCancelMap(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	assert.NotNil(t, launcher.cancelFns)
	assert.Empty(t, launcher.cancelFns)
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

func TestPipelineLauncher_Cleanup_RemovesEntry(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})

	_, cancel := context.WithCancel(context.Background())
	launcher.mu.Lock()
	launcher.cancelFns["run-to-clean"] = cancel
	launcher.cancelFns["run-to-keep"] = cancel
	launcher.mu.Unlock()

	launcher.Cleanup("run-to-clean")

	launcher.mu.Lock()
	_, exists := launcher.cancelFns["run-to-clean"]
	_, kept := launcher.cancelFns["run-to-keep"]
	launcher.mu.Unlock()

	assert.False(t, exists, "cleaned up entry should be gone")
	assert.True(t, kept, "other entries should remain")
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
