package commands

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/onboarding"
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
		adapterAlternativesHint(m, adapters, missing))
}

// adapterAlternativesHint suggests only adapters that would actually work:
// --adapter X for adapters configured in wave.yaml whose binary is on PATH,
// and 'wave init --reconfigure --adapter X' for installed init adapters the
// manifest does not configure yet (--adapter alone fails for those, the
// executor needs the adapter block). Missing binaries are never suggested.
func adapterAlternativesHint(m *manifest.Manifest, adapters map[string]preflight.AdapterDef, missing []string) string {
	isMissing := make(map[string]bool, len(missing))
	for _, b := range missing {
		isMissing[b] = true
	}
	installed := func(bin string) bool {
		if isMissing[bin] || bin == "mock" {
			return false
		}
		_, err := lookPath(bin)
		return err == nil
	}

	var runWith []string
	names := make([]string, 0, len(m.Adapters))
	for name := range m.Adapters {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if installed(adapters[name].Binary) {
			runWith = append(runWith, "--adapter "+name)
		}
	}
	var reconfigure []string
	for _, name := range onboarding.SupportedInitAdapters() {
		if _, configured := m.Adapters[name]; configured {
			continue
		}
		if installed(name) {
			reconfigure = append(reconfigure, name)
		}
	}

	hint := "Install the adapter CLI"
	if len(runWith) > 0 {
		hint += ", or run with " + strings.Join(runWith, " or ")
	}
	for _, name := range reconfigure {
		hint += fmt.Sprintf(", or switch the project with 'wave init --reconfigure --adapter %s'", name)
	}
	return hint
}
