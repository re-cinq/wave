package suggest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/doctor"
	"github.com/recinq/wave/internal/forge"
)

// Proposal contains a prioritized list of pipeline proposals.
type Proposal struct {
	Pipelines []ProposedPipeline `json:"pipelines"`
	Rationale string             `json:"rationale"`
}

// ProposedPipeline is a single pipeline recommendation.
type ProposedPipeline struct {
	Name     string `json:"name"`
	Reason   string `json:"reason"`
	Input    string `json:"input,omitempty"`
	Priority int    `json:"priority"`
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
		opts.PipelinesDir = ".wave/pipelines"
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

		// Priority 1: CI failing
		if cb.CI.Status == "failing" {
			if name := resolvePipeline(catalog, prefix, "debug"); name != "" {
				proposals = append(proposals, ProposedPipeline{
					Name:     name,
					Reason:   fmt.Sprintf("CI has %d recent failures", cb.CI.Failures),
					Input:    "Fix failing CI",
					Priority: 1,
				})
			}
		}

		// Priority 2: Poor quality issues
		if cb.Issues.PoorQuality > 0 {
			if name := resolvePipeline(catalog, prefix, "rewrite"); name != "" {
				proposals = append(proposals, ProposedPipeline{
					Name:     name,
					Reason:   fmt.Sprintf("%d issues with poor quality scores", cb.Issues.PoorQuality),
					Input:    "Rewrite poor quality issues",
					Priority: 2,
				})
			}
		}

		// Priority 3: Open issues without linked PRs
		if cb.Issues.Open > 0 {
			if name := resolvePipeline(catalog, prefix, "implement"); name != "" {
				proposals = append(proposals, ProposedPipeline{
					Name:     name,
					Reason:   fmt.Sprintf("%d open issues to work on", cb.Issues.Open),
					Input:    "Implement open issues",
					Priority: 3,
				})
			} else if name := resolvePipeline(catalog, prefix, "research"); name != "" {
				proposals = append(proposals, ProposedPipeline{
					Name:     name,
					Reason:   fmt.Sprintf("%d open issues to research", cb.Issues.Open),
					Input:    "Research open issues",
					Priority: 3,
				})
			}
		}

		// Priority 4: PRs needing review
		if cb.PRs.NeedsReview > 0 {
			if name := resolvePipeline(catalog, prefix, "pr-review"); name != "" {
				proposals = append(proposals, ProposedPipeline{
					Name:     name,
					Reason:   fmt.Sprintf("%d PRs awaiting review", cb.PRs.NeedsReview),
					Input:    "Review pending PRs",
					Priority: 4,
				})
			}
		}

		// Priority 5: Stale PRs
		if cb.PRs.Stale > 0 {
			if name := resolvePipeline(catalog, prefix, "refresh"); name != "" {
				proposals = append(proposals, ProposedPipeline{
					Name:     name,
					Reason:   fmt.Sprintf("%d stale PRs (>14 days)", cb.PRs.Stale),
					Input:    "Refresh stale PRs",
					Priority: 5,
				})
			}
		}
	}

	// Priority 6: Clean state — always suggest improvement pipelines
	if len(proposals) == 0 {
		if name := resolvePipeline(catalog, prefix, "improve"); name != "" {
			proposals = append(proposals, ProposedPipeline{
				Name:     name,
				Reason:   "Codebase is healthy — suggest improvements",
				Input:    "General improvement",
				Priority: 6,
			})
		}
		if name := resolvePipeline(catalog, prefix, "refactor"); name != "" {
			proposals = append(proposals, ProposedPipeline{
				Name:     name,
				Reason:   "Codebase is healthy — suggest refactoring",
				Input:    "General refactoring",
				Priority: 6,
			})
		}
	}

	// Filter to available pipelines and apply limit
	var available []ProposedPipeline
	for _, p := range proposals {
		if inCatalog(catalog, p.Name) {
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

// resolvePipeline looks for a forge-prefixed pipeline first, then falls back to generic.
func resolvePipeline(catalog []string, prefix, base string) string {
	if prefix != "" {
		prefixed := prefix + "-" + base
		if inCatalog(catalog, prefixed) {
			return prefixed
		}
	}
	if inCatalog(catalog, base) {
		return base
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

// FilterByForge filters proposals to those matching the forge's pipeline prefix.
func FilterByForge(proposals []ProposedPipeline, fi *forge.ForgeInfo) []ProposedPipeline {
	if fi == nil || fi.PipelinePrefix == "" {
		return proposals
	}
	prefix := fi.PipelinePrefix + "-"
	var filtered []ProposedPipeline
	for _, p := range proposals {
		if strings.HasPrefix(p.Name, prefix) || !hasAnyForgePrefix(p.Name) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func hasAnyForgePrefix(name string) bool {
	prefixes := []string{"gh-", "gl-", "bb-", "gt-"}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
