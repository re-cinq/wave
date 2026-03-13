package continuous

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider returns items from a predefined list, then nil.
type mockProvider struct {
	items []*WorkItem
	idx   int
	mu    sync.Mutex
}

func (m *mockProvider) Next(_ context.Context) (*WorkItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.idx >= len(m.items) {
		return nil, nil
	}
	item := m.items[m.idx]
	m.idx++
	return item, nil
}

// errorProvider returns an error on Next.
type errorProvider struct {
	err error
}

func (e *errorProvider) Next(_ context.Context) (*WorkItem, error) {
	return nil, e.err
}

// mockEmitter captures emitted events for inspection.
type mockEmitter struct {
	events []event.Event
	mu     sync.Mutex
}

func (m *mockEmitter) Emit(ev event.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, ev)
}

func (m *mockEmitter) getEvents() []event.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]event.Event, len(m.events))
	copy(cp, m.events)
	return cp
}

// mockStore tracks processed items in memory.
type mockStore struct {
	processed map[string]bool
	mu        sync.Mutex
}

func newMockStore() *mockStore {
	return &mockStore{processed: make(map[string]bool)}
}

func (m *mockStore) MarkItemProcessed(_, itemKey, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processed[itemKey] = true
	return nil
}

func (m *mockStore) IsItemProcessed(_, itemKey string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.processed[itemKey], nil
}

func TestRunnerProcessesAllItems(t *testing.T) {
	items := []*WorkItem{
		{Key: "owner/repo#1", Input: "https://github.com/owner/repo/issues/1"},
		{Key: "owner/repo#2", Input: "https://github.com/owner/repo/issues/2"},
		{Key: "owner/repo#3", Input: "https://github.com/owner/repo/issues/3"},
	}

	provider := &mockProvider{items: items}
	emitter := &mockEmitter{}
	store := newMockStore()

	var processedInputs []string
	var mu sync.Mutex
	factory := func(input string) error {
		mu.Lock()
		processedInputs = append(processedInputs, input)
		mu.Unlock()
		return nil
	}

	r := NewRunner(RunnerConfig{
		Provider:        provider,
		PipelineFactory: factory,
		Emitter:         emitter,
		Store:           store,
		PipelineName:    "test-pipeline",
		Delay:           0,
	})

	err := r.Run(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 3, len(processedInputs))
	assert.Equal(t, "https://github.com/owner/repo/issues/1", processedInputs[0])
	assert.Equal(t, "https://github.com/owner/repo/issues/2", processedInputs[1])
	assert.Equal(t, "https://github.com/owner/repo/issues/3", processedInputs[2])

	// Verify all items marked as processed
	for _, item := range items {
		assert.True(t, store.processed[item.Key])
	}

	// Verify events emitted
	events := emitter.getEvents()
	assert.True(t, len(events) >= 7) // started + 3*(iteration_started + iteration_completed) + exhausted
	assert.Equal(t, event.StateContinuousStarted, events[0].State)
	assert.Equal(t, event.StateContinuousExhausted, events[len(events)-1].State)
}

func TestRunnerContextCancellationStopsLoop(t *testing.T) {
	// Provider returns items forever
	infiniteItems := make([]*WorkItem, 100)
	for i := range infiniteItems {
		infiniteItems[i] = &WorkItem{Key: "key-" + string(rune('0'+i%10)), Input: "input"}
	}
	provider := &mockProvider{items: infiniteItems}
	emitter := &mockEmitter{}

	callCount := 0
	var mu sync.Mutex
	factory := func(_ string) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	r := NewRunner(RunnerConfig{
		Provider:        provider,
		PipelineFactory: factory,
		Emitter:         emitter,
		PipelineName:    "test-pipeline",
		Delay:           10 * time.Millisecond,
	})

	err := r.Run(ctx)
	require.NoError(t, err) // Graceful shutdown returns nil

	mu.Lock()
	assert.True(t, callCount > 0, "should have processed at least one item")
	assert.True(t, callCount < 100, "should not have processed all items")
	mu.Unlock()

	events := emitter.getEvents()
	lastEvent := events[len(events)-1]
	assert.Equal(t, event.StateContinuousStopped, lastEvent.State)
}

func TestRunnerHaltOnError(t *testing.T) {
	items := []*WorkItem{
		{Key: "owner/repo#1", Input: "input-1"},
		{Key: "owner/repo#2", Input: "input-2"},
	}
	provider := &mockProvider{items: items}
	emitter := &mockEmitter{}

	factory := func(_ string) error {
		return errors.New("pipeline failed")
	}

	r := NewRunner(RunnerConfig{
		Provider:        provider,
		PipelineFactory: factory,
		Emitter:         emitter,
		PipelineName:    "test-pipeline",
		HaltOnError:     true,
	})

	err := r.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "halted on iteration 1")

	// Second item should not have been attempted
	provider.mu.Lock()
	assert.Equal(t, 1, provider.idx)
	provider.mu.Unlock()
}

func TestRunnerSkipAndContinue(t *testing.T) {
	items := []*WorkItem{
		{Key: "owner/repo#1", Input: "input-1"},
		{Key: "owner/repo#2", Input: "input-2"},
		{Key: "owner/repo#3", Input: "input-3"},
	}
	provider := &mockProvider{items: items}
	emitter := &mockEmitter{}
	store := newMockStore()

	callCount := 0
	factory := func(_ string) error {
		callCount++
		if callCount == 2 {
			return errors.New("second pipeline failed")
		}
		return nil
	}

	r := NewRunner(RunnerConfig{
		Provider:        provider,
		PipelineFactory: factory,
		Emitter:         emitter,
		Store:           store,
		PipelineName:    "test-pipeline",
		HaltOnError:     false,
	})

	err := r.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, callCount)

	// Check that failed event was emitted
	events := emitter.getEvents()
	var failedCount int
	for _, ev := range events {
		if ev.State == event.StateContinuousIterationFailed {
			failedCount++
		}
	}
	assert.Equal(t, 1, failedCount)
}

func TestRunnerDelayBetweenIterations(t *testing.T) {
	items := []*WorkItem{
		{Key: "owner/repo#1", Input: "input-1"},
		{Key: "owner/repo#2", Input: "input-2"},
	}
	provider := &mockProvider{items: items}
	emitter := &mockEmitter{}

	factory := func(_ string) error { return nil }

	start := time.Now()
	r := NewRunner(RunnerConfig{
		Provider:        provider,
		PipelineFactory: factory,
		Emitter:         emitter,
		PipelineName:    "test-pipeline",
		Delay:           50 * time.Millisecond,
	})

	err := r.Run(context.Background())
	require.NoError(t, err)
	elapsed := time.Since(start)

	// Should have waited at least 50ms * 2 iterations
	assert.True(t, elapsed >= 100*time.Millisecond, "expected delay, got %v", elapsed)
}

func TestRunnerEmptyProviderReturnsImmediately(t *testing.T) {
	provider := &mockProvider{items: nil}
	emitter := &mockEmitter{}

	factory := func(_ string) error { return nil }

	start := time.Now()
	r := NewRunner(RunnerConfig{
		Provider:        provider,
		PipelineFactory: factory,
		Emitter:         emitter,
		PipelineName:    "test-pipeline",
		Delay:           time.Second, // Large delay to verify it doesn't wait
	})

	err := r.Run(context.Background())
	require.NoError(t, err)
	elapsed := time.Since(start)

	assert.True(t, elapsed < 500*time.Millisecond, "should return immediately, took %v", elapsed)

	events := emitter.getEvents()
	assert.Equal(t, 2, len(events)) // started + exhausted
	assert.Equal(t, event.StateContinuousStarted, events[0].State)
	assert.Equal(t, event.StateContinuousExhausted, events[1].State)
}

func TestRunnerMaxIterations(t *testing.T) {
	items := []*WorkItem{
		{Key: "owner/repo#1", Input: "input-1"},
		{Key: "owner/repo#2", Input: "input-2"},
		{Key: "owner/repo#3", Input: "input-3"},
	}
	provider := &mockProvider{items: items}
	emitter := &mockEmitter{}

	callCount := 0
	factory := func(_ string) error {
		callCount++
		return nil
	}

	r := NewRunner(RunnerConfig{
		Provider:        provider,
		PipelineFactory: factory,
		Emitter:         emitter,
		PipelineName:    "test-pipeline",
		MaxIterations:   2,
	})

	err := r.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)

	events := emitter.getEvents()
	lastEvent := events[len(events)-1]
	assert.Equal(t, event.StateContinuousExhausted, lastEvent.State)
	assert.Contains(t, lastEvent.Message, "Max iterations reached")
}

func TestRunnerProviderError(t *testing.T) {
	provider := &errorProvider{err: errors.New("API rate limited")}
	emitter := &mockEmitter{}

	r := NewRunner(RunnerConfig{
		Provider:        provider,
		PipelineFactory: func(_ string) error { return nil },
		Emitter:         emitter,
		PipelineName:    "test-pipeline",
	})

	err := r.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API rate limited")
}
