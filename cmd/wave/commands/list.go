package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/recinq/wave/internal/display"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	_ "modernc.org/sqlite"
)

// JSON output structures
type PipelineInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	StepCount   int      `json:"step_count"`
	Steps       []string `json:"steps"`
}

type PersonaInfo struct {
	Name         string   `json:"name"`
	Adapter      string   `json:"adapter"`
	Description  string   `json:"description"`
	Temperature  float64  `json:"temperature"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
	DeniedTools  []string `json:"denied_tools,omitempty"`
}

type AdapterInfo struct {
	Name         string `json:"name"`
	Binary       string `json:"binary"`
	Mode         string `json:"mode"`
	OutputFormat string `json:"output_format"`
	Available    bool   `json:"available"`
}

// RunInfo holds information about a pipeline run
type RunInfo struct {
	RunID      string `json:"run_id"`
	Pipeline   string `json:"pipeline"`
	Status     string `json:"status"`
	StartedAt  string `json:"started_at"`
	Duration   string `json:"duration"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

type ListOutput struct {
	Pipelines []PipelineInfo `json:"pipelines,omitempty"`
	Personas  []PersonaInfo  `json:"personas,omitempty"`
	Adapters  []AdapterInfo  `json:"adapters,omitempty"`
	Runs      []RunInfo      `json:"runs,omitempty"`
}

type ListOptions struct {
	Manifest string
	Format   string
}

// ListRunsOptions holds options for the list runs subcommand
type ListRunsOptions struct {
	Limit    int
	Pipeline string
	Status   string
	Format   string
}

// ListRunsFlags holds flags specific to the runs subcommand that can be set on main list command
var listRunsLimit int
var listRunsPipeline string
var listRunsStatus string

// printLogo prints the Wave ASCII logo header
func printLogo() {
	f := display.NewFormatter()
	logo := []string{
		"╦ ╦╔═╗╦  ╦╔═╗",
		"║║║╠═╣╚╗╔╝║╣",
		"╚╩╝╩ ╩ ╚╝ ╚═╝",
	}
	fmt.Println()
	for _, line := range logo {
		fmt.Printf("  %s\n", f.Primary(line))
	}
}

func NewListCmd() *cobra.Command {
	var opts ListOptions

	cmd := &cobra.Command{
		Use:   "list [pipelines|personas|adapters|runs]",
		Short: "List pipelines and personas",
		Long: `List available pipelines, personas, and their configurations.
Shows pipeline steps, persona bindings, and execution status.

Arguments:
  pipelines   List available pipelines
  personas    List configured personas
  adapters    List configured adapters
  runs        List recent pipeline executions

With no arguments, lists pipelines and personas.

For 'list runs', additional flags are available:
  --limit N           Maximum number of runs to show (default 10)
  --run-pipeline P    Filter to specific pipeline
  --run-status S      Filter by status (running, completed, failed, cancelled)`,
		ValidArgs: []string{"pipelines", "personas", "adapters", "runs"},
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := ""
			if len(args) > 0 {
				filter = args[0]
			}
			return runList(opts, filter)
		},
	}

	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().StringVar(&opts.Format, "format", "table", "Output format (table, json)")

	// Flags for 'list runs' (only used when filter is "runs")
	cmd.Flags().IntVar(&listRunsLimit, "limit", 10, "Maximum number of runs to show (for 'list runs')")
	cmd.Flags().StringVar(&listRunsPipeline, "run-pipeline", "", "Filter to specific pipeline (for 'list runs')")
	cmd.Flags().StringVar(&listRunsStatus, "run-status", "", "Filter by status (for 'list runs')")

	return cmd
}

func runList(opts ListOptions, filter string) error {
	showAll := filter == ""
	showPipelines := showAll || filter == "pipelines"
	showPersonas := showAll || filter == "personas"
	showAdapters := showAll || filter == "adapters"
	showRuns := showAll || filter == "runs"

	// Handle runs-only filter separately (redirect to runListRuns which prints its own logo)
	if filter == "runs" {
		return runListRuns(ListRunsOptions{
			Limit:    listRunsLimit,
			Pipeline: listRunsPipeline,
			Status:   listRunsStatus,
			Format:   opts.Format,
		})
	}

	// For JSON output, collect all data first
	if opts.Format == "json" {
		output := ListOutput{}

		if showPipelines {
			pipelines, err := collectPipelines()
			if err != nil {
				return err
			}
			output.Pipelines = pipelines
		}

		// Load manifest for personas/adapters
		manifestData, err := os.ReadFile(opts.Manifest)
		if err == nil {
			var m manifestData2
			yaml.Unmarshal(manifestData, &m)

			if showPersonas {
				output.Personas = collectPersonas(m.Personas)
			}
			if showAdapters {
				output.Adapters = collectAdapters(m.Adapters)
			}
		}

		if showRuns {
			runs, err := collectRuns(ListRunsOptions{
				Limit:    listRunsLimit,
				Pipeline: listRunsPipeline,
				Status:   listRunsStatus,
			})
			if err == nil {
				output.Runs = runs
			}
		}

		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Table format (default) - print logo once
	printLogo()

	// Load manifest for adapters/personas
	manifestData, err := os.ReadFile(opts.Manifest)
	if err != nil && (showPersonas || showAdapters) {
		fmt.Printf("(manifest not found: %s)\n", opts.Manifest)
		return nil
	}

	var m manifestData2
	if err == nil {
		yaml.Unmarshal(manifestData, &m)
	}

	// Order: adapters, pipelines, personas, runs
	if showAdapters {
		listAdaptersTable(m.Adapters)
		if showAll {
			fmt.Println()
		}
	}

	if showPipelines {
		if err := listPipelinesTable(); err != nil {
			return err
		}
		if showAll {
			fmt.Println()
		}
	}

	if showPersonas {
		listPersonasTable(m.Personas)
		if showAll {
			fmt.Println()
		}
	}

	if showRuns {
		runs, err := collectRuns(ListRunsOptions{
			Limit:    listRunsLimit,
			Pipeline: listRunsPipeline,
			Status:   listRunsStatus,
		})
		if err == nil {
			listRunsTable(runs)
		}
	}

	return nil
}

type manifestData2 struct {
	Adapters map[string]struct {
		Binary       string `yaml:"binary"`
		Mode         string `yaml:"mode"`
		OutputFormat string `yaml:"output_format"`
	} `yaml:"adapters"`
	Personas map[string]struct {
		Adapter          string  `yaml:"adapter"`
		Description      string  `yaml:"description"`
		SystemPromptFile string  `yaml:"system_prompt_file"`
		Temperature      float64 `yaml:"temperature"`
		Permissions      struct {
			AllowedTools []string `yaml:"allowed_tools"`
			Deny         []string `yaml:"deny"`
		} `yaml:"permissions"`
	} `yaml:"personas"`
}

func collectPipelines() ([]PipelineInfo, error) {
	pipelineDir := ".wave/pipelines"
	entries, err := os.ReadDir(pipelineDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read pipelines directory: %w", err)
	}

	var pipelines []PipelineInfo
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		pipelinePath := filepath.Join(pipelineDir, entry.Name())

		data, err := os.ReadFile(pipelinePath)
		if err != nil {
			continue
		}

		var p struct {
			Metadata struct {
				Description string `yaml:"description"`
			} `yaml:"metadata"`
			Steps []struct {
				ID string `yaml:"id"`
			} `yaml:"steps"`
		}
		if err := yaml.Unmarshal(data, &p); err != nil {
			continue
		}

		stepIDs := []string{}
		for _, s := range p.Steps {
			stepIDs = append(stepIDs, s.ID)
		}

		pipelines = append(pipelines, PipelineInfo{
			Name:        name,
			Description: p.Metadata.Description,
			StepCount:   len(p.Steps),
			Steps:       stepIDs,
		})
	}

	return pipelines, nil
}

func collectPersonas(personas map[string]struct {
	Adapter          string  `yaml:"adapter"`
	Description      string  `yaml:"description"`
	SystemPromptFile string  `yaml:"system_prompt_file"`
	Temperature      float64 `yaml:"temperature"`
	Permissions      struct {
		AllowedTools []string `yaml:"allowed_tools"`
		Deny         []string `yaml:"deny"`
	} `yaml:"permissions"`
}) []PersonaInfo {
	var result []PersonaInfo

	names := make([]string, 0, len(personas))
	for name := range personas {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		persona := personas[name]
		result = append(result, PersonaInfo{
			Name:         name,
			Adapter:      persona.Adapter,
			Description:  persona.Description,
			Temperature:  persona.Temperature,
			AllowedTools: persona.Permissions.AllowedTools,
			DeniedTools:  persona.Permissions.Deny,
		})
	}

	return result
}

func collectAdapters(adapters map[string]struct {
	Binary       string `yaml:"binary"`
	Mode         string `yaml:"mode"`
	OutputFormat string `yaml:"output_format"`
}) []AdapterInfo {
	var result []AdapterInfo

	names := make([]string, 0, len(adapters))
	for name := range adapters {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		adapter := adapters[name]
		available := true
		if _, err := exec.LookPath(adapter.Binary); err != nil {
			available = false
		}
		result = append(result, AdapterInfo{
			Name:         name,
			Binary:       adapter.Binary,
			Mode:         adapter.Mode,
			OutputFormat: adapter.OutputFormat,
			Available:    available,
		})
	}

	return result
}

func listPipelinesTable() error {
	pipelineDir := ".wave/pipelines"
	entries, err := os.ReadDir(pipelineDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read pipelines directory: %w", err)
	}

	f := display.NewFormatter()

	// Header
	fmt.Println()
	fmt.Printf("%s\n", f.Bold("Pipelines"))
	fmt.Printf("%s\n", f.Muted(strings.Repeat("─", 60)))

	if len(entries) == 0 {
		fmt.Printf("  %s\n", f.Muted("(none found in "+pipelineDir+"/)"))
		return nil
	}

	// Sort entries by name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// Collect valid pipelines
	type pipelineEntry struct {
		name        string
		description string
		steps       []string
	}
	var pipelines []pipelineEntry

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		pipelinePath := filepath.Join(pipelineDir, entry.Name())

		data, err := os.ReadFile(pipelinePath)
		if err != nil {
			pipelines = append(pipelines, pipelineEntry{name: name, description: "(error reading)"})
			continue
		}

		var p struct {
			Metadata struct {
				Description string `yaml:"description"`
			} `yaml:"metadata"`
			Steps []struct {
				ID      string `yaml:"id"`
				Persona string `yaml:"persona"`
			} `yaml:"steps"`
		}
		if err := yaml.Unmarshal(data, &p); err != nil {
			pipelines = append(pipelines, pipelineEntry{name: name, description: "(error parsing)"})
			continue
		}

		stepIDs := []string{}
		for _, s := range p.Steps {
			stepIDs = append(stepIDs, s.ID)
		}

		pipelines = append(pipelines, pipelineEntry{
			name:        name,
			description: p.Metadata.Description,
			steps:       stepIDs,
		})
	}

	// Render each pipeline
	for _, p := range pipelines {
		// Pipeline name with step count badge
		stepBadge := f.Muted(fmt.Sprintf("[%d steps]", len(p.steps)))
		fmt.Printf("\n  %s %s\n", f.Primary(p.name), stepBadge)

		// Description
		if p.description != "" {
			fmt.Printf("    %s\n", f.Muted(p.description))
		}

		// Steps flow
		if len(p.steps) > 0 {
			stepsFlow := formatStepsFlow(p.steps, f)
			fmt.Printf("    %s\n", stepsFlow)
		}
	}

	fmt.Println()
	return nil
}

// formatStepsFlow formats pipeline steps as a visual flow with arrows
func formatStepsFlow(steps []string, f *display.Formatter) string {
	if len(steps) == 0 {
		return ""
	}

	var parts []string
	for i, step := range steps {
		if i == 0 {
			parts = append(parts, f.Success("○")+f.Muted(" "+step))
		} else {
			parts = append(parts, f.Muted("→ "+step))
		}
	}

	return strings.Join(parts, " ")
}

func listPersonasTable(personas map[string]struct {
	Adapter          string  `yaml:"adapter"`
	Description      string  `yaml:"description"`
	SystemPromptFile string  `yaml:"system_prompt_file"`
	Temperature      float64 `yaml:"temperature"`
	Permissions      struct {
		AllowedTools []string `yaml:"allowed_tools"`
		Deny         []string `yaml:"deny"`
	} `yaml:"permissions"`
}) {
	f := display.NewFormatter()

	// Header
	fmt.Println()
	fmt.Printf("%s\n", f.Bold("Personas"))
	fmt.Printf("%s\n", f.Muted(strings.Repeat("─", 60)))

	if len(personas) == 0 {
		fmt.Printf("  %s\n", f.Muted("(none defined)"))
		return
	}

	// Sort by name for stable output
	names := make([]string, 0, len(personas))
	for name := range personas {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		persona := personas[name]

		// Persona name
		fmt.Printf("\n  %s\n", f.Primary(name))

		// Metadata line: adapter, temperature, permissions
		metaParts := []string{}
		metaParts = append(metaParts, fmt.Sprintf("adapter: %s", persona.Adapter))
		metaParts = append(metaParts, fmt.Sprintf("temp: %.1f", persona.Temperature))

		// Permission summary
		permSummary := formatPermissionSummary(
			persona.Permissions.AllowedTools,
			persona.Permissions.Deny,
		)
		if permSummary != "" {
			metaParts = append(metaParts, permSummary)
		}

		fmt.Printf("    %s\n", f.Muted(strings.Join(metaParts, " • ")))

		// Description
		if persona.Description != "" {
			fmt.Printf("    %s\n", persona.Description)
		}
	}

	fmt.Println()
}

// formatPermissionSummary creates a concise summary of persona permissions.
func formatPermissionSummary(allowed []string, denied []string) string {
	allowCount := len(allowed)
	denyCount := len(denied)

	if allowCount == 0 && denyCount == 0 {
		return "tools:(default)"
	}

	parts := []string{}
	if allowCount > 0 {
		parts = append(parts, fmt.Sprintf("allow:%d", allowCount))
	}
	if denyCount > 0 {
		parts = append(parts, fmt.Sprintf("deny:%d", denyCount))
	}

	return strings.Join(parts, " ")
}

// listAdaptersTable lists all configured adapters with binary availability check.
func listAdaptersTable(adapters map[string]struct {
	Binary       string `yaml:"binary"`
	Mode         string `yaml:"mode"`
	OutputFormat string `yaml:"output_format"`
}) {
	f := display.NewFormatter()

	// Header
	fmt.Println()
	fmt.Printf("%s\n", f.Bold("Adapters"))
	fmt.Printf("%s\n", f.Muted(strings.Repeat("─", 60)))

	if len(adapters) == 0 {
		fmt.Printf("  %s\n", f.Muted("(none defined)"))
		return
	}

	// Sort by name for stable output
	names := make([]string, 0, len(adapters))
	for name := range adapters {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		adapter := adapters[name]

		// Check binary availability
		available := true
		if _, err := exec.LookPath(adapter.Binary); err != nil {
			available = false
		}

		// Status icon
		var statusIcon string
		if available {
			statusIcon = f.Success("✓")
		} else {
			statusIcon = f.Error("✗")
		}

		// Adapter name with status
		fmt.Printf("\n  %s %s\n", statusIcon, f.Primary(name))

		// Metadata
		metaParts := []string{
			fmt.Sprintf("binary: %s", adapter.Binary),
			fmt.Sprintf("mode: %s", adapter.Mode),
			fmt.Sprintf("format: %s", adapter.OutputFormat),
		}
		fmt.Printf("    %s\n", f.Muted(strings.Join(metaParts, " • ")))

		if !available {
			fmt.Printf("    %s\n", f.Error("binary not found in PATH"))
		}
	}

	fmt.Println()
}

// runListRuns executes the 'list runs' subcommand
func runListRuns(opts ListRunsOptions) error {
	runs, err := collectRuns(opts)
	if err != nil {
		return err
	}

	if opts.Format == "json" {
		output := ListOutput{Runs: runs}
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Table format
	printLogo()
	listRunsTable(runs)
	return nil
}

// collectRuns collects run information from the state database or workspace metadata
func collectRuns(opts ListRunsOptions) ([]RunInfo, error) {
	var runs []RunInfo

	// First try to read from the state database
	dbPath := ".wave/state.db"
	if _, err := os.Stat(dbPath); err == nil {
		dbRuns, err := collectRunsFromDB(dbPath, opts)
		if err == nil && len(dbRuns) > 0 {
			return dbRuns, nil
		}
		// Fall through to workspace fallback if DB query failed or returned no results
	}

	// Fallback: read from workspace directory metadata
	runs, err := collectRunsFromWorkspaces(opts)
	if err != nil {
		return nil, err
	}

	return runs, nil
}

// collectRunsFromDB reads run information from the state database
func collectRunsFromDB(dbPath string, opts ListRunsOptions) ([]RunInfo, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Try pipeline_run table first (new schema from spec 016)
	query := `
		SELECT run_id, pipeline_name, status, started_at, completed_at
		FROM pipeline_run
		WHERE 1=1
	`
	args := []interface{}{}

	if opts.Pipeline != "" {
		query += " AND pipeline_name = ?"
		args = append(args, opts.Pipeline)
	}

	if opts.Status != "" {
		query += " AND LOWER(status) = LOWER(?)"
		args = append(args, opts.Status)
	}

	query += " ORDER BY started_at DESC LIMIT ?"
	args = append(args, opts.Limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		// Fallback to pipeline_state table (old schema)
		return collectRunsFromPipelineState(db, opts)
	}
	defer rows.Close()

	var runs []RunInfo
	for rows.Next() {
		var runID, pipelineName, status string
		var startedAt int64
		var completedAt sql.NullInt64

		if err := rows.Scan(&runID, &pipelineName, &status, &startedAt, &completedAt); err != nil {
			continue
		}

		startTime := time.Unix(startedAt, 0)
		var duration string
		var durationMs int64

		if completedAt.Valid {
			endTime := time.Unix(completedAt.Int64, 0)
			durationMs = endTime.Sub(startTime).Milliseconds()
			duration = formatDuration(endTime.Sub(startTime))
		} else if strings.ToLower(status) == "running" {
			durationMs = time.Since(startTime).Milliseconds()
			duration = formatDuration(time.Since(startTime)) + " (running)"
		} else {
			duration = "-"
		}

		runs = append(runs, RunInfo{
			RunID:      runID,
			Pipeline:   pipelineName,
			Status:     status,
			StartedAt:  startTime.Format("2006-01-02 15:04:05"),
			Duration:   duration,
			DurationMs: durationMs,
		})
	}

	return runs, nil
}

// collectRunsFromPipelineState reads from the legacy pipeline_state table
func collectRunsFromPipelineState(db *sql.DB, opts ListRunsOptions) ([]RunInfo, error) {
	query := `
		SELECT pipeline_id, pipeline_name, status, created_at, updated_at
		FROM pipeline_state
		WHERE 1=1
	`
	args := []interface{}{}

	if opts.Pipeline != "" {
		query += " AND pipeline_name = ?"
		args = append(args, opts.Pipeline)
	}

	if opts.Status != "" {
		query += " AND LOWER(status) = LOWER(?)"
		args = append(args, opts.Status)
	}

	query += " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, opts.Limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []RunInfo
	for rows.Next() {
		var pipelineID, pipelineName, status string
		var createdAt, updatedAt int64

		if err := rows.Scan(&pipelineID, &pipelineName, &status, &createdAt, &updatedAt); err != nil {
			continue
		}

		startTime := time.Unix(createdAt, 0)
		endTime := time.Unix(updatedAt, 0)

		var duration string
		var durationMs int64
		if createdAt != updatedAt {
			durationMs = endTime.Sub(startTime).Milliseconds()
			duration = formatDuration(endTime.Sub(startTime))
		} else {
			duration = "-"
		}

		runs = append(runs, RunInfo{
			RunID:      pipelineID,
			Pipeline:   pipelineName,
			Status:     status,
			StartedAt:  startTime.Format("2006-01-02 15:04:05"),
			Duration:   duration,
			DurationMs: durationMs,
		})
	}

	return runs, nil
}

// collectRunsFromWorkspaces reads run information from workspace directory metadata
func collectRunsFromWorkspaces(opts ListRunsOptions) ([]RunInfo, error) {
	wsDir := ".wave/workspaces"
	entries, err := os.ReadDir(wsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	type wsInfo struct {
		name      string
		modTime   time.Time
		startTime time.Time
	}

	var workspaces []wsInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Apply pipeline filter
		if opts.Pipeline != "" && entry.Name() != opts.Pipeline {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Get creation time (start time) from directory
		wsPath := filepath.Join(wsDir, entry.Name())
		startTime := getDirectoryCreationTime(wsPath)
		if startTime.IsZero() {
			startTime = info.ModTime() // Fallback to mod time
		}

		workspaces = append(workspaces, wsInfo{
			name:      entry.Name(),
			modTime:   info.ModTime(),
			startTime: startTime,
		})
	}

	// Sort by modification time (most recent first)
	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].modTime.After(workspaces[j].modTime)
	})

	// Apply limit
	if opts.Limit > 0 && len(workspaces) > opts.Limit {
		workspaces = workspaces[:opts.Limit]
	}

	var runs []RunInfo
	for _, ws := range workspaces {
		// Infer status from workspace contents
		wsPath := filepath.Join(wsDir, ws.name)
		status, endTime := inferWorkspaceStatus(wsPath, ws.name)

		// Apply status filter
		if opts.Status != "" && strings.ToLower(status) != strings.ToLower(opts.Status) {
			continue
		}

		// Calculate duration
		var duration string
		var durationMs int64
		if !endTime.IsZero() && !ws.startTime.IsZero() {
			d := endTime.Sub(ws.startTime)
			durationMs = d.Milliseconds()
			duration = formatDuration(d)
		} else {
			duration = "-"
		}

		runs = append(runs, RunInfo{
			RunID:      ws.name,
			Pipeline:   ws.name,
			Status:     status,
			StartedAt:  ws.startTime.Format("2006-01-02 15:04:05"),
			Duration:   duration,
			DurationMs: durationMs,
		})
	}

	return runs, nil
}

// inferWorkspaceStatus determines the status of a run by examining its workspace
func inferWorkspaceStatus(wsPath string, pipelineName string) (status string, endTime time.Time) {
	// Try to find and load the pipeline definition to know expected steps
	pipelinePath := ".wave/pipelines/" + pipelineName + ".yaml"
	pipelineData, err := os.ReadFile(pipelinePath)
	if err != nil {
		// Can't determine expected steps, check if any step dirs exist
		stepDirs, _ := os.ReadDir(wsPath)
		if len(stepDirs) == 0 {
			return "pending", time.Time{}
		}
		// Has step dirs but can't verify completion
		return "unknown", getLatestFileTime(wsPath)
	}

	// Parse pipeline to get expected steps
	var p struct {
		Steps []struct {
			ID string `yaml:"id"`
		} `yaml:"steps"`
	}
	if err := yaml.Unmarshal(pipelineData, &p); err != nil {
		return "unknown", getLatestFileTime(wsPath)
	}

	expectedSteps := make(map[string]bool)
	for _, step := range p.Steps {
		expectedSteps[step.ID] = false
	}

	// Check which steps have directories in the workspace
	stepDirs, err := os.ReadDir(wsPath)
	if err != nil {
		return "unknown", time.Time{}
	}

	completedSteps := 0
	var latestTime time.Time
	for _, dir := range stepDirs {
		if !dir.IsDir() {
			continue
		}
		stepID := dir.Name()
		if _, expected := expectedSteps[stepID]; expected {
			// Check if step has output (indicates completion)
			stepPath := filepath.Join(wsPath, stepID)
			if hasStepOutput(stepPath) {
				expectedSteps[stepID] = true
				completedSteps++
			}
			// Track latest modification time
			if info, err := dir.Info(); err == nil {
				if info.ModTime().After(latestTime) {
					latestTime = info.ModTime()
				}
			}
		}
	}

	// Determine overall status
	if completedSteps == 0 {
		return "pending", time.Time{}
	}
	if completedSteps == len(expectedSteps) {
		return "completed", latestTime
	}
	// Some steps completed but not all - could be running or failed
	return "partial", latestTime
}

// hasStepOutput checks if a step directory contains output files
func hasStepOutput(stepPath string) bool {
	// Check for common output locations
	outputDirs := []string{"", "output", "artifacts"}
	for _, subdir := range outputDirs {
		checkPath := stepPath
		if subdir != "" {
			checkPath = filepath.Join(stepPath, subdir)
		}
		entries, err := os.ReadDir(checkPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				// Found at least one file
				return true
			}
		}
	}
	return false
}

// getLatestFileTime finds the most recent modification time in a directory tree
func getLatestFileTime(dirPath string) time.Time {
	var latest time.Time
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return latest
}

// getDirectoryCreationTime gets the creation time of a directory (best effort)
func getDirectoryCreationTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	// On most systems, we can only reliably get ModTime
	// But we can approximate creation time by finding the oldest file
	var oldest time.Time
	filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if oldest.IsZero() || fi.ModTime().Before(oldest) {
			oldest = fi.ModTime()
		}
		return nil
	})
	// Use the directory's own mod time as a fallback
	if oldest.IsZero() {
		return info.ModTime()
	}
	return oldest
}

// listRunsTable displays run information in table format
func listRunsTable(runs []RunInfo) {
	f := display.NewFormatter()

	// Header
	fmt.Println()
	fmt.Printf("%s\n", f.Bold("Recent Pipeline Runs"))
	fmt.Printf("%s\n", f.Muted(strings.Repeat("─", 80)))

	if len(runs) == 0 {
		fmt.Printf("  %s\n\n", f.Muted("(no runs found)"))
		return
	}

	// Table header
	fmt.Printf("  %s  %s  %s  %s  %s\n",
		f.Muted(fmt.Sprintf("%-24s", "RUN_ID")),
		f.Muted(fmt.Sprintf("%-16s", "PIPELINE")),
		f.Muted(fmt.Sprintf("%-12s", "STATUS")),
		f.Muted(fmt.Sprintf("%-18s", "STARTED")),
		f.Muted("DURATION"),
	)

	for _, run := range runs {
		// Truncate long run IDs
		runID := run.RunID
		if len(runID) > 24 {
			runID = runID[:21] + "..."
		}

		// Truncate long pipeline names
		pipeline := run.Pipeline
		if len(pipeline) > 16 {
			pipeline = pipeline[:13] + "..."
		}

		// Format status with color
		status := run.Status
		var statusStr string
		switch strings.ToLower(status) {
		case "completed":
			statusStr = f.Success(fmt.Sprintf("%-12s", status))
		case "failed":
			statusStr = f.Error(fmt.Sprintf("%-12s", status))
		case "running":
			statusStr = f.Primary(fmt.Sprintf("%-12s", status))
		case "cancelled":
			statusStr = f.Warning(fmt.Sprintf("%-12s", status))
		default:
			statusStr = f.Muted(fmt.Sprintf("%-12s", status))
		}

		fmt.Printf("  %-24s  %-16s  %s  %-18s  %s\n",
			runID, pipeline, statusStr, run.StartedAt, f.Muted(run.Duration))
	}

	fmt.Println()
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, mins)
}
