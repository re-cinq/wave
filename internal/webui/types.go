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
	RunID                string     `json:"run_id"`
	PipelineName         string     `json:"pipeline_name"`
	Status               string     `json:"status"`
	CurrentStep          string     `json:"current_step,omitempty"`
	TotalTokens          int        `json:"total_tokens"`
	StartedAt            time.Time  `json:"started_at"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	Duration             string     `json:"duration,omitempty"`
	Tags                 []string   `json:"tags,omitempty"`
	Progress             int        `json:"progress,omitempty"`
	ErrorMessage         string     `json:"error_message,omitempty"`
	InputPreview         string     `json:"input_preview,omitempty"`
	Input                string     `json:"input,omitempty"`
	LinkedURL            string     `json:"linked_url,omitempty"`
	FormattedStartedAt   string     `json:"formatted_started_at,omitempty"`
	FormattedCompletedAt string     `json:"formatted_completed_at,omitempty"`
	BranchName           string     `json:"branch_name,omitempty"`
	StepsCompleted       int        `json:"steps_completed,omitempty"`
	StepsTotal           int        `json:"steps_total,omitempty"`
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
	RunID       string            `json:"run_id"`
	StepID      string            `json:"step_id"`
	Persona     string            `json:"persona"`
	State       string            `json:"state"`
	Progress    int               `json:"progress"`
	Action      string            `json:"current_action,omitempty"`
	StartedAt          *time.Time        `json:"started_at,omitempty"`
	FormattedStartedAt string             `json:"formatted_started_at,omitempty"`
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

// StepEventsResponse is the JSON response for the paginated step events API.
type StepEventsResponse struct {
	Events  []EventSummary `json:"events"`
	HasMore bool           `json:"has_more"`
	Offset  int            `json:"offset"`
	Limit   int            `json:"limit"`
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

// SkillSummary is a summary of a skill for the API.
type SkillSummary struct {
	Name          string   `json:"name"`
	CommandsGlob  string   `json:"commands_glob,omitempty"`
	CommandFiles  []string `json:"command_files,omitempty"`
	InstallCmd    string   `json:"install_cmd,omitempty"`
	CheckCmd      string   `json:"check_cmd,omitempty"`
	PipelineUsage []string `json:"pipeline_usage,omitempty"`
}

// SkillListResponse is the JSON response for the skill list API.
type SkillListResponse struct {
	Skills []SkillSummary `json:"skills"`
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
	Skills       []string `json:"skills,omitempty"`
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

// ResumeRunRequest is the request body for resuming a run from a specific step.
type ResumeRunRequest struct {
	FromStep string `json:"from_step"`
	Force    bool   `json:"force"`
}

// ResumeRunResponse is the JSON response after resuming a run.
type ResumeRunResponse struct {
	RunID         string    `json:"run_id"`
	OriginalRunID string    `json:"original_run_id"`
	PipelineName  string    `json:"pipeline_name"`
	FromStep      string    `json:"from_step"`
	Status        string    `json:"status"`
	StartedAt     time.Time `json:"started_at"`
}

// CompositionPipeline holds details about a pipeline that uses composition primitives.
type CompositionPipeline struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Category    string              `json:"category,omitempty"`
	StepCount   int                 `json:"step_count"`
	Steps       []CompositionStep   `json:"steps"`
	Skills      []string            `json:"skills,omitempty"`
}

// CompositionStep describes a step in a composition pipeline with its primitive type.
type CompositionStep struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // "iterate", "branch", "gate", "loop", "aggregate", "sub_pipeline", "persona"
	SubPipeline string            `json:"sub_pipeline,omitempty"`
	Persona     string            `json:"persona,omitempty"`
	Details     map[string]string `json:"details,omitempty"`
}

// CompositionListResponse is the JSON response for the composition pipeline list API.
type CompositionListResponse struct {
	Pipelines []CompositionPipeline `json:"pipelines"`
}

// PipelineStartInfo holds pipeline metadata used by the enhanced start form.
type PipelineStartInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	StepCount   int    `json:"step_count"`
}

// IssueSummary is a summary of a GitHub issue for the API.
type IssueSummary struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	State     string   `json:"state"`
	Author    string   `json:"author"`
	Labels    []string `json:"labels"`
	Comments  int      `json:"comments"`
	CreatedAt string   `json:"created_at"`
	URL       string   `json:"url"`
}

// IssueListResponse is the JSON response for the issue list API.
type IssueListResponse struct {
	Issues      []IssueSummary `json:"issues"`
	RepoSlug    string         `json:"repo_slug,omitempty"`
	Message     string         `json:"message,omitempty"`
	FilterState string         `json:"filter_state,omitempty"`
	Page        int            `json:"page,omitempty"`
	HasMore     bool           `json:"has_more"`
}

// PRSummary is a summary of a GitHub pull request for the API.
type PRSummary struct {
	Number       int    `json:"number"`
	Title        string `json:"title"`
	State        string `json:"state"`
	Author       string `json:"author"`
	Draft        bool   `json:"draft"`
	Merged       bool   `json:"merged"`
	HeadBranch   string `json:"head_branch"`
	BaseBranch   string `json:"base_branch"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	ChangedFiles int    `json:"changed_files"`
	CreatedAt    string `json:"created_at"`
	URL          string `json:"url"`
}

// PRListResponse is the JSON response for the PR list API.
type PRListResponse struct {
	PullRequests []PRSummary `json:"pull_requests"`
	RepoSlug     string      `json:"repo_slug,omitempty"`
	Message      string      `json:"message,omitempty"`
	FilterState  string      `json:"filter_state,omitempty"`
	Page         int         `json:"page,omitempty"`
	HasMore      bool        `json:"has_more"`
}

// HealthCheckResult is the result of a single health check for the API.
type HealthCheckResult struct {
	Name    string            `json:"name"`
	Status  string            `json:"status"` // "ok", "warn", "error"
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// HealthListResponse is the JSON response for the health check API.
type HealthListResponse struct {
	Checks []HealthCheckResult `json:"checks"`
}
