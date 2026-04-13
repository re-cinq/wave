package pipeline

import (
	"fmt"
	"regexp"
	"strconv"
)

// ConvergenceTracker monitors rework loop progress by tracking validation
// scores across rounds. If scores plateau (no meaningful improvement over
// a sliding window), the tracker signals a stall so the loop can bail early
// instead of burning tokens on fruitless retries.
type ConvergenceTracker struct {
	scores         []float64
	window         int
	minImprovement float64
}

// NewConvergenceTracker creates a tracker with the given window size and
// minimum improvement threshold. Window is the number of recent scores
// to compare; minImprovement is the minimum score delta required to
// consider the loop as making progress (e.g. 0.05 = 5%).
func NewConvergenceTracker(window int, minImprovement float64) *ConvergenceTracker {
	if window < 2 {
		window = 2
	}
	if minImprovement <= 0 {
		minImprovement = 0.05
	}
	return &ConvergenceTracker{
		window:         window,
		minImprovement: minImprovement,
	}
}

// RecordScore appends a new score (0.0–1.0) to the history.
func (ct *ConvergenceTracker) RecordScore(score float64) {
	ct.scores = append(ct.scores, score)
}

// IsStalled returns true if the last `window` scores show no meaningful
// improvement. Requires at least `window` scores recorded.
func (ct *ConvergenceTracker) IsStalled() bool {
	if len(ct.scores) < ct.window {
		return false
	}

	recent := ct.scores[len(ct.scores)-ct.window:]
	first := recent[0]
	last := recent[len(recent)-1]

	improvement := last - first
	return improvement < ct.minImprovement
}

// LastScore returns the most recently recorded score, or 0 if none.
func (ct *ConvergenceTracker) LastScore() float64 {
	if len(ct.scores) == 0 {
		return 0
	}
	return ct.scores[len(ct.scores)-1]
}

// Rounds returns the number of scores recorded.
func (ct *ConvergenceTracker) Rounds() int {
	return len(ct.scores)
}

// Summary returns a human-readable summary of convergence state.
func (ct *ConvergenceTracker) Summary() string {
	if len(ct.scores) == 0 {
		return "no scores recorded"
	}
	return fmt.Sprintf("%.0f%% after %d round(s)", ct.LastScore()*100, len(ct.scores))
}

// scorePattern matches "Score: 75%" in ValidationError messages.
var scorePattern = regexp.MustCompile(`Score:\s*(\d+)%`)

// ExtractScoreFromError attempts to parse a score from a contract validation
// error message (e.g. "Score: 75% (threshold: 100%)"). Returns 0 and false
// if no score is found.
func ExtractScoreFromError(errMsg string) (float64, bool) {
	matches := scorePattern.FindStringSubmatch(errMsg)
	if len(matches) < 2 {
		return 0, false
	}
	pct, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, false
	}
	return pct / 100.0, true
}
