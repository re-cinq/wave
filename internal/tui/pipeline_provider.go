package tui

import (
	"sort"
	"time"

	"github.com/recinq/wave/internal/pipelinecatalog"
	"github.com/recinq/wave/internal/state"
)

// RunningPipeline is a TUI-specific projection of a running pipeline run.
type RunningPipeline struct {
	RunID         string
	Name          string
	Input         string
	BranchName    string
	StartedAt     time.Time
	PID           int    // OS process ID of detached executor (0 = unknown)
	Detached      bool   // True when running as a detached subprocess
	SequenceGroup string // Compose group run ID, empty for standalone runs
}

// FinishedPipeline is a TUI-specific projection of a completed pipeline run.
type FinishedPipeline struct {
	RunID         string
	Name          string
	Input         string
	BranchName    string
	Status        string
	StartedAt     time.Time
	CompletedAt   time.Time
	Duration      time.Duration
	SequenceGroup string // Compose group run ID, empty for standalone runs
}

// PipelineDataProvider fetches pipeline data for the list component.
type PipelineDataProvider interface {
	FetchRunningPipelines() ([]RunningPipeline, error)
	FetchFinishedPipelines(limit int) ([]FinishedPipeline, error)
	FetchAvailablePipelines() ([]pipelinecatalog.PipelineInfo, error)
}

// DefaultPipelineDataProvider implements PipelineDataProvider using a state store and pipeline discovery.
type DefaultPipelineDataProvider struct {
	store        state.RunStore
	pipelinesDir string
}

// NewDefaultPipelineDataProvider creates a new provider wrapping the given state store and pipelines directory.
func NewDefaultPipelineDataProvider(store state.RunStore, pipelinesDir string) *DefaultPipelineDataProvider {
	return &DefaultPipelineDataProvider{
		store:        store,
		pipelinesDir: pipelinesDir,
	}
}

// staleRunCutoff is the age threshold after which a running/pending run
// with no live process is considered stale and filtered from the fleet list.
const staleRunCutoff = 1 * time.Hour

// FetchRunningPipelines returns currently running pipelines, sorted newest-first.
// Runs whose executor process is no longer alive (detached with a dead PID) or
// that are older than staleRunCutoff with no PID are filtered out to prevent
// stale queued/pending/running entries from persisting indefinitely.
func (p *DefaultPipelineDataProvider) FetchRunningPipelines() ([]RunningPipeline, error) {
	records, err := p.store.GetRunningRuns()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var result []RunningPipeline
	for _, r := range records {
		rp := RunningPipeline{
			RunID:      r.RunID,
			Name:       r.PipelineName,
			Input:      r.Input,
			BranchName: r.BranchName,
			StartedAt:  r.StartedAt,
			PID:        r.PID,
			Detached:   r.PID > 0,
		}

		// If the run has a PID, check whether the process is still alive.
		// Dead-process runs are stale leftovers from crashed sessions.
		if rp.PID > 0 && !IsProcessAlive(rp.PID) {
			continue
		}

		// Runs without a PID that are older than staleRunCutoff are almost
		// certainly orphaned from a previous session that exited ungracefully.
		if rp.PID == 0 && now.Sub(rp.StartedAt) > staleRunCutoff {
			continue
		}

		result = append(result, rp)
	}

	// Sort newest-first (GetRunningRuns already does this, but be explicit)
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.After(result[j].StartedAt)
	})

	return result, nil
}

// FetchFinishedPipelines returns the most recent finished pipeline runs.
func (p *DefaultPipelineDataProvider) FetchFinishedPipelines(limit int) ([]FinishedPipeline, error) {
	// Fetch more than needed to account for filtering
	records, err := p.store.ListRuns(state.ListRunsOptions{Limit: limit * 3})
	if err != nil {
		return nil, err
	}

	var result []FinishedPipeline
	for _, r := range records {
		if r.Status != "completed" && r.Status != "failed" && r.Status != "cancelled" {
			continue
		}

		fp := FinishedPipeline{
			RunID:      r.RunID,
			Name:       r.PipelineName,
			Input:      r.Input,
			BranchName: r.BranchName,
			Status:     r.Status,
			StartedAt:  r.StartedAt,
		}

		if r.CompletedAt != nil {
			fp.CompletedAt = *r.CompletedAt
			fp.Duration = r.CompletedAt.Sub(r.StartedAt)
		} else if r.CancelledAt != nil {
			fp.CompletedAt = *r.CancelledAt
			fp.Duration = r.CancelledAt.Sub(r.StartedAt)
		}

		result = append(result, fp)
		if len(result) >= limit {
			break
		}
	}

	return result, nil
}

// FetchAvailablePipelines returns all configured pipelines from the manifest directory.
func (p *DefaultPipelineDataProvider) FetchAvailablePipelines() ([]pipelinecatalog.PipelineInfo, error) {
	return pipelinecatalog.DiscoverPipelines(p.pipelinesDir)
}
