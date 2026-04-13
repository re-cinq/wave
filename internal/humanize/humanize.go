// Package humanize provides human-readable formatting helpers for durations,
// counts, and sizes. It is intentionally kept at a low level so that both
// domain packages (e.g. pipeline) and presentation packages (e.g. display)
// can import it without violating ADR-003 layer rules.
package humanize

import (
	"fmt"
	"time"
)

// Duration formats a time.Duration as a short human-readable string.
// Returns "-" for the zero duration.
func Duration(d time.Duration) string {
	if d == 0 {
		return "-"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	if minutes >= 60 {
		hours := minutes / 60
		minutes %= 60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

// TokenCount formats an integer token count as a short human-readable string
// (e.g. "1k", "45k", "1.5M"). Returns "-" for zero.
func TokenCount(n int) string {
	if n == 0 {
		return "-"
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000.0)
}

// FileSize formats a byte count as a short human-readable string
// (e.g. "512 B", "2.0 KB", "1.5 MB"). Returns "-" for zero.
func FileSize(bytes int64) string {
	if bytes == 0 {
		return "-"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}
