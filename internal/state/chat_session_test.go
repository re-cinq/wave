package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveChatSession(t *testing.T) {
	testCases := []struct {
		name    string
		session ChatSession
	}{
		{
			name: "save session with all fields",
			session: ChatSession{
				SessionID:     "sess-001",
				RunID:         "", // set dynamically
				StepFilter:    "analyze",
				WorkspacePath: "/tmp/ws/test",
				Model:         "claude-opus-4-6",
				CreatedAt:     time.Now().Truncate(time.Second),
			},
		},
		{
			name: "save session with empty optional fields",
			session: ChatSession{
				SessionID:     "sess-002",
				RunID:         "",
				StepFilter:    "",
				WorkspacePath: "/tmp/ws/minimal",
				Model:         "",
				CreatedAt:     time.Now().Truncate(time.Second),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store, cleanup := setupTestStore(t)
			defer cleanup()

			runID, err := store.CreateRun("test-pipeline", "test input")
			require.NoError(t, err)

			tc.session.RunID = runID
			err = store.SaveChatSession(&tc.session)
			require.NoError(t, err)

			got, err := store.GetChatSession(tc.session.SessionID)
			require.NoError(t, err)
			assert.Equal(t, tc.session.SessionID, got.SessionID)
			assert.Equal(t, runID, got.RunID)
			assert.Equal(t, tc.session.StepFilter, got.StepFilter)
			assert.Equal(t, tc.session.WorkspacePath, got.WorkspacePath)
			assert.Equal(t, tc.session.Model, got.Model)
			assert.Equal(t, tc.session.CreatedAt.Unix(), got.CreatedAt.Unix())
			assert.Nil(t, got.LastResumedAt)
		})
	}
}

func TestListChatSessions(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)

	// Insert two sessions with different creation times
	err = store.SaveChatSession(&ChatSession{
		SessionID:     "sess-older",
		RunID:         runID,
		WorkspacePath: "/tmp/ws/1",
		CreatedAt:     now.Add(-time.Minute),
	})
	require.NoError(t, err)

	err = store.SaveChatSession(&ChatSession{
		SessionID:     "sess-newer",
		RunID:         runID,
		WorkspacePath: "/tmp/ws/2",
		CreatedAt:     now,
	})
	require.NoError(t, err)

	sessions, err := store.ListChatSessions(runID)
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	// Ordered by created_at DESC — newer first
	assert.Equal(t, "sess-newer", sessions[0].SessionID)
	assert.Equal(t, "sess-older", sessions[1].SessionID)
}

func TestGetChatSession_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.GetChatSession("nonexistent-session")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-session")
}

func TestListChatSessions_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	sessions, err := store.ListChatSessions(runID)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestSaveChatSession_Upsert(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)

	// Save initial session without last_resumed_at
	err = store.SaveChatSession(&ChatSession{
		SessionID:     "sess-upsert",
		RunID:         runID,
		StepFilter:    "build",
		WorkspacePath: "/tmp/ws/upsert",
		Model:         "claude-opus-4-6",
		CreatedAt:     now,
	})
	require.NoError(t, err)

	// Save again with last_resumed_at set
	resumedAt := now.Add(5 * time.Minute)
	err = store.SaveChatSession(&ChatSession{
		SessionID:     "sess-upsert",
		RunID:         runID,
		StepFilter:    "build",
		WorkspacePath: "/tmp/ws/upsert",
		Model:         "claude-opus-4-6",
		CreatedAt:     now,
		LastResumedAt: &resumedAt,
	})
	require.NoError(t, err)

	got, err := store.GetChatSession("sess-upsert")
	require.NoError(t, err)
	assert.Equal(t, "sess-upsert", got.SessionID)
	require.NotNil(t, got.LastResumedAt)
	assert.Equal(t, resumedAt.Unix(), got.LastResumedAt.Unix())

	// Verify only one session exists for this run
	sessions, err := store.ListChatSessions(runID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
}
