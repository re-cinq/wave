//go:build analytics

package webui

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/recinq/wave/internal/state"
)

// TokenAnalytics holds all aggregated token usage data for the analytics page.
type TokenAnalytics struct {
	// Summary stats
	TotalTokens      int
	TotalRuns        int
	TokensThisWeek   int
	TokensThisMonth  int
	RunsThisWeek     int
	RunsThisMonth    int
	EstCostThisWeek  string // formatted dollar amount
	EstCostThisMonth string

	// Top pipelines by average tokens
	TopPipelines []PipelineTokenStat

	// Top personas by total tokens
	TopPersonas []PersonaTokenStat

	// Recent runs for bar chart (last 20)
	RecentRuns []RunTokenPoint

	// Max tokens in recent runs (for chart scaling)
	MaxRunTokens int
}

// PipelineTokenStat holds aggregated token stats for a single pipeline.
type PipelineTokenStat struct {
	Name        string
	AvgTokens   int
	TotalTokens int
	RunCount    int
	Pct         int // percentage of max for bar width
}

// PersonaTokenStat holds aggregated token stats for a single persona.
type PersonaTokenStat struct {
	Name        string
	TotalTokens int
	StepCount   int
	AvgTokens   int
	Pct         int
	EstCost     string
}

// RunTokenPoint holds token data for a single run (used in charts).
type RunTokenPoint struct {
	RunID        string
	PipelineName string
	Tokens       int
	Pct          int // percentage of max for bar height
	Status       string
	StartedAt    string // formatted date
}

// handleAnalyticsPage handles GET /analytics - serves the token usage analytics page.
func (s *Server) handleAnalyticsPage(w http.ResponseWriter, r *http.Request) {
	analytics := s.buildTokenAnalytics()

	data := struct {
		ActivePage string
		Analytics  TokenAnalytics
	}{
		ActivePage: "analytics",
		Analytics:  analytics,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/analytics.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		log.Printf("[webui] template error rendering analytics page: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// handleAPIAnalytics handles GET /api/analytics - returns token analytics as JSON.
func (s *Server) handleAPIAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics := s.buildTokenAnalytics()
	writeJSON(w, http.StatusOK, analytics)
}

// buildTokenAnalytics aggregates token usage data from runs and performance metrics.
func (s *Server) buildTokenAnalytics() TokenAnalytics {
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, -1, 0)

	var analytics TokenAnalytics

	// Fetch recent runs (up to 200 for aggregation)
	runs, err := s.store.ListRuns(state.ListRunsOptions{Limit: 200})
	if err != nil {
		log.Printf("[webui] analytics: failed to list runs: %v", err)
		return analytics
	}

	analytics.TotalRuns = len(runs)

	// Aggregate by pipeline and time periods
	pipelineTokens := make(map[string]int)
	pipelineRuns := make(map[string]int)

	for _, run := range runs {
		tokens := run.TotalTokens
		analytics.TotalTokens += tokens
		pipelineTokens[run.PipelineName] += tokens
		pipelineRuns[run.PipelineName]++

		if run.StartedAt.After(weekAgo) {
			analytics.TokensThisWeek += tokens
			analytics.RunsThisWeek++
		}
		if run.StartedAt.After(monthAgo) {
			analytics.TokensThisMonth += tokens
			analytics.RunsThisMonth++
		}
	}

	// Cost estimation: $3/MTok input, $15/MTok output
	// Assume ~80% input, ~20% output as a rough split
	analytics.EstCostThisWeek = estimateCost(analytics.TokensThisWeek)
	analytics.EstCostThisMonth = estimateCost(analytics.TokensThisMonth)

	// Top pipelines by average tokens
	for name, total := range pipelineTokens {
		count := pipelineRuns[name]
		avg := 0
		if count > 0 {
			avg = total / count
		}
		analytics.TopPipelines = append(analytics.TopPipelines, PipelineTokenStat{
			Name:        name,
			AvgTokens:   avg,
			TotalTokens: total,
			RunCount:    count,
		})
	}
	sort.Slice(analytics.TopPipelines, func(i, j int) bool {
		return analytics.TopPipelines[i].AvgTokens > analytics.TopPipelines[j].AvgTokens
	})
	if len(analytics.TopPipelines) > 5 {
		analytics.TopPipelines = analytics.TopPipelines[:5]
	}
	// Calculate percentage for bar widths
	if len(analytics.TopPipelines) > 0 {
		maxAvg := analytics.TopPipelines[0].AvgTokens
		for i := range analytics.TopPipelines {
			if maxAvg > 0 {
				analytics.TopPipelines[i].Pct = (analytics.TopPipelines[i].AvgTokens * 100) / maxAvg
			}
		}
	}

	// Per-persona breakdown from performance metrics
	metrics, err := s.store.GetRecentPerformanceHistory(state.PerformanceQueryOptions{
		Limit: 1000,
	})
	if err != nil {
		log.Printf("[webui] analytics: failed to get performance metrics: %v", err)
	} else {
		personaTokens := make(map[string]int)
		personaSteps := make(map[string]int)
		for _, m := range metrics {
			persona := resolveForgeVars(m.Persona)
			if persona == "" {
				continue
			}
			personaTokens[persona] += m.TokensUsed
			personaSteps[persona]++
		}

		for name, total := range personaTokens {
			steps := personaSteps[name]
			avg := 0
			if steps > 0 {
				avg = total / steps
			}
			analytics.TopPersonas = append(analytics.TopPersonas, PersonaTokenStat{
				Name:        name,
				TotalTokens: total,
				StepCount:   steps,
				AvgTokens:   avg,
				EstCost:     estimateCost(total),
			})
		}
		sort.Slice(analytics.TopPersonas, func(i, j int) bool {
			return analytics.TopPersonas[i].TotalTokens > analytics.TopPersonas[j].TotalTokens
		})
		if len(analytics.TopPersonas) > 5 {
			analytics.TopPersonas = analytics.TopPersonas[:5]
		}
		if len(analytics.TopPersonas) > 0 {
			maxTotal := analytics.TopPersonas[0].TotalTokens
			for i := range analytics.TopPersonas {
				if maxTotal > 0 {
					analytics.TopPersonas[i].Pct = (analytics.TopPersonas[i].TotalTokens * 100) / maxTotal
				}
			}
		}
	}

	// Recent runs for bar chart (last 20, oldest first for left-to-right time)
	recentLimit := 20
	if len(runs) < recentLimit {
		recentLimit = len(runs)
	}
	recentRuns := runs[:recentLimit]
	// Reverse to show oldest-to-newest left-to-right
	for i, j := 0, len(recentRuns)-1; i < j; i, j = i+1, j-1 {
		recentRuns[i], recentRuns[j] = recentRuns[j], recentRuns[i]
	}

	maxTokens := 0
	for _, run := range recentRuns {
		if run.TotalTokens > maxTokens {
			maxTokens = run.TotalTokens
		}
	}
	analytics.MaxRunTokens = maxTokens

	for _, run := range recentRuns {
		pct := 0
		if maxTokens > 0 {
			pct = (run.TotalTokens * 100) / maxTokens
		}
		if pct < 2 && run.TotalTokens > 0 {
			pct = 2 // minimum visible bar
		}
		analytics.RecentRuns = append(analytics.RecentRuns, RunTokenPoint{
			RunID:        run.RunID,
			PipelineName: run.PipelineName,
			Tokens:       run.TotalTokens,
			Pct:          pct,
			Status:       run.Status,
			StartedAt:    run.StartedAt.Format("Jan 2 15:04"),
		})
	}

	return analytics
}

// estimateCost estimates the dollar cost for a given token count.
// Uses Claude Sonnet pricing: $3/MTok input, $15/MTok output.
// Assumes approximately 80% input tokens, 20% output tokens.
func estimateCost(tokens int) string {
	if tokens == 0 {
		return "$0.00"
	}
	inputTokens := float64(tokens) * 0.80
	outputTokens := float64(tokens) * 0.20
	cost := (inputTokens * 3.0 / 1_000_000) + (outputTokens * 15.0 / 1_000_000)
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}
