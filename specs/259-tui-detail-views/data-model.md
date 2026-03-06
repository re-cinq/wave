# Data Model: TUI Alternative Master-Detail Views

**Date**: 2026-03-06  
**Feature**: #259 — TUI Alternative Master-Detail Views  
**Branch**: `259-tui-detail-views`

## View Type Enum

```go
// ViewType identifies the active content view.
type ViewType int

const (
    ViewPipelines ViewType = iota // Default — existing pipeline list+detail
    ViewPersonas                   // Persona introspection
    ViewContracts                  // Contract inspection
    ViewSkills                     // Skill discovery
    ViewHealth                     // System health diagnostics
)

// String returns the display name for the view (used as status bar label).
func (v ViewType) String() string // "Pipelines", "Personas", etc.
```

## View Changed Message

```go
// ViewChangedMsg is emitted when the user switches views via Tab.
type ViewChangedMsg struct {
    View ViewType
}
```

## Persona Data Structures

```go
// PersonaInfo is the TUI data projection for a persona.
type PersonaInfo struct {
    Name         string
    Description  string
    Adapter      string
    Model        string
    AllowedTools []string
    DeniedTools  []string
    PipelineUsage []PipelineStepRef // Which pipeline steps use this persona
}

// PersonaStats holds aggregated run statistics for a persona.
type PersonaStats struct {
    TotalRuns      int
    SuccessfulRuns int
    AvgDurationMs  int64
    LastRunAt      time.Time
}

// PipelineStepRef identifies a pipeline/step pair.
type PipelineStepRef struct {
    PipelineName string
    StepID       string
}
```

**Data Sources**:
- `PersonaInfo`: Manifest `Personas` map + pipeline step scan for usage
- `PersonaStats`: `StateStore.GetRecentPerformanceHistory(PerformanceQueryOptions{Persona: name})`, aggregated in-process

## Contract Data Structures

```go
// ContractInfo is the TUI data projection for a contract.
type ContractInfo struct {
    Label         string   // Schema filename or "pipeline:step" for inline
    Type          string   // "json_schema", "test_suite", "quality_gate", etc.
    SchemaPath    string   // File path (empty for inline contracts)
    Source        string   // Inline source content (empty for file-backed)
    SchemaPreview string   // First ~30 lines of schema content
    PipelineUsage []PipelineStepRef
}
```

**Data Sources**:
- Pipeline YAML files: scan all `Step.Handover.Contract` configs
- Schema file content: read from `SchemaPath` if exists

## Skill Data Structures

```go
// SkillInfo is the TUI data projection for a skill.
type SkillInfo struct {
    Name          string
    CommandsGlob  string   // e.g., ".claude/commands/speckit.*.md"
    CommandFiles  []string // Resolved from glob
    InstallCmd    string
    CheckCmd      string
    PipelineUsage []string // Pipeline names that require this skill
}
```

**Data Sources**:
- Pipeline YAML files: `Requires.Skills` map
- Filesystem: `filepath.Glob()` to resolve command files

## Health Check Data Structures

```go
// HealthStatus represents the result of a health check.
type HealthCheckStatus int

const (
    HealthOK   HealthCheckStatus = iota
    HealthWarn
    HealthErr
    HealthChecking // Initial state before check completes
)

// HealthCheck holds the state and result of a single health check.
type HealthCheck struct {
    Name        string
    Description string
    Status      HealthCheckStatus
    Message     string            // Summary line
    Details     map[string]string // Key-value diagnostic pairs
    LastChecked time.Time
}

// HealthCheckResultMsg carries the result of an async health check.
type HealthCheckResultMsg struct {
    Name    string
    Status  HealthCheckStatus
    Message string
    Details map[string]string
}
```

**Health Checks** (fixed set of 6):

| # | Name | Check Method | OK | WARN | FAIL |
|---|------|-------------|----|----|------|
| 1 | Git Repository | `git rev-parse --is-inside-work-tree` | Valid repo, clean | Dirty working tree | Not a git repo |
| 2 | Adapter Binary | `exec.LookPath(binary)` per adapter | All found | — | Binary not found |
| 3 | SQLite Database | Open + test query | DB accessible | — | Connection failed |
| 4 | Wave Configuration | `manifest.LoadManifest()` | Valid config | Warnings | Load failed |
| 5 | Required Tools | `exec.LookPath(tool)` per pipeline tool | All available | — | Tool missing |
| 6 | Required Skills | Skill check command | All installed | — | Not installed |

## Data Provider Interfaces

```go
// PersonaDataProvider fetches persona data for the Personas view.
type PersonaDataProvider interface {
    FetchPersonas() ([]PersonaInfo, error)
    FetchPersonaStats(name string) (*PersonaStats, error)
}

// ContractDataProvider fetches contract data for the Contracts view.
type ContractDataProvider interface {
    FetchContracts() ([]ContractInfo, error)
}

// SkillDataProvider fetches skill data for the Skills view.
type SkillDataProvider interface {
    FetchSkills() ([]SkillInfo, error)
}

// HealthDataProvider executes health checks for the Health view.
type HealthDataProvider interface {
    RunCheck(name string) HealthCheckResultMsg
    CheckNames() []string // Returns ordered list of check names
}
```

## View Model Structure (per alternative view)

Each alternative view follows the same two-model pattern as pipelines:

```go
// Generic list model pattern (applied per-view with different item types):
type <View>ListModel struct {
    width, height int
    items         []<ItemType>  // Full data set
    cursor        int
    navigable     []<ItemType>  // Filtered subset
    filtering     bool
    filterInput   textinput.Model
    filterQuery   string
    focused       bool
    scrollOffset  int
    provider      <View>DataProvider
    loaded        bool          // Lazy init guard
}

// Generic detail model pattern:
type <View>DetailModel struct {
    width, height int
    focused       bool
    viewport      viewport.Model
    selected      *<ItemType>   // Currently displayed item
}
```

## ContentModel Extensions

```go
type ContentModel struct {
    // Existing fields
    width, height int
    list          PipelineListModel
    detail        PipelineDetailModel
    focus         FocusPane
    launcher      *PipelineLauncher

    // New fields for view switching
    currentView   ViewType

    // Lazy-initialized alternative view models (nil until first access)
    personaList   *PersonaListModel
    personaDetail *PersonaDetailModel
    contractList  *ContractListModel
    contractDetail *ContractDetailModel
    skillList     *SkillListModel
    skillDetail   *SkillDetailModel
    healthList    *HealthListModel
    healthDetail  *HealthDetailModel

    // Data providers for alternative views (injected at construction)
    personaProvider  PersonaDataProvider
    contractProvider ContractDataProvider
    skillProvider    SkillDataProvider
    healthProvider   HealthDataProvider
}
```

## Message Flow

```
Tab key press
  → ContentModel intercepts (before child routing)
  → Check: pipeline detail paneState == stateConfiguring? Forward to form
  → Otherwise: increment currentView, reset focus to left pane
  → Emit ViewChangedMsg{View: newView}
  → If new view models are nil, lazy-init and trigger data fetch
  → StatusBar receives ViewChangedMsg, updates contextLabel
```
