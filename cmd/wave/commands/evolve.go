package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// EvolveProposal represents a single suggested ontology change.
type EvolveProposal struct {
	Type        string  `json:"type"` // "prune", "promote", "demote", "low_signal"
	Context     string  `json:"context"`
	Invariant   string  `json:"invariant,omitempty"`
	Reason      string  `json:"reason"`
	SuccessRate float64 `json:"success_rate,omitempty"`
	TotalRuns   int     `json:"total_runs"`
}

// EvolveResult holds all evolution proposals.
type EvolveResult struct {
	TotalRuns    int                   `json:"total_runs"`
	Proposals    []EvolveProposal      `json:"proposals"`
	ContextStats []state.OntologyStats `json:"context_stats"`
}

// Evolution thresholds.
const (
	evolvePruneMinRuns      = 50
	evolvePromoteMinRuns    = 20
	evolvePromoteMinSuccess = 100.0
	evolveDemoteMinRuns     = 10
	evolveDemoteMaxSuccess  = 80.0
	evolveLowSignalMinRuns  = 5
)

func runEvolve(cmd *cobra.Command, m *manifest.Manifest, manifestPath string) error {
	outputCfg := GetOutputConfig(cmd)
	format := ResolveFormat(cmd, "text")
	if outputCfg.Format == OutputFormatJSON {
		format = "json"
	}

	f := display.NewFormatter()

	// Open state store in read-only mode
	store, err := state.NewReadOnlyStateStore(".wave/state.db")
	if err != nil {
		return NewCLIError(CodeStateDBError,
			fmt.Sprintf("failed to open state database: %s", err),
			"Ensure .wave/state.db exists (run a pipeline first)")
	}
	defer store.Close()

	// Get all ontology stats
	stats, err := store.GetOntologyStatsAll()
	if err != nil {
		return NewCLIError(CodeStateDBError,
			fmt.Sprintf("failed to query ontology stats: %s", err),
			"Check .wave/state.db integrity")
	}

	result := buildEvolveResult(m, stats)

	switch format {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	default:
		renderEvolveText(cmd.OutOrStdout(), result, f)
	}

	return nil
}

func buildEvolveResult(m *manifest.Manifest, stats []state.OntologyStats) *EvolveResult {
	result := &EvolveResult{
		ContextStats: stats,
	}

	// Calculate total runs across all contexts (use max as proxy for system-wide activity)
	for _, s := range stats {
		if s.TotalRuns > result.TotalRuns {
			result.TotalRuns = s.TotalRuns
		}
	}

	// Build stats lookup by context name
	statsMap := make(map[string]*state.OntologyStats)
	for i := range stats {
		statsMap[stats[i].ContextName] = &stats[i]
	}

	if m.Ontology == nil {
		return result
	}

	// Check each declared context against lineage data
	for _, ctx := range m.Ontology.Contexts {
		stat, found := statsMap[ctx.Name]

		if !found || stat.TotalRuns == 0 {
			// Context declared but never injected
			if result.TotalRuns >= evolvePruneMinRuns {
				result.Proposals = append(result.Proposals, EvolveProposal{
					Type:      "prune",
					Context:   ctx.Name,
					Reason:    "never injected into any pipeline step",
					TotalRuns: 0,
				})
			} else {
				result.Proposals = append(result.Proposals, EvolveProposal{
					Type:      "low_signal",
					Context:   ctx.Name,
					Reason:    "insufficient run data for analysis",
					TotalRuns: 0,
				})
			}
			continue
		}

		if stat.TotalRuns < evolveLowSignalMinRuns {
			result.Proposals = append(result.Proposals, EvolveProposal{
				Type:        "low_signal",
				Context:     ctx.Name,
				Reason:      fmt.Sprintf("only %d runs — need at least %d for reliable signal", stat.TotalRuns, evolveLowSignalMinRuns),
				TotalRuns:   stat.TotalRuns,
				SuccessRate: stat.SuccessRate,
			})
			continue
		}

		// Check for demotion: low success rate over sufficient runs
		if stat.SuccessRate < evolveDemoteMaxSuccess && stat.TotalRuns >= evolveDemoteMinRuns {
			result.Proposals = append(result.Proposals, EvolveProposal{
				Type:        "demote",
				Context:     ctx.Name,
				Reason:      fmt.Sprintf("%.1f%% success rate below %.0f%% threshold (%d runs)", stat.SuccessRate, evolveDemoteMaxSuccess, stat.TotalRuns),
				TotalRuns:   stat.TotalRuns,
				SuccessRate: stat.SuccessRate,
			})
		}

		// Check for promotion: perfect success over many runs
		if stat.SuccessRate >= evolvePromoteMinSuccess && stat.TotalRuns >= evolvePromoteMinRuns {
			result.Proposals = append(result.Proposals, EvolveProposal{
				Type:        "promote",
				Context:     ctx.Name,
				Reason:      fmt.Sprintf("%.0f%% success rate across %d runs — candidate for invariant promotion", stat.SuccessRate, stat.TotalRuns),
				TotalRuns:   stat.TotalRuns,
				SuccessRate: stat.SuccessRate,
			})
		}
	}

	return result
}

func renderEvolveText(w io.Writer, result *EvolveResult, f *display.Formatter) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Analyzing %d pipeline runs with ontology data...\n\n", result.TotalRuns)

	if len(result.ContextStats) > 0 {
		fmt.Fprintf(w, "  %s\n\n", f.Colorize("Context Statistics:", "\033[1;37m"))
		for _, s := range result.ContextStats {
			icon := f.Success("v")
			if s.SuccessRate < evolveDemoteMaxSuccess {
				icon = f.Error("x")
			}
			fmt.Fprintf(w, "    %s %s — %.1f%% success rate (%d runs)\n",
				icon, f.Primary(s.ContextName), s.SuccessRate, s.TotalRuns)
		}
		fmt.Fprintln(w)
	}

	if len(result.Proposals) == 0 {
		fmt.Fprintf(w, "  %s No changes proposed. Ontology is stable.\n\n", f.Success("v"))
		return
	}

	fmt.Fprintf(w, "  %s\n\n", f.Colorize("Proposed Changes:", "\033[1;37m"))
	for _, p := range result.Proposals {
		switch p.Type {
		case "prune":
			fmt.Fprintf(w, "    %s %s %s: %s\n", f.Error("x"), f.Warning("[prune]"), f.Primary(p.Context), p.Reason)
		case "promote":
			fmt.Fprintf(w, "    %s %s %s: %s\n", f.Success("^"), f.Success("[promote]"), f.Primary(p.Context), p.Reason)
		case "demote":
			fmt.Fprintf(w, "    %s %s %s: %s\n", f.Warning("!"), f.Warning("[demote]"), f.Primary(p.Context), p.Reason)
		case "low_signal":
			fmt.Fprintf(w, "    %s %s %s: %s\n", f.Muted("?"), f.Muted("[low signal]"), f.Primary(p.Context), p.Reason)
		}
	}
	fmt.Fprintln(w)
}
