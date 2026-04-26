// Package checks provides shared host-capability probes for the
// internal/preflight (gating + remediation) and internal/doctor
// (read-only diagnostics) packages. Each probe returns a typed status
// struct so consumers can attach their own messaging, severity, and
// fix-strings without duplicating probe logic.
package checks

import (
	"os"
	"os/exec"
	"path/filepath"
)

// RunCmdFunc executes a command by name with arguments and returns an error
// if it fails. It is the single seam used by probes for command execution
// to allow test fakes.
type RunCmdFunc func(name string, args ...string) error

// RunCmdEnvFunc executes a command with extra environment variables appended
// to the inherited environment. Used by SkillInstalledWithToolBin's PATH
// fallback path.
type RunCmdEnvFunc func(env []string, name string, args ...string) error

// LookPathFunc resolves a binary name on PATH. Mirrors exec.LookPath.
type LookPathFunc func(name string) (string, error)

// DockerStatus reports the outcome of a Docker daemon probe.
type DockerStatus struct {
	// BinaryFound is true when the docker CLI binary is on PATH.
	BinaryFound bool
	// DaemonUp is true when `docker info` succeeds (binary present AND daemon reachable).
	DaemonUp bool
}

// SkillStatus reports the outcome of a skill probe.
type SkillStatus struct {
	// HasCheck is true when the skill config provides a non-empty check command.
	HasCheck bool
	// Installed is true when HasCheck and the check command exits 0.
	Installed bool
}

// DefaultRunCmd executes a command via os/exec and returns its run error.
func DefaultRunCmd(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

// DefaultRunCmdWithEnv executes a command with extra environment variables
// appended to os.Environ(). Used as the default for SkillInstalledWithToolBin's
// PATH fallback.
func DefaultRunCmdWithEnv(env []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), env...)
	return cmd.Run()
}

// DockerDaemon probes whether the Docker CLI is on PATH and the daemon is up.
// runCmd defaults to DefaultRunCmd; lookPath defaults to exec.LookPath. Pass
// nil for either to use the default.
func DockerDaemon(runCmd RunCmdFunc, lookPath LookPathFunc) DockerStatus {
	if runCmd == nil {
		runCmd = DefaultRunCmd
	}
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	if _, err := lookPath("docker"); err != nil {
		return DockerStatus{}
	}
	if err := runCmd("docker", "info"); err != nil {
		return DockerStatus{BinaryFound: true}
	}
	return DockerStatus{BinaryFound: true, DaemonUp: true}
}

// Skill probes whether a skill is installed by running its check command via
// `sh -c`. An empty check command yields HasCheck=false and Installed=false.
// runCmd defaults to DefaultRunCmd.
func Skill(runCmd RunCmdFunc, check string) SkillStatus {
	if check == "" {
		return SkillStatus{}
	}
	if runCmd == nil {
		runCmd = DefaultRunCmd
	}
	if err := runCmd("sh", "-c", check); err != nil {
		return SkillStatus{HasCheck: true}
	}
	return SkillStatus{HasCheck: true, Installed: true}
}

// SkillInstalled is a bool-returning convenience over Skill.
func SkillInstalled(runCmd RunCmdFunc, check string) bool {
	return Skill(runCmd, check).Installed
}

// SkillInstalledWithToolBin first runs the skill check via runCmd. If that
// fails, it retries with $HOME/.local/bin prepended to PATH using runCmdEnv
// to handle install tools (uv, pip, cargo) that drop binaries there in
// sandboxed/detached environments. runCmd defaults to DefaultRunCmd;
// runCmdEnv defaults to DefaultRunCmdWithEnv.
func SkillInstalledWithToolBin(runCmd RunCmdFunc, runCmdEnv RunCmdEnvFunc, check string) bool {
	if check == "" {
		return false
	}
	if runCmd == nil {
		runCmd = DefaultRunCmd
	}
	if SkillInstalled(runCmd, check) {
		return true
	}
	home := os.Getenv("HOME")
	if home == "" {
		return false
	}
	if runCmdEnv == nil {
		runCmdEnv = DefaultRunCmdWithEnv
	}
	toolBin := filepath.Join(home, ".local", "bin")
	enhancedPath := toolBin + string(os.PathListSeparator) + os.Getenv("PATH")
	return runCmdEnv([]string{"PATH=" + enhancedPath}, "sh", "-c", check) == nil
}
