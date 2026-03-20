//go:build ignore

// Package contract defines the API contracts for skill source adapters.
// This file is a design artifact — it documents the interfaces and types
// that the implementation must satisfy.
package contract

import "context"

// --- FR-001: SourceAdapter interface ---

// SourceAdapter handles installation of skills from a specific source type.
// Each adapter handles exactly one source prefix.
type SourceAdapter interface {
	// Install fetches skills from the source reference and writes them to the store.
	//
	// Parameters:
	//   - ctx: carries timeout deadline (2min default) and cancellation signal
	//   - ref: adapter-specific locator (part after the prefix)
	//   - store: skill store to write installed skills into
	//
	// Returns:
	//   - *InstallResult with installed skills and warnings on success
	//   - *DependencyError when a required CLI tool is missing
	//   - error for all other failures (network, parse, validation)
	//
	// Contracts:
	//   - MUST create temp dir via os.MkdirTemp and clean up via defer os.RemoveAll
	//   - MUST validate all SKILL.md content via skill.Parse before writing
	//   - MUST respect ctx deadline for subprocess and network operations
	//   - MUST return actionable error when CLI dependency is missing (FR-011)
	Install(ctx context.Context, ref string, store Store) (*InstallResult, error)

	// Prefix returns the source prefix this adapter handles.
	// Examples: "tessl", "github", "file", "https://"
	//
	// Contracts:
	//   - MUST return a non-empty string
	//   - MUST be unique across all registered adapters
	Prefix() string
}

// --- FR-002: SourceRouter ---

// SourceRouter parses source strings and dispatches to the correct adapter.
//
// Parsing rules (FR-002):
//   - First check for URL scheme prefixes: "https://", "http://"
//   - Then split on first ":" for standard prefixes
//   - Bare names (no prefix) are NOT routed — they are local store lookups (FR-014)
//
// Contracts:
//   - MUST return error for unknown prefixes listing all recognized prefixes (FR-015)
//   - MUST support all 7 prefixes: tessl, bmad, openspec, speckit, github, file, https:// (FR-003)
type SourceRouter interface {
	Install(ctx context.Context, source string, store Store) (*InstallResult, error)
	Parse(source string) (SourceAdapter, string, error)
	Prefixes() []string
}

// --- Return types ---

// InstallResult is the outcome of an adapter invocation.
type InstallResult struct {
	Skills   []Skill  // Successfully installed skills
	Warnings []string // Non-fatal issues encountered
}

// --- Error types ---

// DependencyError indicates a required CLI tool is not available.
//
// Contracts:
//   - Binary MUST be the exact binary name checked via exec.LookPath
//   - Instructions MUST provide a concrete install command
type DependencyError struct {
	Binary       string // e.g., "tessl", "git", "npx"
	Instructions string // e.g., "npm i -g @tessl/cli"
}

// --- Types referenced from existing skill package (not redefined) ---

// Store is the skill store interface — same as skill.Store.
type Store interface {
	Read(name string) (Skill, error)
	Write(skill Skill) error
	List() ([]Skill, error)
	Delete(name string) error
}

// Skill is the parsed SKILL.md representation — same as skill.Skill.
type Skill struct {
	Name          string
	Description   string
	Body          string
	License       string
	Compatibility string
	Metadata      map[string]string
	AllowedTools  []string
	SourcePath    string
	ResourcePaths []string
}
