// Package git centralizes git command invocations used by Wave subsystems
// (notably the TUI providers) so that all git execution flows through a
// single audited point. Tests can swap the underlying runner via SetRunner.
package git

import (
	"os/exec"
	"strings"
	"sync"
)

// Runner abstracts git command execution. Implementations return the raw
// stdout (callers trim/parse) or an error. Tests can substitute a fake to
// exercise providers without invoking the git binary.
type Runner func(args ...string) ([]byte, error)

var (
	runnerMu sync.RWMutex
	runner   Runner = defaultRunner
)

// defaultRunner exec's the real git binary.
func defaultRunner(args ...string) ([]byte, error) {
	return exec.Command("git", args...).Output()
}

// SetRunner installs a Runner used by every package-level helper. It returns
// the previous Runner so test code can restore it. Callers MUST restore the
// previous runner (typically with t.Cleanup) to avoid cross-test leakage.
func SetRunner(r Runner) Runner {
	runnerMu.Lock()
	defer runnerMu.Unlock()
	prev := runner
	if r == nil {
		runner = defaultRunner
	} else {
		runner = r
	}
	return prev
}

// run executes git through the active runner.
func run(args ...string) ([]byte, error) {
	runnerMu.RLock()
	r := runner
	runnerMu.RUnlock()
	return r(args...)
}

// Branch returns the current branch name (`git rev-parse --abbrev-ref HEAD`).
// Returns an error if the working directory is not a git repository.
func Branch() (string, error) {
	out, err := run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ShortHash returns the abbreviated commit hash for HEAD
// (`git rev-parse --short HEAD`).
func ShortHash() (string, error) {
	out, err := run("rev-parse", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// IsDirty returns true when the working tree has any uncommitted changes
// (staged, unstaged, or untracked) according to `git status --porcelain`.
func IsDirty() (bool, error) {
	out, err := run("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

// FirstRemote returns the name of the first configured remote
// (typically "origin"). Returns an empty string when no remote is configured.
func FirstRemote() (string, error) {
	out, err := run("remote")
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", nil
	}
	return lines[0], nil
}

// VerifyRef returns true when the given ref resolves to a commit
// (`git rev-parse --verify <ref>`). A missing ref returns false with nil err.
func VerifyRef(ref string) (bool, error) {
	_, err := run("rev-parse", "--verify", ref)
	if err != nil {
		// rev-parse exits non-zero when the ref doesn't exist; that's a
		// well-formed "no" rather than an error condition for callers.
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
