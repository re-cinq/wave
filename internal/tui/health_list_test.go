package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type healthListTestProvider struct {
	names []string
}

func (p *healthListTestProvider) CheckNames() []string {
	return p.names
}

func (p *healthListTestProvider) RunCheck(name string) HealthCheckResultMsg {
	return HealthCheckResultMsg{
		Name:    name,
		Status:  HealthCheckOK,
		Message: "OK",
	}
}

func TestHealthListModel_CompletionDetection(t *testing.T) {
	provider := &healthListTestProvider{
		names: []string{"Check A", "Check B", "Check C"},
	}
	m := NewHealthListModel(provider)

	assert.Equal(t, 3, m.totalChecks)
	assert.Equal(t, 0, m.completedCount)

	// Send first two checks — no completion yet
	m, cmd := m.Update(HealthCheckResultMsg{Name: "Check A", Status: HealthCheckOK, Message: "ok"})
	assert.Nil(t, cmd)
	assert.Equal(t, 1, m.completedCount)

	m, cmd = m.Update(HealthCheckResultMsg{Name: "Check B", Status: HealthCheckWarn, Message: "warn"})
	assert.Nil(t, cmd)
	assert.Equal(t, 2, m.completedCount)

	// Send last check — should emit HealthPhaseCompleteMsg
	m, cmd = m.Update(HealthCheckResultMsg{Name: "Check C", Status: HealthCheckOK, Message: "ok"})
	require.NotNil(t, cmd)
	assert.Equal(t, 3, m.completedCount)

	msg := cmd()
	completeMsg, ok := msg.(HealthPhaseCompleteMsg)
	require.True(t, ok, "should emit HealthPhaseCompleteMsg")
	assert.True(t, completeMsg.AllPassed, "allPassed should be true (no errors)")
	// 2 OK + 1 warning = allPassed true (only errors make it false)
	assert.Contains(t, completeMsg.Summary, "2/3 passed, 1 warning")
}

func TestHealthListModel_CompletionWithFailure(t *testing.T) {
	provider := &healthListTestProvider{
		names: []string{"Check A", "Check B"},
	}
	m := NewHealthListModel(provider)

	m, _ = m.Update(HealthCheckResultMsg{Name: "Check A", Status: HealthCheckErr, Message: "failed"})
	m, cmd := m.Update(HealthCheckResultMsg{Name: "Check B", Status: HealthCheckOK, Message: "ok"})
	require.NotNil(t, cmd)

	msg := cmd()
	completeMsg, ok := msg.(HealthPhaseCompleteMsg)
	require.True(t, ok)
	assert.False(t, completeMsg.AllPassed, "allPassed should be false with errors")
	assert.Contains(t, completeMsg.Summary, "failed")
}

func TestHealthListModel_AllPassed(t *testing.T) {
	provider := &healthListTestProvider{names: []string{"A"}}
	m := NewHealthListModel(provider)
	m.checks[0].Status = HealthCheckOK
	assert.True(t, m.allPassed())

	m.checks[0].Status = HealthCheckErr
	assert.False(t, m.allPassed())
}

func TestHealthListModel_Timeout(t *testing.T) {
	provider := &healthListTestProvider{names: []string{"A", "B"}}
	m := NewHealthListModel(provider)

	// Complete only one check
	m, _ = m.Update(HealthCheckResultMsg{Name: "A", Status: HealthCheckOK, Message: "ok"})

	// Simulate timeout
	m, cmd := m.Update(healthTimeoutMsg{})
	require.NotNil(t, cmd)

	// Check B should be marked as warning
	assert.Equal(t, HealthCheckWarn, m.checks[1].Status)
	assert.Equal(t, "Check timed out", m.checks[1].Message)

	msg := cmd()
	completeMsg, ok := msg.(HealthPhaseCompleteMsg)
	require.True(t, ok)
	assert.Contains(t, completeMsg.Summary, "timed out")
}
