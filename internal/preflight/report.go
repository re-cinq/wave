package preflight

import (
	"fmt"
	"time"

	"github.com/recinq/wave/internal/manifest"
)

// SystemReadinessReport holds the aggregated results of all system readiness checks.
type SystemReadinessReport struct {
	Timestamp time.Time `json:"timestamp"`
	AllPassed bool      `json:"all_passed"`
	Checks    []Result  `json:"checks"`
	Summary   string    `json:"summary"`
}

// SystemReadinessOpts provides configuration for running system readiness checks.
type SystemReadinessOpts struct {
	Adapters  map[string]manifest.Adapter // Configured adapters to health-check
	RemoteURL string                      // Git remote URL for forge detection
	Skills    []string                    // Required skill names
	Tools     []string                    // Required tool names
	WaveDir   string                      // Path to .wave directory
}

// RunSystemReadiness runs all system readiness check categories and produces
// an aggregated report. Individual check failures do not prevent other categories
// from running.
func (c *Checker) RunSystemReadiness(opts SystemReadinessOpts) (*SystemReadinessReport, error) {
	report := &SystemReadinessReport{
		Timestamp: time.Now(),
		AllPassed: true,
	}

	// Check adapter health
	if len(opts.Adapters) > 0 {
		results, _ := c.CheckAdapterHealth(opts.Adapters)
		report.Checks = append(report.Checks, results...)
	}

	// Check forge CLI
	if opts.RemoteURL != "" {
		results, _ := c.CheckForgeCLI(opts.RemoteURL)
		report.Checks = append(report.Checks, results...)
	}

	// Check tools
	if len(opts.Tools) > 0 {
		results, _ := c.CheckTools(opts.Tools)
		report.Checks = append(report.Checks, results...)
	}

	// Check skills
	if len(opts.Skills) > 0 {
		results, _ := c.CheckSkills(opts.Skills)
		report.Checks = append(report.Checks, results...)
	}

	// Check Wave initialization
	if opts.WaveDir != "" {
		results, _ := c.CheckWaveInit(opts.WaveDir)
		report.Checks = append(report.Checks, results...)
	}

	// Compute overall pass/fail
	passed := 0
	failed := 0
	for _, check := range report.Checks {
		if check.OK {
			passed++
		} else {
			failed++
			report.AllPassed = false
		}
	}

	report.Summary = fmt.Sprintf("%d/%d checks passed", passed, passed+failed)
	if failed > 0 {
		report.Summary += fmt.Sprintf(" (%d failed)", failed)
	}

	return report, nil
}
