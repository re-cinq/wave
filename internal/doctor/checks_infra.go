package doctor

import (
	"fmt"
	"os"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/suggest"
)

func checkOnboarding(opts *Options) suggest.CheckResult {
	if opts.isOnboarded(opts.WaveDir) {
		return suggest.CheckResult{
			Name:     "Wave Initialized",
			Category: "system",
			Status:   suggest.StatusOK,
			Message:  "Wave has been initialized",
		}
	}

	// Grandfather existing projects with wave.yaml
	if _, err := os.Stat(opts.ManifestPath); err == nil {
		return suggest.CheckResult{
			Name:     "Wave Initialized",
			Category: "system",
			Status:   suggest.StatusOK,
			Message:  "Wave project detected",
		}
	}

	return suggest.CheckResult{
		Name:     "Wave Initialized",
		Category: "system",
		Status:   suggest.StatusErr,
		Message:  "Wave has not been initialized",
		Fix:      "Run 'wave init' to set up the project",
	}
}

func checkGit(opts *Options) suggest.CheckResult {
	err := opts.runCmd("git", "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return suggest.CheckResult{
			Name:     "Git Repository",
			Category: "system",
			Status:   suggest.StatusErr,
			Message:  "Not a git repository",
			Fix:      "Run 'git init' to initialize a repository",
		}
	}

	return suggest.CheckResult{
		Name:     "Git Repository",
		Category: "system",
		Status:   suggest.StatusOK,
		Message:  "Valid git repository",
	}
}

func checkManifest(opts *Options) (*manifest.Manifest, suggest.CheckResult) {
	m, err := manifest.Load(opts.ManifestPath)
	if err != nil {
		return nil, suggest.CheckResult{
			Name:     "Manifest Valid",
			Category: "system",
			Status:   suggest.StatusErr,
			Message:  fmt.Sprintf("Failed to load manifest: %v", err),
			Fix:      "Run 'wave init' to create a valid wave.yaml, or fix syntax errors",
		}
	}

	return m, suggest.CheckResult{
		Name:     "Manifest Valid",
		Category: "system",
		Status:   suggest.StatusOK,
		Message:  fmt.Sprintf("Manifest loaded (%d personas, %d adapters)", len(m.Personas), len(m.Adapters)),
	}
}

func checkAdapters(opts *Options, m *manifest.Manifest) []suggest.CheckResult {
	if m == nil || len(m.Adapters) == 0 {
		return []suggest.CheckResult{{
			Name:     "Adapter Binaries",
			Category: "system",
			Status:   suggest.StatusOK,
			Message:  "No adapters configured",
		}}
	}

	var results []suggest.CheckResult
	for name, adapter := range m.Adapters {
		path, err := opts.lookPath(adapter.Binary)
		if err != nil {
			results = append(results, suggest.CheckResult{
				Name:     fmt.Sprintf("Adapter: %s", name),
				Category: "system",
				Status:   suggest.StatusErr,
				Message:  fmt.Sprintf("Binary %q not found on PATH", adapter.Binary),
				Fix:      fmt.Sprintf("Install %s or update wave.yaml adapter configuration", adapter.Binary),
			})
		} else {
			results = append(results, suggest.CheckResult{
				Name:     fmt.Sprintf("Adapter: %s", name),
				Category: "system",
				Status:   suggest.StatusOK,
				Message:  fmt.Sprintf("Found at %s", path),
			})
		}
	}
	return results
}

func checkForge(opts *Options) (forge.ForgeInfo, []suggest.CheckResult) {
	fi, err := opts.detectForge()
	if err != nil {
		return fi, []suggest.CheckResult{{
			Name:     "Forge Detection",
			Category: "forge",
			Status:   suggest.StatusWarn,
			Message:  fmt.Sprintf("Could not detect forge: %v", err),
		}}
	}

	if fi.Type == forge.ForgeUnknown {
		return fi, []suggest.CheckResult{{
			Name:     "Forge Detection",
			Category: "forge",
			Status:   suggest.StatusWarn,
			Message:  "Could not identify forge type from git remote",
		}}
	}

	if fi.Type == forge.ForgeLocal {
		return fi, []suggest.CheckResult{{
			Name:     "Forge Detection",
			Category: "forge",
			Status:   suggest.StatusOK,
			Message:  "Local mode — no git remote configured (forge-dependent pipelines will be filtered out)",
		}}
	}

	results := []suggest.CheckResult{{
		Name:     "Forge Detection",
		Category: "forge",
		Status:   suggest.StatusOK,
		Message:  fmt.Sprintf("Detected %s (%s/%s)", fi.Type, fi.Owner, fi.Repo),
	}}

	// Check for CLI tool
	if fi.CLITool != "" {
		_, err := opts.lookPath(fi.CLITool)
		if err != nil {
			results = append(results, suggest.CheckResult{
				Name:     fmt.Sprintf("Forge CLI: %s", fi.CLITool),
				Category: "forge",
				Status:   suggest.StatusWarn,
				Message:  fmt.Sprintf("CLI tool %q not found on PATH", fi.CLITool),
				Fix:      fmt.Sprintf("Install %s for full forge integration", fi.CLITool),
			})
		} else {
			results = append(results, suggest.CheckResult{
				Name:     fmt.Sprintf("Forge CLI: %s", fi.CLITool),
				Category: "forge",
				Status:   suggest.StatusOK,
				Message:  fmt.Sprintf("CLI tool %q available", fi.CLITool),
			})
		}
	}

	return fi, results
}
