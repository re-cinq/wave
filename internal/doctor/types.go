package doctor

import (
	"os/exec"

	"github.com/recinq/wave/internal/forge"
)

// Options configures which checks to run.
type Options struct {
	ManifestPath string
	WaveDir      string
	PipelinesDir string
	Fix          bool
	SkipCodebase bool

	// ForgeClient is the forge API client for codebase analysis.
	ForgeClient forge.Client

	// LookPath overrides exec.LookPath for testing.
	LookPath func(file string) (string, error)
	// RunCmd overrides command execution for testing.
	RunCmd func(name string, args ...string) error
	// DetectForge overrides forge detection for testing.
	DetectForge func() (forge.ForgeInfo, error)
	// CheckOnboarded reports whether the project has completed onboarding.
	// Callers must inject this; doctor does not import the onboarding package
	// (severs the tui→doctor→onboarding edge).
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
	return false
}
