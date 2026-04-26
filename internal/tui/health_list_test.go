package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/suggest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock provider
// ---------------------------------------------------------------------------

type mockHealthProvider struct {
	names   []string
	results map[string]HealthCheckResultMsg
}

func (p *mockHealthProvider) RunCheck(name string) HealthCheckResultMsg {
	if r, ok := p.results[name]; ok {
		return r
	}
	return HealthCheckResultMsg{Name: name, Status: suggest.StatusOK, Message: "ok"}
}

func (p *mockHealthProvider) CheckNames() []string {
	return p.names
}

func newMockHealthProvider(names ...string) *mockHealthProvider {
	return &mockHealthProvider{
		names:   names,
		results: make(map[string]HealthCheckResultMsg),
	}
}

// newHealthListModelWithSize builds a HealthListModel and sets a size so View works.
func newHealthListModelWithSize(provider HealthDataProvider) HealthListModel {
	m := NewHealthListModel(provider)
	m.SetSize(40, 20)
	return m
}

// executeCmd runs a tea.Cmd and returns the resulting message.
func executeCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// ===========================================================================
// T005: Health completion tracking
// ===========================================================================

func TestHealthListModel_AllOK_EmitsHealthAllCompleteMsg_NoErrors(t *testing.T) {
	names := []string{"check-a", "check-b", "check-c"}
	provider := newMockHealthProvider(names...)
	m := newHealthListModelWithSize(provider)

	// Deliver OK results for all checks
	var lastCmd tea.Cmd
	for _, name := range names {
		m, lastCmd = m.Update(HealthCheckResultMsg{
			Name:    name,
			Status:  suggest.StatusOK,
			Message: "ok",
		})
	}

	// The last update should have emitted HealthAllCompleteMsg
	require.NotNil(t, lastCmd, "final check result should emit a command")
	msg := executeCmd(lastCmd)
	complete, ok := msg.(HealthAllCompleteMsg)
	require.True(t, ok, "expected HealthAllCompleteMsg, got %T", msg)
	assert.False(t, complete.HasErrors, "HasErrors should be false when all checks pass")
}

func TestHealthListModel_OneError_EmitsHealthAllCompleteMsg_HasErrors(t *testing.T) {
	names := []string{"check-a", "check-b"}
	provider := newMockHealthProvider(names...)
	m := newHealthListModelWithSize(provider)

	// Deliver one OK and one error
	m, _ = m.Update(HealthCheckResultMsg{Name: "check-a", Status: suggest.StatusOK, Message: "ok"})
	_, cmd := m.Update(HealthCheckResultMsg{Name: "check-b", Status: suggest.StatusErr, Message: "failed"})

	require.NotNil(t, cmd, "completing all checks should emit a command")
	msg := executeCmd(cmd)
	complete, ok := msg.(HealthAllCompleteMsg)
	require.True(t, ok, "expected HealthAllCompleteMsg, got %T", msg)
	assert.True(t, complete.HasErrors, "HasErrors should be true when one check errors")
}

func TestHealthListModel_WarnStatus_EmitsHealthAllCompleteMsg_NoErrors(t *testing.T) {
	names := []string{"check-a"}
	provider := newMockHealthProvider(names...)
	m := newHealthListModelWithSize(provider)

	_, cmd := m.Update(HealthCheckResultMsg{Name: "check-a", Status: suggest.StatusWarn, Message: "warn"})

	require.NotNil(t, cmd)
	msg := executeCmd(cmd)
	complete, ok := msg.(HealthAllCompleteMsg)
	require.True(t, ok, "expected HealthAllCompleteMsg, got %T", msg)
	// Warn is not Err, so HasErrors should be false
	assert.False(t, complete.HasErrors)
}

func TestHealthListModel_PartialCompletion_NoHealthAllCompleteMsg(t *testing.T) {
	names := []string{"check-a", "check-b", "check-c"}
	provider := newMockHealthProvider(names...)
	m := newHealthListModelWithSize(provider)

	// Deliver only first result — check-b and check-c are still Checking
	m, cmd := m.Update(HealthCheckResultMsg{Name: "check-a", Status: suggest.StatusOK, Message: "ok"})

	// No HealthAllCompleteMsg should be emitted yet
	if cmd != nil {
		msg := executeCmd(cmd)
		_, ok := msg.(HealthAllCompleteMsg)
		assert.False(t, ok, "HealthAllCompleteMsg should not be emitted when checks are still running")
	}

	// Deliver second but not third
	m, cmd = m.Update(HealthCheckResultMsg{Name: "check-b", Status: suggest.StatusOK, Message: "ok"})
	if cmd != nil {
		msg := executeCmd(cmd)
		_, ok := msg.(HealthAllCompleteMsg)
		assert.False(t, ok, "HealthAllCompleteMsg should not be emitted with one check still running")
	}

	_ = m
}

func TestHealthListModel_RerunKey_ResetsCompletionTracking(t *testing.T) {
	names := []string{"check-a", "check-b"}
	provider := newMockHealthProvider(names...)
	m := newHealthListModelWithSize(provider)

	// Complete all checks
	m, _ = m.Update(HealthCheckResultMsg{Name: "check-a", Status: suggest.StatusOK, Message: "ok"})
	m, _ = m.Update(HealthCheckResultMsg{Name: "check-b", Status: suggest.StatusOK, Message: "ok"})

	// Verify all resolved
	for _, check := range m.checks {
		assert.NotEqual(t, HealthCheckChecking, check.Status, "all checks should be resolved before re-run")
	}

	// Press 'r' to re-run all checks
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	// All checks should be reset to Checking
	for _, check := range m.checks {
		assert.Equal(t, HealthCheckChecking, check.Status, "re-run should reset all checks to Checking")
	}

	// The returned cmd should be a batch to re-run checks (non-nil)
	assert.NotNil(t, cmd, "r key should return batch commands to re-run checks")
}

func TestHealthListModel_EmptyProvider_NoHealthAllCompleteMsg(t *testing.T) {
	provider := newMockHealthProvider() // no checks
	m := newHealthListModelWithSize(provider)

	// checkAllComplete returns nil for empty checks list
	cmd := m.checkAllComplete()
	assert.Nil(t, cmd, "empty check list should not emit HealthAllCompleteMsg")
}

func TestHealthListModel_TableDriven_CompletionVariants(t *testing.T) {
	tests := []struct {
		name          string
		statuses      []suggest.Status
		wantHasErrors bool
	}{
		{
			name:          "all OK",
			statuses:      []suggest.Status{suggest.StatusOK, suggest.StatusOK},
			wantHasErrors: false,
		},
		{
			name:          "all warn",
			statuses:      []suggest.Status{suggest.StatusWarn, suggest.StatusWarn},
			wantHasErrors: false,
		},
		{
			name:          "all error",
			statuses:      []suggest.Status{suggest.StatusErr, suggest.StatusErr},
			wantHasErrors: true,
		},
		{
			name:          "mixed ok and error",
			statuses:      []suggest.Status{suggest.StatusOK, suggest.StatusErr},
			wantHasErrors: true,
		},
		{
			name:          "mixed warn and error",
			statuses:      []suggest.Status{suggest.StatusWarn, suggest.StatusErr},
			wantHasErrors: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			names := make([]string, len(tc.statuses))
			for i := range tc.statuses {
				names[i] = "check-" + string(rune('a'+i))
			}
			provider := newMockHealthProvider(names...)
			m := newHealthListModelWithSize(provider)

			var lastCmd tea.Cmd
			for i, status := range tc.statuses {
				m, lastCmd = m.Update(HealthCheckResultMsg{
					Name:   names[i],
					Status: status,
				})
			}

			require.NotNil(t, lastCmd)
			msg := executeCmd(lastCmd)
			complete, ok := msg.(HealthAllCompleteMsg)
			require.True(t, ok, "expected HealthAllCompleteMsg")
			assert.Equal(t, tc.wantHasErrors, complete.HasErrors)
		})
	}
}

func TestHealthListModel_Navigation_UpDown(t *testing.T) {
	provider := newMockHealthProvider("check-a", "check-b", "check-c")
	m := newHealthListModelWithSize(provider)

	assert.Equal(t, 0, m.cursor)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.cursor)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.cursor)
}

func TestHealthListModel_Navigation_UpAtTopStays(t *testing.T) {
	provider := newMockHealthProvider("check-a", "check-b")
	m := newHealthListModelWithSize(provider)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.cursor, "cursor should not go below 0")
}

func TestHealthListModel_Navigation_DownAtBottomStays(t *testing.T) {
	provider := newMockHealthProvider("check-a", "check-b")
	m := newHealthListModelWithSize(provider)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.cursor, "cursor should not exceed last index")
}

func TestHealthListModel_UnfocusedIgnoresKeys(t *testing.T) {
	provider := newMockHealthProvider("check-a", "check-b")
	m := newHealthListModelWithSize(provider)
	m.SetFocused(false)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, m.cursor, "unfocused model should ignore key events")
}

// ===========================================================================
// New health check names (epic #589 capabilities)
// ===========================================================================

func TestHealthListModel_NewChecksAppearInList(t *testing.T) {
	// Verify the three new checks appear in the provider's check list
	provider := newMockHealthProvider(
		"Git Repository",
		"Adapter Binary",
		"SQLite Database",
		"Wave Configuration",
		"Required Tools",
		"Required Skills",
		"Adapter Registry",
		"Retry Policies",
		"Engine Capabilities",
	)
	m := newHealthListModelWithSize(provider)

	// All 9 checks should be initialized
	assert.Equal(t, 9, len(m.checks), "expected 9 health checks")

	// New checks should be at the end
	assert.Equal(t, "Adapter Registry", m.checks[6].Name)
	assert.Equal(t, "Retry Policies", m.checks[7].Name)
	assert.Equal(t, "Engine Capabilities", m.checks[8].Name)
}

func TestHealthListModel_NewChecksComplete(t *testing.T) {
	names := []string{
		"Git Repository",
		"Adapter Binary",
		"SQLite Database",
		"Wave Configuration",
		"Required Tools",
		"Required Skills",
		"Adapter Registry",
		"Retry Policies",
		"Engine Capabilities",
	}
	provider := newMockHealthProvider(names...)
	m := newHealthListModelWithSize(provider)

	// Deliver results for all 9 checks
	var lastCmd tea.Cmd
	for _, name := range names {
		m, lastCmd = m.Update(HealthCheckResultMsg{
			Name:    name,
			Status:  suggest.StatusOK,
			Message: "ok",
		})
	}

	// Should emit HealthAllCompleteMsg after all 9 resolve
	require.NotNil(t, lastCmd, "all checks completed should emit a command")
	msg := executeCmd(lastCmd)
	complete, ok := msg.(HealthAllCompleteMsg)
	require.True(t, ok, "expected HealthAllCompleteMsg, got %T", msg)
	assert.False(t, complete.HasErrors)
}

func TestHealthListModel_RetryPoliciesWarnDoesNotBlockCompletion(t *testing.T) {
	names := []string{"Adapter Registry", "Retry Policies", "Engine Capabilities"}
	provider := newMockHealthProvider(names...)
	m := newHealthListModelWithSize(provider)

	m, _ = m.Update(HealthCheckResultMsg{Name: "Adapter Registry", Status: suggest.StatusOK, Message: "ok"})
	m, _ = m.Update(HealthCheckResultMsg{Name: "Retry Policies", Status: suggest.StatusWarn, Message: "raw max_attempts"})
	_, cmd := m.Update(HealthCheckResultMsg{Name: "Engine Capabilities", Status: suggest.StatusOK, Message: "ok"})

	require.NotNil(t, cmd)
	msg := executeCmd(cmd)
	complete, ok := msg.(HealthAllCompleteMsg)
	require.True(t, ok, "expected HealthAllCompleteMsg")
	// Warn is not an error
	assert.False(t, complete.HasErrors, "warn should not count as error")
}
