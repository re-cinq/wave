package suggest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/doctor"
)

// Proposal contains a prioritized list of pipeline proposals.
type Proposal struct {
	Pipelines []ProposedPipeline `json:"pipelines"`
	Rationale string             `json:"rationale"`
}

// ProposedPipeline is a single pipeline recommendation.
type ProposedPipeline struct {
	Name     string   `json:"name"`
	Reason   string   `json:"reason"`
	Input    string   `json:"input,omitempty"`
	Priority int      `json:"priority"`
	Type     string   `json:"type,omitempty"`     // "single", "sequence", or "parallel"
	Sequence []string `json:"sequence,omitempty"` // Pipeline names for multi-pipeline proposals
}

// EngineOptions configures the suggestion engine.
type EngineOptions struct {
	Report       *doctor.Report
	PipelinesDir string
	Limit        int
}

// Suggest generates pipeline proposals based on codebase health.
func Suggest(opts EngineOptions) (*Proposal, error) {
	if opts.Report == nil {
		return nil, fmt.Errorf("report is required")
	}
	if opts.Limit <= 0 {
		opts.Limit = 5
	}
	if opts.PipelinesDir == "" {
		opts.PipelinesDir = ".agents/pipelines"
	}

	catalog := discoverPipelines(opts.PipelinesDir)
	prefix := ""
	if opts.Report.ForgeInfo != nil {
		prefix = opts.Report.ForgeInfo.PipelinePrefix
	}

	var proposals []ProposedPipeline

	// Apply rules in priority order
	if opts.Report.Codebase != nil {
		cb := opts.Report.Codebase

		// Priority 3: Open issues without linked PRs
		if cb.Issues.Open > 0 {
			implName := resolvePipeline(catalog, prefix, "impl-issue")
			researchName := resolvePipeline(catalog, prefix, "plan-research")
			if implName != "" {
				proposals = append(proposals, ProposedPipeline{
					Name:     implName,
					Reason:   fmt.Sprintf("%d open issues to work on", cb.Issues.Open),
					Input:    "Implement open issues",
					Priority: 3,
				})
			}
			// Also propose research if available (enables sequence chaining)
			if researchName != "" {
				prio := 3
				if implName != "" {
					prio = 4 // Lower priority when implement is also available
				}
				proposals = append(proposals, ProposedPipeline{
					Name:     researchName,
					Reason:   fmt.Sprintf("%d open issues to research", cb.Issues.Open),
					Input:    "Research open issues",
					Priority: prio,
				})
			}
		}

		// Priority 4: PRs needing review
		if cb.PRs.NeedsReview > 0 {
			if name := resolvePipeline(catalog, prefix, "ops-pr-review"); name != "" {
				proposals = append(proposals, ProposedPipeline{
					Name:     name,
					Reason:   fmt.Sprintf("%d PRs awaiting review", cb.PRs.NeedsReview),
					Input:    "Review pending PRs",
					Priority: 4,
				})
			}
		}
	}

	// Detect sequence chains: if both research and implement are proposed, create
	// a sequence proposal (research → implement).
	proposals = detectSequences(proposals, catalog, prefix)

	// Detect parallel-eligible groups: if implement and ops-pr-review are both
	// proposed, mark them as parallel-eligible.
	proposals = detectParallelGroups(proposals)

	// Filter to available pipelines and apply limit
	var available []ProposedPipeline
	for _, p := range proposals {
		if p.Type == "sequence" || p.Type == "parallel" {
			// Multi-pipeline proposals: check all members exist
			allExist := true
			for _, name := range p.Sequence {
				if !inCatalog(catalog, name) {
					allExist = false
					break
				}
			}
			if allExist {
				available = append(available, p)
			}
		} else if inCatalog(catalog, p.Name) {
			available = append(available, p)
		}
	}
	if len(available) > opts.Limit {
		available = available[:opts.Limit]
	}

	rationale := "No actionable suggestions"
	if len(available) > 0 {
		rationale = fmt.Sprintf("%d pipeline(s) suggested based on codebase analysis", len(available))
	}

	return &Proposal{
		Pipelines: available,
		Rationale: rationale,
	}, nil
}

// discoverPipelines lists pipeline names from the pipelines directory.
func discoverPipelines(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext == ".yaml" || ext == ".yml" {
			name := strings.TrimSuffix(entry.Name(), ext)
			names = append(names, name)
		}
	}
	return names
}

// taxonomyPrefixes are the category prefixes used in the pipeline naming taxonomy.
var taxonomyPrefixes = []string{"plan-", "impl-", "audit-", "doc-", "test-", "ops-"}

// resolvePipeline looks for a pipeline by semantic base name, trying:
// 1. Exact bare name (e.g. "debug")
// 2. Taxonomy-prefixed variants (e.g. "ops-debug", "impl-debug")
// 3. Forge-prefixed variants (e.g. "gh-debug") — legacy fallback
func resolvePipeline(catalog []string, prefix, base string) string {
	// Try bare name first (unified pipeline)
	if inCatalog(catalog, base) {
		return base
	}
	// Try taxonomy-prefixed variants
	for _, tp := range taxonomyPrefixes {
		prefixed := tp + base
		if inCatalog(catalog, prefixed) {
			return prefixed
		}
	}
	// Fall back to forge-prefixed (legacy)
	if prefix != "" {
		prefixed := prefix + "-" + base
		if inCatalog(catalog, prefixed) {
			return prefixed
		}
	}
	return ""
}

// inCatalog checks if a pipeline name exists in the catalog.
func inCatalog(catalog []string, name string) bool {
	for _, n := range catalog {
		if n == name {
			return true
		}
	}
	return false
}

// detectSequences checks if complementary pipelines are both proposed (e.g.
// research + implement) and adds a sequence proposal that chains them.
func detectSequences(proposals []ProposedPipeline, _ []string, _ string) []ProposedPipeline {
	byBase := make(map[string]int) // base name → index in proposals
	for i, p := range proposals {
		base := stripForgePrefix(p.Name)
		byBase[base] = i
	}

	// Known chains: research → implement
	chains := [][2]string{{"plan-research", "impl-issue"}}

	for _, chain := range chains {
		firstIdx, hasFirst := byBase[chain[0]]
		secondIdx, hasSecond := byBase[chain[1]]
		if !hasFirst || !hasSecond {
			continue
		}
		first := proposals[firstIdx]
		second := proposals[secondIdx]
		proposals = append(proposals, ProposedPipeline{
			Name:     first.Name + " → " + second.Name,
			Reason:   fmt.Sprintf("Chain: %s then %s", first.Name, second.Name),
			Input:    first.Input,
			Priority: first.Priority, // Use higher priority of the pair
			Type:     "sequence",
			Sequence: []string{first.Name, second.Name},
		})
	}

	return proposals
}

// detectParallelGroups marks proposals that can run concurrently. Currently
// detects implement + ops-pr-review as parallel-eligible.
func detectParallelGroups(proposals []ProposedPipeline) []ProposedPipeline {
	byBase := make(map[string]int)
	for i, p := range proposals {
		if p.Type != "" {
			continue // Skip existing sequence proposals
		}
		base := stripForgePrefix(p.Name)
		byBase[base] = i
	}

	// Known parallel groups
	groups := [][2]string{{"impl-issue", "ops-pr-review"}}

	for _, group := range groups {
		firstIdx, hasFirst := byBase[group[0]]
		secondIdx, hasSecond := byBase[group[1]]
		if !hasFirst || !hasSecond {
			continue
		}
		first := proposals[firstIdx]
		second := proposals[secondIdx]
		proposals = append(proposals, ProposedPipeline{
			Name:     first.Name + " ∥ " + second.Name,
			Reason:   fmt.Sprintf("Parallel: %s and %s can run concurrently", first.Name, second.Name),
			Input:    first.Input,
			Priority: first.Priority,
			Type:     "parallel",
			Sequence: []string{first.Name, second.Name},
		})
	}

	return proposals
}

// forgePrefixes contains the known forge pipeline name prefixes.
var forgePrefixes = []string{"gh-", "gl-", "bb-", "gt-"}

// stripForgePrefix removes forge prefixes (gh-, gl-, bb-, gt-, cb-) from a pipeline name.
func stripForgePrefix(name string) string {
	for _, p := range forgePrefixes {
		if strings.HasPrefix(name, p) {
			return name[len(p):]
		}
	}
	return name
}
