package webui

import (
	"log"
	"net/http"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/worksource"
)

// BridgeData backs templates/bridge.html.
type BridgeData struct {
	ActivePage string

	// Hero stats
	RunningCount    int
	ImplCount       int // running impl-issue pipelines
	ReviewCount     int // running pr-review pipelines
	CompletedToday int
	Spend           string // e.g. "$2.41" — displayed as-is from config
	BudgetCap       string // e.g. "$10.00" — displayed as-is from config
	JudgeAvg        string // e.g. "0.87"
	JudgeTrend      string // e.g. "-0.04" with direction

	// Activity
	RunningRuns    []RunSummary
	InFlightStats  []InFlightStat // pipeline breakdown for hero
	RecentLandings []RunSummary   // completed runs with status/judge info

	// Discover panel
	OpenWorkItems   int
	StaleBranches   int
	HealthHints     int
	BudgetWarning   bool

	// Proposals
	PendingProposalCount int

	// Heatmap (24 cells for last 24h, each 0-4 activity level)
	Heatmap [24]int

	// Activity stream (last N events for SSE)
	ActivityStream []ActivityEvent
}

// InFlightStat is a pipeline-type breakdown line shown in the hero.
type InFlightStat struct {
	Pipeline string
	Count    int
	Label    string // e.g. "implement" or "review"
}

// ActivityEvent is one live-activity row.
type ActivityEvent struct {
	Time    string // "14:22:04"
	Source  string // pipeline name, colored
	SourceClass string // CSS class: g (green), b (blue), y (yellow), r (red)
	Message string // human-readable event text
}

// handleBridge serves GET / — the post-onboard landing that shows fleet health.
func (s *Server) handleBridge(w http.ResponseWriter, r *http.Request) {
	tmpl, ok := s.assets.templates["templates/bridge.html"]
	if !ok || tmpl == nil {
		http.Error(w, "bridge template missing", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Running count and recent runs
	runningCount := 0
	completedToday := 0
	implCount := 0
	reviewCount := 0
	var recentLandings []RunSummary
	var runningRuns []RunSummary
	var heatmap [24]int

	if s.runtime.store != nil {
		runs, err := s.runtime.store.ListRuns(state.ListRunsOptions{Limit: 100})
		if err != nil {
			log.Printf("[webui] / bridge list runs: %v", err)
		} else {
			for _, run := range runs {
				if run.Status == "running" {
					runningCount++
					runningRuns = append(runningRuns, runToSummary(run))
					// pipeline-type breakdown
					switch run.PipelineName {
					case "impl-issue", "impl-finding":
						implCount++
					case "pr-review":
						reviewCount++
					}
				}
				if run.CompletedAt != nil && run.CompletedAt.After(todayStart) {
					completedToday++
				}
				// heatmap: bucket by hour of day (0-23)
				if !run.StartedAt.IsZero() {
					hour := run.StartedAt.Hour()
					heatmap[hour]++
				}
			}
			// Recent landings: last 10 completed (most recent first)
			for i := len(runs) - 1; i >= 0; i-- {
				run := runs[i]
				if run.Status != "running" {
					recentLandings = append(recentLandings, runToSummary(run))
					if len(recentLandings) >= 10 {
						break
					}
				}
			}
		}
	}

	// Active bindings — open work items
	openWorkItems := 0
	if s.runtime.worksource != nil {
		bindings, err := s.runtime.worksource.ListBindings(ctx, worksource.BindingFilter{})
		if err == nil {
			for _, b := range bindings {
				if b.Active {
					openWorkItems++
				}
			}
		}
	}

	// Proposal count
	pendingProposals := 0
	if s.runtime.store != nil {
		proposals, err := s.runtime.store.ListProposalsByStatus(state.ProposalProposed, 100)
		if err == nil {
			pendingProposals = len(proposals)
		}
	}

	inFlightStats := []InFlightStat{}
	if implCount > 0 {
		inFlightStats = append(inFlightStats, InFlightStat{Pipeline: "impl", Count: implCount, Label: "implement"})
	}
	if reviewCount > 0 {
		inFlightStats = append(inFlightStats, InFlightStat{Pipeline: "pr-review", Count: reviewCount, Label: "review"})
	}

	data := BridgeData{
		ActivePage:         "bridge",
		RunningCount:       runningCount,
		ImplCount:          implCount,
		ReviewCount:        reviewCount,
		CompletedToday:     completedToday,
		Spend:              "$0.00",   // TODO: wire from run cost tracking when available
		BudgetCap:          "$10.00",  // TODO: wire from config
		JudgeAvg:           "—",       // TODO: compute from run verdict aggregates
		JudgeTrend:         "",        // TODO: compute from 7d trend
		InFlightStats:      inFlightStats,
		RunningRuns:        runningRuns,
		RecentLandings:     recentLandings,
		OpenWorkItems:      openWorkItems,
		StaleBranches:      0,         // TODO: wire from forge client
		HealthHints:        0,         // TODO: wire from health checks
		BudgetWarning:      false,     // TODO: compare spend vs cap
		PendingProposalCount: pendingProposals,
		Heatmap:            heatmap,
		ActivityStream:     []ActivityEvent{}, // SSE-powered live stream
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("[webui] / bridge render: %v", err)
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}
