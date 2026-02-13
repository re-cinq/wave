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

// =============================================================================
// Pipeline Detail Types (spec 091)
// =============================================================================

// PipelineDetailResponse is the JSON response for the pipeline detail API.
type PipelineDetailResponse struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	StepCount   int                  `json:"step_count"`
	Input       PipelineInputDetail  `json:"input"`
	Steps       []PipelineStepDetail `json:"steps"`
	DAG         *DAGData             `json:"dag,omitempty"`
	LastRun     *RunSummary          `json:"last_run,omitempty"`
}

// PipelineInputDetail holds pipeline input configuration.
type PipelineInputDetail struct {
	Source  string             `json:"source"`
	Schema  *InputSchemaDetail `json:"schema,omitempty"`
	Example string             `json:"example,omitempty"`
}

// InputSchemaDetail holds input schema details.
type InputSchemaDetail struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// PipelineStepDetail holds full step configuration for the pipeline detail view.
type PipelineStepDetail struct {
	ID           string              `json:"id"`
	Persona      string              `json:"persona"`
	Dependencies []string            `json:"dependencies,omitempty"`
	Workspace    WorkspaceDetail     `json:"workspace"`
	Contract     *ContractDetail     `json:"contract,omitempty"`
	Artifacts    []ArtifactDefDetail `json:"artifacts,omitempty"`
	Memory       MemoryDetail        `json:"memory"`
}

// WorkspaceDetail holds workspace configuration for display.
type WorkspaceDetail struct {
	Type   string        `json:"type,omitempty"`
	Root   string        `json:"root,omitempty"`
	Mounts []MountDetail `json:"mounts,omitempty"`
}

// MountDetail holds mount configuration.
type MountDetail struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Mode   string `json:"mode,omitempty"`
}

// ContractDetail holds contract configuration for display.
type ContractDetail struct {
	Type       string `json:"type"`
	Schema     string `json:"schema,omitempty"`
	SchemaPath string `json:"schema_path,omitempty"`
	MustPass   bool   `json:"must_pass"`
	MaxRetries int    `json:"max_retries,omitempty"`
}

// ArtifactDefDetail holds output artifact definitions.
type ArtifactDefDetail struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// MemoryDetail holds memory/injection configuration.
type MemoryDetail struct {
	Strategy string             `json:"strategy"`
	Injected []InjectedArtifact `json:"injected,omitempty"`
}

// InjectedArtifact holds artifact injection references.
type InjectedArtifact struct {
	FromStep string `json:"from_step"`
	Artifact string `json:"artifact"`
	As       string `json:"as"`
}

// =============================================================================
// Persona Detail Types (spec 091)
// =============================================================================

// PersonaDetailResponse is the JSON response for the persona detail API.
type PersonaDetailResponse struct {
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	Adapter          string         `json:"adapter"`
	Model            string         `json:"model,omitempty"`
	Temperature      float64        `json:"temperature"`
	SystemPrompt     string         `json:"system_prompt,omitempty"`
	SystemPromptFile string         `json:"system_prompt_file"`
	AllowedTools     []string       `json:"allowed_tools,omitempty"`
	DeniedTools      []string       `json:"denied_tools,omitempty"`
	Hooks            *HooksDetail   `json:"hooks,omitempty"`
	Sandbox          *SandboxDetail `json:"sandbox,omitempty"`
	UsedInPipelines  []string       `json:"used_in_pipelines,omitempty"`
}

// HooksDetail holds hook configuration for display.
type HooksDetail struct {
	PreToolUse  []HookRuleDetail `json:"pre_tool_use,omitempty"`
	PostToolUse []HookRuleDetail `json:"post_tool_use,omitempty"`
}

// HookRuleDetail holds a single hook rule.
type HookRuleDetail struct {
	Matcher string `json:"matcher"`
	Command string `json:"command"`
}

// SandboxDetail holds sandbox configuration for display.
type SandboxDetail struct {
	AllowedDomains []string `json:"allowed_domains,omitempty"`
}

// =============================================================================
// Statistics Types (spec 091)
// =============================================================================

// RunStatistics holds aggregate run counts.
type RunStatistics struct {
	Total       int     `json:"total"`
	Succeeded   int     `json:"succeeded"`
	Failed      int     `json:"failed"`
	Cancelled   int     `json:"cancelled"`
	Pending     int     `json:"pending"`
	Running     int     `json:"running"`
	SuccessRate float64 `json:"success_rate"`
}

// RunTrendPoint holds a single data point in the run trend.
type RunTrendPoint struct {
	Date        string  `json:"date"`
	Total       int     `json:"total"`
	Succeeded   int     `json:"succeeded"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

// PipelineStatistics holds per-pipeline aggregate stats.
type PipelineStatistics struct {
	PipelineName  string  `json:"pipeline_name"`
	RunCount      int     `json:"run_count"`
	SuccessRate   float64 `json:"success_rate"`
	AvgDurationMs int64   `json:"avg_duration_ms"`
	AvgTokens     int     `json:"avg_tokens"`
}

// StatisticsResponse is the JSON response for the statistics API.
type StatisticsResponse struct {
	Aggregate RunStatistics        `json:"aggregate"`
	Trends    []RunTrendPoint      `json:"trends"`
	Pipelines []PipelineStatistics `json:"pipelines"`
	TimeRange string               `json:"time_range"`
}

// =============================================================================
// Enhanced Run Detail Types (spec 091)
// =============================================================================

// EnhancedStepDetail extends StepDetail with introspection data.
type EnhancedStepDetail struct {
	StepDetail
	ContractResult  *ContractResultDetail `json:"contract_result,omitempty"`
	RecoveryHints   []RecoveryHintDetail  `json:"recovery_hints,omitempty"`
	Performance     *StepPerfDetail       `json:"performance,omitempty"`
	WorkspacePath   string                `json:"workspace_path,omitempty"`
	WorkspaceExists bool                  `json:"workspace_exists"`
}

// ContractResultDetail holds contract validation outcome for a step.
type ContractResultDetail struct {
	Type         string `json:"type"`
	Passed       bool   `json:"passed"`
	ErrorMessage string `json:"error_message,omitempty"`
	Schema       string `json:"schema,omitempty"`
}

// RecoveryHintDetail holds a recovery suggestion for display.
type RecoveryHintDetail struct {
	Label   string `json:"label"`
	Command string `json:"command"`
	Type    string `json:"type"`
}

// StepPerfDetail holds performance metrics for a specific step execution.
type StepPerfDetail struct {
	DurationMs         int64 `json:"duration_ms"`
	TokensUsed         int   `json:"tokens_used"`
	FilesModified      int   `json:"files_modified"`
	ArtifactsGenerated int   `json:"artifacts_generated"`
}

// =============================================================================
// Workspace Browsing Types (spec 091)
// =============================================================================

// WorkspaceTreeResponse is the JSON response for workspace directory listings.
type WorkspaceTreeResponse struct {
	Path    string           `json:"path"`
	Entries []WorkspaceEntry `json:"entries"`
	Error   string           `json:"error,omitempty"`
}

// WorkspaceEntry represents a file or directory in the workspace tree.
type WorkspaceEntry struct {
	Name      string `json:"name"`
	IsDir     bool   `json:"is_dir"`
	Size      int64  `json:"size"`
	Extension string `json:"extension,omitempty"`
}

// WorkspaceFileResponse is the JSON response for workspace file content.
type WorkspaceFileResponse struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	MimeType  string `json:"mime_type"`
	Size      int64  `json:"size"`
	Truncated bool   `json:"truncated"`
	Error     string `json:"error,omitempty"`
}
