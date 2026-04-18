package commands

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/recinq/wave/internal/onboarding"
)

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

// checkOnboarding verifies that onboarding has been completed.
// It returns an error if onboarding is incomplete, directing the user to run 'wave init'.
// Existing projects that have a wave.yaml but no .onboarded marker are grandfathered in.
func checkOnboarding() error {
	if onboarding.IsOnboarded(".agents") {
		return nil
	}

	// Grandfather existing projects: if wave.yaml exists but no .onboarded marker,
	// assume the project was set up before onboarding was introduced.
	if _, err := os.Stat("wave.yaml"); err == nil {
		return nil
	}

	return fmt.Errorf("onboarding not complete\n\nRun 'wave init' to complete setup before running pipelines")
}
