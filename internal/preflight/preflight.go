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
	Name    string // Tool or skill name
	Kind    string // "tool" or "skill"
	OK      bool
	Message string
}

// Checker validates that pipeline dependencies are satisfied before execution.
type Checker struct {
	skills  map[string]manifest.SkillConfig
	runCmd  func(name string, args ...string) error // for testing
}

// NewChecker creates a preflight checker with the given skill configurations.
func NewChecker(skills map[string]manifest.SkillConfig) *Checker {
	return &Checker{
		skills: skills,
		runCmd: defaultRunCmd,
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
		_, err := exec.LookPath(tool)
		if err != nil {
			results = append(results, Result{
				Name:    tool,
				Kind:    "tool",
				OK:      false,
				Message: fmt.Sprintf("tool %q not found on PATH", tool),
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
				Name:    name,
				Kind:    "skill",
				OK:      false,
				Message: fmt.Sprintf("skill %q not declared in wave.yaml skills section", name),
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
			results = append(results, Result{
				Name:    name,
				Kind:    "skill",
				OK:      false,
				Message: fmt.Sprintf("skill %q install failed: %v", name, err),
			})
			failed = append(failed, name)
			continue
		}

		// Run init command if configured
		if cfg.Init != "" {
			if err := c.runShellCommand(cfg.Init); err != nil {
				results = append(results, Result{
					Name:    name,
					Kind:    "skill",
					OK:      false,
					Message: fmt.Sprintf("skill %q init failed: %v", name, err),
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
				Name:    name,
				Kind:    "skill",
				OK:      false,
				Message: fmt.Sprintf("skill %q still not detected after install", name),
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
