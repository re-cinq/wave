package onboarding

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaselineService_IsOnboarded(t *testing.T) {
	dir := t.TempDir()
	svc := NewBaselineService(io.Discard)

	assert.False(t, svc.IsOnboarded(dir))

	require.NoError(t, MarkDoneAt(dir))
	assert.True(t, svc.IsOnboarded(dir))
}

func TestBaselineService_MarkDone_Idempotent(t *testing.T) {
	dir := t.TempDir()
	svc := NewBaselineService(io.Discard)

	require.NoError(t, svc.MarkDone(dir))
	require.NoError(t, svc.MarkDone(dir), "MarkDone must be safe to call twice")

	sentinel := filepath.Join(dir, SentinelFile)
	_, err := os.Stat(sentinel)
	require.NoError(t, err)
}

func TestClearSentinel(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, MarkDoneAt(dir))
	require.NoError(t, ClearSentinel(dir))
	assert.False(t, IsOnboardedAt(dir))

	// Clearing an absent sentinel must not error.
	require.NoError(t, ClearSentinel(dir))
}

func TestBaselineService_StatusUnknownSession(t *testing.T) {
	svc := NewBaselineService(io.Discard)
	_, err := svc.Status("does-not-exist")
	assert.Error(t, err)
}

func TestBaselineService_ResumeUnknownSession(t *testing.T) {
	svc := NewBaselineService(io.Discard)
	_, err := svc.Resume(context.Background(), "does-not-exist")
	assert.Error(t, err)
}

func TestNoopUI_PromptDefaults(t *testing.T) {
	u := NoopUI{}

	got, err := u.PromptString(Question{ID: "name", Default: "wave"})
	require.NoError(t, err)
	assert.Equal(t, "wave", got)

	choice, err := u.PromptChoice(Question{
		ID:      "adapter",
		Default: "claude",
		Choices: []string{"claude", "opencode"},
	})
	require.NoError(t, err)
	assert.Equal(t, "claude", choice)

	// When Default isn't in Choices, NoopUI falls back to the first choice.
	choice2, err := u.PromptChoice(Question{
		ID:      "tier",
		Default: "platinum",
		Choices: []string{"cheapest", "balanced", "strongest"},
	})
	require.NoError(t, err)
	assert.Equal(t, "cheapest", choice2)

	// Notify must accept any event without erroring.
	require.NoError(t, u.Notify(Event{Kind: "info", Message: "hi"}))
}

// captureUI is a UI that records every Notify event for assertions.
type captureUI struct {
	NoopUI
	events []Event
}

func (c *captureUI) Notify(e Event) error {
	c.events = append(c.events, e)
	return nil
}

func TestBaselineService_StartSession_NotifiesUI(t *testing.T) {
	t.Skip("StartSession invokes Greenfield which needs a git repo + asset embed; covered by integration tests in cmd/wave")

	dir := t.TempDir()
	svc := NewBaselineService(io.Discard)
	ui := &captureUI{}

	sess, err := svc.StartSession(context.Background(), dir, StartOptions{
		Adapter:    "claude",
		Workspace:  ".agents/workspaces",
		OutputPath: filepath.Join(dir, "wave.yaml"),
		UI:         ui,
	})
	require.NoError(t, err)
	require.NotNil(t, sess)
	assert.Equal(t, SessionDone, sess.Status)
	assert.GreaterOrEqual(t, len(ui.events), 2)
}
