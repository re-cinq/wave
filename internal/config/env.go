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

import "os"

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
