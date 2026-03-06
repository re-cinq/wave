# Implementation Plan: TUI Alternative Master-Detail Views

**Branch**: `259-tui-detail-views` | **Date**: 2026-03-06 | **Spec**: `specs/259-tui-detail-views/spec.md`
**Input**: Feature specification from `/specs/259-tui-detail-views/spec.md`

## Summary

Implement four alternative views accessible via `Tab` cycling: Personas, Contracts, Skills, and Health. Each view follows the two-pane master-detail pattern established by the Pipelines view (#254/#255). `ContentModel` gains a `currentView` field and lazy-initialized view models. Tab intercepts before child routing to cycle views (unless a form is active). The status bar context label updates per view. Health checks run asynchronously with `r` to re-run. Per-view data providers enable mock testing. View state (cursor, scroll, selection) is preserved across tab switches.

## Technical Context

**Language/Version**: Go 1.25+ (existing project)  
**Primary Dependencies**: `charmbracelet/bubbletea` v1.3.10, `charmbracelet/bubbles/viewport`, `charmbracelet/bubbles/textinput` (all existing)  
**Storage**: SQLite via `internal/state` (existing — `GetRecentPerformanceHistory` for persona stats)  
**Testing**: `go test` with `testify/assert`, `testify/require`  
**Target Platform**: Linux/macOS terminal (80–300 columns, 24–100 rows)  
**Project Type**: Single Go binary — changes in `internal/tui/`  
**Constraints**: No new external dependencies; must not break existing tests (`go test ./...`)

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ Pass | No new runtime dependencies. Uses existing bubbletea, stdlib `os/exec`, `filepath.Glob`. |
| P2: Manifest as SSOT | ✅ Pass | Persona, contract, skill data all sourced from manifest/pipeline YAML. No new config files. |
| P3: Persona-Scoped Execution | N/A | TUI is a display layer, not pipeline execution. |
| P4: Fresh Memory at Step Boundary | N/A | TUI views are user-interactive, not pipeline steps. |
| P5: Navigator-First Architecture | N/A | TUI is not a pipeline. |
| P6: Contracts at Every Handover | N/A | No pipeline step handovers. |
| P7: Relay via Dedicated Summarizer | N/A | TUI component, no context compaction. |
| P8: Ephemeral Workspaces | ✅ Pass | Read-only access to workspace filesystem for health checks. |
| P9: Credentials Never Touch Disk | ✅ Pass | No credential handling. |
| P10: Observable Progress | ✅ Pass | Health checks provide system observability to the user. |
| P11: Bounded Recursion | N/A | No pipeline execution. |
| P12: Minimal Step State Machine | N/A | No step state transitions. |
| P13: Test Ownership | ✅ Pass | All new code will have tests; existing tests must continue to pass. |

No violations. No complexity tracking entries needed.

## Project Structure

### Documentation (this feature)

```
specs/259-tui-detail-views/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
internal/tui/
├── views.go                    # NEW — ViewType enum, ViewChangedMsg, shared view types (PipelineStepRef)
├── content.go                  # MODIFY — add currentView, Tab interception, view model routing, IsFiltering(), CurrentFocus()
├── content_test.go             # MODIFY — Tab cycling tests, view state preservation, IsFiltering delegation
├── app.go                      # MODIFY — forward ViewChangedMsg to statusbar, use IsFiltering()/CurrentFocus() for quit guard
├── app_test.go                 # MODIFY — test ViewChangedMsg forwarding, quit guard with alternative views
├── statusbar.go                # MODIFY — handle ViewChangedMsg to update contextLabel, view-specific hints
├── statusbar_test.go           # MODIFY — test context label updates, view-specific hint rendering
├── persona_list.go             # NEW — PersonaListModel (left pane for Personas view)
├── persona_detail.go           # NEW — PersonaDetailModel (right pane for Personas view)
├── persona_provider.go         # NEW — PersonaDataProvider interface, DefaultPersonaDataProvider
├── persona_provider_test.go    # NEW — tests for persona data provider
├── persona_test.go             # NEW — tests for PersonaListModel and PersonaDetailModel
├── contract_list.go            # NEW — ContractListModel (left pane for Contracts view)
├── contract_detail.go          # NEW — ContractDetailModel (right pane for Contracts view)
├── contract_provider.go        # NEW — ContractDataProvider interface, DefaultContractDataProvider
├── contract_provider_test.go   # NEW — tests for contract data provider
├── contract_test.go            # NEW — tests for ContractListModel and ContractDetailModel
├── skill_list.go               # NEW — SkillListModel (left pane for Skills view)
├── skill_detail.go             # NEW — SkillDetailModel (right pane for Skills view)
├── skill_provider.go           # NEW — SkillDataProvider interface, DefaultSkillDataProvider
├── skill_provider_test.go      # NEW — tests for skill data provider
├── skill_test.go               # NEW — tests for SkillListModel and SkillDetailModel
├── health_list.go              # NEW — HealthListModel (left pane for Health view)
├── health_detail.go            # NEW — HealthDetailModel (right pane for Health view)
├── health_provider.go          # NEW — HealthDataProvider interface, DefaultHealthDataProvider, 6 checks
├── health_provider_test.go     # NEW — tests for health data provider
└── health_test.go              # NEW — tests for HealthListModel and HealthDetailModel
```

**Structure Decision**: Each view gets its own files for list model, detail model, and data provider. This follows the established pattern from the pipeline view (`pipeline_list.go`, `pipeline_detail.go`, `pipeline_provider.go`, `pipeline_detail_provider.go`). Shared types (ViewType, PipelineStepRef) go in a new `views.go` file. This keeps files focused and avoids bloating content.go.

## Implementation Approach

### Phase 1: View Infrastructure — ViewType, Tab Cycling, ContentModel Refactoring

**Files**: `views.go` (NEW), `content.go` (MODIFY), `content_test.go` (MODIFY), `app.go` (MODIFY), `app_test.go` (MODIFY), `statusbar.go` (MODIFY), `statusbar_test.go` (MODIFY)

1. **Create `views.go`** with:
   - `ViewType` enum: `ViewPipelines`, `ViewPersonas`, `ViewContracts`, `ViewSkills`, `ViewHealth`
   - `ViewType.String()` method for status bar labels
   - `ViewChangedMsg{View ViewType}` message type
   - `PipelineStepRef{PipelineName, StepID}` struct (shared across persona/contract/skill views)

2. **Modify `ContentModel`** in `content.go`:
   - Add `currentView ViewType` field (default: `ViewPipelines`)
   - Add pointer fields for alternative view models (initially nil for lazy init)
   - Add provider fields for alternative view data sources (injected at construction)
   - Update `NewContentModel()` signature to accept optional alternative view providers
   - Add `IsFiltering() bool` — delegates to the active view's list model's filtering state
   - Add `CurrentFocus() FocusPane` — returns `m.focus`

3. **Tab key interception** in `ContentModel.Update()`:
   - Insert Tab check *before* the existing `tea.KeyMsg` switch (before focus-based routing at line 138)
   - Logic: if `msg.Type == tea.KeyTab`:
     - If `currentView == ViewPipelines` and `m.detail.paneState == stateConfiguring`: forward Tab to detail (form navigation)
     - Otherwise: call `m.cycleView()` which increments `currentView`, resets focus to left pane, lazy-inits the new view if needed, emits `ViewChangedMsg`
   - `cycleView()` returns `tea.Cmd` batch: `ViewChangedMsg`, `FocusChangedMsg{Left}`, and data fetch cmd if view was just initialized

4. **View rendering** in `ContentModel.View()`:
   - Switch on `currentView` to render the appropriate list+detail pair
   - Apply same left/right pane sizing via `leftPaneWidth()` and dimming when right pane focused

5. **Message routing** in `ContentModel.Update()`:
   - Pipeline-specific messages (`PipelineDataMsg`, `PipelineRefreshTickMsg`, `PipelineSelectedMsg`, `DetailDataMsg`, `ConfigureFormMsg`, `LaunchRequestMsg`, etc.) only route to pipeline models
   - View-specific messages (persona/contract/skill/health data messages) route to their respective view models
   - `tea.WindowSizeMsg` (via `SetSize`) propagates to all initialized view models

6. **Modify `AppModel`** in `app.go`:
   - Change quit guard from `!m.content.list.filtering && m.content.focus == FocusPaneLeft` to `!m.content.IsFiltering() && m.content.CurrentFocus() == FocusPaneLeft`
   - Forward `ViewChangedMsg` to status bar (add to the `switch msg.(type)` case at line 94)

7. **Modify `StatusBarModel`** in `statusbar.go`:
   - Handle `ViewChangedMsg`: update `m.contextLabel` to `msg.View.String()`
   - Add view-specific hints:
     - Personas/Contracts/Skills left pane: `"↑↓: navigate  Enter: view  /: filter  Tab: view  q: quit  ctrl+c: exit"`
     - Health left pane: `"↑↓: navigate  Enter: view  r: recheck  Tab: view  q: quit  ctrl+c: exit"`
     - Any alternative view right pane: `"↑↓: scroll  Esc: back  Tab: view  q: quit  ctrl+c: exit"`
   - Add `currentView ViewType` field to track which hints to show

8. **Tests**:
   - Tab cycles through all 5 views (SC-001)
   - Tab wraps from Health back to Pipelines
   - Tab forwards to form when pipeline detail is configuring (SC-011)
   - Tab from right pane resets focus to left (US1.4)
   - View state preserved across tab switches (SC-002)
   - `IsFiltering()` delegates to active view's list
   - `ViewChangedMsg` updates status bar context label (SC-009)
   - `q` quits from alternative views when not filtering

### Phase 2: Personas View

**Files**: `persona_list.go`, `persona_detail.go`, `persona_provider.go`, `persona_provider_test.go`, `persona_test.go` (all NEW)

1. **`PersonaDataProvider` interface** in `persona_provider.go`:
   - `FetchPersonas() ([]PersonaInfo, error)` — loads manifest personas + pipeline usage scan
   - `FetchPersonaStats(name string) (*PersonaStats, error)` — aggregates from `GetRecentPerformanceHistory`

2. **`DefaultPersonaDataProvider`** in `persona_provider.go`:
   - Constructor takes `*manifest.Manifest`, `state.StateStore` (nullable), and `pipelinesDir string`
   - `FetchPersonas()`:
     - Iterate `manifest.Personas` map, sort alphabetically by key
     - For each persona, build `PersonaInfo` with name, description, adapter, model, permissions
     - Scan all pipeline YAML files to build `PipelineUsage` (step.Persona == name)
   - `FetchPersonaStats()`:
     - If store is nil, return `nil, nil`
     - Call `store.GetRecentPerformanceHistory(PerformanceQueryOptions{Persona: name, Limit: 1000})`
     - Aggregate: count total, count successful, avg duration_ms, max started_at

3. **`PersonaListModel`** in `persona_list.go`:
   - Follows `PipelineListModel` pattern: `width/height`, `cursor`, `navigable`, `filtering`, `filterInput`, `focused`, `scrollOffset`
   - Items are `PersonaInfo` structs sorted alphabetically
   - `Init()` returns `fetchPersonaData` command (async)
   - `Update()` handles `PersonaDataMsg`, arrow keys, `/` filter, Enter (emits `PersonaSelectedMsg`)
   - `View()` renders list with simple name entries, `"▶ "` cursor indicator

4. **`PersonaDetailModel`** in `persona_detail.go`:
   - Two states: empty (no selection) and showing detail
   - Embeds `viewport.Model` for scrolling
   - On `PersonaSelectedMsg`: trigger stats fetch, update viewport content
   - `renderPersonaDetail()`: title, description, adapter, model, permissions (allow/deny), pipeline usage list, run stats section

5. **Messages**: `PersonaDataMsg{Personas []PersonaInfo, Err error}`, `PersonaSelectedMsg{Name string}`, `PersonaStatsMsg{Name string, Stats *PersonaStats, Err error}`

6. **Tests** (SC-003, SC-004):
   - Mock provider returns known personas → verify list renders all names alphabetically
   - Select persona → detail shows correct metadata, permissions, pipeline usage
   - Mock stats → verify aggregated display (total runs, success rate, avg duration)
   - No stats → shows "No runs recorded"
   - Empty persona map → shows "No personas configured" placeholder

### Phase 3: Contracts View

**Files**: `contract_list.go`, `contract_detail.go`, `contract_provider.go`, `contract_provider_test.go`, `contract_test.go` (all NEW)

1. **`ContractDataProvider` interface** in `contract_provider.go`:
   - `FetchContracts() ([]ContractInfo, error)` — scans pipeline YAML for contract configs

2. **`DefaultContractDataProvider`** in `contract_provider.go`:
   - Constructor takes `pipelinesDir string`
   - `FetchContracts()`:
     - Scan all pipeline YAML files, parse `pipeline.Pipeline` structs
     - For each step with `Handover.Contract.Type != ""`:
       - If `SchemaPath != ""`: use schema filename as label, read first ~30 lines as preview
       - If `SchemaPath == ""` and `Source != ""`: use `"pipeline:step"` as label, use Source as preview
     - Deduplicate by SchemaPath: merge pipeline usage refs for same schema
     - Sort alphabetically by label

3. **`ContractListModel`** in `contract_list.go`:
   - Same pattern as PersonaListModel but with `ContractInfo` items
   - Renders: contract label, type badge (e.g., `[json_schema]`)

4. **`ContractDetailModel`** in `contract_detail.go`:
   - Shows: label, type, schema path, schema preview (scrollable), pipeline usage
   - Schema preview renders in a code-style block (monospace, dimmed)

5. **Messages**: `ContractDataMsg{Contracts []ContractInfo, Err error}`, `ContractSelectedMsg{Label string}`

6. **Tests** (SC-005):
   - Contracts deduplicated by schema path, combined usage
   - Inline contracts appear with `pipeline:step` label
   - Schema preview loads from file
   - Empty contracts → placeholder message

### Phase 4: Skills View

**Files**: `skill_list.go`, `skill_detail.go`, `skill_provider.go`, `skill_provider_test.go`, `skill_test.go` (all NEW)

1. **`SkillDataProvider` interface** in `skill_provider.go`:
   - `FetchSkills() ([]SkillInfo, error)` — scans pipeline YAML for skill requirements

2. **`DefaultSkillDataProvider`** in `skill_provider.go`:
   - Constructor takes `pipelinesDir string`
   - `FetchSkills()`:
     - Scan all pipeline YAML files
     - For each pipeline with `Requires.Skills`, extract skill name/config
     - Deduplicate by skill name, aggregate pipeline usage
     - For each skill, resolve `CommandsGlob` via `filepath.Glob()`
     - Sort alphabetically

3. **`SkillListModel`** in `skill_list.go`:
   - Same list pattern with `SkillInfo` items
   - Renders: skill name, command count badge

4. **`SkillDetailModel`** in `skill_detail.go`:
   - Shows: name, commands glob, resolved command files, install/check commands, pipeline usage

5. **Messages**: `SkillDataMsg{Skills []SkillInfo, Err error}`, `SkillSelectedMsg{Name string}`

6. **Tests** (SC-006):
   - Skills deduplicated across pipelines
   - Glob resolution discovers command files
   - Missing command files → "No commands found"
   - Empty skills → placeholder message

### Phase 5: Health View

**Files**: `health_list.go`, `health_detail.go`, `health_provider.go`, `health_provider_test.go`, `health_test.go` (all NEW)

1. **`HealthDataProvider` interface** in `health_provider.go`:
   - `RunCheck(name string) HealthCheckResultMsg`
   - `CheckNames() []string`

2. **`DefaultHealthDataProvider`** in `health_provider.go`:
   - Constructor takes `*manifest.Manifest`, `state.StateStore`, `pipelinesDir string`
   - 6 check functions:
     - `checkGitRepository()`: `git rev-parse`, branch name, remote URL, dirty status
     - `checkAdapterBinary()`: `exec.LookPath` for each adapter in manifest
     - `checkSQLiteDatabase()`: attempt `store.ListRuns(ListRunsOptions{Limit: 1})`
     - `checkWaveConfiguration()`: count personas/pipelines/adapters from manifest
     - `checkRequiredTools()`: `exec.LookPath` for each tool in all pipeline `Requires.Tools`
     - `checkRequiredSkills()`: run skill check commands from all pipeline `Requires.Skills`

3. **`HealthListModel`** in `health_list.go`:
   - Fixed list (no filtering, no section headers) — 6 entries
   - Each entry shows status icon: `●` OK (green), `▲` WARN (yellow), `✗` FAIL (red), `…` checking (gray)
   - `r` key: re-runs all checks (emits batch of check commands)
   - On init (first access): emits batch of 6 async check commands

4. **`HealthDetailModel`** in `health_detail.go`:
   - Shows: check name, status, message, detail key-value pairs, last-checked timestamp
   - Uses viewport for scrolling

5. **Messages**: `HealthCheckResultMsg{Name, Status, Message, Details}`, `HealthSelectedMsg{Name string}`

6. **Tests** (SC-007, SC-008):
   - Mock provider returns known results → verify status icons
   - `r` key re-runs all checks
   - Async result arrival updates individual entries
   - All checks initially show "checking..."

### Phase 6: Integration — Wiring ContentModel to Views

**Files**: `content.go` (MODIFY)

1. Wire all view models into `ContentModel`:
   - `SetSize()` propagates to all initialized view models
   - `Init()` only initializes pipeline models (alternatives are lazy)
   - View-specific `Update()` routing: pipeline messages → pipeline models only, persona messages → persona models only, etc.

2. `cycleView()` helper:
   - Increments `currentView` modulo 5
   - If target view models are nil: creates them, returns data fetch `tea.Cmd`
   - Resets focus to left pane of new view
   - Returns `tea.Batch(ViewChangedMsg, FocusChangedMsg, initCmd...)`

3. `View()` rendering:
   - Switch on `currentView`:
     - `ViewPipelines`: existing `m.list.View()` + `m.detail.View()`
     - `ViewPersonas`: `m.personaList.View()` + `m.personaDetail.View()`
     - etc.
   - Apply focus dimming to left pane consistently

4. Integration with `NewContentModel`:
   - Add optional provider parameters
   - In `RunTUI()`: construct providers from manifest, state store, pipelines dir

### Phase 7: Status Bar View-Specific Hints

**Files**: `statusbar.go` (MODIFY), `statusbar_test.go` (MODIFY)

1. Add `currentView ViewType` field to `StatusBarModel`

2. Handle `ViewChangedMsg` in `Update()`: set `m.currentView` and `m.contextLabel`

3. Update `View()` hint logic:
   - Priority chain: formActive → liveOutputActive → finishedDetailActive → view-specific
   - For non-pipeline views in left pane: show view-appropriate hints
   - Health view: add `r: recheck` to hints
   - All alternative views in right pane: show scroll + back + tab + quit

4. Tests (SC-009):
   - ViewChangedMsg updates contextLabel
   - Health view shows `r: recheck` hint
   - Alternative view right pane shows appropriate hints

## Complexity Tracking

_No constitution violations. No complexity tracking entries needed._
