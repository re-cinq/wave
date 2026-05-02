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
	ActivePage      string
	RunningCount    int
	CompletedToday int
	ActiveBindings int
	ProposalCount  int
	RunningRuns    []RunSummary
	RecentRuns     []RunSummary
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
	var recentRuns []RunSummary
	var runningRuns []RunSummary
	if s.runtime.store != nil {
		runs, err := s.runtime.store.ListRuns(state.ListRunsOptions{Limit: 50})
		if err != nil {
			log.Printf("[webui] / bridge list runs: %v", err)
		} else {
			for _, run := range runs {
				if run.Status == "running" {
					runningCount++
					runningRuns = append(runningRuns, runToSummary(run))
				}
				if run.CompletedAt != nil && run.CompletedAt.After(todayStart) {
					completedToday++
				}
			}
			// Recent: last 10 completed (reverse to get most recent first)
			for i := len(runs) - 1; i >= 0; i-- {
				run := runs[i]
				if run.Status != "running" {
					recentRuns = append(recentRuns, runToSummary(run))
					if len(recentRuns) >= 10 {
						break
					}
				}
			}
		}
	}

	// Active bindings
	activeBindings := 0
	if s.runtime.worksource != nil {
		bindings, err := s.runtime.worksource.ListBindings(ctx, worksource.BindingFilter{})
		if err == nil {
			for _, b := range bindings {
				if b.Active {
					activeBindings++
				}
			}
		}
	}

	data := BridgeData{
		ActivePage:      "bridge",
		RunningCount:    runningCount,
		CompletedToday:  completedToday,
		ActiveBindings: activeBindings,
		ProposalCount:  0,
		RunningRuns:     runningRuns,
		RecentRuns:      recentRuns,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("[webui] / bridge render: %v", err)
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}
