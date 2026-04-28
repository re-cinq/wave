package listing

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/recinq/wave/internal/state"
	"gopkg.in/yaml.v3"
)

// DefaultStateDBPath is the default Wave state database location.
const DefaultStateDBPath = ".agents/state.db"

// DefaultWorkspacesDir is the default Wave workspaces directory.
const DefaultWorkspacesDir = ".agents/workspaces"

// ListRuns returns a slice of RunInfo, preferring the StateStore-backed source
// and falling back to workspace directory scans when the database is missing
// or returns no rows.
func ListRuns(opts RunsOptions) ([]RunInfo, error) {
	if _, err := os.Stat(DefaultStateDBPath); err == nil {
		dbRuns, err := listRunsFromDB(DefaultStateDBPath, opts)
		if err == nil && len(dbRuns) > 0 {
			return dbRuns, nil
		}
	}
	return listRunsFromWorkspaces(opts)
}

// listRunsFromDB reads run information from the state database via StateStore.
func listRunsFromDB(dbPath string, opts RunsOptions) ([]RunInfo, error) {
	store, err := state.NewReadOnlyStateStore(dbPath)
	if err != nil {
		return nil, err
	}
	defer store.Close()

	listOpts := state.ListRunsOptions{
		PipelineName: opts.Pipeline,
		Status:       strings.ToLower(opts.Status),
		Limit:        opts.Limit,
	}

	records, err := store.ListRuns(listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to query pipeline runs: %w", err)
	}

	runs := make([]RunInfo, 0, len(records))
	for _, r := range records {
		var duration string
		var durationMs int64

		switch {
		case r.CompletedAt != nil:
			d := r.CompletedAt.Sub(r.StartedAt)
			durationMs = d.Milliseconds()
			duration = FormatDuration(d)
		case strings.ToLower(r.Status) == "running":
			d := time.Since(r.StartedAt)
			durationMs = d.Milliseconds()
			duration = FormatDuration(d) + " (running)"
		default:
			duration = "-"
		}

		runs = append(runs, RunInfo{
			RunID:      r.RunID,
			Pipeline:   r.PipelineName,
			Status:     r.Status,
			StartedAt:  r.StartedAt.Format("2006-01-02 15:04:05"),
			Duration:   duration,
			DurationMs: durationMs,
		})
	}

	return runs, nil
}

// listRunsFromWorkspaces reads run information from workspace directory metadata.
func listRunsFromWorkspaces(opts RunsOptions) ([]RunInfo, error) {
	entries, err := os.ReadDir(DefaultWorkspacesDir)
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

		if opts.Pipeline != "" && entry.Name() != opts.Pipeline {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		wsPath := filepath.Join(DefaultWorkspacesDir, entry.Name())
		startTime := directoryCreationTime(wsPath)
		if startTime.IsZero() {
			startTime = info.ModTime()
		}

		workspaces = append(workspaces, wsInfo{
			name:      entry.Name(),
			modTime:   info.ModTime(),
			startTime: startTime,
		})
	}

	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].modTime.After(workspaces[j].modTime)
	})

	if opts.Limit > 0 && len(workspaces) > opts.Limit {
		workspaces = workspaces[:opts.Limit]
	}

	var runs []RunInfo
	for _, ws := range workspaces {
		wsPath := filepath.Join(DefaultWorkspacesDir, ws.name)
		status, endTime := inferWorkspaceStatus(wsPath, ws.name)

		if opts.Status != "" && !strings.EqualFold(status, opts.Status) {
			continue
		}

		var duration string
		var durationMs int64
		if !endTime.IsZero() && !ws.startTime.IsZero() {
			d := endTime.Sub(ws.startTime)
			durationMs = d.Milliseconds()
			duration = FormatDuration(d)
		} else {
			duration = "-"
		}

		runs = append(runs, RunInfo{
			RunID:      ws.name,
			Pipeline:   ExtractPipelineName(ws.name),
			Status:     status,
			StartedAt:  ws.startTime.Format("2006-01-02 15:04:05"),
			Duration:   duration,
			DurationMs: durationMs,
		})
	}

	return runs, nil
}

// inferWorkspaceStatus determines the status of a run by examining its workspace.
func inferWorkspaceStatus(wsPath string, pipelineName string) (status string, endTime time.Time) {
	baseName := ExtractPipelineName(pipelineName)
	pipelinePath := filepath.Join(DefaultPipelineDir, baseName+".yaml")
	pipelineData, err := os.ReadFile(pipelinePath)
	if err != nil {
		stepDirs, _ := os.ReadDir(wsPath)
		if len(stepDirs) == 0 {
			return "pending", time.Time{}
		}
		return "unknown", latestFileTime(wsPath)
	}

	var p struct {
		Steps []struct {
			ID string `yaml:"id"`
		} `yaml:"steps"`
	}
	if err := yaml.Unmarshal(pipelineData, &p); err != nil {
		return "unknown", latestFileTime(wsPath)
	}

	expectedSteps := make(map[string]bool)
	for _, step := range p.Steps {
		expectedSteps[step.ID] = false
	}

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
			stepPath := filepath.Join(wsPath, stepID)
			if hasStepOutput(stepPath) {
				expectedSteps[stepID] = true
				completedSteps++
			}
			if info, err := dir.Info(); err == nil {
				if info.ModTime().After(latestTime) {
					latestTime = info.ModTime()
				}
			}
		}
	}

	if completedSteps == 0 {
		return "pending", time.Time{}
	}
	if completedSteps == len(expectedSteps) {
		return "completed", latestTime
	}
	return "partial", latestTime
}

// hasStepOutput checks if a step directory contains any output files.
func hasStepOutput(stepPath string) bool {
	outputDirs := []string{"", ".agents/output", ".agents/artifacts"}
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
				return true
			}
		}
	}
	return false
}

// latestFileTime finds the most recent modification time in a directory tree.
func latestFileTime(dirPath string) time.Time {
	var latest time.Time
	_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
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

// directoryCreationTime approximates the creation time of a directory by
// finding the oldest file inside, falling back to the directory's mod time.
func directoryCreationTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	var oldest time.Time
	_ = filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if oldest.IsZero() || fi.ModTime().Before(oldest) {
			oldest = fi.ModTime()
		}
		return nil
	})
	if oldest.IsZero() {
		return info.ModTime()
	}
	return oldest
}
