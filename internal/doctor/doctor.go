package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/github"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/onboarding"
)

// Status represents the severity of a check result.
type Status int

const (
	StatusOK   Status = iota
	StatusWarn
	StatusErr
)

// String returns a human-readable label for the status.
func (s Status) String() string {
	switch s {
	case StatusOK:
		return "ok"
	case StatusWarn:
		return "warn"
	case StatusErr:
		return "error"
	default:
		return "unknown"
	}
}

// MarshalJSON implements json.Marshaler for Status.
func (s Status) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// CheckResult represents the outcome of a single health check.
type CheckResult struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Status   Status `json:"status"`
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
}

// Report aggregates all check results and forge detection info.
type Report struct {
	Results   []CheckResult    `json:"results"`
	Summary   Status           `json:"summary"`
	ForgeInfo *forge.ForgeInfo  `json:"forge,omitempty"`
	Codebase  *CodebaseHealth  `json:"codebase,omitempty"`
}

// Options configures which checks to run.
type Options struct {
	ManifestPath   string
	WaveDir        string
	PipelinesDir   string
	Fix            bool
	SkipCodebase   bool

	// GHClient is the GitHub API client for codebase analysis.
	GHClient *github.Client

	// LookPath overrides exec.LookPath for testing.
	LookPath func(file string) (string, error)
	// RunCmd overrides command execution for testing.
	RunCmd func(name string, args ...string) error
	// DetectForge overrides forge detection for testing.
	DetectForge func() (forge.ForgeInfo, error)
	// CheckOnboarded overrides onboarding check for testing.
	CheckOnboarded func(waveDir string) bool
}

func (o *Options) lookPath(file string) (string, error) {
	if o.LookPath != nil {
		return o.LookPath(file)
	}
	return exec.LookPath(file)
}

func (o *Options) runCmd(name string, args ...string) error {
	if o.RunCmd != nil {
		return o.RunCmd(name, args...)
	}
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func (o *Options) detectForge() (forge.ForgeInfo, error) {
	if o.DetectForge != nil {
		return o.DetectForge()
	}
	return forge.DetectFromGitRemotes()
}

func (o *Options) isOnboarded(waveDir string) bool {
	if o.CheckOnboarded != nil {
		return o.CheckOnboarded(waveDir)
	}
	return onboarding.IsOnboarded(waveDir)
}

// RunChecks executes all health checks and returns a report.
func RunChecks(ctx context.Context, opts Options) (*Report, error) {
	if opts.ManifestPath == "" {
		opts.ManifestPath = "wave.yaml"
	}
	if opts.WaveDir == "" {
		opts.WaveDir = ".wave"
	}
	if opts.PipelinesDir == "" {
		opts.PipelinesDir = ".wave/pipelines"
	}

	report := &Report{}

	// 1. Wave initialization
	report.Results = append(report.Results, checkOnboarding(&opts))

	// 2. Git repository
	report.Results = append(report.Results, checkGit(&opts))

	// 3. Manifest
	m, result := checkManifest(&opts)
	report.Results = append(report.Results, result)

	// 4. Adapter binaries
	report.Results = append(report.Results, checkAdapters(&opts, m)...)

	// 5. Forge detection + CLI
	fi, forgeResults := checkForge(&opts)
	report.Results = append(report.Results, forgeResults...)
	if fi.Type != forge.ForgeUnknown {
		report.ForgeInfo = &fi
	}

	// 6. Codebase health (forge API)
	if !opts.SkipCodebase && fi.Type != forge.ForgeUnknown {
		codebase, err := AnalyzeCodebase(ctx, CodebaseOptions{
			ForgeInfo: fi,
			GHClient:  opts.GHClient,
		})
		if err == nil && codebase != nil {
			report.Codebase = codebase
		}
	}

	// 7. Required tools
	report.Results = append(report.Results, checkRequiredTools(&opts)...)

	// 8. Required skills
	report.Results = append(report.Results, checkRequiredSkills(&opts)...)

	// Compute summary
	report.Summary = StatusOK
	for _, r := range report.Results {
		if r.Status == StatusErr && report.Summary < StatusErr {
			report.Summary = StatusErr
		} else if r.Status == StatusWarn && report.Summary < StatusWarn {
			report.Summary = StatusWarn
		}
	}

	return report, nil
}

func checkOnboarding(opts *Options) CheckResult {
	if opts.isOnboarded(opts.WaveDir) {
		return CheckResult{
			Name:     "Wave Initialized",
			Category: "system",
			Status:   StatusOK,
			Message:  "Wave has been initialized",
		}
	}

	// Grandfather existing projects with wave.yaml
	if _, err := os.Stat(opts.ManifestPath); err == nil {
		return CheckResult{
			Name:     "Wave Initialized",
			Category: "system",
			Status:   StatusOK,
			Message:  "Wave project detected",
		}
	}

	return CheckResult{
		Name:     "Wave Initialized",
		Category: "system",
		Status:   StatusErr,
		Message:  "Wave has not been initialized",
		Fix:      "Run 'wave init' to set up the project",
	}
}

func checkGit(opts *Options) CheckResult {
	err := opts.runCmd("git", "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return CheckResult{
			Name:     "Git Repository",
			Category: "system",
			Status:   StatusErr,
			Message:  "Not a git repository",
			Fix:      "Run 'git init' to initialize a repository",
		}
	}

	return CheckResult{
		Name:     "Git Repository",
		Category: "system",
		Status:   StatusOK,
		Message:  "Valid git repository",
	}
}

func checkManifest(opts *Options) (*manifest.Manifest, CheckResult) {
	m, err := manifest.Load(opts.ManifestPath)
	if err != nil {
		return nil, CheckResult{
			Name:     "Manifest Valid",
			Category: "system",
			Status:   StatusErr,
			Message:  fmt.Sprintf("Failed to load manifest: %v", err),
			Fix:      "Run 'wave init' to create a valid wave.yaml, or fix syntax errors",
		}
	}

	return m, CheckResult{
		Name:     "Manifest Valid",
		Category: "system",
		Status:   StatusOK,
		Message:  fmt.Sprintf("Manifest loaded (%d personas, %d adapters)", len(m.Personas), len(m.Adapters)),
	}
}

func checkAdapters(opts *Options, m *manifest.Manifest) []CheckResult {
	if m == nil || len(m.Adapters) == 0 {
		return []CheckResult{{
			Name:     "Adapter Binaries",
			Category: "system",
			Status:   StatusOK,
			Message:  "No adapters configured",
		}}
	}

	var results []CheckResult
	for name, adapter := range m.Adapters {
		path, err := opts.lookPath(adapter.Binary)
		if err != nil {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Adapter: %s", name),
				Category: "system",
				Status:   StatusErr,
				Message:  fmt.Sprintf("Binary %q not found on PATH", adapter.Binary),
				Fix:      fmt.Sprintf("Install %s or update wave.yaml adapter configuration", adapter.Binary),
			})
		} else {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Adapter: %s", name),
				Category: "system",
				Status:   StatusOK,
				Message:  fmt.Sprintf("Found at %s", path),
			})
		}
	}
	return results
}

func checkForge(opts *Options) (forge.ForgeInfo, []CheckResult) {
	fi, err := opts.detectForge()
	if err != nil {
		return fi, []CheckResult{{
			Name:     "Forge Detection",
			Category: "forge",
			Status:   StatusWarn,
			Message:  fmt.Sprintf("Could not detect forge: %v", err),
		}}
	}

	if fi.Type == forge.ForgeUnknown {
		return fi, []CheckResult{{
			Name:     "Forge Detection",
			Category: "forge",
			Status:   StatusWarn,
			Message:  "Could not identify forge type from git remote",
		}}
	}

	results := []CheckResult{{
		Name:     "Forge Detection",
		Category: "forge",
		Status:   StatusOK,
		Message:  fmt.Sprintf("Detected %s (%s/%s)", fi.Type, fi.Owner, fi.Repo),
	}}

	// Check for CLI tool
	if fi.CLITool != "" {
		_, err := opts.lookPath(fi.CLITool)
		if err != nil {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Forge CLI: %s", fi.CLITool),
				Category: "forge",
				Status:   StatusWarn,
				Message:  fmt.Sprintf("CLI tool %q not found on PATH", fi.CLITool),
				Fix:      fmt.Sprintf("Install %s for full forge integration", fi.CLITool),
			})
		} else {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Forge CLI: %s", fi.CLITool),
				Category: "forge",
				Status:   StatusOK,
				Message:  fmt.Sprintf("CLI tool %q available", fi.CLITool),
			})
		}
	}

	return fi, results
}

func checkRequiredTools(opts *Options) []CheckResult {
	tools := collectRequiredTools(opts.PipelinesDir)
	if len(tools) == 0 {
		return []CheckResult{{
			Name:     "Required Tools",
			Category: "system",
			Status:   StatusOK,
			Message:  "No tools required by pipelines",
		}}
	}

	var results []CheckResult
	for _, tool := range tools {
		_, err := opts.lookPath(tool)
		if err != nil {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Tool: %s", tool),
				Category: "system",
				Status:   StatusErr,
				Message:  fmt.Sprintf("Required tool %q not found on PATH", tool),
				Fix:      fmt.Sprintf("Install %s", tool),
			})
		} else {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Tool: %s", tool),
				Category: "system",
				Status:   StatusOK,
				Message:  fmt.Sprintf("Tool %q available", tool),
			})
		}
	}
	return results
}

func checkRequiredSkills(opts *Options) []CheckResult {
	skills := collectRequiredSkills(opts.PipelinesDir)
	if len(skills) == 0 {
		return []CheckResult{{
			Name:     "Required Skills",
			Category: "system",
			Status:   StatusOK,
			Message:  "No skills required by pipelines",
		}}
	}

	var results []CheckResult
	for name, check := range skills {
		if check == "" {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Skill: %s", name),
				Category: "system",
				Status:   StatusWarn,
				Message:  fmt.Sprintf("Skill %q has no check command", name),
			})
			continue
		}

		err := opts.runCmd("sh", "-c", check)
		if err != nil {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Skill: %s", name),
				Category: "system",
				Status:   StatusErr,
				Message:  fmt.Sprintf("Skill %q not installed", name),
				Fix:      fmt.Sprintf("Install skill %q or run 'wave run' with auto-install", name),
			})
		} else {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Skill: %s", name),
				Category: "system",
				Status:   StatusOK,
				Message:  fmt.Sprintf("Skill %q installed", name),
			})
		}
	}
	return results
}
