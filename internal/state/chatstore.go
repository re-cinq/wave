package state

// ChatStore is the domain-scoped persistence surface for bidirectional chat
// sessions. Consumers that only read/write chat state should depend on this
// interface rather than the aggregate StateStore.
type ChatStore interface {
	SaveChatSession(session *ChatSession) error
	GetChatSession(sessionID string) (*ChatSession, error)
	ListChatSessions(runID string) ([]ChatSession, error)
}
