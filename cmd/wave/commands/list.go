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
	showAdapters := filter == "adapters"
	showRuns := filter == "runs"

	// Handle runs filter separately (redirect to runListRuns)
	if showRuns {
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

		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Table format (default)
	if showPipelines {
		if err := listPipelines(); err != nil {
			return err
		}
		if showAll {
			fmt.Println()
		}
	}

	// Load manifest for personas/adapters
	manifestData, err := os.ReadFile(opts.Manifest)
	if err != nil && (showPersonas || showAdapters) {
		fmt.Printf("(manifest not found: %s)\n", opts.Manifest)
		return nil
	}

	var m manifestData2
	if err == nil {
		yaml.Unmarshal(manifestData, &m)
	}

	if showPersonas {
		listPersonas(m.Personas)
		if showAll && showAdapters {
			fmt.Println()
		}
	}

	if showAdapters {
		listAdapters(m.Adapters)
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

func listPipelines() error {
	pipelineDir := ".wave/pipelines"
	entries, err := os.ReadDir(pipelineDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read pipelines directory: %w", err)
	}

	fmt.Printf("Pipelines:\n")
	if len(entries) == 0 {
		fmt.Printf("  (none found in %s/)\n", pipelineDir)
		return nil
	}

	// Sort entries by name
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
			fmt.Printf("  %-20s (error reading)\n", name)
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
			fmt.Printf("  %-20s (error parsing)\n", name)
			continue
		}

		desc := p.Metadata.Description
		if desc == "" {
			desc = "(no description)"
		}

		stepIDs := []string{}
		for _, s := range p.Steps {
			stepIDs = append(stepIDs, s.ID)
		}
		fmt.Printf("  %-20s %d steps  %s\n", name, len(p.Steps), desc)
		fmt.Printf("  %-20s steps: %s\n", "", strings.Join(stepIDs, " → "))
	}

	return nil
}

func listPersonas(personas map[string]struct {
	Adapter          string  `yaml:"adapter"`
	Description      string  `yaml:"description"`
	SystemPromptFile string  `yaml:"system_prompt_file"`
	Temperature      float64 `yaml:"temperature"`
	Permissions      struct {
		AllowedTools []string `yaml:"allowed_tools"`
		Deny         []string `yaml:"deny"`
	} `yaml:"permissions"`
}) {
	fmt.Printf("Personas:\n")
	if len(personas) == 0 {
		fmt.Printf("  (none defined)\n")
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
		desc := persona.Description
		if desc == "" {
			desc = "(no description)"
		}
		// T089: Add permission summary
		permSummary := formatPermissionSummary(
			persona.Permissions.AllowedTools,
			persona.Permissions.Deny,
		)
		fmt.Printf("  %-20s adapter:%-10s temp:%.1f  %s\n",
			name,
			persona.Adapter,
			persona.Temperature,
			permSummary,
		)
		fmt.Printf("  %-20s %s\n", "", desc)
	}
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

// listAdapters lists all configured adapters with binary availability check.
func listAdapters(adapters map[string]struct {
	Binary       string `yaml:"binary"`
	Mode         string `yaml:"mode"`
	OutputFormat string `yaml:"output_format"`
}) {
	fmt.Printf("Adapters:\n")
	if len(adapters) == 0 {
		fmt.Printf("  (none defined)\n")
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
		// T087: Check binary availability
		available := "OK"
		if _, err := exec.LookPath(adapter.Binary); err != nil {
			available = "[X] not found"
		}
		fmt.Printf("  %-20s binary:%-10s mode:%-10s format:%-6s %s\n",
			name, adapter.Binary, adapter.Mode, adapter.OutputFormat, available)
	}
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
	listRuns(runs)
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
		name    string
		modTime time.Time
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

		workspaces = append(workspaces, wsInfo{
			name:    entry.Name(),
			modTime: info.ModTime(),
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
		// Infer status from workspace name or default to "unknown"
		status := "unknown"
		if opts.Status != "" && strings.ToLower(status) != strings.ToLower(opts.Status) {
			continue
		}

		runs = append(runs, RunInfo{
			RunID:     ws.name,
			Pipeline:  ws.name,
			Status:    status,
			StartedAt: ws.modTime.Format("2006-01-02 15:04:05"),
			Duration:  "-",
		})
	}

	return runs, nil
}

// listRuns displays run information in table format
func listRuns(runs []RunInfo) {
	fmt.Printf("Recent Pipeline Runs:\n")
	if len(runs) == 0 {
		fmt.Printf("  (no runs found)\n")
		return
	}

	// Print header
	fmt.Printf("  %-36s %-20s %-12s %-20s %s\n",
		"RUN_ID", "PIPELINE", "STATUS", "STARTED", "DURATION")
	fmt.Printf("  %s %s %s %s %s\n",
		strings.Repeat("-", 36),
		strings.Repeat("-", 20),
		strings.Repeat("-", 12),
		strings.Repeat("-", 20),
		strings.Repeat("-", 15))

	for _, run := range runs {
		// Truncate long run IDs
		runID := run.RunID
		if len(runID) > 36 {
			runID = runID[:33] + "..."
		}

		// Truncate long pipeline names
		pipeline := run.Pipeline
		if len(pipeline) > 20 {
			pipeline = pipeline[:17] + "..."
		}

		// Format status with color hints (for terminal)
		status := run.Status
		if len(status) > 12 {
			status = status[:9] + "..."
		}

		fmt.Printf("  %-36s %-20s %-12s %-20s %s\n",
			runID, pipeline, status, run.StartedAt, run.Duration)
	}
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
