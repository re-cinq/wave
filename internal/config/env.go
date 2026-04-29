// Package config centralizes process-wide environment variable reads.
//
// Direct os.Getenv calls scattered across packages create drift hazards
// (the same env var may be read with subtly different semantics in different
// places) and obstruct testing. This package consolidates the reads behind
// a single typed surface.
//
// Two consumption styles are supported:
//
//   - Env: a snapshot struct populated by FromEnv(). Use for static reads
//     resolved once at startup (e.g. terminal capability detection).
//   - Lookup: a thin os.Getenv wrapper for sites that read dynamically by
//     key name (e.g. token introspection, admin credential probing).
//
// Only env vars actually consumed by the codebase are surfaced. New env vars
// must be added here intentionally rather than scattered across packages.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Env is a snapshot of relevant process environment variables, captured by
// FromEnv. Consumers receive Env by value or read individual fields via the
// accessor methods so tests can substitute alternative snapshots.
type Env struct {
	// AnthropicAPIKey is the API key used by the LLM-judge contract validator
	// when an OAuth-based Claude CLI session is unavailable.
	AnthropicAPIKey string

	// GitHubToken is the token used by the GitHub adapter when no explicit
	// token is supplied. Resolved from GITHUB_TOKEN.
	GitHubToken string

	// HOME and PATH are forwarded into curated subprocess environments
	// (hooks, detached runs) so child processes can locate user binaries
	// and config dirs without inheriting the full host environment.
	Home string
	Path string

	// Term, ColorTerm, Lang, LCAll, ColorFGBG, ITermProfile influence
	// terminal capability detection.
	Term         string
	ColorTerm    string
	Lang         string
	LCAll        string
	ColorFGBG    string
	ITermProfile string

	// NoColor disables ANSI color output when set to any non-empty value.
	NoColor string

	// NoUnicode disables Unicode glyph fallbacks when set.
	NoUnicode string

	// ForceTTY overrides terminal detection. "1"/"true" = TTY, anything else
	// non-empty = non-TTY. Empty string = auto-detect.
	ForceTTY string

	// Columns and Lines are fallback terminal-size hints when ioctl fails.
	Columns string
	Lines   string
}

// FromEnv captures the current process environment into an Env snapshot.
// Call once at startup or per-test to obtain a stable view of env state.
func FromEnv() Env {
	return Env{
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		GitHubToken:     os.Getenv("GITHUB_TOKEN"),
		Home:            os.Getenv("HOME"),
		Path:            os.Getenv("PATH"),
		Term:            os.Getenv("TERM"),
		ColorTerm:       os.Getenv("COLORTERM"),
		Lang:            os.Getenv("LANG"),
		LCAll:           os.Getenv("LC_ALL"),
		ColorFGBG:       os.Getenv("COLORFGBG"),
		ITermProfile:    os.Getenv("ITERM_PROFILE"),
		NoColor:         os.Getenv("NO_COLOR"),
		NoUnicode:       os.Getenv("NO_UNICODE"),
		ForceTTY:        os.Getenv("WAVE_FORCE_TTY"),
		Columns:         os.Getenv("COLUMNS"),
		Lines:           os.Getenv("LINES"),
	}
}

// Lookup returns the value of an environment variable by name. It exists for
// sites that probe env keys dynamically (introspection, credential surface
// detection) where a typed snapshot field cannot be pre-declared.
//
// Prefer reading fields from Env where the key is known statically.
func Lookup(key string) string {
	return os.Getenv(key)
}

// EnvPresent reports whether the named environment variable is set to a
// non-empty value. Use for presence-only probes (e.g. token introspection
// preconditions, admin credential surface detection) where the value itself
// is not consumed at the call site.
func EnvPresent(key string) bool {
	return os.Getenv(key) != ""
}

// HomeOr returns the HOME value or the supplied fallback if HOME is unset.
// Convenience for subprocess env construction.
func (e Env) HomeOr(fallback string) string {
	if e.Home != "" {
		return e.Home
	}
	return fallback
}

// TermOr returns TERM or the supplied fallback. Used by hook env construction
// where a sensible default ("xterm-256color") is preferred over an empty TERM.
func (e Env) TermOr(fallback string) string {
	if e.Term != "" {
		return e.Term
	}
	return fallback
}

// SubprocessHomePath returns the HOME and PATH values intended to be forwarded
// into curated subprocess environments. The values are returned as-is from the
// inherited process environment — callers that need a fallback (for example,
// when HOME is unset) should apply it explicitly. Centralising the read here
// keeps the inherited-env contract auditable from a single place.
func SubprocessHomePath() (home, path string) {
	env := FromEnv()
	return env.Home, env.Path
}

// MigrationEnv groups the four WAVE_MIGRATION_* environment variables that
// configure the schema migration subsystem. A nil pointer field means the
// corresponding env var was unset and the consumer should keep its default.
//
// The boolean fields accept "true", "1", or "yes" (case-insensitive) as truthy
// values; any other non-empty value is interpreted as false. The explicit
// pointer-vs-zero-value distinction lets callers tell "unset" apart from
// "explicitly set to false".
type MigrationEnv struct {
	Enabled              *bool
	AutoMigrate          *bool
	SkipValidation       *bool
	MaxVersion           *int
	MaxVersionParseError error // non-nil when WAVE_MAX_MIGRATION_VERSION was set but failed to parse
	MaxVersionRawValue   string
}

// LoadMigrationEnv reads the four WAVE_MIGRATION_* environment variables and
// returns a MigrationEnv with each field populated only when its underlying
// env var was set to a non-empty value. The version int parser surfaces an
// explicit error rather than silently dropping malformed values.
func LoadMigrationEnv() MigrationEnv {
	out := MigrationEnv{}
	if v := os.Getenv("WAVE_MIGRATION_ENABLED"); v != "" {
		b := parseBoolish(v)
		out.Enabled = &b
	}
	if v := os.Getenv("WAVE_AUTO_MIGRATE"); v != "" {
		b := parseBoolish(v)
		out.AutoMigrate = &b
	}
	if v := os.Getenv("WAVE_SKIP_MIGRATION_VALIDATION"); v != "" {
		b := parseBoolish(v)
		out.SkipValidation = &b
	}
	if v := os.Getenv("WAVE_MAX_MIGRATION_VERSION"); v != "" {
		out.MaxVersionRawValue = v
		n, err := strconv.Atoi(v)
		if err != nil {
			out.MaxVersionParseError = fmt.Errorf("WAVE_MAX_MIGRATION_VERSION=%q: %w", v, err)
		} else {
			out.MaxVersion = &n
		}
	}
	return out
}

// parseBoolish returns true when v (case-insensitive, trimmed) matches one of
// the accepted truthy spellings: "true", "1", "yes". Any other non-empty value
// returns false. Empty strings should be filtered out by the caller.
func parseBoolish(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}
