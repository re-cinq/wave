package webui

import (
	"time"

	"github.com/recinq/wave/internal/pipeline"
)

// RunListResponse is the JSON response for the run list API.
type RunListResponse struct {
	Runs       []RunSummary `json:"runs"`
	NextCursor string       `json:"next_cursor,omitempty"`
	HasMore    bool         `json:"has_more"`
}

// RunSummary is a summary of a pipeline run for list views.
type RunSummary struct {
	RunID                string       `json:"run_id"`
	PipelineName         string       `json:"pipeline_name"`
	Status               string       `json:"status"`
	CurrentStep          string       `json:"current_step,omitempty"`
	TotalTokens          int          `json:"total_tokens"`
	StartedAt            time.Time    `json:"started_at"`
	CompletedAt          *time.Time   `json:"completed_at,omitempty"`
	Duration             string       `json:"duration,omitempty"`
	Tags                 []string     `json:"tags,omitempty"`
	Progress             int          `json:"progress,omitempty"`
	ErrorMessage         string       `json:"error_message,omitempty"`
	InputPreview         string       `json:"input_preview,omitempty"`
	Input                string       `json:"input,omitempty"`
	LinkedURL            string       `json:"linked_url,omitempty"`
	FormattedStartedAt   string       `json:"formatted_started_at,omitempty"`
	FormattedCompletedAt string       `json:"formatted_completed_at,omitempty"`
	BranchName           string       `json:"branch_name,omitempty"`
	StepsCompleted       int          `json:"steps_completed,omitempty"`
	StepsTotal           int          `json:"steps_total,omitempty"`
	ParentRunID          string       `json:"parent_run_id,omitempty"`
	ParentStepID         string       `json:"parent_step_id,omitempty"`
	ChildRuns            []RunSummary `json:"child_runs,omitempty"`
	Adapters             []string     `json:"adapters,omitempty"`
	Models               []string     `json:"models,omitempty"`
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
	RunID              string                `json:"run_id"`
	StepID             string                `json:"step_id"`
	Persona            string                `json:"persona"`
	State              string                `json:"state"`
	Progress           int                   `json:"progress"`
	Action             string                `json:"current_action,omitempty"`
	StartedAt          *time.Time            `json:"started_at,omitempty"`
	FormattedStartedAt string                `json:"formatted_started_at,omitempty"`
	CompletedAt        *time.Time            `json:"completed_at,omitempty"`
	Duration           string                `json:"duration,omitempty"`
	TokensUsed         int                   `json:"tokens_used"`
	Error              string                `json:"error,omitempty"`
	FailureClass       string                `json:"failure_class,omitempty"`
	Artifacts          []ArtifactSummary     `json:"artifacts,omitempty"`
	StepType           string                `json:"step_type,omitempty"`            // "conditional", "command", "gate", "pipeline", or ""
	Script             string                `json:"script,omitempty"`               // Shell script for command steps
	SubPipeline        string                `json:"sub_pipeline,omitempty"`         // Referenced pipeline for pipeline steps
	GatePrompt         string                `json:"gate_prompt,omitempty"`          // Gate prompt/message
	GateChoices        string                `json:"gate_choices,omitempty"`         // Comma-separated gate choice labels
	GateChoicesData    []pipeline.GateChoice `json:"gate_choices_data,omitempty"`    // Structured gate choice data for interactive UI
	GateFreeform       bool                  `json:"gate_freeform,omitempty"`        // Whether freeform text input is allowed
	EdgeInfo           string                `json:"edge_info,omitempty"`            // Edge conditions for conditional steps
	Contract           string                `json:"contract,omitempty"`             // Contract path/type
	ContractSchemaName string                `json:"contract_schema_name,omitempty"` // Human-readable contract name
	Model              string                `json:"model,omitempty"`                // Resolved model ID (e.g., "claude-haiku-4-5")
	ConfiguredModel    string                `json:"configured_model,omitempty"`     // Tier from pipeline config (e.g., "cheapest")
	Adapter            string                `json:"adapter,omitempty"`              // Adapter used for this step
	Dependencies       []string              `json:"dependencies,omitempty"`         // Step dependencies (step IDs)
	InputArtifacts     []InputArtifactRef    `json:"input_artifacts,omitempty"`      // Injected artifacts from upstream steps (source-step/name)
	VisitCount         int                   `json:"visit_count,omitempty"`          // Current visit count for graph loop steps
	MaxVisits          int                   `json:"max_visits,omitempty"`           // Max visit limit for graph loop steps
	GanttLeft          float64               `json:"gantt_left,omitempty"`           // Gantt bar left offset (percentage)
	GanttWidth         float64               `json:"gantt_width,omitempty"`          // Gantt bar width (percentage)
	Output             string                `json:"output,omitempty"`               // Output artifact names for this step
	ReviewVerdict      string                `json:"review_verdict,omitempty"`       // "pass", "fail", or "warn"
	ReviewIssues       []string              `json:"review_issues,omitempty"`        // Issue descriptions from LLM review
	ReviewerPersona    string                `json:"reviewer_persona,omitempty"`     // Persona used for review step
	ReviewTokens       int                   `json:"review_tokens,omitempty"`        // Tokens used in review step
	ReviewIssueCount   int                   `json:"review_issue_count,omitempty"`   // Number of review issues found
}

// InputArtifactRef identifies an artifact injected from an upstream step.
// Used to render each IN artifact as an individually clickable chip on the
// run detail page (same URL shape as OUT: source step + artifact name).
type InputArtifactRef struct {
	Step string `json:"step"` // Source step ID (where the artifact was produced)
	Name string `json:"name"` // Artifact name
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
	Model      string    `json:"model,omitempty"`
	Adapter    string    `json:"adapter,omitempty"`
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
	Name           string   `json:"name"`
	CommandsGlob   string   `json:"commands_glob,omitempty"`
	CommandFiles   []string `json:"command_files,omitempty"`
	InstallCmd     string   `json:"install_cmd,omitempty"`
	InstallPkg     string   `json:"install_pkg,omitempty"`    // parsed: package name
	InstallSource  string   `json:"install_source,omitempty"` // parsed: source repo (cleaned)
	InstallMethod  string   `json:"install_method,omitempty"` // parsed: uv/pip/npm/brew etc.
	CheckCmd       string   `json:"check_cmd,omitempty"`
	PipelineUsage  []string `json:"pipeline_usage,omitempty"`
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
	Prompt       string   `json:"prompt,omitempty"`
}

// PersonaListResponse is the JSON response for the persona list API.
type PersonaListResponse struct {
	Personas []PersonaSummary `json:"personas"`
}

// StartPipelineRequest is the request body for starting a pipeline.
type StartPipelineRequest struct {
	Input             string `json:"input"`
	Model             string `json:"model,omitempty"`
	Adapter           string `json:"adapter,omitempty"`
	DryRun            bool   `json:"dry_run,omitempty"`
	FromStep          string `json:"from_step,omitempty"`
	Force             bool   `json:"force,omitempty"`
	Detach            bool   `json:"detach,omitempty"`
	Timeout           int    `json:"timeout,omitempty"`
	Steps             string `json:"steps,omitempty"`
	Exclude           string `json:"exclude,omitempty"`
	OnFailure         string `json:"on_failure,omitempty"`
	Continuous        bool   `json:"continuous,omitempty"`
	Source            string `json:"source,omitempty"`
	MaxIterations     int    `json:"max_iterations,omitempty"`
	Delay             string `json:"delay,omitempty"`
	Mock              bool   `json:"mock,omitempty"`
	PreserveWorkspace bool   `json:"preserve_workspace,omitempty"`
	AutoApprove       bool   `json:"auto_approve,omitempty"`
	NoRetro           bool   `json:"no_retro,omitempty"`
	ForceModel        bool   `json:"force_model,omitempty"`
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
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Category    string            `json:"category,omitempty"`
	StepCount   int               `json:"step_count"`
	Steps       []CompositionStep `json:"steps"`
	Skills      []string          `json:"skills,omitempty"`
	RunCount    int               `json:"run_count,omitempty"`
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

// LabelBadge is a label with an optional display color for the web UI.
type LabelBadge struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// IssueSummary is a summary of a GitHub issue for the API.
type IssueSummary struct {
	Number    int          `json:"number"`
	Title     string       `json:"title"`
	State     string       `json:"state"`
	Author    string       `json:"author"`
	Labels    []LabelBadge `json:"labels"`
	Comments  int          `json:"comments"`
	CreatedAt string       `json:"created_at"`
	URL       string       `json:"url"`
	// Wave stats
	RunCount    int    `json:"run_count,omitempty"`
	LastStatus  string `json:"last_status,omitempty"`
	TotalTokens int64  `json:"total_tokens,omitempty"`
}

// IssueDetail holds full issue information for the detail page.
type IssueDetail struct {
	Number    int          `json:"number"`
	Title     string       `json:"title"`
	State     string       `json:"state"`
	Body      string       `json:"body"`
	Author    string       `json:"author"`
	Labels    []LabelBadge `json:"labels"`
	Assignees []string     `json:"assignees,omitempty"`
	Comments  int          `json:"comments"`
	CreatedAt string       `json:"created_at"`
	UpdatedAt string       `json:"updated_at"`
	URL       string       `json:"url"`
}

// IssueListResponse is the JSON response for the issue list API.
type IssueListResponse struct {
	Issues      []IssueSummary `json:"issues"`
	RepoSlug    string         `json:"repo_slug,omitempty"`
	Message     string         `json:"message,omitempty"`
	FilterState string         `json:"filter_state,omitempty"`
	Page        int            `json:"page,omitempty"`
	HasMore     bool           `json:"has_more"`
	TotalOpen   int            `json:"total_open,omitempty"`
	TotalClosed int            `json:"total_closed,omitempty"`
}

// PRSummary is a summary of a GitHub pull request for the API.
type PRSummary struct {
	Number       int          `json:"number"`
	Title        string       `json:"title"`
	State        string       `json:"state"`
	Author       string       `json:"author"`
	Labels       []LabelBadge `json:"labels,omitempty"`
	Draft        bool         `json:"draft"`
	Merged       bool         `json:"merged"`
	HeadBranch   string       `json:"head_branch"`
	BaseBranch   string       `json:"base_branch"`
	Additions    int          `json:"additions"`
	Deletions    int          `json:"deletions"`
	ChangedFiles int          `json:"changed_files"`
	CreatedAt    string       `json:"created_at"`
	URL          string       `json:"url"`
	Comments     int          `json:"comments"`
	// CI check status: "success", "failure", "pending", or ""
	CheckStatus string `json:"check_status,omitempty"`
	// Wave stats
	RunCount    int    `json:"run_count,omitempty"`
	LastStatus  string `json:"last_status,omitempty"`
	TotalTokens int64  `json:"total_tokens,omitempty"`
}

// CommentSummary is a summary of a comment on an issue or PR.
type CommentSummary struct {
	Author    string `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	TimeISO   string `json:"time_iso"`
	HTMLURL   string `json:"url"`
}

// CommitSummary is a summary of a commit on a PR for the detail page.
type CommitSummary struct {
	SHA      string `json:"sha"`
	ShortSHA string `json:"short_sha"`
	Message  string `json:"message"`
	Author   string `json:"author"`
	Date     string `json:"date"`
	TimeISO  string `json:"time_iso"`
	HTMLURL  string `json:"url"`
}

// PRCheck represents a CI/CD status check on a PR.
type PRCheck struct {
	Name       string `json:"name"`
	Status     string `json:"status"`     // "queued", "in_progress", "completed"
	Conclusion string `json:"conclusion"` // "success", "failure", "neutral", "cancelled", "skipped", "timed_out", "action_required"
	URL        string `json:"url"`
}

// PRDetail holds full PR information for the detail page.
type PRDetail struct {
	Number       int          `json:"number"`
	Title        string       `json:"title"`
	State        string       `json:"state"`
	Body         string       `json:"body"`
	Author       string       `json:"author"`
	Labels       []LabelBadge `json:"labels,omitempty"`
	Draft        bool         `json:"draft"`
	Merged       bool         `json:"merged"`
	HeadBranch   string       `json:"head_branch"`
	BaseBranch   string       `json:"base_branch"`
	Additions    int          `json:"additions"`
	Deletions    int          `json:"deletions"`
	ChangedFiles int          `json:"changed_files"`
	Commits      int          `json:"commits"`
	Comments     int          `json:"comments"`
	CreatedAt    string       `json:"created_at"`
	UpdatedAt    string       `json:"updated_at"`
	URL          string       `json:"url"`
	Checks       []PRCheck    `json:"checks,omitempty"`
}

// PRListResponse is the JSON response for the PR list API.
type PRListResponse struct {
	PullRequests []PRSummary `json:"pull_requests"`
	RepoSlug     string      `json:"repo_slug,omitempty"`
	Message      string      `json:"message,omitempty"`
	FilterState  string      `json:"filter_state,omitempty"`
	Page         int         `json:"page,omitempty"`
	HasMore      bool        `json:"has_more"`
	TotalOpen    int         `json:"total_open,omitempty"`
	TotalClosed  int         `json:"total_closed,omitempty"`
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

// DiffSummary represents the aggregate changed-file list for a pipeline run.
type DiffSummary struct {
	Files          []FileSummary `json:"files"`
	TotalFiles     int           `json:"total_files"`
	TotalAdditions int           `json:"total_additions"`
	TotalDeletions int           `json:"total_deletions"`
	BaseBranch     string        `json:"base_branch"`
	HeadBranch     string        `json:"head_branch"`
	Available      bool          `json:"available"`
	Message        string        `json:"message,omitempty"`
}

// FileSummary represents a single changed file in the diff summary.
type FileSummary struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Binary    bool   `json:"binary"`
}

// FileDiff represents the diff content for a single file.
type FileDiff struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
	Size      int    `json:"size"`
	Binary    bool   `json:"binary"`
	OldPath   string `json:"old_path,omitempty"`
}

// PersonaDetailData holds all data for the persona detail page.
type PersonaDetailData struct {
	ActivePage     string
	Persona        PersonaSummary
	TokenScopes    []string
	AllowedDomains []string
	UsedBy         []PersonaUsageRef
}

// PersonaUsageRef links a persona to a pipeline step that uses it.
type PersonaUsageRef struct {
	Pipeline string
	StepID   string
}

// ContractDetailPage holds all data for the contract detail page.
type ContractDetailPage struct {
	ActivePage string
	Contract   ContractDetailResponse
	UsedBy     []ContractUsageRef
}

// ContractUsageRef links a contract to a pipeline step that uses it.
type ContractUsageRef struct {
	Pipeline     string
	StepID       string
	ContractType string
}

// StepArtifactGroup groups artifacts by the step that produced them.
type StepArtifactGroup struct {
	StepID    string
	Artifacts []ArtifactSummary
}

// GateApproveRequest is the request body for approving a gate.
type GateApproveRequest struct {
	Choice string `json:"choice"`         // Choice key (required)
	Text   string `json:"text,omitempty"` // Freeform text (optional)
}

// GateApproveResponse is the JSON response after approving a gate.
type GateApproveResponse struct {
	RunID  string `json:"run_id"`
	StepID string `json:"step_id"`
	Choice string `json:"choice"`
	Label  string `json:"label"`
}
type GateResolveRequest struct {
	Approve bool `json:"approve"`
}

type RunLogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	StepID     string    `json:"step_id,omitempty"`
	State      string    `json:"state"`
	Persona    string    `json:"persona,omitempty"`
	Message    string    `json:"message"`
	TokensUsed int       `json:"tokens_used,omitempty"`
	DurationMs int64     `json:"duration_ms,omitempty"`
}

type RunLogsResponse struct {
	RunID string        `json:"run_id"`
	Logs  []RunLogEntry `json:"logs"`
}

type SubmitRunRequest struct {
	Pipeline          string `json:"pipeline"`
	Input             string `json:"input"`
	Model             string `json:"model,omitempty"`
	Adapter           string `json:"adapter,omitempty"`
	DryRun            bool   `json:"dry_run,omitempty"`
	FromStep          string `json:"from_step,omitempty"`
	Force             bool   `json:"force,omitempty"`
	Detach            bool   `json:"detach,omitempty"`
	Timeout           int    `json:"timeout,omitempty"`
	Steps             string `json:"steps,omitempty"`
	Exclude           string `json:"exclude,omitempty"`
	OnFailure         string `json:"on_failure,omitempty"`
	Continuous        bool   `json:"continuous,omitempty"`
	Source            string `json:"source,omitempty"`
	MaxIterations     int    `json:"max_iterations,omitempty"`
	Delay             string `json:"delay,omitempty"`
	Mock              bool   `json:"mock,omitempty"`
	PreserveWorkspace bool   `json:"preserve_workspace,omitempty"`
	AutoApprove       bool   `json:"auto_approve,omitempty"`
	NoRetro           bool   `json:"no_retro,omitempty"`
	ForceModel        bool   `json:"force_model,omitempty"`
}

type SubmitRunResponse struct {
	RunID        string    `json:"run_id"`
	PipelineName string    `json:"pipeline_name"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
}

// StartIssueRequest is the request body for starting a pipeline from an issue.
type StartIssueRequest struct {
	IssueURL      string `json:"issue_url"`
	PipelineName  string `json:"pipeline_name"`
	Model         string `json:"model,omitempty"`
	Adapter       string `json:"adapter,omitempty"`
	DryRun        bool   `json:"dry_run,omitempty"`
	FromStep      string `json:"from_step,omitempty"`
	Force         bool   `json:"force,omitempty"`
	Detach        bool   `json:"detach,omitempty"`
	Timeout       int    `json:"timeout,omitempty"`
	Steps         string `json:"steps,omitempty"`
	Exclude       string `json:"exclude,omitempty"`
	OnFailure     string `json:"on_failure,omitempty"`
	Continuous    bool   `json:"continuous,omitempty"`
	Source        string `json:"source,omitempty"`
	MaxIterations int    `json:"max_iterations,omitempty"`
	Delay         string `json:"delay,omitempty"`
}

// StartPRRequest is the request body for starting a pipeline from a PR.
type StartPRRequest struct {
	PRURL         string `json:"pr_url"`
	PipelineName  string `json:"pipeline_name"`
	Model         string `json:"model,omitempty"`
	Adapter       string `json:"adapter,omitempty"`
	DryRun        bool   `json:"dry_run,omitempty"`
	FromStep      string `json:"from_step,omitempty"`
	Force         bool   `json:"force,omitempty"`
	Detach        bool   `json:"detach,omitempty"`
	Timeout       int    `json:"timeout,omitempty"`
	Steps         string `json:"steps,omitempty"`
	Exclude       string `json:"exclude,omitempty"`
	OnFailure     string `json:"on_failure,omitempty"`
	Continuous    bool   `json:"continuous,omitempty"`
	Source        string `json:"source,omitempty"`
	MaxIterations int    `json:"max_iterations,omitempty"`
	Delay         string `json:"delay,omitempty"`
}

// ForkRunRequest is the request body for forking a run from a specific step.
type ForkRunRequest struct {
	FromStep string `json:"from_step"`
}

// ForkRunResponse is the JSON response after forking a run.
type ForkRunResponse struct {
	RunID        string    `json:"run_id"`
	SourceRunID  string    `json:"source_run_id"`
	FromStep     string    `json:"from_step"`
	PipelineName string    `json:"pipeline_name"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
}

// RewindRunRequest is the request body for rewinding a run to a specific step.
type RewindRunRequest struct {
	ToStep string `json:"to_step"`
}

// RewindRunResponse is the JSON response after rewinding a run.
type RewindRunResponse struct {
	RunID        string   `json:"run_id"`
	ToStep       string   `json:"to_step"`
	StepsDeleted []string `json:"steps_deleted"`
	Status       string   `json:"status"`
}

// ForkPointResponse represents an available fork point returned by the API.
type ForkPointResponse struct {
	StepID    string `json:"step_id"`
	StepIndex int    `json:"step_index"`
	HasSHA    bool   `json:"has_sha"`
}

// ForkPointsResponse is the JSON response for listing fork points.
type ForkPointsResponse struct {
	RunID      string              `json:"run_id"`
	ForkPoints []ForkPointResponse `json:"fork_points"`
}

// RetroTrendEntry holds a single retrospective entry for the smoothness trend chart.
type RetroTrendEntry struct {
	RunID      string `json:"run_id"`
	Pipeline   string `json:"pipeline"`
	Smoothness string `json:"smoothness"`
	HeightPct  int    `json:"height_pct"`
	CreatedAt  string `json:"created_at"`
}

// FrictionCount holds an aggregated friction point type with its occurrence count.
type FrictionCount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// PipelineSuccessRate holds aggregated success metrics for a pipeline.
type PipelineSuccessRate struct {
	Pipeline    string `json:"pipeline"`
	TotalRuns   int    `json:"total_runs"`
	SuccessPct  int    `json:"success_pct"`
	AvgDuration string `json:"avg_duration"`
}

// RetroListEntry holds a retrospective record for the list view.
type RetroListEntry struct {
	RunID      string `json:"run_id"`
	Pipeline   string `json:"pipeline"`
	Smoothness string `json:"smoothness"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}
