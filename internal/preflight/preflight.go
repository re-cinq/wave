package preflight

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/recinq/wave/internal/checks"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/tools"
)

// SkillError represents a preflight failure due to missing skills.
// It wraps an underlying error and preserves the list of missing skill names.
type SkillError struct {
	MissingSkills []string
	Err           error
}

// Error implements the error interface.
func (e *SkillError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("missing required skills: %s", strings.Join(e.MissingSkills, ", "))
}

// Unwrap returns the underlying error for errors.Unwrap support.
func (e *SkillError) Unwrap() error {
	return e.Err
}

// ToolError represents a preflight failure due to missing tools.
// It wraps an underlying error and preserves the list of missing tool names.
type ToolError struct {
	MissingTools []string
	Err          error
}

// Error implements the error interface.
func (e *ToolError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("missing required tools: %s", strings.Join(e.MissingTools, ", "))
}

// Unwrap returns the underlying error for errors.Unwrap support.
func (e *ToolError) Unwrap() error {
	return e.Err
}

// Result represents the outcome of a single preflight check.
type Result struct {
	Name    string // Tool or skill name
	Kind    string // "tool" or "skill"
	OK      bool
	Message string
}

// Checker validates that pipeline dependencies are satisfied before execution.
type Checker struct {
	skills map[string]skill.SkillConfig
	runCmd func(name string, args ...string) error // for testing
}

// NewChecker creates a preflight checker with the given skill configurations.
func NewChecker(skills map[string]skill.SkillConfig) *Checker {
	return &Checker{
		skills: skills,
		runCmd: defaultRunCmd,
	}
}

// defaultRunCmd executes a command and returns an error if it fails.
func defaultRunCmd(name string, args ...string) error {
	return checks.DefaultRunCmd(name, args...)
}

// runCmdWithOutput executes a command and returns combined stdout+stderr output.
func runCmdWithOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// CheckTools verifies that all required CLI tools are available on PATH.
func (c *Checker) CheckTools(toolNames []string) ([]Result, error) {
	checks := tools.CheckOnPath(nil, toolNames)
	results := make([]Result, 0, len(checks))
	var missing []string

	for _, ch := range checks {
		if ch.Found {
			results = append(results, Result{
				Name:    ch.Name,
				Kind:    "tool",
				OK:      true,
				Message: fmt.Sprintf("tool %q found", ch.Name),
			})
		} else {
			results = append(results, Result{
				Name:    ch.Name,
				Kind:    "tool",
				OK:      false,
				Message: fmt.Sprintf("tool %q not found on PATH", ch.Name),
			})
			missing = append(missing, ch.Name)
		}
	}

	if len(missing) > 0 {
		return results, &ToolError{MissingTools: missing}
	}
	return results, nil
}

// BrowserBinaries is the search order for browser binaries on PATH.
var BrowserBinaries = []string{
	"chromium",
	"chromium-browser",
	"google-chrome",
	"google-chrome-stable",
}

// CheckBrowserBinary verifies that a Chromium/Chrome binary is available on PATH.
// Returns the found binary path and a Result. If not found, includes platform-specific
// install instructions in the error message.
func (c *Checker) CheckBrowserBinary() (string, Result) {
	for _, name := range BrowserBinaries {
		if path, err := exec.LookPath(name); err == nil {
			return path, Result{
				Name:    name,
				Kind:    "tool",
				OK:      true,
				Message: fmt.Sprintf("browser binary %q found at %s", name, path),
			}
		}
	}

	installHint := "Install a Chromium-based browser:\n" +
		"  Ubuntu/Debian: sudo apt install chromium-browser\n" +
		"  Fedora/RHEL:   sudo dnf install chromium\n" +
		"  macOS:         brew install --cask chromium\n" +
		"  Arch:          sudo pacman -S chromium\n" +
		"  Nix:           nix-env -iA nixpkgs.chromium"

	return "", Result{
		Name:    "chromium",
		Kind:    "tool",
		OK:      false,
		Message: fmt.Sprintf("no browser binary found on PATH (searched: %s)\n%s", strings.Join(BrowserBinaries, ", "), installHint),
	}
}

// CheckDockerDaemon verifies that the Docker daemon is available and running.
func (c *Checker) CheckDockerDaemon() Result {
	status := checks.DockerDaemon(c.runCmd, nil)
	switch {
	case !status.BinaryFound:
		return Result{
			Name:    "docker",
			Kind:    "tool",
			OK:      false,
			Message: "docker binary not found on PATH\n  Install Docker: https://docs.docker.com/get-docker/",
		}
	case !status.DaemonUp:
		return Result{
			Name:    "docker",
			Kind:    "tool",
			OK:      false,
			Message: "docker daemon not running\n  Linux: systemctl start docker\n  macOS: Open Docker Desktop\n  WSL2: Start Docker Desktop for Windows",
		}
	default:
		return Result{
			Name:    "docker",
			Kind:    "tool",
			OK:      true,
			Message: "docker daemon available",
		}
	}
}

// CheckBubblewrap verifies that the bubblewrap binary is available on PATH.
func (c *Checker) CheckBubblewrap() Result {
	_, err := exec.LookPath("bwrap")
	if err != nil {
		return Result{
			Name:    "bwrap",
			Kind:    "tool",
			OK:      false,
			Message: "bubblewrap (bwrap) not found on PATH\n  Consider using Docker sandbox instead: runtime.sandbox.backend: docker",
		}
	}

	return Result{
		Name:    "bwrap",
		Kind:    "tool",
		OK:      true,
		Message: "bubblewrap available",
	}
}

// CheckSkills verifies that all required skills are installed, attempting auto-install if configured.
// Note: init commands are NOT run here — they run inside the worktree after creation.
func (c *Checker) CheckSkills(skills []string) ([]Result, error) {
	var results []Result
	var failed []string

	for _, name := range skills {
		cfg, exists := c.skills[name]
		if !exists {
			results = append(results, Result{
				Name:    name,
				Kind:    "skill",
				OK:      false,
				Message: fmt.Sprintf("skill %q not declared in pipeline requires.skills section", name),
			})
			failed = append(failed, name)
			continue
		}

		// Check if skill is already installed
		if c.isSkillInstalled(cfg) {
			results = append(results, Result{
				Name:    name,
				Kind:    "skill",
				OK:      true,
				Message: fmt.Sprintf("skill %q installed", name),
			})
			continue
		}

		// Attempt auto-install if install command is configured
		if cfg.Install == "" {
			if cfg.Optional {
				results = append(results, Result{
					Name:    name,
					Kind:    "skill",
					OK:      true,
					Message: fmt.Sprintf("optional skill %q not installed (skipped)", name),
				})
				continue
			}
			results = append(results, Result{
				Name:    name,
				Kind:    "skill",
				OK:      false,
				Message: fmt.Sprintf("skill %q not installed and no install command configured", name),
			})
			failed = append(failed, name)
			continue
		}

		// Run install command
		if err := c.runShellCommand(cfg.Install); err != nil {
			if cfg.Optional {
				results = append(results, Result{
					Name:    name,
					Kind:    "skill",
					OK:      true,
					Message: fmt.Sprintf("optional skill %q install failed (skipped): %v", name, err),
				})
				continue
			}
			// Capture output for diagnostics on failure
			installOutput, _ := runCmdWithOutput("sh", "-c", cfg.Install)
			msg := fmt.Sprintf("skill %q install failed: %v", name, err)
			if installOutput != "" {
				msg += "; output: " + truncateOutput(installOutput, 200)
			}
			results = append(results, Result{
				Name:    name,
				Kind:    "skill",
				OK:      false,
				Message: msg,
			})
			failed = append(failed, name)
			continue
		}

		// Re-check after install. First try with existing PATH, then retry
		// with $HOME/.local/bin added since install tools (uv, pip, cargo)
		// place binaries there and sandboxed/detached environments may not
		// include it in PATH.
		if c.isSkillInstalled(cfg) || c.isSkillInstalledWithToolBin(cfg) {
			results = append(results, Result{
				Name:    name,
				Kind:    "skill",
				OK:      true,
				Message: fmt.Sprintf("skill %q installed successfully", name),
			})
		} else {
			if cfg.Optional {
				results = append(results, Result{
					Name:    name,
					Kind:    "skill",
					OK:      true,
					Message: fmt.Sprintf("optional skill %q not detected after install (skipped)", name),
				})
				continue
			}
			// Capture check output for diagnostics
			checkOutput, checkErr := runCmdWithOutput("sh", "-c", cfg.Check)
			msg := fmt.Sprintf("skill %q still not detected after install", name)
			if checkErr != nil {
				msg += fmt.Sprintf(" (check error: %v)", checkErr)
			}
			if checkOutput != "" {
				msg += "; check output: " + truncateOutput(checkOutput, 200)
			}
			results = append(results, Result{
				Name:    name,
				Kind:    "skill",
				OK:      false,
				Message: msg,
			})
			failed = append(failed, name)
		}
	}

	if len(failed) > 0 {
		return results, &SkillError{
			MissingSkills: failed,
		}
	}
	return results, nil
}

// isSkillInstalled runs the skill's check command to verify installation.
func (c *Checker) isSkillInstalled(cfg skill.SkillConfig) bool {
	return checks.SkillInstalled(c.runCmd, cfg.Check)
}

// isSkillInstalledWithToolBin checks skill installation with $HOME/.local/bin
// added to PATH. Install tools (uv, pip, cargo) place binaries there, and
// in sandboxed or detached environments this directory may not be in PATH
// even after a successful install.
func (c *Checker) isSkillInstalledWithToolBin(cfg skill.SkillConfig) bool {
	return checks.SkillInstalledWithToolBin(c.runCmd, nil, cfg.Check)
}

// runShellCommand executes a shell command string via sh -c.
func (c *Checker) runShellCommand(command string) error {
	return c.runCmd("sh", "-c", command)
}

// truncateOutput trims output to maxLen characters, adding ellipsis if truncated.
func truncateOutput(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Run executes all preflight checks for the given tool and skill requirements.
// When both tools and skills fail, returns a PreflightError wrapping both typed
// errors so callers can extract either via errors.As().
func (c *Checker) Run(tools, skills []string) ([]Result, error) {
	var allResults []Result
	var skillErr *SkillError
	var toolErr *ToolError

	if len(tools) > 0 {
		toolResults, err := c.CheckTools(tools)
		allResults = append(allResults, toolResults...)
		if err != nil {
			var te *ToolError
			if errors.As(err, &te) {
				toolErr = te
			}
		}
	}

	if len(skills) > 0 {
		skillResults, err := c.CheckSkills(skills)
		allResults = append(allResults, skillResults...)
		if err != nil {
			var se *SkillError
			if errors.As(err, &se) {
				skillErr = se
			}
		}
	}

	// Return composite error when both fail so callers can extract either
	if skillErr != nil && toolErr != nil {
		return allResults, &PreflightError{SkillErr: skillErr, ToolErr: toolErr}
	}
	if skillErr != nil {
		return allResults, skillErr
	}
	if toolErr != nil {
		return allResults, toolErr
	}
	return allResults, nil
}

// CollectAdapterBinaries returns the unique set of adapter binary names
// referenced by the given pipeline steps. It resolves each step's persona
// to find the adapter, then collects the adapter binary. If a step has a
// direct adapter override, that takes precedence over the persona's adapter.
//
// This allows preflight checks to verify that all adapter binaries are
// available on PATH before pipeline execution begins.
func CollectAdapterBinaries(
	personas map[string]Persona,
	adapters map[string]AdapterDef,
	steps []StepRef,
) []string {
	seen := make(map[string]bool)
	var binaries []string
	for _, step := range steps {
		adapterName := ""
		// Step-level adapter override takes precedence
		if step.Adapter != "" {
			adapterName = step.Adapter
		} else if step.Persona != "" {
			if p, ok := personas[step.Persona]; ok {
				adapterName = p.Adapter
			}
		}
		if adapterName == "" {
			continue
		}
		if a, ok := adapters[adapterName]; ok && a.Binary != "" {
			if !seen[a.Binary] {
				seen[a.Binary] = true
				binaries = append(binaries, a.Binary)
			}
		}
	}
	return binaries
}

// Persona is a minimal representation of a manifest persona for preflight binary collection.
type Persona struct {
	Adapter string
}

// AdapterDef is a minimal representation of a manifest adapter for preflight binary collection.
type AdapterDef struct {
	Binary string
}

// StepRef is a minimal representation of a pipeline step for preflight binary collection.
type StepRef struct {
	Persona string
	Adapter string // Step-level adapter override
}

// CheckAdapterBinaries verifies that all referenced adapter binaries are available on PATH.
// It collects unique adapter binaries from the given adapter map and checks each.
func (c *Checker) CheckAdapterBinaries(adapterBinaries []string) ([]Result, error) {
	seen := make(map[string]bool)
	var unique []string
	for _, binary := range adapterBinaries {
		if binary == "" || seen[binary] {
			continue
		}
		seen[binary] = true
		unique = append(unique, binary)
	}
	return c.CheckTools(unique)
}

// PreflightError is a composite error returned when both tools and skills fail.
// It implements errors.As() for both SkillError and ToolError so callers can
// extract either typed error from the chain.
type PreflightError struct {
	SkillErr *SkillError
	ToolErr  *ToolError
}

// Error implements the error interface.
func (e *PreflightError) Error() string {
	parts := make([]string, 0, 2)
	if e.ToolErr != nil {
		parts = append(parts, e.ToolErr.Error())
	}
	if e.SkillErr != nil {
		parts = append(parts, e.SkillErr.Error())
	}
	return strings.Join(parts, "; ")
}

// As implements errors.As support so callers can extract either SkillError or ToolError.
func (e *PreflightError) As(target interface{}) bool {
	switch t := target.(type) {
	case **SkillError:
		if e.SkillErr != nil {
			*t = e.SkillErr
			return true
		}
	case **ToolError:
		if e.ToolErr != nil {
			*t = e.ToolErr
			return true
		}
	}
	return false
}
