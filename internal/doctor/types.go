package doctor

import (
	"os/exec"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/onboarding"
)

// Status represents the severity of a check result.
type Status int

const (
	StatusOK Status = iota
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
	ForgeInfo *forge.ForgeInfo `json:"forge,omitempty"`
	Codebase  *CodebaseHealth  `json:"codebase,omitempty"`
}

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
