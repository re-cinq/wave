package onboarding

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
