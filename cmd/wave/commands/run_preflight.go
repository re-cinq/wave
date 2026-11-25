package commands

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/preflight"
)

// lookPath is exec.LookPath, swappable in tests.
var lookPath = exec.LookPath

// checkRunAdapters verifies, before any step starts, that every adapter
// binary the pipeline will invoke is on PATH. Without it a missing adapter
// only surfaced mid-step as a raw fork/exec error.
func checkRunAdapters(p *pipeline.Pipeline, m *manifest.Manifest, adapterOverride string) error {
	personas := make(map[string]preflight.Persona, len(m.Personas))
	for name, per := range m.Personas {
		personas[name] = preflight.Persona{Adapter: per.Adapter}
	}
	adapters := make(map[string]preflight.AdapterDef, len(m.Adapters)+1)
	for name, a := range m.Adapters {
		bin := a.Binary
		if bin == "" {
			bin = name
		}
		adapters[name] = preflight.AdapterDef{Binary: bin}
	}
	if adapterOverride != "" {
		if _, ok := adapters[adapterOverride]; !ok {
			adapters[adapterOverride] = preflight.AdapterDef{Binary: adapterOverride}
		}
	}

	steps := make([]preflight.StepRef, 0, len(p.Steps))
	for _, s := range p.Steps {
		ref := preflight.StepRef{Persona: s.Persona, Adapter: s.Adapter}
		if adapterOverride != "" {
			ref.Adapter = adapterOverride
		}
		steps = append(steps, ref)
	}

	var missing []string
	for _, bin := range preflight.CollectAdapterBinaries(personas, adapters, steps) {
		if bin == "mock" {
			continue
		}
		if _, err := lookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return NewCLIError(CodePreflightFailed,
		fmt.Sprintf("adapter binary not found on PATH: %s", strings.Join(missing, ", ")),
		"Install the adapter CLI, run with --adapter claude or --adapter codex, or switch the project with 'wave init --reconfigure --adapter <name>'")
}
