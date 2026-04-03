# WebUI User Stories & E2E Test Expectations

**Created**: 2026-04-03
**Scope**: All 14 webui pages — user stories, current gaps, and e2e test expectations
**Status**: Living document — updated as features land

---

## US-1: Runs List Page (`/runs`)

**As a user**, I want to see all pipeline runs in a sortable, filterable list so I can monitor activity and find specific runs.

### Current Features
- [x] Table with columns: Status, Pipeline, Run ID, Started, Duration, Progress, Tokens, Adapter, Model
- [x] Status filter dropdown (pending, running, completed, failed, cancelled)
- [x] Pipeline name filter dropdown
- [x] Date "since" filter
- [x] Text search (pipeline name, run ID)
- [x] Sortable columns (status, pipeline, started, duration)
- [x] Cursor-based pagination ("Load more runs")
- [x] Export CSV / Export JSON
- [x] Filter chips with clear buttons
- [x] Empty states: no runs, no matching runs
- [x] "Start Pipeline" dialog with pipeline selector + input textarea
- [x] SSE live-updating status badges for running runs
- [x] Child run display (sub-pipelines)

### Missing vs CLI (`wave run`)
- [ ] **Start form: Model override** — CLI `--model haiku` lets users pick a model. Webui has no model selector in the start form
- [ ] **Start form: Adapter override** — CLI `--adapter opencode` lets users pick an adapter. Webui has no adapter selector
- [ ] **Start form: Dry-run toggle** — CLI `--dry-run` validates without executing. Webui has no dry-run option
- [ ] **Start form: Timeout override** — CLI `--timeout 30` sets a per-run timeout. Webui uses manifest default only
- [ ] **Start form: Step selection** — CLI `--steps clarify,plan` and `-x implement` let users pick steps. Webui runs all steps
- [ ] **Start form: Mock mode** — CLI `--mock` uses mock adapter for testing. Webui has no mock toggle
- [ ] **Start form: Preserve workspace** — CLI `--preserve-workspace` reuses previous workspace. Webui has no option
- [ ] **Start form: Continuous mode** — CLI `--continuous --source github:label=bug` for batch processing. Webui has no equivalent
- [ ] **Start form: No-retro toggle** — CLI `--no-retro` skips retrospective. Webui has no option
- [ ] **Start form: Auto-approve** — CLI `--auto-approve` skips approval gates. Webui has no option
- [ ] **Start form: Detach** — CLI `--detach` runs in background. Webui runs are always "detached" (fire-and-forget via API)
- [ ] **Smart input routing** — CLI auto-detects GitHub URLs, issue numbers, etc. and suggests pipelines. Webui requires manual pipeline selection

### E2E Test Cases
- `runs_list_loads`: Page loads with table, no JS errors
- `runs_list_filters`: Status/pipeline/date filters work, chip indicators appear
- `runs_list_sort`: Click sortable headers, verify sort order changes
- `runs_list_search`: Type in search box, table filters client-side
- `runs_list_pagination`: If >20 runs exist, "Load more" appears and works
- `runs_list_export_csv`: Click export CSV, verify download
- `runs_list_export_json`: Click export JSON, verify download
- `runs_list_empty`: With no runs, shows empty state with "Start Pipeline" button
- `runs_list_no_match`: With active filters and no matches, shows "No matching runs"
- `runs_list_sse_running`: Running runs show live-updating status badge
- `runs_list_child_runs`: Parent run shows indented child runs

---

## US-2: Run Detail Page (`/runs/{id}`)

**As a user**, I want to see the full details of a single run including step statuses, logs, artifacts, and recovery options.

### Current Features
- [x] Run summary bar: Pipeline, Status, Started, Duration, Tokens, Adapters, Models
- [x] Step cards with status, duration, token usage, persona, model, adapter badges
- [x] Step log viewer with auto-scroll toggle
- [x] Artifact download links per step
- [x] Cancel button (for running runs)
- [x] Retry button (for failed/cancelled runs)
- [x] Resume from step dialog (for failed/cancelled runs)
- [x] Fork from step dialog (for runs with checkpoints)
- [x] Rewind to step dialog (for runs with checkpoints)
- [x] Recovery hints based on error type (timeout, adapter, etc.)
- [x] SSE live updates for running steps
- [x] Copy run ID button
- [x] DAG visualization (collapsible, horizontal layout)
- [x] Approval gate buttons (approve/reject)
- [x] Diff browser for code changes
- [x] Child run links

### Missing vs CLI
- [ ] **Step-level overflow logs** — CLI shows full step output. Webui truncates logs, no "expand all" or "consolidate overflows" option
- [ ] **Artifact preview** — CLI shows artifact content inline. Webui only has download links
- [ ] **Run metadata** — CLI shows all flags used (--model, --adapter, --from-step, etc.). Webui doesn't display what options were used to start the run

### E2E Test Cases
- `run_detail_loads`: Page loads with summary bar, step cards, no JS errors
- `run_detail_running`: Running run shows live log stream, auto-scroll works
- `run_detail_cancel`: Click cancel, run status changes to cancelled
- `run_detail_retry`: Failed run → click retry → new run created
- `run_detail_resume`: Failed run → resume from step → new run from that step
- `run_detail_fork`: Run with checkpoints → fork from step → new run created
- `run_detail_rewind`: Run with checkpoints → rewind → steps after point deleted
- `run_detail_logs`: Click step → logs appear in viewer → scroll works
- `run_detail_autoscroll`: Toggle auto-scroll off, scroll manually, verify it stays
- `run_detail_dag`: DAG visualization renders, steps are clickable
- `run_detail_gate_approve`: Approval gate → approve → step proceeds
- `run_detail_gate_reject`: Approval gate → reject → run fails
- `run_detail_diff`: Run with code changes → diff browser shows files
- `run_detail_artifacts`: Step with artifacts → download links work
- `run_detail_child_runs`: Parent run links to child run detail pages
- `run_detail_copy_id`: Click copy button → run ID in clipboard

---

## US-3: Pipelines Page (`/pipelines`)

**As a user**, I want to browse all available pipelines, see their metadata, and navigate to run them.

### Current Features
- [x] Pipeline cards with name, description, step count, category
- [x] Category badges (forge, research, ops, etc.)
- [x] Composition badge for composed pipelines
- [x] Skills badge list
- [x] Run count with link to filtered runs
- [x] "Start" button on each card (opens quick-start dialog)
- [x] Search/filter

### Missing
- [ ] **Pipeline YAML preview** — No way to see the raw pipeline definition from the webui
- [ ] **Step list preview** — Cards show step count but not step names/order

### E2E Test Cases
- `pipelines_list_loads`: Page loads with pipeline cards
- `pipelines_search`: Search filters cards
- `pipelines_start`: Click "Start" → quick-start dialog appears
- `pipelines_run_count`: Click run count → navigates to /runs?pipeline=<name>
- `pipelines_composition`: Composition pipeline shows "Composition" badge

---

## US-4: Pipeline Detail Page (`/pipelines/{name}`)

**As a user**, I want to see the full definition of a pipeline including steps, dependencies, and run history.

### Current Features
- [x] Pipeline name, description, category
- [x] Step count, run count (linked to filtered runs)
- [x] Composition type badge
- [x] Skills list
- [x] Quick-start button
- [x] Recent runs table

### Missing
- [ ] **Step list with dependencies** — No visualization of steps, their personas, or DAG
- [ ] **Contract info** — No display of what contracts a pipeline enforces
- [ ] **Edit/disable pipeline** — No admin controls on the detail page

### E2E Test Cases
- `pipeline_detail_loads`: Page loads with pipeline info
- `pipeline_detail_start`: Click "Start" → quick-start dialog
- `pipeline_detail_runs`: Recent runs table shows and links work

---

## US-5: Compose Page (`/compose`)

**As a user**, I want to create and run pipeline compositions (sequences of pipelines).

### Current Features
- [x] List of composition pipelines
- [x] Each card shows component pipelines
- [x] Run button per composition

### Missing
- [ ] **Visual composition builder** — No drag-and-drop or form-based composition editor
- [ ] **Composition history** — No link to past composition runs
- [ ] **Artifact flow visualization** — No display of how artifacts flow between component pipelines

### E2E Test Cases
- `compose_list_loads`: Page loads with composition cards
- `compose_run`: Click "Run" on a composition → starts execution

---

## US-6: Issues Page (`/issues`)

**As a user**, I want to browse GitHub issues and launch pipelines from them.

### Current Features
- [x] Issue list from GitHub API
- [x] Issue detail page with description, labels, comments
- [x] Status badges (open/closed)
- [x] "Implement" button to start pipeline from issue

### Missing
- [ ] **Filter by label/milestone** — No label-based filtering
- [ ] **Filter by assignee** — No assignee filter
- [ ] **Sort options** — No sort by updated/created/comments
- [ ] **Issue search** — No text search across issues
- [ ] **Pagination** — No pagination for large issue lists

### E2E Test Cases
- `issues_list_loads`: Page loads with issue list
- `issues_detail`: Click issue → detail page with description
- `issues_start_pipeline`: Click "Implement" → pipeline start dialog

---

## US-7: PRs Page (`/prs`)

**As a user**, I want to browse pull requests and review them.

### Current Features
- [x] PR list from GitHub API
- [x] PR detail page with description, files changed, status
- [x] State badges (open/closed/merged)
- [x] Review form (approve/request changes/comment)

### Missing
- [ ] **Filter by state/label/reviewer** — No filtering
- [ ] **PR search** — No text search
- [ ] **Inline diff view** — No diff viewer in PR detail (only file list)
- [ ] **CI status** — No GitHub Actions/check status display

### E2E Test Cases
- `prs_list_loads`: Page loads with PR list
- `prs_detail`: Click PR → detail page
- `prs_review_approve`: Submit approve review → success feedback
- `prs_review_comment`: Submit comment review → success feedback

---

## US-8: Analytics Page (`/analytics`)

**As a user**, I want to see aggregate statistics about pipeline runs.

### Current Features
- [x] Run count, success rate, average duration charts
- [x] Token usage statistics
- [x] Pipeline usage breakdown

### Missing
- [ ] **Date range selector** — Charts show all-time data, no date range filter
- [ ] **Export charts** — No way to export analytics data

### E2E Test Cases
- `analytics_loads`: Page loads with charts rendered
- `analytics_charts`: Charts display data (not empty)

---

## US-9: Retros Page (`/retros`)

**As a user**, I want to browse pipeline retrospective reports.

### Current Features
- [x] Retrospective list with pipeline name, run ID, date
- [x] Retrospective detail with LLM-generated analysis
- [x] "Narrate" button to generate/update narration

### Missing
- [ ] **Filter by pipeline** — No pipeline filter on retros list
- [ ] **Search** — No text search across retros

### E2E Test Cases
- `retros_list_loads`: Page loads with retrospective list
- `retros_detail`: Click retro → detail page
- `retros_narrate`: Click "Narrate" → narration generates

---

## US-10: Personas Page (`/personas`)

**As a user**, I want to browse configured personas and see their properties.

### Current Features
- [x] Persona cards with name, description
- [x] Persona detail page with full configuration
- [x] Search filter

### Missing
- [ ] **"See runs" link** — No link to filtered runs for a persona
- [ ] **Persona usage stats** — No display of how many runs used each persona

### E2E Test Cases
- `personas_list_loads`: Page loads with persona cards
- `personas_search`: Search filters cards
- `personas_detail`: Click persona → detail page

---

## US-11: Contracts Page (`/contracts`)

**As a user**, I want to browse pipeline contracts (test/validation requirements).

### Current Features
- [x] Contract cards with name, description
- [x] Contract detail page with test commands, expected outcomes

### Missing
- [ ] **Contract status** — No pass/fail rate display
- [ ] **Link to pipeline** — No link from contract to pipelines that enforce it

### E2E Test Cases
- `contracts_list_loads`: Page loads with contract cards
- `contracts_detail`: Click contract → detail page

---

## US-12: Skills Page (`/skills`)

**As a user**, I want to browse and install skills.

### Current Features
- [x] Skill list with name, description, source
- [x] Install button per skill

### Missing
- [ ] **Skill detail page** — No dedicated detail view
- [ ] **Installed indicator** — No badge showing if a skill is already installed

### E2E Test Cases
- `skills_list_loads`: Page loads with skill list
- `skills_install`: Click install → success feedback

---

## US-13: Webhooks Page (`/webhooks`)

**As a user**, I want to manage webhook integrations for pipeline events.

### Current Features
- [x] Webhook list with name, URL, events, status
- [x] Create/edit/delete webhooks
- [x] Enable/disable toggle
- [x] Test webhook delivery
- [x] Webhook detail page with delivery history

### E2E Test Cases
- `webhooks_list_loads`: Page loads with webhook table
- `webhooks_create`: Fill form → save → webhook appears in list
- `webhooks_edit`: Click edit → form pre-fills → save → updates
- `webhooks_delete`: Click delete → confirm → webhook removed
- `webhooks_toggle`: Click enable/disable → status changes
- `webhooks_test`: Click test → delivery appears in detail
- `webhooks_detail`: Click webhook → delivery history table

---

## US-14: Health Page (`/health`)

**As a user**, I want to see the health status of Wave infrastructure.

### Current Features
- [x] Health check list with pass/fail status
- [x] Check details on failure

### Missing
- [ ] **Auto-refresh** — No periodic refresh for running health checks
- [ ] **"Fix" action** — No suggested remediation for failed checks

### E2E Test Cases
- `health_loads`: Page loads with health checks
- `health_passing`: All checks show green/passed
- `health_failing`: Failed checks show red/failed with details

---

## US-15: Ontology Page (`/ontology`)

**As a user**, I want to see the bounded context model and domain relationships.

### Current Features
- [x] Bounded context stats (entities, relationships)
- [x] Context list with descriptions

### E2E Test Cases
- `ontology_loads`: Page loads with context data

---

## US-16: Admin Page (`/admin`)

**As a user**, I want to manage server configuration and monitor the system.

### Current Features
- [x] Server configuration table
- [x] Adapter binary paths
- [x] Credential status checks
- [x] Emergency stop button (cancel all running pipelines)
- [x] Audit log (recent lifecycle events)
- [x] Enable/disable pipelines

### Missing
- [ ] **Pipeline enable/disable bulk actions** — Only one at a time
- [ ] **Config editing** — View-only, no inline editing

### E2E Test Cases
- `admin_loads`: Page loads with all sections
- `admin_config`: Server config table renders
- `admin_credentials`: Credential checks display
- `admin_emergency_stop`: Click stop → confirms → cancels running runs
- `admin_audit`: Audit log shows recent events
- `admin_disable_pipeline`: Disable a pipeline → re-enable it

---

## US-17: Compare Page (`/compare`)

**As a user**, I want to compare two pipeline runs side by side.

### Current Features
- [x] Run selector (two dropdowns)
- [x] Side-by-side step comparison
- [x] Duration/token comparison

### E2E Test Cases
- `compare_loads`: Page loads with selectors
- `compare_select_runs`: Select two runs → comparison table renders

---

## Cross-Cutting Concerns

### Sidebar Navigation
- [x] 5 collapsible groups with icons
- [x] Active page highlighted
- [x] Group collapse state persisted
- [x] Desktop: full sidebar, Tablet: icons only, Mobile: overlay
- [x] Theme toggle in footer
- [x] Keyboard shortcuts (? for help, g+r, g+p, g+h, t)

### Responsive Design
- [x] Desktop (>1024px): Full layout
- [x] Tablet (768-1024px): Collapsed sidebar
- [x] Mobile (<768px): Overlay sidebar + topbar
- [ ] **Table horizontal scroll** — Some tables overflow on mobile without scroll indicators
- [ ] **Dialog sizing** — Some dialogs may overflow on small screens

### Theme
- [x] Dark/light theme toggle
- [x] Theme preference persisted in localStorage
- [x] All pages respect theme
- [ ] **System preference detection** — No `prefers-color-scheme` auto-detection

### Error States
- [ ] **Network error handling** — No global error boundary or offline indicator
- [ ] **API error toast** — Some API failures show no feedback
- [ ] **404 handling** — Custom 404 page exists but could be more helpful

---

## Priority Matrix

| Priority | User Story | Rationale |
|----------|-----------|-----------|
| P0 | US-1: Runs list | Most visited page, start form missing critical CLI parity |
| P0 | US-2: Run detail | Most visited page, log overflow is known pain point |
| P1 | US-6: Issues | "Implement" button is core workflow |
| P1 | US-3: Pipelines | Pipeline browsing is high-frequency |
| P2 | US-7: PRs | Review workflow |
| P2 | US-13: Webhooks | Configuration management |
| P2 | US-16: Admin | System management |
| P3 | US-4: Pipeline detail | Secondary navigation |
| P3 | US-5: Compose | Advanced workflow |
| P3 | US-10: Personas | Rarely visited |
| P3 | US-11: Contracts | Rarely visited |
| P3 | US-12: Skills | Rarely visited |
| P3 | US-8: Analytics | Insights |
| P3 | US-9: Retros | Insights |
| P3 | US-14: Health | Infra |
| P3 | US-15: Ontology | Infra |
| P3 | US-17: Compare | Advanced |
