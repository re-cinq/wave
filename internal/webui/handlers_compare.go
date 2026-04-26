package webui

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/recinq/wave/internal/humanize"
	"github.com/recinq/wave/internal/state"
)

// CompareStepRow holds the comparison data for a single pipeline step.
type CompareStepRow struct {
	StepID        string `json:"step_id"`
	LeftState     string `json:"left_state"`
	RightState    string `json:"right_state"`
	LeftDuration  string `json:"left_duration"`
	RightDuration string `json:"right_duration"`
	LeftTokens    int    `json:"left_tokens"`
	RightTokens   int    `json:"right_tokens"`
	DeltaDuration string `json:"delta_duration"` // e.g. "+12s", "-3m 10s"
	DeltaTokens   string `json:"delta_tokens"`   // e.g. "+1.2k", "-500"
	DurationClass string `json:"duration_class"` // "compare-improvement", "compare-regression", or ""
	TokensClass   string `json:"tokens_class"`   // "compare-improvement", "compare-regression", or ""
	StateDiff     bool   `json:"state_diff"`     // true if left and right have different terminal states
}

// CompareResponse is the JSON response for the compare API.
type CompareResponse struct {
	Left          RunSummary       `json:"left"`
	Right         RunSummary       `json:"right"`
	Steps         []CompareStepRow `json:"steps"`
	DurationDelta string           `json:"duration_delta"`
	DurationClass string           `json:"duration_class"`
	TokensDelta   string           `json:"tokens_delta"`
	TokensClass   string           `json:"tokens_class"`
}

// comparePageData holds all fields the compare template can use.
type comparePageData struct {
	ActivePage       string
	ShowSelector     bool
	Error            string
	Runs             []RunSummary
	Left             RunSummary
	Right            RunSummary
	Rows             []CompareStepRow
	DurationDelta    string
	DurationClass    string
	TokensDelta      string
	TokensClass      string
	SamePipelineRuns []RunSummary
}

// handleComparePage handles GET /compare - renders a side-by-side comparison of two runs.
func (s *Server) handleComparePage(w http.ResponseWriter, r *http.Request) {
	leftID := r.URL.Query().Get("left")
	rightID := r.URL.Query().Get("right")

	if leftID == "" || rightID == "" {
		s.renderComparePage(w, comparePageData{
			ActivePage:   "runs",
			ShowSelector: true,
			Runs:         s.listRecentRuns(),
		})
		return
	}

	leftRun, err := s.store.GetRun(leftID)
	if err != nil {
		s.renderComparePage(w, comparePageData{
			ActivePage:   "runs",
			ShowSelector: true,
			Error:        "Left run not found: " + leftID,
			Runs:         s.listRecentRuns(),
		})
		return
	}

	rightRun, err := s.store.GetRun(rightID)
	if err != nil {
		s.renderComparePage(w, comparePageData{
			ActivePage:   "runs",
			ShowSelector: true,
			Error:        "Right run not found: " + rightID,
			Runs:         s.listRecentRuns(),
		})
		return
	}

	leftSummary := runToSummary(*leftRun)
	rightSummary := runToSummary(*rightRun)

	leftSteps := s.buildStepDetails(leftID, leftRun.PipelineName)
	rightSteps := s.buildStepDetails(rightID, rightRun.PipelineName)

	// Backfill tokens from steps if run-level is zero
	if leftSummary.TotalTokens == 0 {
		for _, sd := range leftSteps {
			leftSummary.TotalTokens += sd.TokensUsed
		}
	}
	if rightSummary.TotalTokens == 0 {
		for _, sd := range rightSteps {
			rightSummary.TotalTokens += sd.TokensUsed
		}
	}

	rows := buildCompareRows(leftSteps, rightSteps)

	// Compute run-level deltas
	durationDelta, durationClass := computeRunDurationDelta(leftRun, rightRun)
	tokensDelta, tokensClass := computeTokensDelta(leftSummary.TotalTokens, rightSummary.TotalTokens)

	// Fetch other runs of the same pipeline for the "Compare with..." dropdowns
	var samePipelineRuns []RunSummary
	if leftRun.PipelineName == rightRun.PipelineName {
		samePipelineRuns = s.listSamePipelineRuns(leftRun.PipelineName, leftID, rightID)
	}

	s.renderComparePage(w, comparePageData{
		ActivePage:       "runs",
		Left:             leftSummary,
		Right:            rightSummary,
		Rows:             rows,
		DurationDelta:    durationDelta,
		DurationClass:    durationClass,
		TokensDelta:      tokensDelta,
		TokensClass:      tokensClass,
		SamePipelineRuns: samePipelineRuns,
	})
}

// renderComparePage renders the compare template with the given data.
func (s *Server) renderComparePage(w http.ResponseWriter, data comparePageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := s.templates["templates/compare.html"]
	if tmpl == nil {
		http.Error(w, "compare template not found", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		log.Printf("[webui] template error rendering compare page: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// listRecentRuns fetches recent runs for the compare selector.
func (s *Server) listRecentRuns() []RunSummary {
	runs, err := s.store.ListRuns(state.ListRunsOptions{Limit: 50})
	if err != nil {
		log.Printf("[webui] compare: failed to list recent runs: %v", err)
		return nil
	}
	summaries := make([]RunSummary, 0, len(runs))
	for _, r := range runs {
		summaries = append(summaries, runToSummary(r))
	}
	return summaries
}

// handleAPICompare handles GET /api/compare - returns comparison data as JSON.
func (s *Server) handleAPICompare(w http.ResponseWriter, r *http.Request) {
	leftID := r.URL.Query().Get("left")
	rightID := r.URL.Query().Get("right")

	if leftID == "" || rightID == "" {
		writeJSONError(w, http.StatusBadRequest, "both left and right run IDs are required")
		return
	}

	leftRun, err := s.store.GetRun(leftID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "left run not found")
		return
	}

	rightRun, err := s.store.GetRun(rightID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "right run not found")
		return
	}

	leftSummary := runToSummary(*leftRun)
	rightSummary := runToSummary(*rightRun)

	leftSteps := s.buildStepDetails(leftID, leftRun.PipelineName)
	rightSteps := s.buildStepDetails(rightID, rightRun.PipelineName)

	if leftSummary.TotalTokens == 0 {
		for _, sd := range leftSteps {
			leftSummary.TotalTokens += sd.TokensUsed
		}
	}
	if rightSummary.TotalTokens == 0 {
		for _, sd := range rightSteps {
			rightSummary.TotalTokens += sd.TokensUsed
		}
	}

	rows := buildCompareRows(leftSteps, rightSteps)

	durationDelta, durationClass := computeRunDurationDelta(leftRun, rightRun)
	tokensDelta, tokensClass := computeTokensDelta(leftSummary.TotalTokens, rightSummary.TotalTokens)

	resp := CompareResponse{
		Left:          leftSummary,
		Right:         rightSummary,
		Steps:         rows,
		DurationDelta: durationDelta,
		DurationClass: durationClass,
		TokensDelta:   tokensDelta,
		TokensClass:   tokensClass,
	}

	writeJSON(w, http.StatusOK, resp)
}

// buildCompareRows matches steps by ID and computes per-step deltas.
func buildCompareRows(leftSteps, rightSteps []StepDetail) []CompareStepRow {
	rightMap := make(map[string]StepDetail, len(rightSteps))
	for _, s := range rightSteps {
		rightMap[s.StepID] = s
	}

	// Track which right steps have been matched
	matched := make(map[string]bool)

	var rows []CompareStepRow

	// First pass: iterate left steps and match with right
	for _, ls := range leftSteps {
		row := CompareStepRow{
			StepID:       ls.StepID,
			LeftState:    ls.State,
			LeftDuration: ls.Duration,
			LeftTokens:   ls.TokensUsed,
		}

		if rs, ok := rightMap[ls.StepID]; ok {
			matched[ls.StepID] = true
			row.RightState = rs.State
			row.RightDuration = rs.Duration
			row.RightTokens = rs.TokensUsed
			row.StateDiff = (ls.State != rs.State)

			// Duration delta
			row.DeltaDuration, row.DurationClass = computeStepDurationDelta(ls, rs)
			// Token delta
			row.DeltaTokens, row.TokensClass = computeTokensDelta(ls.TokensUsed, rs.TokensUsed)
		} else {
			row.RightState = "-"
			row.StateDiff = true
		}

		rows = append(rows, row)
	}

	// Second pass: right-only steps
	for _, rs := range rightSteps {
		if matched[rs.StepID] {
			continue
		}
		rows = append(rows, CompareStepRow{
			StepID:        rs.StepID,
			LeftState:     "-",
			RightState:    rs.State,
			RightDuration: rs.Duration,
			RightTokens:   rs.TokensUsed,
			StateDiff:     true,
		})
	}

	return rows
}

// computeStepDurationDelta computes a human-readable duration delta between two steps.
// A negative delta (right is faster) is an improvement; positive is a regression.
func computeStepDurationDelta(left, right StepDetail) (string, string) {
	leftDur := stepDuration(left)
	rightDur := stepDuration(right)

	if leftDur == 0 && rightDur == 0 {
		return "", ""
	}
	if leftDur == 0 || rightDur == 0 {
		return "", ""
	}

	delta := rightDur - leftDur
	if delta == 0 {
		return "same", ""
	}

	absDelta := delta
	if absDelta < 0 {
		absDelta = -absDelta
	}

	prefix := "+"
	class := "compare-regression"
	if delta < 0 {
		prefix = "-"
		class = "compare-improvement"
	}

	return prefix + humanize.Duration(absDelta), class
}

// computeRunDurationDelta computes the delta between two runs' total durations.
func computeRunDurationDelta(left, right *state.RunRecord) (string, string) {
	leftDur := runDuration(left)
	rightDur := runDuration(right)

	if leftDur == 0 && rightDur == 0 {
		return "", ""
	}
	if leftDur == 0 || rightDur == 0 {
		return "", ""
	}

	delta := rightDur - leftDur
	if delta == 0 {
		return "same", ""
	}

	absDelta := delta
	if absDelta < 0 {
		absDelta = -absDelta
	}

	prefix := "+"
	class := "compare-regression"
	if delta < 0 {
		prefix = "-"
		class = "compare-improvement"
	}

	return prefix + humanize.Duration(absDelta), class
}

// computeTokensDelta computes a human-readable token delta.
// Fewer tokens on the right is an improvement.
func computeTokensDelta(leftTokens, rightTokens int) (string, string) {
	if leftTokens == 0 && rightTokens == 0 {
		return "", ""
	}

	delta := rightTokens - leftTokens
	if delta == 0 {
		return "same", ""
	}

	prefix := "+"
	class := "compare-regression"
	if delta < 0 {
		prefix = "-"
		class = "compare-improvement"
	}

	absDelta := delta
	if absDelta < 0 {
		absDelta = -absDelta
	}

	return prefix + formatTokensDelta(absDelta), class
}

// formatTokensDelta formats a token count as a compact string (e.g. "1.2k", "45k").
func formatTokensDelta(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	k := float64(tokens) / 1000.0
	if k < 10 {
		s := fmt.Sprintf("%.1f", k)
		// Trim trailing ".0"
		if len(s) > 2 && s[len(s)-1] == '0' && s[len(s)-2] == '.' {
			s = s[:len(s)-2]
		}
		return s + "k"
	}
	return fmt.Sprintf("%dk", int(math.Round(k)))
}

// stepDuration extracts the actual duration from a StepDetail.
func stepDuration(sd StepDetail) time.Duration {
	if sd.StartedAt == nil {
		return 0
	}
	if sd.CompletedAt != nil {
		return sd.CompletedAt.Sub(*sd.StartedAt)
	}
	return 0
}

// runDuration extracts the actual duration from a RunRecord.
func runDuration(r *state.RunRecord) time.Duration {
	if r.CompletedAt != nil {
		return r.CompletedAt.Sub(r.StartedAt)
	}
	return 0
}

// listSamePipelineRuns fetches recent runs of the same pipeline, excluding the two being compared.
func (s *Server) listSamePipelineRuns(pipelineName, excludeLeft, excludeRight string) []RunSummary {
	runs, err := s.store.ListRuns(state.ListRunsOptions{
		PipelineName: pipelineName,
		Limit:        50,
	})
	if err != nil {
		log.Printf("[webui] compare: failed to list runs for pipeline %q: %v", pipelineName, err)
		return nil
	}

	var summaries []RunSummary
	for _, r := range runs {
		if r.RunID == excludeLeft || r.RunID == excludeRight {
			continue
		}
		summaries = append(summaries, runToSummary(r))
	}
	return summaries
}
