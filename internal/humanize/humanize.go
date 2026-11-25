// Package humanize provides human-readable formatting helpers for durations,
// counts, and sizes. It is intentionally kept at a low level so that both
// domain packages (e.g. pipeline) and presentation packages (e.g. display)
// can import it without violating ADR-003 layer rules.
package humanize

import (
	"fmt"
)

// DurationMs formats a duration given in milliseconds as a short
// human-readable string. Returns "-" for zero or negative input,
// "<1s" for sub-second values.
func DurationMs(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	if ms < 1000 {
		return "<1s"
	}
	return formatTotalSeconds(int(ms / 1000))
}

// formatTotalSeconds renders a second count as the canonical short form:
// "5s", "3m15s", "2h30m". Callers handle zero / sub-second pre-formatting.
func formatTotalSeconds(totalSeconds int) string {
	if totalSeconds < 60 {
		return fmt.Sprintf("%ds", totalSeconds)
	}
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	if minutes >= 60 {
		hours := minutes / 60
		minutes %= 60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}
