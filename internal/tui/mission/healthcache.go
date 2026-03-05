package mission

import (
	"context"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/meta"
	"github.com/recinq/wave/internal/platform"
)

// HealthCacheMsg carries a completed health check result.
type HealthCacheMsg struct {
	Report *meta.HealthReport
	Err    error
}

// HealthCache runs health checks asynchronously and caches the result.
type HealthCache struct {
	mu           sync.Mutex
	report       *meta.HealthReport
	lastRefresh  time.Time
	loading      bool
	manifestPath string
	version      string
}

// NewHealthCache creates a new health cache.
func NewHealthCache(manifestPath, version string) *HealthCache {
	return &HealthCache{
		manifestPath: manifestPath,
		version:      version,
	}
}

// Report returns the cached health report, or nil if not yet loaded.
func (hc *HealthCache) Report() *meta.HealthReport {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	return hc.report
}

// IsLoading returns true if a health check is in progress.
func (hc *HealthCache) IsLoading() bool {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	return hc.loading
}

// RefreshCmd returns a tea.Cmd that runs health checks in the background.
func (hc *HealthCache) RefreshCmd() tea.Cmd {
	hc.mu.Lock()
	if hc.loading {
		hc.mu.Unlock()
		return nil
	}
	hc.loading = true
	hc.mu.Unlock()

	return func() tea.Msg {
		profile, _ := platform.DetectFromGit()

		checker := meta.NewHealthChecker(
			meta.WithManifestPath(hc.manifestPath),
			meta.WithVersion(hc.version),
			meta.WithPlatformProfile(profile),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		report, err := checker.RunHealthChecks(ctx, meta.DefaultHealthCheckConfig())

		hc.mu.Lock()
		hc.loading = false
		if err == nil {
			hc.report = report
			hc.lastRefresh = time.Now()
		}
		hc.mu.Unlock()

		return HealthCacheMsg{Report: report, Err: err}
	}
}
