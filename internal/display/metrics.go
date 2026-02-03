// Package display provides performance monitoring for progress display overhead.
package display

import (
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceMetrics tracks performance overhead of the progress display system.
// Target: <5% overhead relative to total execution time.
type PerformanceMetrics struct {
	mu sync.RWMutex

	// Rendering statistics
	TotalRenderCalls   int64         // Total number of render operations
	TotalRenderTimeNs  int64         // Total time spent rendering (nanoseconds)
	LastRenderTimeNs   int64         // Last render operation duration
	MaxRenderTimeNs    int64         // Maximum render time observed
	MinRenderTimeNs    int64         // Minimum render time observed
	AvgRenderTimeNs    int64         // Average render time
	RenderTimeHistory  []int64       // Recent render times for trending
	HistoryMaxSize     int           // Maximum history size
	LastRenderTime     time.Time     // Timestamp of last render
	FailedRenderCalls  int64         // Number of failed render attempts

	// Memory statistics
	CurrentMemoryBytes int64         // Current memory usage estimate
	PeakMemoryBytes    int64         // Peak memory usage observed
	MemoryGrowthRate   float64       // Memory growth rate (bytes/second)

	// Overall timing
	StartTime          time.Time     // When metrics tracking started
	ExecutionStartTime time.Time     // When actual execution started
	TotalExecutionNs   int64         // Total execution time (nanoseconds)

	// Overhead calculation
	OverheadRatio      float64       // Render overhead / total execution time
	TargetOverhead     float64       // Target maximum overhead (default: 0.05 = 5%)
	IsOverheadExceeded bool          // Whether overhead target is exceeded

	// Event statistics
	TotalEvents        int64         // Total events processed
	DroppedEvents      int64         // Events dropped due to backpressure
	QueuedEvents       int64         // Events currently queued
	MaxQueueDepth      int64         // Maximum queue depth observed

	// Update frequency
	UpdateCount        int64         // Total number of display updates
	UpdateIntervalMs   int64         // Target update interval in milliseconds
	ActualIntervalMs   int64         // Actual average update interval
}

// NewPerformanceMetrics creates a new performance metrics tracker.
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		StartTime:        time.Now(),
		TargetOverhead:   0.05, // 5% target overhead
		HistoryMaxSize:   100,
		RenderTimeHistory: make([]int64, 0, 100),
		MinRenderTimeNs:  1<<63 - 1, // Max int64
	}
}

// SetExecutionStart marks the start of actual execution (not including setup).
func (pm *PerformanceMetrics) SetExecutionStart() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.ExecutionStartTime = time.Now()
}

// RecordRenderStart returns a function to call when rendering completes.
// Usage: defer pm.RecordRenderStart()()
func (pm *PerformanceMetrics) RecordRenderStart() func() {
	startTime := time.Now()
	return func() {
		pm.RecordRenderComplete(time.Since(startTime))
	}
}

// RecordRenderComplete records a completed render operation.
func (pm *PerformanceMetrics) RecordRenderComplete(duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	durationNs := duration.Nanoseconds()

	// Update counters
	atomic.AddInt64(&pm.TotalRenderCalls, 1)
	atomic.AddInt64(&pm.TotalRenderTimeNs, durationNs)
	atomic.StoreInt64(&pm.LastRenderTimeNs, durationNs)

	// Update min/max
	if durationNs < pm.MinRenderTimeNs {
		pm.MinRenderTimeNs = durationNs
	}
	if durationNs > pm.MaxRenderTimeNs {
		pm.MaxRenderTimeNs = durationNs
	}

	// Update average
	if pm.TotalRenderCalls > 0 {
		pm.AvgRenderTimeNs = pm.TotalRenderTimeNs / pm.TotalRenderCalls
	}

	// Update history
	pm.RenderTimeHistory = append(pm.RenderTimeHistory, durationNs)
	if len(pm.RenderTimeHistory) > pm.HistoryMaxSize {
		// Keep only recent history
		pm.RenderTimeHistory = pm.RenderTimeHistory[len(pm.RenderTimeHistory)-pm.HistoryMaxSize:]
	}

	pm.LastRenderTime = time.Now()

	// Calculate overhead
	pm.calculateOverhead()
}

// RecordRenderFailure records a failed render attempt.
func (pm *PerformanceMetrics) RecordRenderFailure() {
	atomic.AddInt64(&pm.FailedRenderCalls, 1)
}

// RecordEvent records a processed event.
func (pm *PerformanceMetrics) RecordEvent() {
	atomic.AddInt64(&pm.TotalEvents, 1)
}

// RecordDroppedEvent records an event that was dropped.
func (pm *PerformanceMetrics) RecordDroppedEvent() {
	atomic.AddInt64(&pm.DroppedEvents, 1)
}

// UpdateQueueDepth updates the current queue depth.
func (pm *PerformanceMetrics) UpdateQueueDepth(depth int64) {
	atomic.StoreInt64(&pm.QueuedEvents, depth)
	if depth > atomic.LoadInt64(&pm.MaxQueueDepth) {
		atomic.StoreInt64(&pm.MaxQueueDepth, depth)
	}
}

// RecordMemoryUsage updates memory usage statistics.
func (pm *PerformanceMetrics) RecordMemoryUsage(bytes int64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.CurrentMemoryBytes = bytes
	if bytes > pm.PeakMemoryBytes {
		pm.PeakMemoryBytes = bytes
	}

	// Calculate growth rate
	elapsed := time.Since(pm.StartTime).Seconds()
	if elapsed > 0 {
		pm.MemoryGrowthRate = float64(bytes) / elapsed
	}
}

// SetTotalExecutionTime sets the total execution time for overhead calculation.
func (pm *PerformanceMetrics) SetTotalExecutionTime(duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.TotalExecutionNs = duration.Nanoseconds()
	pm.calculateOverhead()
}

// calculateOverhead computes the overhead ratio.
// Must be called with lock held.
func (pm *PerformanceMetrics) calculateOverhead() {
	if pm.TotalExecutionNs > 0 {
		pm.OverheadRatio = float64(pm.TotalRenderTimeNs) / float64(pm.TotalExecutionNs)
		pm.IsOverheadExceeded = pm.OverheadRatio > pm.TargetOverhead
	}
}

// GetOverheadRatio returns the current overhead ratio (render time / execution time).
func (pm *PerformanceMetrics) GetOverheadRatio() float64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.OverheadRatio
}

// IsOverheadTargetExceeded returns true if overhead exceeds the target.
func (pm *PerformanceMetrics) IsOverheadTargetExceeded() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.IsOverheadExceeded
}

// GetAverageRenderTime returns the average render time in milliseconds.
func (pm *PerformanceMetrics) GetAverageRenderTime() float64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return float64(pm.AvgRenderTimeNs) / 1e6 // Convert to milliseconds
}

// GetStats returns a snapshot of current metrics.
func (pm *PerformanceMetrics) GetStats() PerformanceStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return PerformanceStats{
		TotalRenders:       pm.TotalRenderCalls,
		FailedRenders:      pm.FailedRenderCalls,
		AvgRenderTimeMs:    float64(pm.AvgRenderTimeNs) / 1e6,
		MinRenderTimeMs:    float64(pm.MinRenderTimeNs) / 1e6,
		MaxRenderTimeMs:    float64(pm.MaxRenderTimeNs) / 1e6,
		LastRenderTimeMs:   float64(pm.LastRenderTimeNs) / 1e6,
		TotalEvents:        pm.TotalEvents,
		DroppedEvents:      pm.DroppedEvents,
		QueuedEvents:       pm.QueuedEvents,
		MaxQueueDepth:      pm.MaxQueueDepth,
		OverheadRatio:      pm.OverheadRatio,
		OverheadPercent:    pm.OverheadRatio * 100,
		TargetOverhead:     pm.TargetOverhead,
		IsOverTarget:       pm.IsOverheadExceeded,
		CurrentMemoryMB:    float64(pm.CurrentMemoryBytes) / 1024 / 1024,
		PeakMemoryMB:       float64(pm.PeakMemoryBytes) / 1024 / 1024,
		MemoryGrowthKBs:    pm.MemoryGrowthRate / 1024,
		UptimeSeconds:      time.Since(pm.StartTime).Seconds(),
	}
}

// PerformanceStats is a snapshot of performance metrics.
type PerformanceStats struct {
	TotalRenders       int64   // Total render calls
	FailedRenders      int64   // Failed render calls
	AvgRenderTimeMs    float64 // Average render time in ms
	MinRenderTimeMs    float64 // Minimum render time in ms
	MaxRenderTimeMs    float64 // Maximum render time in ms
	LastRenderTimeMs   float64 // Last render time in ms
	TotalEvents        int64   // Total events processed
	DroppedEvents      int64   // Events dropped
	QueuedEvents       int64   // Events currently queued
	MaxQueueDepth      int64   // Maximum queue depth
	OverheadRatio      float64 // Overhead ratio (0.0-1.0)
	OverheadPercent    float64 // Overhead as percentage
	TargetOverhead     float64 // Target overhead
	IsOverTarget       bool    // Whether overhead exceeds target
	CurrentMemoryMB    float64 // Current memory usage in MB
	PeakMemoryMB       float64 // Peak memory usage in MB
	MemoryGrowthKBs    float64 // Memory growth rate in KB/s
	UptimeSeconds      float64 // Total uptime in seconds
}

// Reset resets all metrics.
func (pm *PerformanceMetrics) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.TotalRenderCalls = 0
	pm.TotalRenderTimeNs = 0
	pm.LastRenderTimeNs = 0
	pm.MaxRenderTimeNs = 0
	pm.MinRenderTimeNs = 1<<63 - 1
	pm.AvgRenderTimeNs = 0
	pm.RenderTimeHistory = make([]int64, 0, pm.HistoryMaxSize)
	pm.FailedRenderCalls = 0
	pm.CurrentMemoryBytes = 0
	pm.PeakMemoryBytes = 0
	pm.MemoryGrowthRate = 0
	pm.StartTime = time.Now()
	pm.TotalExecutionNs = 0
	pm.OverheadRatio = 0
	pm.IsOverheadExceeded = false
	pm.TotalEvents = 0
	pm.DroppedEvents = 0
	pm.QueuedEvents = 0
	pm.MaxQueueDepth = 0
	pm.UpdateCount = 0
}
