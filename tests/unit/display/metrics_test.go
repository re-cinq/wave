package display_test

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/display"
)

// TestNewPerformanceMetrics tests creation of performance metrics.
func TestNewPerformanceMetrics(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	if metrics == nil {
		t.Fatal("expected NewPerformanceMetrics to return non-nil")
	}

	// Verify initial state
	stats := metrics.GetStats()

	if stats.TotalRenders != 0 {
		t.Errorf("expected 0 renders initially, got %d", stats.TotalRenders)
	}

	if stats.TotalEvents != 0 {
		t.Errorf("expected 0 events initially, got %d", stats.TotalEvents)
	}

	if stats.TargetOverhead != 0.05 {
		t.Errorf("expected target overhead 0.05 (5%%), got %f", stats.TargetOverhead)
	}
}

// TestRecordRenderComplete tests render completion tracking.
func TestRecordRenderComplete(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	// Record some render operations
	durations := []time.Duration{
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,
		5 * time.Millisecond,
	}

	for _, duration := range durations {
		metrics.RecordRenderComplete(duration)
	}

	stats := metrics.GetStats()

	// Verify total renders
	if stats.TotalRenders != int64(len(durations)) {
		t.Errorf("expected %d renders, got %d", len(durations), stats.TotalRenders)
	}

	// Verify average
	expectedAvg := 3.0 // (1+2+3+4+5)/5 = 3
	if stats.AvgRenderTimeMs < expectedAvg-0.1 || stats.AvgRenderTimeMs > expectedAvg+0.1 {
		t.Errorf("expected average around %fms, got %fms", expectedAvg, stats.AvgRenderTimeMs)
	}

	// Verify min/max
	if stats.MinRenderTimeMs < 0.9 || stats.MinRenderTimeMs > 1.1 {
		t.Errorf("expected min around 1ms, got %fms", stats.MinRenderTimeMs)
	}

	if stats.MaxRenderTimeMs < 4.9 || stats.MaxRenderTimeMs > 5.1 {
		t.Errorf("expected max around 5ms, got %fms", stats.MaxRenderTimeMs)
	}
}

// TestRecordRenderStartDeferred tests deferred render tracking.
func TestRecordRenderStartDeferred(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	cleanup := metrics.RecordRenderStart()
	time.Sleep(2 * time.Millisecond)
	cleanup()

	stats := metrics.GetStats()

	if stats.TotalRenders != 1 {
		t.Errorf("expected 1 render, got %d", stats.TotalRenders)
	}

	if stats.LastRenderTimeMs < 1.5 || stats.LastRenderTimeMs > 3.0 {
		t.Errorf("expected render time around 2ms, got %fms", stats.LastRenderTimeMs)
	}
}

// TestRecordRenderFailure tests render failure tracking.
func TestRecordRenderFailure(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	// Record some successful renders
	metrics.RecordRenderComplete(1 * time.Millisecond)
	metrics.RecordRenderComplete(2 * time.Millisecond)

	// Record some failures
	metrics.RecordRenderFailure()
	metrics.RecordRenderFailure()
	metrics.RecordRenderFailure()

	stats := metrics.GetStats()

	if stats.TotalRenders != 2 {
		t.Errorf("expected 2 successful renders, got %d", stats.TotalRenders)
	}

	if stats.FailedRenders != 3 {
		t.Errorf("expected 3 failed renders, got %d", stats.FailedRenders)
	}
}

// TestEventTracking tests event counting.
func TestEventTracking(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	// Record events
	for i := 0; i < 100; i++ {
		metrics.RecordEvent()
	}

	// Record dropped events
	for i := 0; i < 5; i++ {
		metrics.RecordDroppedEvent()
	}

	stats := metrics.GetStats()

	if stats.TotalEvents != 100 {
		t.Errorf("expected 100 events, got %d", stats.TotalEvents)
	}

	if stats.DroppedEvents != 5 {
		t.Errorf("expected 5 dropped events, got %d", stats.DroppedEvents)
	}
}

// TestQueueDepthTracking tests queue depth monitoring.
func TestQueueDepthTracking(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	// Simulate varying queue depth
	depths := []int64{0, 5, 10, 15, 20, 15, 10, 5, 0}

	for _, depth := range depths {
		metrics.UpdateQueueDepth(depth)
	}

	stats := metrics.GetStats()

	// Current depth should be the last value
	if stats.QueuedEvents != 0 {
		t.Errorf("expected current queue depth 0, got %d", stats.QueuedEvents)
	}

	// Max depth should be the highest value
	if stats.MaxQueueDepth != 20 {
		t.Errorf("expected max queue depth 20, got %d", stats.MaxQueueDepth)
	}
}

// TestMemoryTracking tests memory usage tracking.
func TestMemoryTracking(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	// Record increasing memory usage
	memoryValues := []int64{
		1024 * 1024,      // 1 MB
		2 * 1024 * 1024,  // 2 MB
		3 * 1024 * 1024,  // 3 MB
		4 * 1024 * 1024,  // 4 MB
	}

	for _, mem := range memoryValues {
		metrics.RecordMemoryUsage(mem)
		time.Sleep(1 * time.Millisecond)
	}

	stats := metrics.GetStats()

	// Current should be last value
	if stats.CurrentMemoryMB < 3.9 || stats.CurrentMemoryMB > 4.1 {
		t.Errorf("expected current memory around 4MB, got %fMB", stats.CurrentMemoryMB)
	}

	// Peak should be highest value
	if stats.PeakMemoryMB < 3.9 || stats.PeakMemoryMB > 4.1 {
		t.Errorf("expected peak memory around 4MB, got %fMB", stats.PeakMemoryMB)
	}

	// Growth rate should be positive
	if stats.MemoryGrowthKBs <= 0 {
		t.Errorf("expected positive memory growth rate, got %fKB/s", stats.MemoryGrowthKBs)
	}
}

// TestOverheadCalculation tests overhead ratio calculation.
func TestOverheadCalculation(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	// Simulate render time: 5ms
	metrics.RecordRenderComplete(5 * time.Millisecond)

	// Set total execution time: 100ms
	metrics.SetTotalExecutionTime(100 * time.Millisecond)

	stats := metrics.GetStats()

	// Expected overhead: 5/100 = 0.05 = 5%
	expectedOverhead := 0.05
	if stats.OverheadRatio < expectedOverhead-0.01 || stats.OverheadRatio > expectedOverhead+0.01 {
		t.Errorf("expected overhead ratio around %f, got %f", expectedOverhead, stats.OverheadRatio)
	}

	if stats.OverheadPercent < 4.9 || stats.OverheadPercent > 5.1 {
		t.Errorf("expected overhead percent around 5%%, got %f%%", stats.OverheadPercent)
	}

	// Should not exceed target (5%)
	if stats.IsOverTarget {
		t.Error("expected overhead to not exceed target")
	}
}

// TestOverheadTargetExceeded tests overhead target violation detection.
func TestOverheadTargetExceeded(t *testing.T) {
	tests := []struct {
		name            string
		renderTimeMs    int64
		executionTimeMs int64
		shouldExceed    bool
	}{
		{"within target 1%", 1, 100, false},
		{"within target 3%", 3, 100, false},
		{"at target 5%", 5, 100, false},
		{"slightly over 5.1%", 51, 1000, true},
		{"significantly over 10%", 10, 100, true},
		{"very high 20%", 20, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := display.NewPerformanceMetrics()

			// Record render time
			metrics.RecordRenderComplete(time.Duration(tt.renderTimeMs) * time.Millisecond)

			// Set execution time
			metrics.SetTotalExecutionTime(time.Duration(tt.executionTimeMs) * time.Millisecond)

			exceeded := metrics.IsOverheadTargetExceeded()

			if exceeded != tt.shouldExceed {
				stats := metrics.GetStats()
				t.Errorf("expected exceeded=%v, got %v (overhead=%f%%)",
					tt.shouldExceed, exceeded, stats.OverheadPercent)
			}
		})
	}
}

// TestGetAverageRenderTime tests average render time calculation.
func TestGetAverageRenderTime(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	// No renders yet
	if avg := metrics.GetAverageRenderTime(); avg != 0 {
		t.Errorf("expected 0 average initially, got %f", avg)
	}

	// Record some renders
	durations := []time.Duration{
		2 * time.Millisecond,
		4 * time.Millisecond,
		6 * time.Millisecond,
	}

	for _, d := range durations {
		metrics.RecordRenderComplete(d)
	}

	avg := metrics.GetAverageRenderTime()
	expected := 4.0 // (2+4+6)/3 = 4

	if avg < expected-0.1 || avg > expected+0.1 {
		t.Errorf("expected average around %fms, got %fms", expected, avg)
	}
}

// TestReset tests metrics reset functionality.
func TestReset(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	// Record some activity
	for i := 0; i < 10; i++ {
		metrics.RecordRenderComplete(1 * time.Millisecond)
		metrics.RecordEvent()
	}
	metrics.RecordMemoryUsage(1024 * 1024)
	metrics.UpdateQueueDepth(10)

	// Verify activity was recorded
	stats := metrics.GetStats()
	if stats.TotalRenders == 0 || stats.TotalEvents == 0 {
		t.Error("expected activity to be recorded before reset")
	}

	// Reset
	metrics.Reset()

	// Verify everything is cleared
	stats = metrics.GetStats()

	if stats.TotalRenders != 0 {
		t.Errorf("expected 0 renders after reset, got %d", stats.TotalRenders)
	}

	if stats.TotalEvents != 0 {
		t.Errorf("expected 0 events after reset, got %d", stats.TotalEvents)
	}

	if stats.CurrentMemoryMB != 0 {
		t.Errorf("expected 0 memory after reset, got %fMB", stats.CurrentMemoryMB)
	}

	if stats.QueuedEvents != 0 {
		t.Errorf("expected 0 queued events after reset, got %d", stats.QueuedEvents)
	}

	if stats.MaxQueueDepth != 0 {
		t.Errorf("expected 0 max queue depth after reset, got %d", stats.MaxQueueDepth)
	}
}

// TestConcurrentMetrics tests metrics under concurrent access.
func TestConcurrentMetrics(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	done := make(chan bool)

	// Spawn multiple goroutines that record metrics concurrently
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				metrics.RecordRenderComplete(1 * time.Millisecond)
				metrics.RecordEvent()
				metrics.UpdateQueueDepth(int64(j))
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	stats := metrics.GetStats()

	// Should have recorded 100 renders (5 * 20)
	if stats.TotalRenders != 100 {
		t.Errorf("expected 100 renders, got %d", stats.TotalRenders)
	}

	// Should have recorded 100 events (5 * 20)
	if stats.TotalEvents != 100 {
		t.Errorf("expected 100 events, got %d", stats.TotalEvents)
	}
}

// TestPerformanceStatsSnapshot tests that stats return a consistent snapshot.
func TestPerformanceStatsSnapshot(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	// Record some metrics
	metrics.RecordRenderComplete(5 * time.Millisecond)
	metrics.SetTotalExecutionTime(100 * time.Millisecond)

	// Get snapshot
	stats1 := metrics.GetStats()

	// Record more metrics
	metrics.RecordRenderComplete(10 * time.Millisecond)

	// Get another snapshot
	stats2 := metrics.GetStats()

	// First snapshot should be unchanged
	if stats1.TotalRenders != 1 {
		t.Errorf("expected stats1 to have 1 render, got %d", stats1.TotalRenders)
	}

	// Second snapshot should reflect new data
	if stats2.TotalRenders != 2 {
		t.Errorf("expected stats2 to have 2 renders, got %d", stats2.TotalRenders)
	}
}

// TestSetExecutionStart tests execution start timestamp recording.
func TestSetExecutionStart(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	// Set execution start
	metrics.SetExecutionStart()

	// Give it a moment
	time.Sleep(10 * time.Millisecond)

	// Stats should show uptime
	stats := metrics.GetStats()

	if stats.UptimeSeconds < 0.01 {
		t.Errorf("expected uptime > 0.01s, got %fs", stats.UptimeSeconds)
	}
}
