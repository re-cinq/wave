package retro

import (
	"context"
	"log"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/metrics"
	"github.com/recinq/wave/internal/state"
)

// runQuerier is the run/step subset of state.RunStore the collector needs.
// Kept narrow so callers can pass any store that exposes these methods.
type runQuerier interface {
	GetRun(runID string) (*state.RunRecord, error)
	GetStepAttempts(runID string, stepID string) ([]state.StepAttemptRecord, error)
}

// combinedStore composes a runQuerier with a *metrics.Store so the Collector
// can satisfy its StateQuerier dependency without forcing callers to wire a
// new aggregate type.
type combinedStore struct {
	runQuerier
	*metrics.Store
}

// Generator orchestrates retrospective generation after pipeline runs.
type Generator struct {
	collector *Collector
	storage   *Storage
	narrator  *Narrator
	config    *manifest.RetrosConfig
}

// NewGenerator creates a Generator with all dependencies. The runStore
// supplies run/step lookups; the metricsStore supplies performance-metric
// reads and retrospective index writes (both tables migrated to
// internal/metrics in #62).
func NewGenerator(runStore runQuerier, metricsStore *metrics.Store, runner adapter.AdapterRunner, retrosDir string, config *manifest.RetrosConfig) *Generator {
	g := &Generator{
		collector: NewCollector(combinedStore{runQuerier: runStore, Store: metricsStore}),
		storage:   NewStorage(retrosDir, metricsStore),
		config:    config,
	}

	if config.IsNarrateEnabled() && runner != nil {
		g.narrator = NewNarrator(runner, config.GetNarrateModel())
	}

	return g
}

// Generate creates a retrospective for the given run.
// Quantitative data is collected synchronously. Narrative generation
// runs asynchronously and does not block.
func (g *Generator) Generate(runID string, pipelineName string) {
	if !g.config.IsEnabled() {
		return
	}

	// Phase 1: Collect quantitative data (synchronous, fast)
	quant, err := g.collector.Collect(runID)
	if err != nil {
		log.Printf("[retro] failed to collect quantitative data for run %s: %v", runID, err)
		return
	}

	// Create retrospective with quantitative data
	retro := &Retrospective{
		RunID:        runID,
		Pipeline:     pipelineName,
		Timestamp:    time.Now(),
		Quantitative: quant,
	}

	// Save quantitative retro immediately
	if err := g.storage.Save(retro); err != nil {
		log.Printf("[retro] failed to save retrospective for run %s: %v", runID, err)
		return
	}

	log.Printf("[retro] quantitative retrospective saved for run %s", runID)

	// Phase 2: Narrative generation (asynchronous, non-blocking)
	if g.narrator != nil {
		go g.generateNarrative(runID, pipelineName, quant)
	}
}

// generateNarrative runs LLM narrative generation asynchronously.
func (g *Generator) generateNarrative(runID string, pipelineName string, quant *QuantitativeData) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	narrative, err := g.narrator.Narrate(ctx, runID, pipelineName, quant)
	if err != nil {
		log.Printf("[retro] narrative generation failed for run %s: %v", runID, err)
		return
	}

	// Load current retro and attach narrative
	retro, err := g.storage.Load(runID)
	if err != nil {
		log.Printf("[retro] failed to load retrospective for narrative update on run %s: %v", runID, err)
		return
	}

	retro.Narrative = narrative
	if err := g.storage.Update(retro); err != nil {
		log.Printf("[retro] failed to update retrospective with narrative for run %s: %v", runID, err)
		return
	}

	log.Printf("[retro] narrative retrospective completed for run %s (smoothness: %s)", runID, narrative.Smoothness)
}

// GenerateNarrativeSync generates the narrative synchronously (for CLI --narrate).
func (g *Generator) GenerateNarrativeSync(ctx context.Context, runID string) error {
	retro, err := g.storage.Load(runID)
	if err != nil {
		return err
	}

	if g.narrator == nil {
		return nil
	}

	narrative, err := g.narrator.Narrate(ctx, runID, retro.Pipeline, retro.Quantitative)
	if err != nil {
		return err
	}

	retro.Narrative = narrative
	return g.storage.Update(retro)
}

// GetStorage returns the storage for direct access.
func (g *Generator) GetStorage() *Storage {
	return g.storage
}
