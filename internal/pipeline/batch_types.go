package pipeline

import (
	"fmt"
	"sort"
	"time"

	"github.com/recinq/wave/internal/manifest"
)

// OnFailure defines the error handling policy for batch pipeline execution.
type OnFailure string

const (
	// OnFailureContinue allows other pipelines to continue when one fails.
	// Dependent pipelines are skipped; independent pipelines proceed.
	OnFailureContinue OnFailure = "continue"

	// OnFailureAbortAll cancels all running pipelines when one fails.
	OnFailureAbortAll OnFailure = "abort-all"
)

// PipelineRunStatus represents the completion status of a pipeline within a batch.
type PipelineRunStatus string

const (
	// RunStatusCompleted indicates the pipeline finished successfully.
	RunStatusCompleted PipelineRunStatus = "completed"

	// RunStatusFailed indicates the pipeline encountered an error.
	RunStatusFailed PipelineRunStatus = "failed"

	// RunStatusSkipped indicates the pipeline was skipped (e.g., dependency failed).
	RunStatusSkipped PipelineRunStatus = "skipped"
)

// PipelineBatchConfig defines a batch of pipelines to execute with dependency
// ordering, concurrency limits, and error handling policy.
type PipelineBatchConfig struct {
	// Pipelines is the list of pipelines to execute in this batch.
	Pipelines []PipelineBatchEntry

	// Dependencies maps a pipeline name to the list of pipeline names it depends on.
	// Pipelines without entries run in the first tier.
	Dependencies map[string][]string

	// MaxConcurrentPipelines limits how many pipelines run at the same time.
	// A value of 0 means unlimited (all pipelines in a tier run concurrently).
	MaxConcurrentPipelines int

	// OnFailure determines what happens when a pipeline fails.
	// Defaults to OnFailureContinue if empty.
	OnFailure OnFailure
}

// PipelineBatchEntry describes a single pipeline within a batch.
type PipelineBatchEntry struct {
	// Name is the unique identifier for this pipeline within the batch.
	Name string

	// Pipeline is the pipeline definition to execute.
	Pipeline *Pipeline

	// Manifest is the manifest configuration for this pipeline.
	Manifest *manifest.Manifest

	// Input is the input string passed to the pipeline execution.
	Input string
}

// PipelineRunResult holds the outcome of a single pipeline execution within a batch.
type PipelineRunResult struct {
	// Name is the pipeline name matching PipelineBatchEntry.Name.
	Name string

	// Status is the completion status of the pipeline.
	Status PipelineRunStatus

	// Error is the error encountered during execution, nil if successful.
	Error error

	// ArtifactPaths maps "stepID:artifactName" to the filesystem path of the artifact.
	ArtifactPaths map[string]string

	// TokensUsed is the total number of tokens consumed by this pipeline.
	TokensUsed int

	// Duration is the wall-clock time the pipeline took to execute.
	Duration time.Duration

	// SkipReason explains why the pipeline was skipped, empty if not skipped.
	SkipReason string
}

// PipelineBatchResult aggregates the results of all pipelines in a batch execution.
type PipelineBatchResult struct {
	// Results contains the per-pipeline execution results.
	Results []PipelineRunResult

	// TotalDuration is the wall-clock duration of the entire batch execution.
	TotalDuration time.Duration

	// TotalTokens is the sum of tokens consumed across all pipelines.
	TotalTokens int

	// CompletedCount is the number of pipelines that finished successfully.
	CompletedCount int

	// FailedCount is the number of pipelines that encountered errors.
	FailedCount int

	// SkippedCount is the number of pipelines that were skipped.
	SkippedCount int
}

// Validate checks the batch configuration for errors.
// It ensures at least one pipeline is present, names are unique, dependencies
// reference existing pipelines, there are no cycles, and configuration values
// are within valid ranges. If OnFailure is empty, it defaults to OnFailureContinue.
func (c *PipelineBatchConfig) Validate() error {
	if len(c.Pipelines) == 0 {
		return fmt.Errorf("batch config must contain at least one pipeline")
	}

	if c.MaxConcurrentPipelines < 0 {
		return fmt.Errorf("max_concurrent_pipelines must be non-negative, got %d", c.MaxConcurrentPipelines)
	}

	// Default error policy
	if c.OnFailure == "" {
		c.OnFailure = OnFailureContinue
	}

	// Build name set and check for duplicates
	names := make(map[string]bool, len(c.Pipelines))
	for _, entry := range c.Pipelines {
		if names[entry.Name] {
			return fmt.Errorf("duplicate pipeline name %q in batch config", entry.Name)
		}
		names[entry.Name] = true
	}

	// Validate dependencies reference existing pipeline names
	if c.Dependencies != nil {
		for name, deps := range c.Dependencies {
			if !names[name] {
				return fmt.Errorf("dependency source %q is not a pipeline in this batch", name)
			}
			for _, dep := range deps {
				if !names[dep] {
					return fmt.Errorf("pipeline %q depends on %q which is not in this batch", name, dep)
				}
			}
		}
	}

	// Detect cycles via tier computation
	deps := c.Dependencies
	if deps == nil {
		deps = make(map[string][]string)
	}
	if _, err := computePipelineTiers(names, deps); err != nil {
		return fmt.Errorf("dependency cycle detected: %w", err)
	}

	return nil
}

// computePipelineTiers groups pipeline names into execution tiers using Kahn's
// algorithm (BFS topological sort). Pipelines within a tier have no dependencies
// on each other and can run concurrently. Tiers are executed sequentially.
// Returns an error if a dependency cycle is detected.
func computePipelineTiers(names map[string]bool, deps map[string][]string) ([][]string, error) {
	// Build in-degree map
	inDegree := make(map[string]int, len(names))
	for name := range names {
		inDegree[name] = 0
	}
	for name, depList := range deps {
		if !names[name] {
			continue
		}
		inDegree[name] = len(depList)
	}

	// Build reverse adjacency: dep -> names that depend on it
	reverse := make(map[string][]string, len(names))
	for name, depList := range deps {
		if !names[name] {
			continue
		}
		for _, dep := range depList {
			reverse[dep] = append(reverse[dep], name)
		}
	}

	var tiers [][]string
	remaining := len(names)

	for remaining > 0 {
		// Collect names with in-degree 0
		var tier []string
		for name, deg := range inDegree {
			if deg == 0 {
				tier = append(tier, name)
			}
		}

		if len(tier) == 0 {
			return nil, fmt.Errorf("cycle detected among remaining %d pipelines", remaining)
		}

		// Sort tier for deterministic ordering
		sort.Strings(tier)

		// Remove processed names from graph
		for _, name := range tier {
			delete(inDegree, name)
			for _, dependent := range reverse[name] {
				inDegree[dependent]--
			}
		}

		tiers = append(tiers, tier)
		remaining -= len(tier)
	}

	return tiers, nil
}
