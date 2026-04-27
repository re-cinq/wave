package webui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/recinq/wave/internal/state"
)

// enrichSummariesWithRuns adds Wave pipeline run stats to issue summaries.
func enrichSummariesWithRuns(summaries []IssueSummary, runs []state.RunRecord, _ string) {
	// Build a map of issue/PR number -> runs
	// Matches both URL patterns (/issues/700, /pull/700) and
	// short-form input ("owner/repo 700")
	runMap := make(map[int][]state.RunRecord)
	for _, run := range runs {
		if run.Input == "" {
			continue
		}
		if num := extractIssueNumber(run.Input); num > 0 {
			runMap[num] = append(runMap[num], run)
		}
	}

	for i := range summaries {
		matching := runMap[summaries[i].Number]
		if len(matching) == 0 {
			continue
		}
		summaries[i].RunCount = len(matching)
		summaries[i].LastStatus = matching[0].Status
		for _, r := range matching {
			summaries[i].TotalTokens += int64(r.TotalTokens)
		}
	}
}

// extractIssueNumber parses an issue/PR number from a run input string.
// Supported formats:
//   - URL: "https://github.com/owner/repo/issues/700"
//   - URL: "https://github.com/owner/repo/pull/700"
//   - Short: "owner/repo 700"
//   - Short with suffix: "owner/repo 700 -- some comment"
//
// Returns 0 if no number could be extracted.
func extractIssueNumber(input string) int {
	// Try URL patterns first: /issues/<N> or /pull/<N>
	for _, sep := range []string{"/issues/", "/pull/"} {
		idx := strings.Index(input, sep)
		if idx >= 0 {
			rest := input[idx+len(sep):]
			// Take only digits
			end := 0
			for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
				end++
			}
			if end > 0 {
				if n, err := strconv.Atoi(rest[:end]); err == nil {
					return n
				}
			}
		}
	}

	// Try short form: "owner/repo <number>" or "owner/repo <number> -- ..."
	// Must contain exactly one "/" before the space+number to qualify as owner/repo
	parts := strings.SplitN(input, " ", 3)
	if len(parts) >= 2 && strings.Count(parts[0], "/") == 1 {
		if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
			return n
		}
	}

	return 0
}

// enrichPRSummariesWithRuns adds Wave pipeline run stats to PR summaries.
// PRs match by /pull/<N> URL, branch name, short-form "owner/repo <N>",
// or outcome records (persisted PR URLs from pipeline outputs).
func enrichPRSummariesWithRuns(summaries []PRSummary, runs []state.RunRecord, store state.RunStore) {
	// Map by number (from URL/short-form) and by branch name
	numMap := make(map[int][]state.RunRecord)
	branchMap := make(map[string][]state.RunRecord)
	runByID := make(map[string]state.RunRecord)
	for _, run := range runs {
		runByID[run.RunID] = run
		if run.Input != "" {
			if num := extractIssueNumber(run.Input); num > 0 {
				numMap[num] = append(numMap[num], run)
			}
		}
		if run.BranchName != "" {
			branchMap[run.BranchName] = append(branchMap[run.BranchName], run)
		}
	}

	for i := range summaries {
		// Deduplicate: collect unique run IDs from all sources
		seen := make(map[string]bool)
		var matching []state.RunRecord
		for _, r := range numMap[summaries[i].Number] {
			if !seen[r.RunID] {
				seen[r.RunID] = true
				matching = append(matching, r)
			}
		}
		if summaries[i].HeadBranch != "" {
			for _, r := range branchMap[summaries[i].HeadBranch] {
				if !seen[r.RunID] {
					seen[r.RunID] = true
					matching = append(matching, r)
				}
			}
		}
		// Check persisted outcomes for runs that produced this PR URL
		if store != nil {
			prPattern := fmt.Sprintf("/pull/%d", summaries[i].Number)
			if outcomes, err := store.GetOutcomesByValue("pr", prPattern); err == nil {
				for _, o := range outcomes {
					if !seen[o.RunID] {
						if run, ok := runByID[o.RunID]; ok {
							seen[o.RunID] = true
							matching = append(matching, run)
						}
					}
				}
			}
		}
		if len(matching) == 0 {
			continue
		}
		summaries[i].RunCount = len(matching)
		summaries[i].LastStatus = matching[0].Status
		for _, r := range matching {
			summaries[i].TotalTokens += int64(r.TotalTokens)
		}
	}
}

// formatTokensShort formats token counts compactly (e.g., "1.2k", "45k").
func formatTokensShort(n int64) string {
	if n == 0 {
		return ""
	}
	if n < 1000 {
		return strconv.FormatInt(n, 10)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
}
