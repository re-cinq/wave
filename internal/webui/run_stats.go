package webui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/recinq/wave/internal/state"
)

// enrichSummariesWithRuns adds Wave pipeline run stats to issue summaries.
func enrichSummariesWithRuns(summaries []IssueSummary, runs []state.RunRecord, matchType string) {
	// Build a map of issue/PR number -> runs
	runMap := make(map[string][]state.RunRecord)
	for _, run := range runs {
		if run.Input == "" {
			continue
		}
		// Match by /issues/<number> or /pull/<number> pattern
		for _, sep := range []string{"/issues/", "/pull/"} {
			idx := strings.Index(run.Input, sep)
			if idx >= 0 {
				rest := run.Input[idx+len(sep):]
				end := strings.IndexByte(rest, '/')
				if end > 0 {
					rest = rest[:end]
				}
				if _, err := strconv.Atoi(rest); err == nil {
					key := sep + rest
					runMap[key] = append(runMap[key], run)
				}
			}
		}
	}

	for i := range summaries {
		key := "/" + matchType + "/" + strconv.Itoa(summaries[i].Number)
		matching := runMap[key]
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

// enrichPRSummariesWithRuns adds Wave pipeline run stats to PR summaries.
func enrichPRSummariesWithRuns(summaries []PRSummary, runs []state.RunRecord) {
	runMap := make(map[string][]state.RunRecord)
	for _, run := range runs {
		if run.Input == "" {
			continue
		}
		idx := strings.Index(run.Input, "/pull/")
		if idx >= 0 {
			rest := run.Input[idx+len("/pull/"):]
			end := strings.IndexByte(rest, '/')
			if end > 0 {
				rest = rest[:end]
			}
			if _, err := strconv.Atoi(rest); err == nil {
				runMap[rest] = append(runMap[rest], run)
			}
		}
	}

	for i := range summaries {
		matching := runMap[strconv.Itoa(summaries[i].Number)]
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
