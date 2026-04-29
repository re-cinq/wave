package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	wgit "github.com/recinq/wave/internal/git"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
)

// StepSummary is a lightweight projection of a pipeline step for display.
type StepSummary struct {
	ID      string
	Persona string
}

// stepTypeLabel returns a descriptive label for composition steps that have no persona.
func stepTypeLabel(s pipeline.Step) string {
	switch {
	case s.SubPipeline != "":
		return "pipeline:" + s.SubPipeline
	case s.Branch != nil:
		return "branch"
	case s.Gate != nil:
		return "gate"
	case s.Loop != nil:
		return "loop"
	case s.Aggregate != nil:
		return "aggregate"
	default:
		return "step"
	}
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
	ID           string
	Status       string // "completed", "failed", "skipped", "pending"
	Duration     time.Duration
	Persona      string
	FailureClass string // e.g. "transient", "deterministic", "contract_failure", "test_failure", "budget_exhausted", "canceled"
}

// ArtifactInfo describes a produced artifact.
type ArtifactInfo struct {
	Name string
	Path string
	Type string
}

// FinishedDetail is the data projection for rendering a finished pipeline's execution summary.
type FinishedDetail struct {
	RunID         string
	Name          string
	Input         string
	Status        string // "completed", "failed", "cancelled"
	Duration      time.Duration
	BranchName    string
	StartedAt     time.Time
	CompletedAt   time.Time
	ErrorMessage  string // Non-empty for failed runs
	FailedStep    string // Step ID that failed
	Steps         []StepResult
	Artifacts     []ArtifactInfo
	WorkspacePath string // Filesystem path to pipeline workspace, empty if deleted
	BranchDeleted bool   // True if the branch no longer exists
}

// DetailDataProvider is the interface for fetching detailed pipeline data.
type DetailDataProvider interface {
	FetchAvailableDetail(name string) (*AvailableDetail, error)
	FetchFinishedDetail(runID string) (*FinishedDetail, error)
	FetchRunEvents(runID string) ([]state.LogRecord, error)
}

// detailStore is the persistence surface needed by the detail provider:
// run lookup + step performance/attempts (RunStore) plus event/artifact
// retrieval (EventStore). Satisfied by the aggregate StateStore.
type detailStore interface {
	state.RunStore
	state.EventStore
}

// DefaultDetailDataProvider implements DetailDataProvider using state store and pipeline directory.
type DefaultDetailDataProvider struct {
	store        detailStore
	pipelinesDir string
}

// NewDefaultDetailDataProvider creates a new provider.
func NewDefaultDetailDataProvider(store detailStore, pipelinesDir string) *DefaultDetailDataProvider {
	return &DefaultDetailDataProvider{store: store, pipelinesDir: pipelinesDir}
}

// FetchAvailableDetail reads all YAML files from pipelinesDir, finds the pipeline with the
// given name, and returns a detailed projection of its configuration.
func (d *DefaultDetailDataProvider) FetchAvailableDetail(name string) (*AvailableDetail, error) {
	pipelines := pipeline.ScanPipelinesDir(d.pipelinesDir)
	if pipelines == nil {
		// Distinguish "no pipelines dir / unreadable" from "not found"
		// — preserve the original error semantics by stat'ing the dir.
		if _, err := os.Stat(d.pipelinesDir); err != nil {
			return nil, err
		}
	}

	for _, p := range pipelines {
		if p.Metadata.Name != name {
			continue
		}

		// Map steps to StepSummary.
		steps := make([]StepSummary, len(p.Steps))
		for i, s := range p.Steps {
			persona := s.Persona
			if persona == "" {
				persona = stepTypeLabel(s)
			}
			steps[i] = StepSummary{ID: s.ID, Persona: persona}
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
		// Look up failure class from step attempts for failed steps
		if status == "failed" {
			attempts, attErr := d.store.GetStepAttempts(runID, m.StepID)
			if attErr == nil && len(attempts) > 0 {
				lastAttempt := attempts[len(attempts)-1]
				if lastAttempt.FailureClass != "" {
					steps[i].FailureClass = lastAttempt.FailureClass
				}
			}
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

	detail := &FinishedDetail{
		RunID:        run.RunID,
		Name:         run.PipelineName,
		Input:        run.Input,
		Status:       run.Status,
		Duration:     duration,
		BranchName:   run.BranchName,
		StartedAt:    run.StartedAt,
		CompletedAt:  completedAt,
		ErrorMessage: run.ErrorMessage,
		FailedStep:   failedStep,
		Steps:        steps,
		Artifacts:    artifacts,
	}

	// Derive workspace path from RunID and BranchName.
	if run.BranchName != "" {
		sanitized := pipeline.SanitizeBranchName(run.BranchName)
		wsPath := filepath.Join(".agents", "workspaces", run.RunID, "__wt_"+sanitized)
		if _, err := os.Stat(wsPath); err == nil {
			detail.WorkspacePath = wsPath
		}
	} else {
		// Fallback: try glob for any worktree under this run.
		matches, _ := filepath.Glob(filepath.Join(".agents", "workspaces", run.RunID, "__wt_*"))
		if len(matches) > 0 {
			if info, err := os.Stat(matches[0]); err == nil && info.IsDir() {
				detail.WorkspacePath = matches[0]
			}
		}
	}

	// Check if the branch still exists via the centralized git helper.
	if run.BranchName != "" {
		exists, vErr := wgit.VerifyRef(run.BranchName)
		if vErr != nil || !exists {
			detail.BranchDeleted = true
		}
	}

	return detail, nil
}

// FetchRunEvents retrieves persisted event log records for a pipeline run.
func (d *DefaultDetailDataProvider) FetchRunEvents(runID string) ([]state.LogRecord, error) {
	return d.store.GetEvents(runID, state.EventQueryOptions{Limit: 500})
}
