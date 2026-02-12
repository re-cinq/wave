package preflight

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/recinq/wave/internal/manifest"
)

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
		return results, fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
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
		return results, fmt.Errorf("missing required skills: %s", strings.Join(failed, ", "))
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
func (c *Checker) Run(tools, skills []string) ([]Result, error) {
	var allResults []Result
	var errors []string

	if len(tools) > 0 {
		toolResults, err := c.CheckTools(tools)
		allResults = append(allResults, toolResults...)
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(skills) > 0 {
		skillResults, err := c.CheckSkills(skills)
		allResults = append(allResults, skillResults...)
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return allResults, fmt.Errorf("preflight check failed: %s", strings.Join(errors, "; "))
	}
	return allResults, nil
}
