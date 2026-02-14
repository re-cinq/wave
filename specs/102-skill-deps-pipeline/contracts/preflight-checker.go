// Package contracts defines the interfaces and types for the skill dependency
// installation feature. These are API contracts — the implementation must
// satisfy these signatures.
//
// NOTE: This file is a design artifact, not production code. It documents
// the expected interfaces that will be implemented in internal/preflight/.

package contracts

// --- Checker API Contract ---

// CheckerOption configures optional behavior on the preflight Checker.
// Used via functional options pattern.
//
// Expected options:
//   - WithEmitter(fn func(name, kind, message string)) — callback for per-dependency events
//   - WithRunCmd(fn func(name string, args ...string) error) — override command execution (for testing)
type CheckerOption interface{}

// CheckerContract defines the public API of the preflight Checker.
//
// Implementation location: internal/preflight/preflight.go
//
// Changes from current API:
//   - NewChecker gains variadic CheckerOption parameters
//   - No changes to CheckTools, CheckSkills, or Run signatures
//   - Event emission happens internally via the emitter callback
type CheckerContract interface {
	// CheckTools verifies that all required CLI tools are available on PATH.
	// Returns one Result per tool. Returns error if any tool is missing.
	CheckTools(tools []string) ([]Result, error)

	// CheckSkills verifies that all required skills are installed.
	// Attempts auto-install for skills with install commands.
	// Returns one Result per skill. Returns error if any skill is unsatisfied.
	CheckSkills(skills []string) ([]Result, error)

	// Run executes all preflight checks for the given tool and skill requirements.
	// Combines CheckTools and CheckSkills results.
	Run(tools, skills []string) ([]Result, error)
}

// Result represents the outcome of a single preflight check.
// No changes from existing type.
type Result struct {
	Name    string // Tool or skill name
	Kind    string // "tool" or "skill"
	OK      bool
	Message string
}
