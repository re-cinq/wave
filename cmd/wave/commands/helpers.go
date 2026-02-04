package commands

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// formatSize formats bytes into human-readable format (KB, MB, etc.)
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// parseDuration parses duration strings like "7d", "24h", "1h30m".
// Extends time.ParseDuration to support day suffix (d).
func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	// Check for day suffix (not supported by time.ParseDuration)
	dayRegex := regexp.MustCompile(`^(\d+)d(.*)$`)
	if matches := dayRegex.FindStringSubmatch(s); len(matches) == 3 {
		days, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("invalid days value: %s", matches[1])
		}
		remaining := matches[2]
		var extraDuration time.Duration
		if remaining != "" {
			var err error
			extraDuration, err = time.ParseDuration(remaining)
			if err != nil {
				return 0, fmt.Errorf("invalid duration: %s", s)
			}
		}
		return time.Duration(days)*24*time.Hour + extraDuration, nil
	}

	return time.ParseDuration(s)
}

// formatTokens formats a token count with appropriate units.
// Uses integer-based thousands suffix (e.g., "1k", "45k", "1M").
func formatTokens(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	if tokens < 1000000 {
		return fmt.Sprintf("%dk", tokens/1000)
	}
	return fmt.Sprintf("%dM", tokens/1000000)
}

// truncateString truncates a string to maxLen and adds "..." if needed.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatElapsed formats a duration as "1m23s" or "1h23m".
func formatElapsed(d time.Duration) string {
	if d < 0 {
		d = -d
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}
