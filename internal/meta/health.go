package meta

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/platform"
	"golang.org/x/sync/errgroup"
)

// HealthReport is the top-level result of all health checks.
type HealthReport struct {
	Timestamp    time.Time                `json:"timestamp"`
	Duration     time.Duration            `json:"duration_ms"`
	Init         InitCheckResult          `json:"init"`
	Dependencies DependencyReport         `json:"dependencies"`
	Codebase     CodebaseMetrics          `json:"codebase"`
	Platform     platform.PlatformProfile `json:"platform"`
	Errors       []HealthCheckError       `json:"errors,omitempty"`
}

// InitCheckResult holds the result of the manifest/init check.
type InitCheckResult struct {
	ManifestFound  bool      `json:"manifest_found"`
	ManifestValid  bool      `json:"manifest_valid"`
	WaveVersion    string    `json:"wave_version"`
	LastConfigDate time.Time `json:"last_config_date,omitempty"`
	Error          string    `json:"error,omitempty"`
}

// DependencyReport holds results for tool and skill dependency checks.
type DependencyReport struct {
	Tools  []DependencyStatus `json:"tools"`
	Skills []DependencyStatus `json:"skills"`
}

// DependencyStatus represents the availability of a single dependency.
type DependencyStatus struct {
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	Available       bool   `json:"available"`
	AutoInstallable bool   `json:"auto_installable"`
	Message         string `json:"message,omitempty"`
}

// CodebaseMetrics holds codebase health metrics gathered from git or an API.
type CodebaseMetrics struct {
	RecentCommits  int            `json:"recent_commits"`
	OpenIssueCount int            `json:"open_issue_count"`
	OpenPRCount    int            `json:"open_pr_count"`
	PRsByStatus    map[string]int `json:"prs_by_status"`
	BranchCount    int            `json:"branch_count"`
	LastCommitDate time.Time      `json:"last_commit_date"`
	APIAvailable   bool           `json:"api_available"`
	Source         string         `json:"source"`
}

// HealthCheckError records an error encountered during a specific check.
type HealthCheckError struct {
	Check   string `json:"check"`
	Message string `json:"message"`
	Timeout bool   `json:"timeout"`
}

// HealthCheckConfig holds timeout settings for each health check phase.
type HealthCheckConfig struct {
	InitTimeout     time.Duration
	DepsTimeout     time.Duration
	CodebaseTimeout time.Duration
	PlatformTimeout time.Duration
}

// DefaultHealthCheckConfig returns sensible default timeout values.
func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		InitTimeout:     5 * time.Second,
		DepsTimeout:     10 * time.Second,
		CodebaseTimeout: 15 * time.Second,
		PlatformTimeout: 5 * time.Second,
	}
}

// HealthChecker is the interface for running health checks.
type HealthChecker interface {
	RunHealthChecks(ctx context.Context, opts HealthCheckConfig) (*HealthReport, error)
}

// GitRunner abstracts git command execution for testability.
type GitRunner interface {
	Run(ctx context.Context, args ...string) (string, error)
}

// GitHubAPI abstracts the subset of GitHub client operations needed by health checks.
type GitHubAPI interface {
	GetRepoStats(ctx context.Context, owner, repo string) (openIssues, openPRs int, err error)
}

// defaultGitRunner executes real git commands.
type defaultGitRunner struct{}

func (r *defaultGitRunner) Run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// HealthCheckerImpl implements the HealthChecker interface.
type HealthCheckerImpl struct {
	manifestPath string
	profile      platform.PlatformProfile
	manifest     *manifest.Manifest
	version      string

	// Injected dependencies for testability.
	lookPath  func(file string) (string, error)
	gitRunner GitRunner
	githubAPI GitHubAPI
}

// initCheckOutput bundles the init result with the loaded manifest.
type initCheckOutput struct {
	result   InitCheckResult
	manifest *manifest.Manifest
}

// HealthCheckerOption is a functional option for HealthCheckerImpl.
type HealthCheckerOption func(*HealthCheckerImpl)

// WithManifestPath sets the manifest file path.
func WithManifestPath(path string) HealthCheckerOption {
	return func(h *HealthCheckerImpl) {
		h.manifestPath = path
	}
}

// WithPlatformProfile sets the platform profile.
func WithPlatformProfile(p platform.PlatformProfile) HealthCheckerOption {
	return func(h *HealthCheckerImpl) {
		h.profile = p
	}
}

// WithVersion sets the Wave version string.
func WithVersion(v string) HealthCheckerOption {
	return func(h *HealthCheckerImpl) {
		h.version = v
	}
}

// WithLookPath overrides exec.LookPath for testing.
func WithLookPath(fn func(string) (string, error)) HealthCheckerOption {
	return func(h *HealthCheckerImpl) {
		h.lookPath = fn
	}
}

// WithGitRunner overrides the git command runner for testing.
func WithGitRunner(r GitRunner) HealthCheckerOption {
	return func(h *HealthCheckerImpl) {
		h.gitRunner = r
	}
}

// WithGitHubAPI overrides the GitHub API client for testing.
func WithGitHubAPI(api GitHubAPI) HealthCheckerOption {
	return func(h *HealthCheckerImpl) {
		h.githubAPI = api
	}
}

// WithManifest sets a pre-loaded manifest (useful for testing or when already loaded).
func WithManifest(m *manifest.Manifest) HealthCheckerOption {
	return func(h *HealthCheckerImpl) {
		h.manifest = m
	}
}

// NewHealthChecker creates a new HealthCheckerImpl with the given options.
func NewHealthChecker(opts ...HealthCheckerOption) *HealthCheckerImpl {
	h := &HealthCheckerImpl{
		manifestPath: "wave.yaml",
		version:      "unknown",
		lookPath:     exec.LookPath,
		gitRunner:    &defaultGitRunner{},
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// checkInit validates the Wave manifest file.
func (h *HealthCheckerImpl) checkInit(ctx context.Context, manifestPath string) initCheckOutput {
	result := InitCheckResult{
		WaveVersion: h.version,
	}

	info, err := os.Stat(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Error = fmt.Sprintf("manifest not found: %s", manifestPath)
			return initCheckOutput{result: result}
		}
		result.Error = fmt.Sprintf("cannot stat manifest: %v", err)
		return initCheckOutput{result: result}
	}
	result.ManifestFound = true
	result.LastConfigDate = info.ModTime()

	m, err := manifest.Load(manifestPath)
	if err != nil {
		result.Error = fmt.Sprintf("manifest validation failed: %v", err)
		return initCheckOutput{result: result}
	}
	result.ManifestValid = true

	return initCheckOutput{result: result, manifest: m}
}

// checkDependencies checks tool and skill availability.
func (h *HealthCheckerImpl) checkDependencies(ctx context.Context, m *manifest.Manifest) DependencyReport {
	report := DependencyReport{
		Tools:  []DependencyStatus{},
		Skills: []DependencyStatus{},
	}

	if m == nil {
		return report
	}

	// Collect required tools from all pipelines by scanning adapter binaries.
	toolSet := make(map[string]struct{})
	for _, adapter := range m.Adapters {
		if adapter.Binary != "" {
			toolSet[adapter.Binary] = struct{}{}
		}
	}
	// Add git as an implicit dependency.
	toolSet["git"] = struct{}{}

	for tool := range toolSet {
		status := DependencyStatus{
			Name: tool,
			Kind: "tool",
		}
		_, err := h.lookPath(tool)
		if err != nil {
			status.Available = false
			status.Message = fmt.Sprintf("tool %q not found on PATH", tool)
		} else {
			status.Available = true
			status.Message = fmt.Sprintf("tool %q found", tool)
		}
		report.Tools = append(report.Tools, status)
	}

	// Check skills.
	for name, cfg := range m.Skills {
		status := DependencyStatus{
			Name:            name,
			Kind:            "skill",
			AutoInstallable: cfg.Install != "",
		}

		if cfg.Check != "" {
			// Use a short timeout for skill check commands.
			checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			cmd := exec.CommandContext(checkCtx, "sh", "-c", cfg.Check)
			err := cmd.Run()
			cancel()

			if err != nil {
				status.Available = false
				status.Message = fmt.Sprintf("skill %q check failed: %v", name, err)
			} else {
				status.Available = true
				status.Message = fmt.Sprintf("skill %q installed", name)
			}
		} else {
			status.Available = false
			status.Message = fmt.Sprintf("skill %q has no check command", name)
		}

		report.Skills = append(report.Skills, status)
	}

	return report
}

// checkCodebase gathers codebase metrics from git or the GitHub API.
func (h *HealthCheckerImpl) checkCodebase(ctx context.Context, prof platform.PlatformProfile) CodebaseMetrics {
	metrics := CodebaseMetrics{
		PRsByStatus: make(map[string]int),
	}

	// Try GitHub API first if platform is GitHub and we have a client.
	if prof.Type == platform.PlatformGitHub && h.githubAPI != nil && prof.Owner != "" && prof.Repo != "" {
		openIssues, openPRs, err := h.githubAPI.GetRepoStats(ctx, prof.Owner, prof.Repo)
		if err == nil {
			metrics.APIAvailable = true
			metrics.Source = "github_api"
			metrics.OpenIssueCount = openIssues
			metrics.OpenPRCount = openPRs
			metrics.PRsByStatus["open"] = openPRs

			// Supplement with git-local data.
			h.fillGitLocalMetrics(ctx, &metrics)
			return metrics
		}
		// Fall through to git-local on API error.
	}

	// Git-local fallback.
	metrics.Source = "git_local"
	h.fillGitLocalMetrics(ctx, &metrics)
	return metrics
}

// fillGitLocalMetrics populates metrics from local git commands.
func (h *HealthCheckerImpl) fillGitLocalMetrics(ctx context.Context, metrics *CodebaseMetrics) {
	// Recent commit count (last 30 days).
	since := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	countStr, err := h.gitRunner.Run(ctx, "rev-list", "--count", "--since="+since, "HEAD")
	if err == nil {
		if n, parseErr := strconv.Atoi(countStr); parseErr == nil {
			metrics.RecentCommits = n
		}
	}

	// Branch count.
	branchOut, err := h.gitRunner.Run(ctx, "branch", "--list")
	if err == nil {
		lines := strings.Split(branchOut, "\n")
		count := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		metrics.BranchCount = count
	}

	// Last commit date.
	dateStr, err := h.gitRunner.Run(ctx, "log", "-1", "--format=%cI")
	if err == nil {
		if t, parseErr := time.Parse(time.RFC3339, dateStr); parseErr == nil {
			metrics.LastCommitDate = t
		}
	}
}

// RunHealthChecks runs all health checks in parallel with per-check timeouts.
func (h *HealthCheckerImpl) RunHealthChecks(ctx context.Context, opts HealthCheckConfig) (*HealthReport, error) {
	start := time.Now()
	report := &HealthReport{
		Timestamp: start,
		Platform:  h.profile,
	}

	var mu sync.Mutex
	addError := func(check, message string, timeout bool) {
		mu.Lock()
		defer mu.Unlock()
		report.Errors = append(report.Errors, HealthCheckError{
			Check:   check,
			Message: message,
			Timeout: timeout,
		})
	}

	// Channel to pass the manifest from init check to dependency check.
	manifestCh := make(chan *manifest.Manifest, 1)

	g, gctx := errgroup.WithContext(ctx)

	// 1. Init check.
	g.Go(func() error {
		initCtx, cancel := context.WithTimeout(gctx, opts.InitTimeout)
		defer cancel()

		done := make(chan initCheckOutput, 1)
		go func() {
			done <- h.checkInit(initCtx, h.manifestPath)
		}()

		select {
		case out := <-done:
			mu.Lock()
			report.Init = out.result
			mu.Unlock()
			manifestCh <- out.manifest
		case <-initCtx.Done():
			addError("init", "init check timed out", true)
			manifestCh <- nil
		}
		return nil
	})

	// 2. Dependencies check — waits for manifest from init via channel.
	g.Go(func() error {
		depsCtx, cancel := context.WithTimeout(gctx, opts.InitTimeout+opts.DepsTimeout)
		defer cancel()

		var m *manifest.Manifest
		select {
		case m = <-manifestCh:
		case <-depsCtx.Done():
			addError("dependencies", "dependencies check timed out waiting for manifest", true)
			return nil
		}

		// If init didn't produce a manifest, fall back to pre-loaded one.
		if m == nil {
			m = h.manifest
		}

		depCtx, depCancel := context.WithTimeout(depsCtx, opts.DepsTimeout)
		defer depCancel()

		done := make(chan DependencyReport, 1)
		go func() {
			done <- h.checkDependencies(depCtx, m)
		}()

		select {
		case result := <-done:
			mu.Lock()
			report.Dependencies = result
			mu.Unlock()
		case <-depCtx.Done():
			addError("dependencies", "dependencies check timed out", true)
		}
		return nil
	})

	// 3. Codebase check.
	g.Go(func() error {
		cbCtx, cancel := context.WithTimeout(gctx, opts.CodebaseTimeout)
		defer cancel()

		done := make(chan CodebaseMetrics, 1)
		go func() {
			done <- h.checkCodebase(cbCtx, h.profile)
		}()

		select {
		case result := <-done:
			mu.Lock()
			report.Codebase = result
			mu.Unlock()
		case <-cbCtx.Done():
			addError("codebase", "codebase check timed out", true)
		}
		return nil
	})

	// 4. Platform check.
	g.Go(func() error {
		platCtx, cancel := context.WithTimeout(gctx, opts.PlatformTimeout)
		defer cancel()

		select {
		case <-platCtx.Done():
			if platCtx.Err() == context.DeadlineExceeded {
				addError("platform", "platform check timed out", true)
			}
		default:
		}
		return nil
	})

	_ = g.Wait()

	report.Duration = time.Since(start)
	return report, nil
}
