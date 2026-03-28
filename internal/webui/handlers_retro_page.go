package webui

import (
	"net/http"
	"sort"
	"time"

	"github.com/recinq/wave/internal/retro"
	"github.com/recinq/wave/internal/state"
)

// handleRetrosPage handles GET /retros - serves the HTML retrospective trends page.
func (s *Server) handleRetrosPage(w http.ResponseWriter, r *http.Request) {
	// Fetch recent retrospectives (last 20 for chart, up to 50 for list)
	records, err := s.store.ListRetrospectives(state.ListRetrosOptions{
		Limit: 50,
	})
	if err != nil {
		http.Error(w, "failed to list retrospectives", http.StatusInternalServerError)
		return
	}

	storage := retro.NewStorage(".wave/retros", s.store)

	// Build smoothness trend entries (last 20, reversed to chronological order for chart)
	chartLimit := 20
	if len(records) < chartLimit {
		chartLimit = len(records)
	}
	chartRecords := records[:chartLimit]

	// Reverse for chronological display (oldest first on the left)
	trendEntries := make([]RetroTrendEntry, len(chartRecords))
	for i, rec := range chartRecords {
		trendEntries[len(chartRecords)-1-i] = RetroTrendEntry{
			RunID:      rec.RunID,
			Pipeline:   rec.PipelineName,
			Smoothness: rec.Smoothness,
			HeightPct:  smoothnessToHeight(rec.Smoothness),
			CreatedAt:  rec.CreatedAt.Format("Jan 2 15:04"),
		}
	}

	// Aggregate friction points across all retros with narratives
	frictionCounts := make(map[string]int)
	pipelineStats := make(map[string]*pipelineAgg)

	for _, rec := range records {
		// Track pipeline stats from index records
		agg, ok := pipelineStats[rec.PipelineName]
		if !ok {
			agg = &pipelineAgg{}
			pipelineStats[rec.PipelineName] = agg
		}
		agg.total++
		if rec.Smoothness == "effortless" || rec.Smoothness == "smooth" {
			agg.successes++
		}

		// Load full retro for friction point aggregation (only complete ones)
		if rec.Status != "complete" {
			continue
		}
		fullRetro, loadErr := storage.Load(rec.RunID)
		if loadErr != nil {
			continue
		}
		if fullRetro.Narrative == nil {
			continue
		}

		// Aggregate duration
		if fullRetro.Quantitative != nil {
			agg.totalDurationMs += fullRetro.Quantitative.TotalDurationMs
			agg.durationCount++
		}

		for _, fp := range fullRetro.Narrative.FrictionPoints {
			frictionCounts[string(fp.Type)]++
		}
	}

	// Sort friction points by count descending
	frictionList := make([]FrictionCount, 0, len(frictionCounts))
	for typ, count := range frictionCounts {
		frictionList = append(frictionList, FrictionCount{Type: typ, Count: count})
	}
	sort.Slice(frictionList, func(i, j int) bool {
		return frictionList[i].Count > frictionList[j].Count
	})

	// Build pipeline success rate table
	pipelineRates := make([]PipelineSuccessRate, 0, len(pipelineStats))
	for name, agg := range pipelineStats {
		pct := 0
		if agg.total > 0 {
			pct = (agg.successes * 100) / agg.total
		}
		avgDur := "-"
		if agg.durationCount > 0 {
			avgMs := agg.totalDurationMs / int64(agg.durationCount)
			avgDur = formatDurationValue(time.Duration(avgMs) * time.Millisecond)
		}
		pipelineRates = append(pipelineRates, PipelineSuccessRate{
			Pipeline:    name,
			TotalRuns:   agg.total,
			SuccessPct:  pct,
			AvgDuration: avgDur,
		})
	}
	sort.Slice(pipelineRates, func(i, j int) bool {
		return pipelineRates[i].TotalRuns > pipelineRates[j].TotalRuns
	})

	// Build list entries
	listEntries := make([]RetroListEntry, len(records))
	for i, rec := range records {
		listEntries[i] = RetroListEntry{
			RunID:      rec.RunID,
			Pipeline:   rec.PipelineName,
			Smoothness: rec.Smoothness,
			Status:     rec.Status,
			CreatedAt:  rec.CreatedAt.Format("Jan 2 15:04"),
		}
	}

	data := struct {
		ActivePage     string
		Trend          []RetroTrendEntry
		FrictionPoints []FrictionCount
		PipelineRates  []PipelineSuccessRate
		Retros         []RetroListEntry
		HasData        bool
	}{
		ActivePage:     "retros",
		Trend:          trendEntries,
		FrictionPoints: frictionList,
		PipelineRates:  pipelineRates,
		Retros:         listEntries,
		HasData:        len(records) > 0,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/retros.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// pipelineAgg accumulates stats for a single pipeline.
type pipelineAgg struct {
	total           int
	successes       int
	totalDurationMs int64
	durationCount   int
}

// smoothnessToHeight converts a smoothness rating to a bar height percentage.
func smoothnessToHeight(smoothness string) int {
	switch smoothness {
	case "effortless":
		return 100
	case "smooth":
		return 80
	case "bumpy":
		return 60
	case "struggled":
		return 40
	case "failed":
		return 20
	default:
		return 10
	}
}

// smoothnessLabel returns a human-readable label for the smoothness CSS class.
// Registered in the template FuncMap in embed.go.
func smoothnessLabel(s string) string {
	switch s {
	case "effortless":
		return "Effortless"
	case "smooth":
		return "Smooth"
	case "bumpy":
		return "Bumpy"
	case "struggled":
		return "Struggled"
	case "failed":
		return "Failed"
	default:
		return s
	}
}

// frictionLabel returns a human-readable label for a friction type.
// Registered in the template FuncMap in embed.go.
func frictionLabel(s string) string {
	switch s {
	case "retry":
		return "Retries"
	case "timeout":
		return "Timeouts"
	case "wrong_approach":
		return "Wrong Approach"
	case "tool_failure":
		return "Tool Failures"
	case "ambiguity":
		return "Ambiguity"
	case "contract_failure":
		return "Contract Failures"
	default:
		return s
	}
}
