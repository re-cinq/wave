# Data Model: Dashboard Inspection, Rendering, Statistics & Run Introspection

**Branch**: `091-dashboard-introspection` | **Date**: 2026-02-14

## Existing Entities (No Modifications)

These entities already exist in the codebase. This feature reads from them but does not alter their schema.

### pipeline_run (SQLite table)
Source: `internal/state/schema.sql`

| Column | Type | Description |
|--------|------|-------------|
| run_id | TEXT PK | Unique run identifier |
| pipeline_name | TEXT | Pipeline name |
| status | TEXT | pending/running/completed/failed/cancelled |
| input | TEXT | Pipeline input |
| current_step | TEXT | Currently executing step |
| total_tokens | INTEGER | Total tokens consumed |
| started_at | INTEGER | Unix timestamp |
| completed_at | INTEGER | Unix timestamp (nullable) |
| cancelled_at | INTEGER | Unix timestamp (nullable) |
| error_message | TEXT | Error message (nullable) |
| tags_json | TEXT | JSON array of tags |

**Indexes**: `pipeline_name`, `status`, `started_at`

### event_log (SQLite table)
Source: `internal/state/schema.sql`

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| run_id | TEXT FK | References pipeline_run |
| timestamp | INTEGER | Unix timestamp |
| step_id | TEXT | Step identifier |
| state | TEXT | Event state |
| persona | TEXT | Persona name |
| message | TEXT | Event message |
| tokens_used | INTEGER | Token count |
| duration_ms | INTEGER | Duration in ms |

### performance_metric (SQLite table)
Source: `internal/state/schema.sql`

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| run_id | TEXT FK | References pipeline_run |
| step_id | TEXT | Step identifier |
| pipeline_name | TEXT | Pipeline name |
| persona | TEXT | Persona name |
| started_at | INTEGER | Unix timestamp |
| completed_at | INTEGER | Unix timestamp (nullable) |
| duration_ms | INTEGER | Duration in ms |
| tokens_used | INTEGER | Token count |
| files_modified | INTEGER | Files changed |
| artifacts_generated | INTEGER | Artifacts created |
| memory_bytes | INTEGER | Memory used |
| success | BOOLEAN | Success flag |
| error_message | TEXT | Error message |

**Indexes**: `run_id`, `step_id`, `pipeline_name`, `started_at`

### step_state (SQLite table)
Source: `internal/state/schema.sql`

| Column | Type | Description |
|--------|------|-------------|
| step_id | TEXT PK | Step identifier |
| pipeline_id | TEXT FK | References pipeline_state |
| state | TEXT | Step state |
| retry_count | INTEGER | Retry count |
| started_at | INTEGER | Unix timestamp (nullable) |
| completed_at | INTEGER | Unix timestamp (nullable) |
| workspace_path | TEXT | Workspace directory path |
| error_message | TEXT | Error message |

### Manifest Types (Go structs)
Source: `internal/manifest/types.go`

- `Manifest` — top-level config with `Personas map[string]Persona`, `Adapters map[string]Adapter`, `Runtime`
- `Persona` — adapter, description, system_prompt_file, temperature, model, permissions, hooks, sandbox
- `Adapter` — binary, mode, output_format, project_files, default_permissions

### Pipeline Types (Go structs)
Source: `internal/pipeline/types.go`

- `Pipeline` — kind, metadata, requires, input, steps
- `Step` — id, persona, dependencies, memory, workspace, exec, output_artifacts, handover, strategy, validation
- `ContractConfig` — type, schema, source, schema_path, validate, command, dir, must_pass
- `InputConfig` — source, schema, example, label_filter, batch_size

---

## New API Response Types

These Go types are added to `internal/webui/types.go`.

### PipelineDetailResponse

```go
// PipelineDetailResponse is the JSON response for the pipeline detail API.
type PipelineDetailResponse struct {
    Name            string                `json:"name"`
    Description     string                `json:"description,omitempty"`
    StepCount       int                   `json:"step_count"`
    Input           PipelineInputDetail   `json:"input"`
    Steps           []PipelineStepDetail  `json:"steps"`
    DAG             *DAGData              `json:"dag,omitempty"`
    LastRun         *RunSummary           `json:"last_run,omitempty"`
}

// PipelineInputDetail holds pipeline input configuration.
type PipelineInputDetail struct {
    Source  string `json:"source"`
    Schema  *InputSchemaDetail `json:"schema,omitempty"`
    Example string `json:"example,omitempty"`
}

// InputSchemaDetail holds input schema details.
type InputSchemaDetail struct {
    Type        string `json:"type,omitempty"`
    Description string `json:"description,omitempty"`
}

// PipelineStepDetail holds full step configuration for the pipeline detail view.
type PipelineStepDetail struct {
    ID            string              `json:"id"`
    Persona       string              `json:"persona"`
    Dependencies  []string            `json:"dependencies,omitempty"`
    Workspace     WorkspaceDetail     `json:"workspace"`
    Contract      *ContractDetail     `json:"contract,omitempty"`
    Artifacts     []ArtifactDefDetail `json:"artifacts,omitempty"`
    Memory        MemoryDetail        `json:"memory"`
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
    Strategy string           `json:"strategy"`
    Injected []InjectedArtifact `json:"injected,omitempty"`
}

// InjectedArtifact holds artifact injection references.
type InjectedArtifact struct {
    FromStep string `json:"from_step"`
    Artifact string `json:"artifact"`
    As       string `json:"as"`
}
```

### PersonaDetailResponse

```go
// PersonaDetailResponse is the JSON response for the persona detail API.
type PersonaDetailResponse struct {
    Name             string   `json:"name"`
    Description      string   `json:"description,omitempty"`
    Adapter          string   `json:"adapter"`
    Model            string   `json:"model,omitempty"`
    Temperature      float64  `json:"temperature"`
    SystemPrompt     string   `json:"system_prompt,omitempty"`
    SystemPromptFile string   `json:"system_prompt_file"`
    AllowedTools     []string `json:"allowed_tools,omitempty"`
    DeniedTools      []string `json:"denied_tools,omitempty"`
    Hooks            *HooksDetail   `json:"hooks,omitempty"`
    Sandbox          *SandboxDetail `json:"sandbox,omitempty"`
    UsedInPipelines  []string `json:"used_in_pipelines,omitempty"`
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
```

### Statistics Types

```go
// RunStatistics holds aggregate run counts.
type RunStatistics struct {
    Total       int     `json:"total"`
    Succeeded   int     `json:"succeeded"`
    Failed      int     `json:"failed"`
    Cancelled   int     `json:"cancelled"`
    Pending     int     `json:"pending"`
    Running     int     `json:"running"`
    SuccessRate float64 `json:"success_rate"` // percentage 0-100
}

// RunTrendPoint holds a single data point in the run trend.
type RunTrendPoint struct {
    Date        string  `json:"date"`        // YYYY-MM-DD
    Total       int     `json:"total"`
    Succeeded   int     `json:"succeeded"`
    Failed      int     `json:"failed"`
    SuccessRate float64 `json:"success_rate"` // percentage 0-100
}

// PipelineStatistics holds per-pipeline aggregate stats.
type PipelineStatistics struct {
    PipelineName  string  `json:"pipeline_name"`
    RunCount      int     `json:"run_count"`
    SuccessRate   float64 `json:"success_rate"`   // percentage 0-100
    AvgDurationMs int64   `json:"avg_duration_ms"`
    AvgTokens     int     `json:"avg_tokens"`
}

// StatisticsResponse is the JSON response for the statistics API.
type StatisticsResponse struct {
    Aggregate  RunStatistics        `json:"aggregate"`
    Trends     []RunTrendPoint      `json:"trends"`
    Pipelines  []PipelineStatistics `json:"pipelines"`
    TimeRange  string               `json:"time_range"` // "24h", "7d", "30d", "all"
}
```

### Enhanced Run Detail Types

```go
// EnhancedStepDetail extends StepDetail with introspection data.
type EnhancedStepDetail struct {
    StepDetail                          // embed base
    ContractResult  *ContractResultDetail `json:"contract_result,omitempty"`
    RecoveryHints   []RecoveryHintDetail  `json:"recovery_hints,omitempty"`
    Performance     *StepPerfDetail       `json:"performance,omitempty"`
    WorkspacePath   string                `json:"workspace_path,omitempty"`
    WorkspaceExists bool                  `json:"workspace_exists"`
}

// ContractResultDetail holds contract validation outcome for a step.
type ContractResultDetail struct {
    Type          string `json:"type"`
    Passed        bool   `json:"passed"`
    ErrorMessage  string `json:"error_message,omitempty"`
    Schema        string `json:"schema,omitempty"`
}

// RecoveryHintDetail holds a recovery suggestion for display.
type RecoveryHintDetail struct {
    Label   string `json:"label"`
    Command string `json:"command"`
    Type    string `json:"type"` // resume, force, workspace, debug
}

// StepPerfDetail holds performance metrics for a specific step execution.
type StepPerfDetail struct {
    DurationMs         int64 `json:"duration_ms"`
    TokensUsed         int   `json:"tokens_used"`
    FilesModified      int   `json:"files_modified"`
    ArtifactsGenerated int   `json:"artifacts_generated"`
}
```

### Workspace Browsing Types

```go
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
```

---

## New StateStore Interface Methods

These methods are added to the `StateStore` interface in `internal/state/store.go`:

```go
// Statistics (spec 091 - Dashboard Introspection)
GetRunStatistics(since time.Time) (*RunStatisticsRecord, error)
GetRunTrends(since time.Time) ([]RunTrendRecord, error)
GetPipelineStatistics(since time.Time) ([]PipelineStatisticsRecord, error)
GetPipelineStepStats(pipelineName string, since time.Time) ([]StepPerformanceStats, error)
GetLastRunForPipeline(pipelineName string) (*RunRecord, error)
```

### New State Types

```go
// RunStatisticsRecord holds aggregate run statistics from SQL.
type RunStatisticsRecord struct {
    Total     int
    Succeeded int
    Failed    int
    Cancelled int
    Pending   int
    Running   int
}

// RunTrendRecord holds a daily trend data point from SQL.
type RunTrendRecord struct {
    Date      string // YYYY-MM-DD
    Total     int
    Succeeded int
    Failed    int
}

// PipelineStatisticsRecord holds per-pipeline aggregate stats from SQL.
type PipelineStatisticsRecord struct {
    PipelineName  string
    RunCount      int
    Succeeded     int
    Failed        int
    AvgDurationMs int64
    AvgTokens     int
}
```

---

## New API Routes

Added to `routes.go`:

```
# Pipeline detail
GET /pipelines/{name}              → handlePipelineDetailPage (HTML)
GET /api/pipelines/{name}          → handleAPIPipelineDetail (JSON)

# Persona detail
GET /personas/{name}               → handlePersonaDetailPage (HTML)
GET /api/personas/{name}           → handleAPIPersonaDetail (JSON)

# Statistics
GET /statistics                    → handleStatisticsPage (HTML)
GET /api/statistics                → handleAPIStatistics (JSON)

# Workspace browsing
GET /api/runs/{id}/workspace/{step}/tree   → handleWorkspaceTree (JSON)
GET /api/runs/{id}/workspace/{step}/file   → handleWorkspaceFile (JSON)
```

---

## New Static Assets

Added to `internal/webui/static/`:

| File | Purpose | Est. Size |
|------|---------|-----------|
| `markdown.js` | Minimal markdown parser with raw/rendered toggle | ~5 KB |
| `highlight.js` | Regex-based syntax highlighter for 9 languages | ~4 KB |
| `stats.js` | Statistics page interactions (time range filter) | ~3 KB |
| `workspace.js` | File tree lazy-loading and file content viewer | ~3 KB |
| `introspect.js` | Step drill-down, toggles, contract inspection | ~2 KB |

---

## New HTML Templates

Added to `internal/webui/templates/`:

| File | Purpose |
|------|---------|
| `pipeline_detail.html` | Pipeline inspection with step detail, contract display, DAG |
| `persona_detail.html` | Persona inspection with system prompt, permissions, hooks |
| `statistics.html` | Statistics dashboard with aggregate counts, trends, per-pipeline |
| `partials/markdown_viewer.html` | Markdown content with raw/rendered toggle |
| `partials/code_viewer.html` | Syntax highlighted content with raw/formatted toggle |
| `partials/step_inspector.html` | Step introspection with drill-down panels |
| `partials/workspace_tree.html` | File tree browser with lazy loading |
| `partials/stats_chart.html` | CSS-based chart components |

---

## Entity Relationship Summary

```
Manifest (in-memory)
├── Personas map[string]Persona     ──→ PersonaDetailResponse
├── Adapters map[string]Adapter
└── Runtime

Pipeline YAML (filesystem)
├── PipelineMetadata                ──→ PipelineDetailResponse
├── Steps[]                         ──→ PipelineStepDetail[]
│   ├── Persona ref                 ──→ cross-link to PersonaDetailResponse
│   ├── ContractConfig              ──→ ContractDetail
│   └── OutputArtifacts[]           ──→ ArtifactDefDetail[]
└── InputConfig                     ──→ PipelineInputDetail

pipeline_run (SQLite)               ──→ RunStatistics, RunTrendPoint, PipelineStatistics
├── event_log[]                     ──→ EventSummary[] (event timeline)
├── performance_metric[]            ──→ StepPerfDetail
├── artifact[]                      ──→ ArtifactSummary[]
└── step_state (workspace_path)     ──→ WorkspaceTreeResponse, WorkspaceFileResponse
```
