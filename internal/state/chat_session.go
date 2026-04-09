package state

import "time"

// ChatSession tracks a bidirectional chat session linked to a pipeline run.
type ChatSession struct {
	SessionID     string // Claude Code session ID
	RunID         string // links to pipeline run
	StepFilter    string // optional step scope
	WorkspacePath string // workspace used for this session
	Model         string // model used
	CreatedAt     time.Time
	LastResumedAt *time.Time
}
