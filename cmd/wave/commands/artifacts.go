package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// ArtifactsOptions holds options for the artifacts command
type ArtifactsOptions struct {
	RunID    string // Specific run (from args, default: most recent)
	Step     string // Filter by step ID
	Export   string // Export directory path
	Format   string // table, json
	Manifest string
}

// ArtifactOutput represents a single artifact for JSON output
type ArtifactOutput struct {
	Step   string `json:"step"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Size   int64  `json:"size_bytes"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

// ArtifactsOutput represents the JSON output for artifacts command
type ArtifactsOutput struct {
	RunID     string           `json:"run_id"`
	Artifacts []ArtifactOutput `json:"artifacts"`
}

// Common artifact file extensions and their types
var artifactPatterns = map[string]string{
	".md":   "markdown",
	".json": "json",
	".yaml": "yaml",
	".yml":  "yaml",
	".txt":  "text",
	".log":  "log",
}

// NewArtifactsCmd creates the artifacts command
func NewArtifactsCmd() *cobra.Command {
	var opts ArtifactsOptions

	cmd := &cobra.Command{
		Use:   "artifacts [run-id]",
		Short: "List and export pipeline artifacts",
		Long: `List artifacts from a pipeline run.

Without arguments, shows artifacts from the most recent run.
With a run-id argument, shows artifacts from that specific run.

Use --step to filter artifacts to a specific step.
Use --export to copy artifacts to a specified directory.
Use --format json for machine-readable output.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.RunID = args[0]
			}
			return runArtifacts(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Step, "step", "", "Filter to specific step ID")
	cmd.Flags().StringVar(&opts.Export, "export", "", "Export artifacts to specified directory")
	cmd.Flags().StringVar(&opts.Format, "format", "table", "Output format (table, json)")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")

	return cmd
}

func runArtifacts(opts ArtifactsOptions) error {
	// Collect artifacts (from state store or filesystem)
	artifacts, runID, err := collectArtifacts(opts)
	if err != nil {
		return err
	}

	// Apply step filter
	if opts.Step != "" {
		var filtered []ArtifactOutput
		for _, a := range artifacts {
			if a.Step == opts.Step {
				filtered = append(filtered, a)
			}
		}
		artifacts = filtered
	}

	// Handle export if requested
	if opts.Export != "" {
		return exportArtifacts(artifacts, opts.Export)
	}

	// Output based on format
	if opts.Format == "json" {
		return outputArtifactsJSON(runID, artifacts)
	}

	return outputArtifactsTable(runID, artifacts)
}

// collectArtifacts gathers artifacts from state store with filesystem fallback
func collectArtifacts(opts ArtifactsOptions) ([]ArtifactOutput, string, error) {
	var artifacts []ArtifactOutput
	var runID string

	// Try to use state store first
	dbPath := ".wave/state.db"
	if _, err := os.Stat(dbPath); err == nil {
		store, err := state.NewStateStore(dbPath)
		if err == nil {
			defer store.Close()

			// Get run ID if not specified
			if opts.RunID == "" {
				runs, err := store.ListRuns(state.ListRunsOptions{Limit: 1})
				if err == nil && len(runs) > 0 {
					runID = runs[0].RunID
				}
			} else {
				runID = opts.RunID
			}

			// Get artifacts from store
			if runID != "" {
				records, err := store.GetArtifacts(runID, opts.Step)
				if err == nil && len(records) > 0 {
					for _, r := range records {
						exists := true
						if _, err := os.Stat(r.Path); os.IsNotExist(err) {
							exists = false
						}
						artifacts = append(artifacts, ArtifactOutput{
							Step:   r.StepID,
							Name:   r.Name,
							Type:   r.Type,
							Size:   r.SizeBytes,
							Path:   r.Path,
							Exists: exists,
						})
					}
					return artifacts, runID, nil
				}
			}
		}
	}

	// Fallback: scan filesystem for artifacts
	return scanWorkspaceArtifacts(opts)
}

// scanWorkspaceArtifacts scans the workspace directories for artifacts
func scanWorkspaceArtifacts(opts ArtifactsOptions) ([]ArtifactOutput, string, error) {
	var artifacts []ArtifactOutput
	wsDir := ".wave/workspaces"

	// If no workspaces directory, return empty
	if _, err := os.Stat(wsDir); os.IsNotExist(err) {
		return artifacts, "", nil
	}

	// Get the most recent pipeline workspace
	pipelineDir := ""
	var mostRecentTime int64

	entries, err := os.ReadDir(wsDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read workspaces directory: %w", err)
	}

	// Find the most recent pipeline or specific run
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// If runID is specified, match against it
		if opts.RunID != "" {
			// RunID format is typically: pipeline-name-timestamp
			// or just match if it starts with the pipeline name
			if entry.Name() == opts.RunID || strings.HasPrefix(opts.RunID, entry.Name()) {
				pipelineDir = filepath.Join(wsDir, entry.Name())
				break
			}
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		modTime := info.ModTime().UnixNano()
		if modTime > mostRecentTime {
			mostRecentTime = modTime
			pipelineDir = filepath.Join(wsDir, entry.Name())
		}
	}

	if pipelineDir == "" {
		return artifacts, "", nil
	}

	runID := filepath.Base(pipelineDir)

	// Scan step directories within the pipeline
	stepEntries, err := os.ReadDir(pipelineDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read pipeline directory: %w", err)
	}

	for _, stepEntry := range stepEntries {
		if !stepEntry.IsDir() {
			continue
		}

		stepID := stepEntry.Name()

		// Apply step filter if specified
		if opts.Step != "" && stepID != opts.Step {
			continue
		}

		stepDir := filepath.Join(pipelineDir, stepID)
		stepArtifacts := scanDirectoryForArtifacts(stepDir, stepID)
		artifacts = append(artifacts, stepArtifacts...)
	}

	// Sort artifacts by step, then name
	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].Step != artifacts[j].Step {
			return artifacts[i].Step < artifacts[j].Step
		}
		return artifacts[i].Name < artifacts[j].Name
	})

	return artifacts, runID, nil
}

// scanDirectoryForArtifacts finds artifact files in a directory
func scanDirectoryForArtifacts(dir string, stepID string) []ArtifactOutput {
	var artifacts []ArtifactOutput

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Check if the file matches artifact patterns
		ext := strings.ToLower(filepath.Ext(path))
		artifactType, ok := artifactPatterns[ext]
		if !ok {
			return nil
		}

		artifacts = append(artifacts, ArtifactOutput{
			Step:   stepID,
			Name:   info.Name(),
			Type:   artifactType,
			Size:   info.Size(),
			Path:   path,
			Exists: true,
		})

		return nil
	})

	return artifacts
}

// outputArtifactsTable prints artifacts in table format
func outputArtifactsTable(runID string, artifacts []ArtifactOutput) error {
	if runID != "" {
		fmt.Printf("Artifacts for run: %s\n\n", runID)
	}

	if len(artifacts) == 0 {
		fmt.Println("No artifacts found")
		return nil
	}

	// Calculate column widths
	maxStep := len("STEP")
	maxName := len("ARTIFACT")
	maxType := len("TYPE")
	maxSize := len("SIZE")

	for _, a := range artifacts {
		if len(a.Step) > maxStep {
			maxStep = len(a.Step)
		}
		if len(a.Name) > maxName {
			maxName = len(a.Name)
		}
		if len(a.Type) > maxType {
			maxType = len(a.Type)
		}
		sizeStr := formatSize(a.Size)
		if len(sizeStr) > maxSize {
			maxSize = len(sizeStr)
		}
	}

	// Print header
	fmt.Printf("%-*s  %-*s  %-*s  %-*s  %s\n",
		maxStep, "STEP",
		maxName, "ARTIFACT",
		maxType, "TYPE",
		maxSize, "SIZE",
		"PATH")

	// Print artifacts
	for _, a := range artifacts {
		status := ""
		if !a.Exists {
			status = " [missing]"
		}
		fmt.Printf("%-*s  %-*s  %-*s  %-*s  %s%s\n",
			maxStep, a.Step,
			maxName, a.Name,
			maxType, a.Type,
			maxSize, formatSize(a.Size),
			a.Path,
			status)
	}

	return nil
}

// outputArtifactsJSON prints artifacts in JSON format
func outputArtifactsJSON(runID string, artifacts []ArtifactOutput) error {
	output := ArtifactsOutput{
		RunID:     runID,
		Artifacts: artifacts,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}

// exportArtifacts copies artifacts to the specified directory
func exportArtifacts(artifacts []ArtifactOutput, exportDir string) error {
	// Create export directory if it doesn't exist
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	var exportedCount int
	var totalSize int64
	var warnings []string
	namesSeen := make(map[string]bool)

	for _, a := range artifacts {
		// Skip missing artifacts
		if !a.Exists {
			warnings = append(warnings, fmt.Sprintf("skipping missing artifact: %s/%s", a.Step, a.Name))
			continue
		}

		// Create step subdirectory
		stepDir := filepath.Join(exportDir, a.Step)
		if err := os.MkdirAll(stepDir, 0755); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to create step directory: %s: %v", stepDir, err))
			continue
		}

		// Handle name collisions by prefixing with step ID
		exportName := a.Name
		exportPath := filepath.Join(stepDir, exportName)

		// If same name already seen in same step (shouldn't happen normally)
		// append a suffix
		key := fmt.Sprintf("%s/%s", a.Step, exportName)
		if namesSeen[key] {
			base := strings.TrimSuffix(exportName, filepath.Ext(exportName))
			ext := filepath.Ext(exportName)
			counter := 1
			for namesSeen[key] {
				exportName = fmt.Sprintf("%s_%d%s", base, counter, ext)
				key = fmt.Sprintf("%s/%s", a.Step, exportName)
				counter++
			}
			exportPath = filepath.Join(stepDir, exportName)
		}
		namesSeen[key] = true

		// Validate artifact path is within workspace (prevent path traversal)
		absPath, err := filepath.Abs(a.Path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("invalid artifact path %s: %v", a.Name, err))
			continue
		}
		workspaceAbs, _ := filepath.Abs(".wave/workspaces")
		if !strings.HasPrefix(absPath, workspaceAbs+string(filepath.Separator)) && !strings.HasPrefix(absPath, workspaceAbs) {
			warnings = append(warnings, fmt.Sprintf("artifact path outside workspace: %s", a.Name))
			continue
		}

		// Copy the file
		if err := copyArtifactFile(a.Path, exportPath); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to copy artifact %s: %v", a.Name, err))
			continue
		}

		exportedCount++
		totalSize += a.Size
	}

	// Print warnings
	for _, w := range warnings {
		fmt.Printf("Warning: %s\n", w)
	}

	// Print summary
	fmt.Printf("\nExported %d artifact(s) to %s (total: %s)\n", exportedCount, exportDir, formatSize(totalSize))

	return nil
}

// copyArtifactFile copies a single file from src to dst
func copyArtifactFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}
