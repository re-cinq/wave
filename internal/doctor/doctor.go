package doctor

import (
	"context"

	"github.com/recinq/wave/internal/forge"
)

// RunChecks executes all health checks and returns a report.
func RunChecks(ctx context.Context, opts Options) (*Report, error) {
	if opts.ManifestPath == "" {
		opts.ManifestPath = "wave.yaml"
	}
	if opts.WaveDir == "" {
		opts.WaveDir = ".agents"
	}
	if opts.PipelinesDir == "" {
		opts.PipelinesDir = ".agents/pipelines"
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
			ForgeInfo:   fi,
			ForgeClient: opts.ForgeClient,
		})
		if err == nil && codebase != nil {
			report.Codebase = codebase
		}
	}

	// 7. Required tools
	report.Results = append(report.Results, checkRequiredTools(&opts)...)

	// 8. Required skills
	report.Results = append(report.Results, checkRequiredSkills(&opts)...)

	// 8b. Docker daemon (informational — many projects don't use Docker)
	report.Results = append(report.Results, checkDockerDaemon(&opts))

	// 10. Adapter registry
	report.Results = append(report.Results, checkAdapterRegistry(m))

	// 11. Retry policies
	report.Results = append(report.Results, checkRetryPolicies(&opts)...)

	// 12. Engine capabilities
	report.Results = append(report.Results, checkEngineCapabilities())

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
