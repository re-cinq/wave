package proposal

import (
	"fmt"
	"sort"
	"time"
)

// Engine produces pipeline execution proposals from a health artifact
// and pipeline catalog. It is a pure Go function with no LLM invocation.
type Engine struct {
	catalog *Catalog
	scorer  Scorer
}

// EngineOption configures the proposal engine.
type EngineOption func(*Engine)

// WithScorer sets a custom scoring strategy.
func WithScorer(s Scorer) EngineOption {
	return func(e *Engine) {
		e.scorer = s
	}
}

// NewEngine creates a new proposal engine with the given catalog and options.
func NewEngine(catalog *Catalog, opts ...EngineOption) *Engine {
	e := &Engine{
		catalog: catalog,
		scorer:  &DefaultScorer{},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// scoredEntry pairs a catalog entry with its relevance score.
type scoredEntry struct {
	entry CatalogEntry
	score float64
}

// Propose generates a pipeline execution proposal based on the health
// artifact and forge type. It filters the catalog by forge, scores each
// pipeline, constructs dependency edges, assigns parallel groups, and
// returns a sorted proposal.
func (e *Engine) Propose(health HealthArtifact, forgeType ForgeType) (*Proposal, error) {
	if e.catalog == nil {
		return nil, fmt.Errorf("engine: catalog is nil")
	}

	entries := e.catalog.Entries()
	filtered := FilterByForge(entries, forgeType)

	// Score each pipeline
	var candidates []scoredEntry
	for _, entry := range filtered {
		s := e.scorer.Score(entry, health)
		if s > 0 {
			candidates = append(candidates, scoredEntry{entry: entry, score: s})
		}
	}

	// Sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].entry.Name < candidates[j].entry.Name
	})

	// Build proposal items with dependency edges and parallel groups
	items := make([]ProposalItem, 0, len(candidates))
	for i, c := range candidates {
		deps := inferDependencies(c.entry, candidates)
		item := ProposalItem{
			Pipeline:      c.entry.Name,
			Rationale:     buildRationale(c.entry, health),
			Priority:      i + 1,
			Score:         c.score,
			ParallelGroup: 0, // Will be assigned below
			DependsOn:     deps,
			Category:      c.entry.Category,
		}
		items = append(items, item)
	}

	// Assign parallel groups based on dependency structure
	assignParallelGroups(items)

	effectiveForge := forgeType
	if effectiveForge == "" {
		effectiveForge = health.ForgeType
	}
	if effectiveForge == "" {
		effectiveForge = ForgeUnknown
	}

	proposal := &Proposal{
		ForgeType:     effectiveForge,
		Proposals:     items,
		Timestamp:     time.Now().UTC(),
		HealthSummary: health.Summary,
	}

	return proposal, nil
}

// phasePriority defines the logical ordering of pipeline categories.
// Lower numbers execute first.
var phasePriority = map[string]int{
	"security-scan": 1,
	"dead-code":     2,
	"test-gen":      2,
	"smoke-test":    2,
	"doc-audit":     3,
	"doc-fix":       4,
	"refactor":      5,
	"improve":       5,
	"debug":         1,
	"hotfix":        1,
}

// inferDependencies determines which other candidate pipelines this entry
// should depend on based on logical phase ordering.
func inferDependencies(entry CatalogEntry, candidates []scoredEntry) []string {
	myPhase, ok := phasePriority[entry.Name]
	if !ok {
		return nil
	}

	var deps []string
	for _, c := range candidates {
		if c.entry.Name == entry.Name {
			continue
		}
		theirPhase, ok := phasePriority[c.entry.Name]
		if !ok {
			continue
		}
		// Depend on items from earlier phases
		if theirPhase < myPhase {
			deps = append(deps, c.entry.Name)
		}
	}
	return deps
}

// assignParallelGroups assigns parallel group identifiers to proposal items.
// Items with no dependencies and no mutual dependency conflicts share group 0.
// Items that depend on earlier items get progressively higher group numbers.
func assignParallelGroups(items []ProposalItem) {
	if len(items) == 0 {
		return
	}

	// Build a name→index map
	nameIndex := make(map[string]int, len(items))
	for i, item := range items {
		nameIndex[item.Pipeline] = i
	}

	// Assign groups via topological layers
	assigned := make([]bool, len(items))
	group := 0
	for {
		var layerIndices []int
		for i, item := range items {
			if assigned[i] {
				continue
			}
			// Check if all dependencies are already assigned
			allDepsAssigned := true
			for _, dep := range item.DependsOn {
				if idx, ok := nameIndex[dep]; ok && !assigned[idx] {
					allDepsAssigned = false
					break
				}
			}
			if allDepsAssigned {
				layerIndices = append(layerIndices, i)
			}
		}

		if len(layerIndices) == 0 {
			// Assign remaining items (cyclic deps or orphans) to the next group
			for i := range items {
				if !assigned[i] {
					items[i].ParallelGroup = group
					assigned[i] = true
				}
			}
			break
		}

		for _, idx := range layerIndices {
			items[idx].ParallelGroup = group
			assigned[idx] = true
		}
		group++
	}
}

// buildRationale generates a human-readable rationale for why a pipeline
// is relevant based on the health signals.
func buildRationale(entry CatalogEntry, health HealthArtifact) string {
	for _, signal := range health.Signals {
		pipelines, ok := signalMapping[signal.Category]
		if !ok {
			continue
		}
		for _, p := range pipelines {
			if p == entry.Name {
				return fmt.Sprintf("%s detected: %s (severity: %s, count: %d)",
					signal.Category, signal.Detail, signal.Severity, signal.Count)
			}
		}
	}
	return fmt.Sprintf("Pipeline %q is relevant to the current codebase health state", entry.Name)
}
