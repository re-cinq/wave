package commands

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEventCollector records events for verification.
type mockEventCollector struct {
	events []event.Event
}

func (m *mockEventCollector) Emit(e event.Event) {
	m.events = append(m.events, e)
}

func TestGetWaveVersion(t *testing.T) {
	version := getWaveVersion()
	require.NotEmpty(t, version, "getWaveVersion should return a non-empty string")
	// In test/dev builds, the version is typically "dev"
	assert.NotEqual(t, "", version)
}

func TestEmitMetaEvent(t *testing.T) {
	collector := &mockEventCollector{}

	emitMetaEvent(collector, "meta.health_started", "Starting health checks")

	require.Len(t, collector.events, 1, "should have emitted exactly one event")

	evt := collector.events[0]
	assert.Equal(t, "meta.health_started", evt.State)
	assert.Equal(t, "Starting health checks", evt.Message)
	assert.Equal(t, "wave", evt.PipelineID)
	assert.False(t, evt.Timestamp.IsZero(), "timestamp should be set")
	assert.WithinDuration(t, time.Now(), evt.Timestamp, 5*time.Second, "timestamp should be recent")
}

func TestEmitMetaEvent_NilEmitter(t *testing.T) {
	// Should not panic when emitter is nil
	assert.NotPanics(t, func() {
		emitMetaEvent(nil, "meta.health_started", "test")
	})
}
