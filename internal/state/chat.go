package state

import (
	"fmt"
	"time"
)

// SaveChatSession persists a chat session record. If a session with the same ID
// already exists, it updates last_resumed_at.
func (s *stateStore) SaveChatSession(session *ChatSession) error {
	var lastResumedAt *int64
	if session.LastResumedAt != nil {
		t := session.LastResumedAt.Unix()
		lastResumedAt = &t
	}
	_, err := s.db.Exec(
		`INSERT INTO chat_session (session_id, run_id, step_filter, workspace_path, model, created_at, last_resumed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET last_resumed_at = excluded.last_resumed_at`,
		session.SessionID, session.RunID, session.StepFilter, session.WorkspacePath, session.Model, session.CreatedAt.Unix(), lastResumedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save chat session: %w", err)
	}
	return nil
}

// GetChatSession retrieves a chat session by its session ID.
func (s *stateStore) GetChatSession(sessionID string) (*ChatSession, error) {
	row := s.db.QueryRow(
		`SELECT session_id, run_id, step_filter, workspace_path, model, created_at, last_resumed_at FROM chat_session WHERE session_id = ?`,
		sessionID,
	)

	var cs ChatSession
	var createdAt int64
	var lastResumedAt *int64
	err := row.Scan(&cs.SessionID, &cs.RunID, &cs.StepFilter, &cs.WorkspacePath, &cs.Model, &createdAt, &lastResumedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat session %s: %w", sessionID, err)
	}
	cs.CreatedAt = time.Unix(createdAt, 0)
	if lastResumedAt != nil {
		t := time.Unix(*lastResumedAt, 0)
		cs.LastResumedAt = &t
	}
	return &cs, nil
}

// ListChatSessions returns all chat sessions for a pipeline run, ordered by creation time descending.
func (s *stateStore) ListChatSessions(runID string) ([]ChatSession, error) {
	rows, err := s.db.Query(
		`SELECT session_id, run_id, step_filter, workspace_path, model, created_at, last_resumed_at FROM chat_session WHERE run_id = ? ORDER BY created_at DESC`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list chat sessions for run %s: %w", runID, err)
	}
	defer rows.Close()

	var sessions []ChatSession
	for rows.Next() {
		var cs ChatSession
		var createdAt int64
		var lastResumedAt *int64
		err := rows.Scan(&cs.SessionID, &cs.RunID, &cs.StepFilter, &cs.WorkspacePath, &cs.Model, &createdAt, &lastResumedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat session: %w", err)
		}
		cs.CreatedAt = time.Unix(createdAt, 0)
		if lastResumedAt != nil {
			t := time.Unix(*lastResumedAt, 0)
			cs.LastResumedAt = &t
		}
		sessions = append(sessions, cs)
	}
	return sessions, nil
}
