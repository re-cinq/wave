package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/recinq/wave/internal/defaults"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/onboarding"
)

// loadDefaultBootstrapManifest synthesises a wave.yaml from embedded defaults
// so a bootstrap pipeline (e.g. onboard-project) can execute on a cold-start
// repo with no prior manifest. The pipeline is expected to write the actual
// wave.yaml as part of its output, so this in-memory copy is short-lived.
//
// As a side effect, the embedded persona / pipeline / contract / prompt
// assets are written to .agents/ so the executor's downstream disk-reads
// (base-protocol.md, prompt source_path, schema_path) succeed.
func loadDefaultBootstrapManifest() (*manifest.Manifest, error) {
	if err := scaffoldEmbeddedAssets(); err != nil {
		return nil, fmt.Errorf("scaffold embedded assets: %w", err)
	}
	personas, err := defaults.GetPersonaConfigs()
	if err != nil {
		return nil, fmt.Errorf("load embedded persona configs: %w", err)
	}
	defaultMap := onboarding.BuildDefaultManifest("claude", ".agents/workspaces", nil, personas)
	data, err := yaml.Marshal(defaultMap)
	if err != nil {
		return nil, fmt.Errorf("synthesise bootstrap manifest: %w", err)
	}
	m, err := manifest.UnmarshalStrict(data)
	if err != nil {
		return nil, fmt.Errorf("parse bootstrap manifest: %w", err)
	}
	m.RootDir = "."
	return m, nil
}

// materialiseBootstrapOutputs walks the workspaces produced by a bootstrap
// pipeline and copies the project-shaping outputs back to the repo root.
// Worktree workspaces are isolated git working trees by design; without this
// post-pipeline step the sentinel + manifest + generated assets would stay
// trapped inside .agents/workspaces/<run-id>/ and never reach the user.
//
// Files materialised when present in any step's workspace:
//   - .agents/.onboarding-done   — marks onboarding complete
//   - wave.yaml                  — project manifest (if pipeline emits one)
//   - .agents/output/*.json      — declared step outputs
//   - .agents/personas/*.md      — custom personas the pipeline generated
//   - .agents/pipelines/*.yaml   — custom pipelines the pipeline generated
//   - .agents/prompts/**/*.md    — custom prompts the pipeline generated
//   - .agents/contracts/*        — custom contracts
func materialiseBootstrapOutputs(runID string) error {
	return materialiseFromRunDir(filepath.Join(".agents", "workspaces", runID))
}

func materialiseFromRunDir(runDir string) error {
	entries, err := os.ReadDir(runDir)
	if err != nil {
		return nil // no workspaces, nothing to copy
	}
	// Material patterns to scan inside each step workspace's .agents/ tree.
	type pat struct {
		rel string // path relative to step workspace
		dir bool   // true means walk a directory recursively
	}
	patterns := []pat{
		{".agents/.onboarding-done", false},
		{"wave.yaml", false},
		{".agents/output", true},
		{".agents/personas", true},
		{".agents/pipelines", true},
		{".agents/prompts", true},
		{".agents/contracts", true},
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		stepWS := filepath.Join(runDir, e.Name())
		// Worktree workspaces add an extra `__wt_<run>` subdir.
		// Pick the inner dir if present.
		if inner, derr := os.ReadDir(stepWS); derr == nil {
			for _, child := range inner {
				if strings.HasPrefix(child.Name(), "__wt_") && child.IsDir() {
					stepWS = filepath.Join(stepWS, child.Name())
					break
				}
			}
		}
		for _, p := range patterns {
			src := filepath.Join(stepWS, p.rel)
			if p.dir {
				_ = copyTreePreservingExisting(src, p.rel)
			} else {
				if _, err := os.Stat(src); err == nil {
					_ = copyFilePreservingExisting(src, p.rel)
				}
			}
		}
	}
	return nil
}

func copyFilePreservingExisting(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return nil // never overwrite project root
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func copyTreePreservingExisting(srcRoot, dstRoot string) error {
	return filepath.Walk(srcRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(srcRoot, path)
		if rerr != nil {
			return nil
		}
		dst := filepath.Join(dstRoot, rel)
		return copyFilePreservingExisting(path, dst)
	})
}

func scaffoldEmbeddedAssets() error {
	personaMD, err := defaults.GetPersonas()
	if err != nil {
		return err
	}
	if err := onboarding.CreateExamplePersonas(personaMD); err != nil {
		return err
	}
	pipelines, err := defaults.GetPipelines()
	if err != nil {
		return err
	}
	if err := onboarding.CreateExamplePipelines(pipelines); err != nil {
		return err
	}
	contracts, err := defaults.GetContracts()
	if err != nil {
		return err
	}
	if err := onboarding.CreateExampleContracts(contracts); err != nil {
		return err
	}
	prompts, err := defaults.GetPrompts()
	if err != nil {
		return err
	}
	if err := onboarding.CreateExamplePrompts(prompts); err != nil {
		return err
	}
	return nil
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

// bootstrapPipelines may run on a cold-start repo (no wave.yaml, no sentinel)
// because they exist to PRODUCE those artefacts. Excluding them from the
// onboarding gate is what lets a fresh project ever reach a primed state.
var bootstrapPipelines = map[string]bool{
	"onboard-project": true,
	"ops-bootstrap":   true,
}

// checkOnboarding verifies that onboarding has been completed.
// It returns an error if onboarding is incomplete, directing the user to run 'wave init'.
// Existing projects that have a wave.yaml but no .onboarded marker are grandfathered in.
// Bootstrap pipelines bypass the gate entirely.
func checkOnboarding(pipelineName string) error {
	if bootstrapPipelines[pipelineName] {
		return nil
	}

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
