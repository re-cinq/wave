package proposals

import (
	"errors"
	"fmt"

	"github.com/recinq/wave/internal/state"
)

// Sentinels for rollback outcomes.
var (
	ErrNoActiveVersion = errors.New("no active pipeline version to roll back")
	ErrNoPriorVersion  = errors.New("no prior pipeline version available")
)

// PriorVersion returns (prior, current, nil) where current is the active
// pipeline_version row and prior is the highest version_id below it. Used
// by `wave proposals rollback` and POST /proposals/rollback to compute
// the activation target before flipping.
//
// Both records are returned by value so the caller can show "v3 -> v2"
// in CLI output / API response.
func PriorVersion(store state.EvolutionStore, pipelineName string) (state.PipelineVersionRecord, state.PipelineVersionRecord, error) {
	if pipelineName == "" {
		return state.PipelineVersionRecord{}, state.PipelineVersionRecord{}, fmt.Errorf("PriorVersion: pipeline name required")
	}
	versions, err := store.ListPipelineVersions(pipelineName)
	if err != nil {
		return state.PipelineVersionRecord{}, state.PipelineVersionRecord{}, fmt.Errorf("list versions: %w", err)
	}
	var current *state.PipelineVersionRecord
	for i := range versions {
		if versions[i].Active {
			current = &versions[i]
			break
		}
	}
	if current == nil {
		return state.PipelineVersionRecord{}, state.PipelineVersionRecord{}, ErrNoActiveVersion
	}

	// versions are sorted newest-first; pick first with Version < current.Version.
	for i := range versions {
		if versions[i].Version < current.Version {
			return versions[i], *current, nil
		}
	}
	return state.PipelineVersionRecord{}, *current, ErrNoPriorVersion
}
