package tui

import (
	"sort"
	"time"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
)

// PersonaInfo is the TUI data projection for a persona.
type PersonaInfo struct {
	Name          string
	Description   string
	Adapter       string
	Model         string
	AllowedTools  []string
	DeniedTools   []string
	PipelineUsage []PipelineStepRef
}

// PersonaStats holds aggregated run statistics for a persona.
type PersonaStats struct {
	TotalRuns      int
	SuccessfulRuns int
	AvgDurationMs  int64
	LastRunAt      time.Time
}

// PersonaDataProvider fetches persona data for the Personas view.
type PersonaDataProvider interface {
	FetchPersonas() ([]PersonaInfo, error)
	FetchPersonaStats(name string) (*PersonaStats, error)
}

// DefaultPersonaDataProvider implements PersonaDataProvider using the manifest and state store.
type DefaultPersonaDataProvider struct {
	manifest     *manifest.Manifest
	store        state.RunStore
	pipelinesDir string
}

// NewDefaultPersonaDataProvider creates a new persona data provider.
func NewDefaultPersonaDataProvider(m *manifest.Manifest, store state.RunStore, pipelinesDir string) *DefaultPersonaDataProvider {
	return &DefaultPersonaDataProvider{
		manifest:     m,
		store:        store,
		pipelinesDir: pipelinesDir,
	}
}

// FetchPersonas returns all personas from the manifest with pipeline usage cross-references.
func (p *DefaultPersonaDataProvider) FetchPersonas() ([]PersonaInfo, error) {
	if p.manifest == nil {
		return nil, nil
	}

	// Build pipeline usage map by scanning all pipeline YAML files
	usageMap := make(map[string][]PipelineStepRef)
	for _, pl := range pipeline.ScanPipelinesDir(p.pipelinesDir) {
		for _, step := range pl.Steps {
			if step.Persona != "" {
				usageMap[step.Persona] = append(usageMap[step.Persona], PipelineStepRef{
					PipelineName: pl.Metadata.Name,
					StepID:       step.ID,
				})
			}
		}
	}

	// Build persona info list
	var personas []PersonaInfo
	for name, persona := range p.manifest.Personas {
		info := PersonaInfo{
			Name:          name,
			Description:   persona.Description,
			Adapter:       persona.Adapter,
			Model:         persona.Model,
			AllowedTools:  persona.Permissions.AllowedTools,
			DeniedTools:   persona.Permissions.Deny,
			PipelineUsage: usageMap[name],
		}
		personas = append(personas, info)
	}

	sort.Slice(personas, func(i, j int) bool {
		return personas[i].Name < personas[j].Name
	})

	return personas, nil
}

// FetchPersonaStats aggregates performance stats for a persona from the state store.
func (p *DefaultPersonaDataProvider) FetchPersonaStats(name string) (*PersonaStats, error) {
	if p.store == nil {
		return nil, nil
	}

	records, err := p.store.GetRecentPerformanceHistory(state.PerformanceQueryOptions{
		Persona: name,
		Limit:   1000,
	})
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil
	}

	stats := &PersonaStats{
		TotalRuns: len(records),
	}
	var totalDuration int64
	for _, r := range records {
		if r.Success {
			stats.SuccessfulRuns++
		}
		totalDuration += r.DurationMs
		if r.StartedAt.After(stats.LastRunAt) {
			stats.LastRunAt = r.StartedAt
		}
	}
	stats.AvgDurationMs = totalDuration / int64(len(records))

	return stats, nil
}
