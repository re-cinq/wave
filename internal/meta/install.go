package meta

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// InstallResult represents the outcome of attempting to install a dependency.
type InstallResult struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	RunCommand(ctx context.Context, command string) error
}

// defaultCommandRunner executes commands via sh -c.
type defaultCommandRunner struct{}

func (r *defaultCommandRunner) RunCommand(ctx context.Context, command string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Installer handles auto-installation of missing dependencies.
type Installer struct {
	runner  CommandRunner
	timeout time.Duration
}

// InstallerOption configures the Installer.
type InstallerOption func(*Installer)

// WithCommandRunner sets a custom command runner (for testing).
func WithCommandRunner(r CommandRunner) InstallerOption {
	return func(i *Installer) { i.runner = r }
}

// WithInstallTimeout sets the timeout for install commands.
func WithInstallTimeout(d time.Duration) InstallerOption {
	return func(i *Installer) { i.timeout = d }
}

// NewInstaller creates a new Installer with the given options.
func NewInstaller(opts ...InstallerOption) *Installer {
	i := &Installer{
		runner:  &defaultCommandRunner{},
		timeout: 60 * time.Second,
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// GetInstallable returns the list of auto-installable dependencies from a DependencyReport.
// It returns dependencies that are not available but have auto-install support.
func GetInstallable(report DependencyReport) []DependencyStatus {
	var result []DependencyStatus
	for _, dep := range report.Tools {
		if !dep.Available && dep.AutoInstallable {
			result = append(result, dep)
		}
	}
	for _, dep := range report.Skills {
		if !dep.Available && dep.AutoInstallable {
			result = append(result, dep)
		}
	}
	return result
}

// Install attempts to install dependencies using their install commands.
// installCommands maps dependency name to the shell command to run.
// Returns results for each attempted installation.
func (i *Installer) Install(ctx context.Context, deps []DependencyStatus, installCommands map[string]string) []InstallResult {
	results := make([]InstallResult, 0, len(deps))

	for _, dep := range deps {
		cmd, ok := installCommands[dep.Name]
		if !ok {
			results = append(results, InstallResult{
				Name:    dep.Name,
				Success: false,
				Message: fmt.Sprintf("no install command configured for %q", dep.Name),
			})
			continue
		}

		installCtx, cancel := context.WithTimeout(ctx, i.timeout)
		err := i.runner.RunCommand(installCtx, cmd)
		cancel()

		if err != nil {
			results = append(results, InstallResult{
				Name:    dep.Name,
				Success: false,
				Message: fmt.Sprintf("install failed: %v", err),
			})
		} else {
			results = append(results, InstallResult{
				Name:    dep.Name,
				Success: true,
				Message: fmt.Sprintf("successfully installed %q", dep.Name),
			})
		}
	}

	return results
}
