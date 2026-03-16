package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	return HealthCheckResultMsg{Name: name, Status: HealthCheckOK, Message: "ok"}
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
			Status:  HealthCheckOK,
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
	m, _ = m.Update(HealthCheckResultMsg{Name: "check-a", Status: HealthCheckOK, Message: "ok"})
	m, cmd := m.Update(HealthCheckResultMsg{Name: "check-b", Status: HealthCheckErr, Message: "failed"})

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

	m, cmd := m.Update(HealthCheckResultMsg{Name: "check-a", Status: HealthCheckWarn, Message: "warn"})

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
	m, cmd := m.Update(HealthCheckResultMsg{Name: "check-a", Status: HealthCheckOK, Message: "ok"})

	// No HealthAllCompleteMsg should be emitted yet
	if cmd != nil {
		msg := executeCmd(cmd)
		_, ok := msg.(HealthAllCompleteMsg)
		assert.False(t, ok, "HealthAllCompleteMsg should not be emitted when checks are still running")
	}

	// Deliver second but not third
	m, cmd = m.Update(HealthCheckResultMsg{Name: "check-b", Status: HealthCheckOK, Message: "ok"})
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
	m, _ = m.Update(HealthCheckResultMsg{Name: "check-a", Status: HealthCheckOK, Message: "ok"})
	m, _ = m.Update(HealthCheckResultMsg{Name: "check-b", Status: HealthCheckOK, Message: "ok"})

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
		statuses      []HealthCheckStatus
		wantHasErrors bool
	}{
		{
			name:          "all OK",
			statuses:      []HealthCheckStatus{HealthCheckOK, HealthCheckOK},
			wantHasErrors: false,
		},
		{
			name:          "all warn",
			statuses:      []HealthCheckStatus{HealthCheckWarn, HealthCheckWarn},
			wantHasErrors: false,
		},
		{
			name:          "all error",
			statuses:      []HealthCheckStatus{HealthCheckErr, HealthCheckErr},
			wantHasErrors: true,
		},
		{
			name:          "mixed ok and error",
			statuses:      []HealthCheckStatus{HealthCheckOK, HealthCheckErr},
			wantHasErrors: true,
		},
		{
			name:          "mixed warn and error",
			statuses:      []HealthCheckStatus{HealthCheckWarn, HealthCheckErr},
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
