package webui

import "time"

// RunListResponse is the JSON response for the run list API.
type RunListResponse struct {
	Runs       []RunSummary `json:"runs"`
	NextCursor string       `json:"next_cursor,omitempty"`
	HasMore    bool         `json:"has_more"`
}

// RunSummary is a summary of a pipeline run for list views.
type RunSummary struct {
	RunID        string     `json:"run_id"`
	PipelineName string     `json:"pipeline_name"`
	Status       string     `json:"status"`
	CurrentStep  string     `json:"current_step,omitempty"`
	TotalTokens  int        `json:"total_tokens"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Duration     string     `json:"duration,omitempty"`
	Tags         []string   `json:"tags,omitempty"`
	Progress     int        `json:"progress,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// RunDetailResponse is the JSON response for the run detail API.
type RunDetailResponse struct {
	Run       RunSummary        `json:"run"`
	Steps     []StepDetail      `json:"steps"`
	Events    []EventSummary    `json:"events"`
	Artifacts []ArtifactSummary `json:"artifacts,omitempty"`
	DAG       *DAGData          `json:"dag,omitempty"`
}

// StepDetail holds detail information about a pipeline step.
type StepDetail struct {
	StepID      string            `json:"step_id"`
	Persona     string            `json:"persona"`
	State       string            `json:"state"`
	Progress    int               `json:"progress"`
	Action      string            `json:"current_action,omitempty"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Duration    string            `json:"duration,omitempty"`
	TokensUsed  int               `json:"tokens_used"`
	Error       string            `json:"error,omitempty"`
	Artifacts   []ArtifactSummary `json:"artifacts,omitempty"`
}

// EventSummary holds summary information about a pipeline event.
type EventSummary struct {
	ID         int64     `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	StepID     string    `json:"step_id,omitempty"`
	State      string    `json:"state"`
	Persona    string    `json:"persona,omitempty"`
	Message    string    `json:"message"`
	TokensUsed int       `json:"tokens_used"`
	DurationMs int64     `json:"duration_ms"`
}

// ArtifactSummary holds summary information about a pipeline artifact.
type ArtifactSummary struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Type      string `json:"type"`
	SizeBytes int64  `json:"size_bytes"`
	Preview   string `json:"preview,omitempty"`
}

// ArtifactContentResponse is the JSON response for artifact content.
type ArtifactContentResponse struct {
	Content  string           `json:"content"`
	Metadata ArtifactMetadata `json:"metadata"`
}

// ArtifactMetadata holds metadata about an artifact file.
type ArtifactMetadata struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	SizeBytes int64  `json:"size_bytes"`
	Truncated bool   `json:"truncated"`
	MimeType  string `json:"mime_type"`
}

// DAGData holds the data for rendering a pipeline DAG.
type DAGData struct {
	Nodes []DAGNode `json:"nodes"`
	Edges []DAGEdge `json:"edges"`
}

// DAGNode represents a node in the pipeline DAG.
type DAGNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Persona  string `json:"persona"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
}

// DAGEdge represents an edge in the pipeline DAG.
type DAGEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// PaginationCursor is the cursor used for paginating run lists.
type PaginationCursor struct {
	Timestamp int64  `json:"t"`
	RunID     string `json:"id"`
}

// PersonaSummary is a summary of a persona for the API.
type PersonaSummary struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Adapter      string   `json:"adapter"`
	Model        string   `json:"model"`
	Temperature  float64  `json:"temperature"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
	DeniedTools  []string `json:"denied_tools,omitempty"`
}

// PersonaListResponse is the JSON response for the persona list API.
type PersonaListResponse struct {
	Personas []PersonaSummary `json:"personas"`
}

// StartPipelineRequest is the request body for starting a pipeline.
type StartPipelineRequest struct {
	Input string `json:"input"`
}

// StartPipelineResponse is the JSON response after starting a pipeline.
type StartPipelineResponse struct {
	RunID        string    `json:"run_id"`
	PipelineName string    `json:"pipeline_name"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
}

// CancelRunRequest is the request body for cancelling a run.
type CancelRunRequest struct {
	Force bool `json:"force"`
}

// CancelRunResponse is the JSON response after cancelling a run.
type CancelRunResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// RetryRunResponse is the JSON response after retrying a run.
type RetryRunResponse struct {
	RunID         string    `json:"run_id"`
	OriginalRunID string    `json:"original_run_id"`
	PipelineName  string    `json:"pipeline_name"`
	Status        string    `json:"status"`
	StartedAt     time.Time `json:"started_at"`
}
