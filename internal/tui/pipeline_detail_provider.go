package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"gopkg.in/yaml.v3"
)

// StepSummary is a lightweight projection of a pipeline step for display.
type StepSummary struct {
	ID      string
	Persona string
}

// AvailableDetail is the data projection for rendering an available pipeline's configuration.
type AvailableDetail struct {
	Name         string
	Description  string
	Category     string
	StepCount    int
	Steps        []StepSummary
	InputSource  string
	InputExample string
	Artifacts    []string // Output artifact names across all steps
	Skills       []string // Required skill names
	Tools        []string // Required tool names
}

// StepResult is a single step's execution result.
type StepResult struct {
	ID       string
	Status   string // "completed", "failed", "skipped", "pending"
	Duration time.Duration
	Persona  string
}

// ArtifactInfo describes a produced artifact.
type ArtifactInfo struct {
	Name string
	Path string
	Type string
}

// FinishedDetail is the data projection for rendering a finished pipeline's execution summary.
type FinishedDetail struct {
	RunID        string
	Name         string
	Status       string // "completed", "failed", "cancelled"
	Duration     time.Duration
	BranchName   string
	StartedAt    time.Time
	CompletedAt  time.Time
	ErrorMessage string // Non-empty for failed runs
	FailedStep   string // Step ID that failed
	Steps        []StepResult
	Artifacts    []ArtifactInfo
}

// DetailDataProvider is the interface for fetching detailed pipeline data.
type DetailDataProvider interface {
	FetchAvailableDetail(name string) (*AvailableDetail, error)
	FetchFinishedDetail(runID string) (*FinishedDetail, error)
}

// DefaultDetailDataProvider implements DetailDataProvider using state store and pipeline directory.
type DefaultDetailDataProvider struct {
	store        state.StateStore
	pipelinesDir string
}

// NewDefaultDetailDataProvider creates a new provider.
func NewDefaultDetailDataProvider(store state.StateStore, pipelinesDir string) *DefaultDetailDataProvider {
	return &DefaultDetailDataProvider{store: store, pipelinesDir: pipelinesDir}
}

// FetchAvailableDetail reads all YAML files from pipelinesDir, finds the pipeline with the
// given name, and returns a detailed projection of its configuration.
func (d *DefaultDetailDataProvider) FetchAvailableDetail(name string) (*AvailableDetail, error) {
	entries, err := os.ReadDir(d.pipelinesDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(d.pipelinesDir, entry.Name()))
		if err != nil {
			continue
		}

		var p pipeline.Pipeline
		if err := yaml.Unmarshal(data, &p); err != nil {
			continue
		}

		if p.Metadata.Name != name {
			continue
		}

		// Map steps to StepSummary.
		steps := make([]StepSummary, len(p.Steps))
		for i, s := range p.Steps {
			steps[i] = StepSummary{ID: s.ID, Persona: s.Persona}
		}

		// Collect output artifact names across all steps.
		var artifacts []string
		for _, s := range p.Steps {
			for _, a := range s.OutputArtifacts {
				artifacts = append(artifacts, a.Name)
			}
		}

		// Get skill names (nil-safe via SkillNames method).
		skills := p.Requires.SkillNames()

		// Get tool names (nil-safe).
		var tools []string
		if p.Requires != nil {
			tools = p.Requires.Tools
		}

		return &AvailableDetail{
			Name:         p.Metadata.Name,
			Description:  p.Metadata.Description,
			Category:     p.Metadata.Category,
			StepCount:    len(p.Steps),
			Steps:        steps,
			InputSource:  p.Input.Source,
			InputExample: p.Input.Example,
			Artifacts:    artifacts,
			Skills:       skills,
			Tools:        tools,
		}, nil
	}

	return nil, fmt.Errorf("pipeline not found: %s", name)
}

// FetchFinishedDetail returns detailed information about a finished pipeline run.
func (d *DefaultDetailDataProvider) FetchFinishedDetail(runID string) (*FinishedDetail, error) {
	run, err := d.store.GetRun(runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("run not found: %s", runID)
	}

	// Fetch performance metrics to build step results.
	metrics, err := d.store.GetPerformanceMetrics(runID, "")
	if err != nil {
		return nil, err
	}

	steps := make([]StepResult, len(metrics))
	var failedStep string
	for i, m := range metrics {
		status := "completed"
		if !m.Success {
			status = "failed"
			if failedStep == "" {
				failedStep = m.StepID
			}
		}
		steps[i] = StepResult{
			ID:       m.StepID,
			Status:   status,
			Duration: time.Duration(m.DurationMs) * time.Millisecond,
			Persona:  m.Persona,
		}
	}

	// Fetch artifacts.
	artifactRecords, err := d.store.GetArtifacts(runID, "")
	if err != nil {
		return nil, err
	}

	var artifacts []ArtifactInfo
	for _, a := range artifactRecords {
		artifacts = append(artifacts, ArtifactInfo{
			Name: a.Name,
			Path: a.Path,
			Type: a.Type,
		})
	}

	// Compute duration and CompletedAt.
	var duration time.Duration
	var completedAt time.Time
	if run.CompletedAt != nil {
		duration = run.CompletedAt.Sub(run.StartedAt)
		completedAt = *run.CompletedAt
	} else if run.CancelledAt != nil {
		duration = run.CancelledAt.Sub(run.StartedAt)
		completedAt = *run.CancelledAt
	}

	return &FinishedDetail{
		RunID:        run.RunID,
		Name:         run.PipelineName,
		Status:       run.Status,
		Duration:     duration,
		BranchName:   run.BranchName,
		StartedAt:    run.StartedAt,
		CompletedAt:  completedAt,
		ErrorMessage: run.ErrorMessage,
		FailedStep:   failedStep,
		Steps:        steps,
		Artifacts:    artifacts,
	}, nil
}
