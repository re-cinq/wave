package dashboard

import "time"

// API response types for the dashboard REST API.

// RunResponse represents a pipeline run in API responses.
type RunResponse struct {
	RunID        string     `json:"run_id"`
	PipelineName string     `json:"pipeline_name"`
	Status       string     `json:"status"`
	Input        string     `json:"input,omitempty"`
	CurrentStep  string     `json:"current_step,omitempty"`
	TotalTokens  int        `json:"total_tokens"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CancelledAt  *time.Time `json:"cancelled_at,omitempty"`
	ErrorMessage string     `json:"error,omitempty"`
	Tags         []string   `json:"tags,omitempty"`
	DurationMs   int64      `json:"duration_ms"`
}

// RunListResponse is the response for GET /api/runs.
type RunListResponse struct {
	Runs  []RunResponse `json:"runs"`
	Total int           `json:"total"`
}

// EventResponse represents an event log entry in API responses.
type EventResponse struct {
	ID         int64     `json:"id"`
	RunID      string    `json:"run_id"`
	Timestamp  time.Time `json:"timestamp"`
	StepID     string    `json:"step_id,omitempty"`
	State      string    `json:"state"`
	Persona    string    `json:"persona,omitempty"`
	Message    string    `json:"message,omitempty"`
	TokensUsed int       `json:"tokens_used,omitempty"`
	DurationMs int64     `json:"duration_ms,omitempty"`
}

// EventListResponse is the response for GET /api/runs/{id}/events.
type EventListResponse struct {
	Events []EventResponse `json:"events"`
}

// StepProgressResponse represents step progress in API responses.
type StepProgressResponse struct {
	StepID                string     `json:"step_id"`
	RunID                 string     `json:"run_id"`
	Persona               string     `json:"persona,omitempty"`
	State                 string     `json:"state"`
	Progress              int        `json:"progress"`
	CurrentAction         string     `json:"current_action,omitempty"`
	Message               string     `json:"message,omitempty"`
	StartedAt             *time.Time `json:"started_at,omitempty"`
	UpdatedAt             time.Time  `json:"updated_at"`
	EstimatedCompletionMs int64      `json:"estimated_completion_ms,omitempty"`
	TokensUsed            int        `json:"tokens_used,omitempty"`
}

// PipelineProgressResponse represents pipeline-level progress.
type PipelineProgressResponse struct {
	RunID                 string    `json:"run_id"`
	TotalSteps            int       `json:"total_steps"`
	CompletedSteps        int       `json:"completed_steps"`
	CurrentStepIndex      int       `json:"current_step_index"`
	OverallProgress       int       `json:"overall_progress"`
	EstimatedCompletionMs int64     `json:"estimated_completion_ms,omitempty"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// RunDetailResponse is the response for GET /api/runs/{id}.
type RunDetailResponse struct {
	Run              RunResponse               `json:"run"`
	Steps            []StepProgressResponse    `json:"steps,omitempty"`
	PipelineProgress *PipelineProgressResponse `json:"pipeline_progress,omitempty"`
}

// ArtifactResponse represents an artifact in API responses.
type ArtifactResponse struct {
	ID        int64     `json:"id"`
	RunID     string    `json:"run_id"`
	StepID    string    `json:"step_id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Type      string    `json:"type,omitempty"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

// ArtifactListResponse is the response for GET /api/runs/{id}/artifacts.
type ArtifactListResponse struct {
	Artifacts []ArtifactResponse `json:"artifacts"`
}

// SSEEvent represents a server-sent event message.
type SSEEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}
