package tui

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/meta"
	"github.com/recinq/wave/internal/platform"
	"github.com/stretchr/testify/assert"
)

func TestRenderHealthReport(t *testing.T) {
	now := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		report         *meta.HealthReport
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "full report with all fields populated",
			report: &meta.HealthReport{
				Timestamp: now,
				Duration:  1500 * time.Millisecond,
				Init: meta.InitCheckResult{
					ManifestFound:  true,
					ManifestValid:  true,
					WaveVersion:    "0.8.0",
					LastConfigDate: now.Add(-24 * time.Hour),
				},
				Dependencies: meta.DependencyReport{
					Tools: []meta.DependencyStatus{
						{Name: "git", Kind: "tool", Available: true, Message: `tool "git" found`},
						{Name: "claude", Kind: "tool", Available: true, Message: `tool "claude" found`},
					},
					Skills: []meta.DependencyStatus{
						{Name: "speckit", Kind: "skill", Available: true, AutoInstallable: true, Message: `skill "speckit" installed`},
					},
				},
				Codebase: meta.CodebaseMetrics{
					RecentCommits:  42,
					OpenIssueCount: 7,
					OpenPRCount:    3,
					PRsByStatus:    map[string]int{"open": 3},
					BranchCount:    12,
					LastCommitDate: now.Add(-2 * time.Hour),
					APIAvailable:   true,
					Source:         "github_api",
				},
				Platform: platform.PlatformProfile{
					Type:           platform.PlatformGitHub,
					Owner:          "recinq",
					Repo:           "wave",
					CLITool:        "gh",
					PipelineFamily: "gh",
				},
			},
			wantContains: []string{
				"Init",
				"Manifest found",
				"Manifest valid",
				"0.8.0",
				"Dependencies",
				"Tools",
				"git",
				"claude",
				"Skills",
				"speckit",
				"Codebase",
				"42",
				"7",
				"3",
				"12",
				"github_api",
				"Platform",
				"github",
				"recinq/wave",
				"gh",
				"Completed in",
			},
			wantNotContain: []string{
				"Errors",
			},
		},
		{
			name: "minimal report with errors and timeouts",
			report: &meta.HealthReport{
				Timestamp: now,
				Duration:  500 * time.Millisecond,
				Init: meta.InitCheckResult{
					ManifestFound: false,
					ManifestValid: false,
					WaveVersion:   "unknown",
					Error:         "manifest not found: wave.yaml",
				},
				Dependencies: meta.DependencyReport{
					Tools:  []meta.DependencyStatus{},
					Skills: []meta.DependencyStatus{},
				},
				Codebase: meta.CodebaseMetrics{
					PRsByStatus: map[string]int{},
				},
				Platform: platform.PlatformProfile{
					Type: platform.PlatformUnknown,
				},
				Errors: []meta.HealthCheckError{
					{Check: "init", Message: "init check timed out", Timeout: true},
					{Check: "codebase", Message: "git not available", Timeout: false},
				},
			},
			wantContains: []string{
				"Init",
				"manifest not found: wave.yaml",
				"unknown",
				"Dependencies",
				"No dependencies detected",
				"Codebase",
				"0", // zero values for metrics
				"Platform",
				"Errors",
				"init check timed out",
				"git not available",
			},
		},
		{
			name: "empty/zero values",
			report: &meta.HealthReport{
				Timestamp: time.Time{},
				Duration:  0,
				Init: meta.InitCheckResult{
					WaveVersion: "",
				},
				Dependencies: meta.DependencyReport{},
				Codebase: meta.CodebaseMetrics{
					PRsByStatus: map[string]int{},
				},
				Platform: platform.PlatformProfile{},
			},
			wantContains: []string{
				"Init",
				"Dependencies",
				"Codebase",
				"Platform",
				"none", // source defaults to "none" when empty
			},
			wantNotContain: []string{
				"Errors",
			},
		},
		{
			name: "dependency status indicators",
			report: &meta.HealthReport{
				Timestamp: now,
				Duration:  100 * time.Millisecond,
				Init: meta.InitCheckResult{
					ManifestFound: true,
					ManifestValid: true,
					WaveVersion:   "0.8.0",
				},
				Dependencies: meta.DependencyReport{
					Tools: []meta.DependencyStatus{
						{Name: "git", Kind: "tool", Available: true, Message: "found"},
						{Name: "claude", Kind: "tool", Available: false, Message: "not found"},
					},
					Skills: []meta.DependencyStatus{
						{Name: "speckit", Kind: "skill", Available: false, AutoInstallable: true, Message: "auto-installable"},
						{Name: "custom", Kind: "skill", Available: false, AutoInstallable: false, Message: "missing"},
					},
				},
				Codebase: meta.CodebaseMetrics{
					PRsByStatus: map[string]int{},
				},
				Platform: platform.PlatformProfile{
					Type: platform.PlatformGitHub,
				},
			},
			wantContains: []string{
				"git",
				"claude",
				"speckit",
				"custom",
				indicatorOK,
				indicatorFail,
				indicatorAutoInstallable,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderHealthReport(tt.report)
			assert.NotEmpty(t, output)

			for _, want := range tt.wantContains {
				assert.Contains(t, output, want, "output should contain %q", want)
			}
			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, output, notWant, "output should not contain %q", notWant)
			}
		})
	}
}

func TestRenderHealthReport_NilReport(t *testing.T) {
	output := RenderHealthReport(nil)
	assert.Contains(t, output, "No health report data available")
}

func TestRenderHealthReport_SectionHeaders(t *testing.T) {
	report := &meta.HealthReport{
		Timestamp: time.Now(),
		Duration:  1 * time.Second,
		Init: meta.InitCheckResult{
			WaveVersion: "0.1.0",
		},
		Dependencies: meta.DependencyReport{},
		Codebase: meta.CodebaseMetrics{
			PRsByStatus: map[string]int{},
		},
		Platform: platform.PlatformProfile{
			Type: platform.PlatformUnknown,
		},
	}

	output := RenderHealthReport(report)

	// All four main sections must always be present.
	assert.Contains(t, output, "Init")
	assert.Contains(t, output, "Dependencies")
	assert.Contains(t, output, "Codebase")
	assert.Contains(t, output, "Platform")
}

func TestRenderHealthReport_ErrorsSectionOnlyWhenErrors(t *testing.T) {
	noErrors := &meta.HealthReport{
		Init:     meta.InitCheckResult{WaveVersion: "v1"},
		Codebase: meta.CodebaseMetrics{PRsByStatus: map[string]int{}},
		Platform: platform.PlatformProfile{Type: platform.PlatformUnknown},
	}
	withErrors := &meta.HealthReport{
		Init:     meta.InitCheckResult{WaveVersion: "v1"},
		Codebase: meta.CodebaseMetrics{PRsByStatus: map[string]int{}},
		Platform: platform.PlatformProfile{Type: platform.PlatformUnknown},
		Errors: []meta.HealthCheckError{
			{Check: "test", Message: "something broke"},
		},
	}

	outputClean := RenderHealthReport(noErrors)
	outputWithErrors := RenderHealthReport(withErrors)

	assert.NotContains(t, outputClean, "Errors")
	assert.Contains(t, outputWithErrors, "Errors")
	assert.Contains(t, outputWithErrors, "something broke")
}

func TestRenderHealthReport_TimeoutIndicator(t *testing.T) {
	report := &meta.HealthReport{
		Init:     meta.InitCheckResult{WaveVersion: "v1"},
		Codebase: meta.CodebaseMetrics{PRsByStatus: map[string]int{}},
		Platform: platform.PlatformProfile{Type: platform.PlatformUnknown},
		Errors: []meta.HealthCheckError{
			{Check: "codebase", Message: "timed out", Timeout: true},
		},
	}

	output := RenderHealthReport(report)
	// Timeout errors use the clock indicator instead of the cross.
	assert.Contains(t, output, "⏱")
	assert.Contains(t, output, "timed out")
}

func TestRenderHealthReport_PlatformRepoDisplay(t *testing.T) {
	report := &meta.HealthReport{
		Init:     meta.InitCheckResult{WaveVersion: "v1"},
		Codebase: meta.CodebaseMetrics{PRsByStatus: map[string]int{}},
		Platform: platform.PlatformProfile{
			Type:           platform.PlatformGitLab,
			Owner:          "myorg",
			Repo:           "myapp",
			PipelineFamily: "gl",
			CLITool:        "glab",
		},
	}

	output := RenderHealthReport(report)
	assert.Contains(t, output, "gitlab")
	assert.Contains(t, output, "myorg/myapp")
	assert.Contains(t, output, "gl")
	assert.Contains(t, output, "glab")
}
