package webui

import (
	"log"
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
		smoothness := rec.Smoothness
		if smoothness == "" {
			// Derive smoothness from quantitative data when narrative hasn't been generated
			if fullRetro, err := storage.Load(rec.RunID); err == nil && fullRetro.Quantitative != nil {
				smoothness = deriveSmoothness(fullRetro.Quantitative)
			}
		}
		trendEntries[len(chartRecords)-1-i] = RetroTrendEntry{
			RunID:      rec.RunID,
			Pipeline:   rec.PipelineName,
			Smoothness: smoothness,
			HeightPct:  smoothnessToHeight(smoothness),
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

		// Load full retro for duration aggregation and friction points
		fullRetro, loadErr := storage.Load(rec.RunID)
		if loadErr != nil {
			continue
		}

		// Determine smoothness: use narrative if available, otherwise derive from quantitative
		smoothness := rec.Smoothness
		if smoothness == "" && fullRetro.Quantitative != nil {
			smoothness = deriveSmoothness(fullRetro.Quantitative)
		}
		if smoothness == "effortless" || smoothness == "smooth" || smoothness == "bumpy" {
			agg.successes++
		}

		// Aggregate duration from quantitative data
		if fullRetro.Quantitative != nil {
			agg.totalDurationMs += fullRetro.Quantitative.TotalDurationMs
			agg.durationCount++
		}

		// Aggregate friction points from narrative (only complete retros)
		if fullRetro.Narrative == nil {
			continue
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
		log.Printf("[webui] template error rendering retros page: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
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

// deriveSmoothness infers a smoothness rating from quantitative metrics
// when the narrative phase hasn't been run.
func deriveSmoothness(q *retro.QuantitativeData) string {
	if q.TotalSteps == 0 {
		return "bumpy"
	}
	failRatio := float64(q.FailureCount) / float64(q.TotalSteps)
	retryRatio := float64(q.TotalRetries) / float64(q.TotalSteps)

	switch {
	case q.FailureCount == 0 && q.TotalRetries == 0:
		return "effortless"
	case failRatio == 0 && retryRatio <= 0.5:
		return "smooth"
	case failRatio < 0.25:
		return "bumpy"
	case failRatio < 0.5:
		return "struggled"
	default:
		return "failed"
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
