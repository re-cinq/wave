package doctor

import (
	"fmt"
	"os"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
)

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

	if fi.Type == forge.ForgeLocal {
		return fi, []CheckResult{{
			Name:     "Forge Detection",
			Category: "forge",
			Status:   StatusOK,
			Message:  "Local mode — no git remote configured (forge-dependent pipelines will be filtered out)",
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
