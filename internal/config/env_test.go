package config

import (
	"testing"
)

func TestFromEnvCapturesValues(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-test")
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("WAVE_FORCE_TTY", "1")
	t.Setenv("COLUMNS", "120")
	t.Setenv("LINES", "40")

	env := FromEnv()

	if env.AnthropicAPIKey != "sk-test" {
		t.Errorf("AnthropicAPIKey = %q, want sk-test", env.AnthropicAPIKey)
	}
	if env.GitHubToken != "ghp_test" {
		t.Errorf("GitHubToken = %q, want ghp_test", env.GitHubToken)
	}
	if env.NoColor != "1" {
		t.Errorf("NoColor = %q, want 1", env.NoColor)
	}
	if env.ForceTTY != "1" {
		t.Errorf("ForceTTY = %q, want 1", env.ForceTTY)
	}
	if env.Columns != "120" {
		t.Errorf("Columns = %q, want 120", env.Columns)
	}
	if env.Lines != "40" {
		t.Errorf("Lines = %q, want 40", env.Lines)
	}
}

func TestHomeOr(t *testing.T) {
	t.Setenv("HOME", "")
	env := FromEnv()
	if got := env.HomeOr("/tmp"); got != "/tmp" {
		t.Errorf("HomeOr(/tmp) with empty HOME = %q, want /tmp", got)
	}

	t.Setenv("HOME", "/home/user")
	env = FromEnv()
	if got := env.HomeOr("/tmp"); got != "/home/user" {
		t.Errorf("HomeOr(/tmp) with HOME set = %q, want /home/user", got)
	}
}

func TestTermOr(t *testing.T) {
	t.Setenv("TERM", "")
	env := FromEnv()
	if got := env.TermOr("xterm-256color"); got != "xterm-256color" {
		t.Errorf("TermOr fallback = %q, want xterm-256color", got)
	}

	t.Setenv("TERM", "screen")
	env = FromEnv()
	if got := env.TermOr("xterm-256color"); got != "screen" {
		t.Errorf("TermOr explicit = %q, want screen", got)
	}
}

func TestLookup(t *testing.T) {
	t.Setenv("WAVE_TEST_LOOKUP_VAR", "value")
	if got := Lookup("WAVE_TEST_LOOKUP_VAR"); got != "value" {
		t.Errorf("Lookup = %q, want value", got)
	}
	if got := Lookup("WAVE_TEST_DOES_NOT_EXIST_XYZ"); got != "" {
		t.Errorf("Lookup of unset = %q, want empty", got)
	}
}
