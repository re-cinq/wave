package pipeline

import (
	"time"

	"github.com/recinq/wave/internal/state"
)

// ChatContext holds assembled context for a wave chat session.
type ChatContext struct {
	Run              *state.RunRecord
	Steps            []ChatStepContext
	Pipeline         *Pipeline
	PipelinePath     string
	Artifacts        []state.ArtifactRecord
	ProjectRoot      string
	ArtifactContents map[string]string  // Artifact name → content/summary
	ChatConfig       *ChatContextConfig // Pipeline-level chat configuration
}

// ChatStepContext holds context for a single step in the chat session.
type ChatStepContext struct {
	StepID        string
	Persona       string
	Adapter       string
	State         string
	Duration      time.Duration
	TokensUsed    int
	WorkspacePath string
	Artifacts     []state.ArtifactRecord
	ErrorMessage  string
}
