package preflight

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/recinq/wave/internal/manifest"
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
	Name        string `json:"name"`                  // Tool or skill name
	Kind        string `json:"kind"`                  // "tool", "skill", "forge", "adapter", or "init"
	OK          bool   `json:"ok"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"` // Optional install/fix guidance
}

// Checker validates that pipeline dependencies are satisfied before execution.
type Checker struct {
	skills   map[string]manifest.SkillConfig
	runCmd   func(name string, args ...string) error // for testing
	lookPath func(file string) (string, error)       // for testing
}

// NewChecker creates a preflight checker with the given skill configurations.
func NewChecker(skills map[string]manifest.SkillConfig) *Checker {
	return &Checker{
		skills:   skills,
		runCmd:   defaultRunCmd,
		lookPath: exec.LookPath,
	}
}

// defaultRunCmd executes a command and returns an error if it fails.
func defaultRunCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// CheckTools verifies that all required CLI tools are available on PATH.
func (c *Checker) CheckTools(tools []string) ([]Result, error) {
	var results []Result
	var missing []string

	for _, tool := range tools {
		_, err := c.lookPath(tool)
		if err != nil {
			results = append(results, Result{
				Name:        tool,
				Kind:        "tool",
				OK:          false,
				Message:     fmt.Sprintf("tool %q not found on PATH", tool),
				Remediation: fmt.Sprintf("Install %q and ensure it is on your PATH", tool),
			})
			missing = append(missing, tool)
		} else {
			results = append(results, Result{
				Name:    tool,
				Kind:    "tool",
				OK:      true,
				Message: fmt.Sprintf("tool %q found", tool),
			})
		}
	}

	if len(missing) > 0 {
		return results, &ToolError{
			MissingTools: missing,
		}
	}
	return results, nil
}

// CheckSkills verifies that all required skills are installed, attempting auto-install if configured.
func (c *Checker) CheckSkills(skills []string) ([]Result, error) {
	var results []Result
	var failed []string

	for _, name := range skills {
		cfg, exists := c.skills[name]
		if !exists {
			results = append(results, Result{
				Name:        name,
				Kind:        "skill",
				OK:          false,
				Message:     fmt.Sprintf("skill %q not declared in wave.yaml skills section", name),
				Remediation: fmt.Sprintf("Add skill %q to the skills section of wave.yaml", name),
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
			results = append(results, Result{
				Name:        name,
				Kind:        "skill",
				OK:          false,
				Message:     fmt.Sprintf("skill %q not installed and no install command configured", name),
				Remediation: fmt.Sprintf("Configure an install command for skill %q in wave.yaml, or install it manually", name),
			})
			failed = append(failed, name)
			continue
		}

		// Run install command
		if err := c.runShellCommand(cfg.Install); err != nil {
			results = append(results, Result{
				Name:        name,
				Kind:        "skill",
				OK:          false,
				Message:     fmt.Sprintf("skill %q install failed: %v", name, err),
				Remediation: fmt.Sprintf("Check that the install command for skill %q is correct in wave.yaml: %s", name, cfg.Install),
			})
			failed = append(failed, name)
			continue
		}

		// Run init command if configured
		if cfg.Init != "" {
			if err := c.runShellCommand(cfg.Init); err != nil {
				results = append(results, Result{
					Name:        name,
					Kind:        "skill",
					OK:          false,
					Message:     fmt.Sprintf("skill %q init failed: %v", name, err),
					Remediation: fmt.Sprintf("Check that the init command for skill %q is correct in wave.yaml: %s", name, cfg.Init),
				})
				failed = append(failed, name)
				continue
			}
		}

		// Re-check after install
		if c.isSkillInstalled(cfg) {
			results = append(results, Result{
				Name:    name,
				Kind:    "skill",
				OK:      true,
				Message: fmt.Sprintf("skill %q installed successfully", name),
			})
		} else {
			results = append(results, Result{
				Name:        name,
				Kind:        "skill",
				OK:          false,
				Message:     fmt.Sprintf("skill %q still not detected after install", name),
				Remediation: fmt.Sprintf("Skill %q check command did not pass after installation — verify the check command: %s", name, cfg.Check),
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
func (c *Checker) isSkillInstalled(cfg manifest.SkillConfig) bool {
	if cfg.Check == "" {
		return false
	}
	return c.runShellCommand(cfg.Check) == nil
}

// runShellCommand executes a shell command string via sh -c.
func (c *Checker) runShellCommand(command string) error {
	return c.runCmd("sh", "-c", command)
}

// CheckAdapterHealth verifies that each configured adapter binary is reachable
// on PATH and can respond to an auth probe command.
func (c *Checker) CheckAdapterHealth(adapters map[string]manifest.Adapter) ([]Result, error) {
	var results []Result

	for name, adapter := range adapters {
		binary := adapter.Binary

		// Check if binary is on PATH
		_, err := c.lookPath(binary)
		if err != nil {
			results = append(results, Result{
				Name:        name,
				Kind:        "adapter",
				OK:          false,
				Message:     fmt.Sprintf("adapter %q binary %q not found on PATH", name, binary),
				Remediation: fmt.Sprintf("Install adapter binary %q and ensure it is on your PATH", binary),
			})
			continue
		}

		// Run auth probe
		var probeErr error
		if adapter.AuthCheck != "" {
			probeErr = c.runCmd("sh", "-c", adapter.AuthCheck)
		} else {
			probeErr = c.runCmd(binary, "--version")
		}

		if probeErr == nil {
			results = append(results, Result{
				Name:    name,
				Kind:    "adapter",
				OK:      true,
				Message: fmt.Sprintf("adapter %q is reachable and responding", name),
			})
		} else {
			remediation := fmt.Sprintf("Run %q to verify it is working correctly", binary)
			switch name {
			case "claude":
				remediation = "Run `claude` to complete authentication, or check your API key"
			case "opencode":
				remediation = "Run `opencode` to complete setup, or check your API key"
			}

			results = append(results, Result{
				Name:        name,
				Kind:        "adapter",
				OK:          false,
				Message:     fmt.Sprintf("adapter %q binary found but health check failed: %v", name, probeErr),
				Remediation: remediation,
			})
		}
	}

	return results, nil
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
