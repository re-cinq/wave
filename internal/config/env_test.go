package config

import (
	"os"
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

func TestEnvPresent(t *testing.T) {
	const key = "WAVE_TEST_PRESENT_VAR"
	_ = os.Unsetenv(key)
	if EnvPresent(key) {
		t.Fatalf("EnvPresent(%s) = true, want false (unset)", key)
	}

	t.Setenv(key, "")
	if EnvPresent(key) {
		t.Errorf("EnvPresent(%s) with empty value = true, want false", key)
	}

	t.Setenv(key, "anything")
	if !EnvPresent(key) {
		t.Errorf("EnvPresent(%s) with value = false, want true", key)
	}
}

func TestSubprocessHomePath(t *testing.T) {
	t.Setenv("HOME", "/home/runner")
	t.Setenv("PATH", "/usr/local/bin:/usr/bin")

	home, path := SubprocessHomePath()
	if home != "/home/runner" {
		t.Errorf("home = %q, want /home/runner", home)
	}
	if path != "/usr/local/bin:/usr/bin" {
		t.Errorf("path = %q, want /usr/local/bin:/usr/bin", path)
	}
}

func TestParseBoolish(t *testing.T) {
	truthy := []string{"true", "TRUE", "True", "1", "yes", "YES", "Yes", "  true  ", "  1\t"}
	for _, v := range truthy {
		if !parseBoolish(v) {
			t.Errorf("parseBoolish(%q) = false, want true", v)
		}
	}

	falsy := []string{"false", "FALSE", "0", "no", "off", "anything", "  false  ", "2"}
	for _, v := range falsy {
		if parseBoolish(v) {
			t.Errorf("parseBoolish(%q) = true, want false", v)
		}
	}
}

func TestLoadMigrationEnv_AllUnset(t *testing.T) {
	for _, k := range []string{
		"WAVE_MIGRATION_ENABLED",
		"WAVE_AUTO_MIGRATE",
		"WAVE_SKIP_MIGRATION_VALIDATION",
		"WAVE_MAX_MIGRATION_VERSION",
	} {
		_ = os.Unsetenv(k)
	}

	got := LoadMigrationEnv()
	if got.Enabled != nil || got.AutoMigrate != nil || got.SkipValidation != nil || got.MaxVersion != nil {
		t.Errorf("LoadMigrationEnv() with all unset returned non-nil pointers: %+v", got)
	}
	if got.MaxVersionParseError != nil {
		t.Errorf("MaxVersionParseError = %v, want nil", got.MaxVersionParseError)
	}
}

func TestLoadMigrationEnv_AllSet(t *testing.T) {
	t.Setenv("WAVE_MIGRATION_ENABLED", "false")
	t.Setenv("WAVE_AUTO_MIGRATE", "yes")
	t.Setenv("WAVE_SKIP_MIGRATION_VALIDATION", "1")
	t.Setenv("WAVE_MAX_MIGRATION_VERSION", "7")

	got := LoadMigrationEnv()
	if got.Enabled == nil || *got.Enabled != false {
		t.Errorf("Enabled = %v, want false", got.Enabled)
	}
	if got.AutoMigrate == nil || *got.AutoMigrate != true {
		t.Errorf("AutoMigrate = %v, want true", got.AutoMigrate)
	}
	if got.SkipValidation == nil || *got.SkipValidation != true {
		t.Errorf("SkipValidation = %v, want true", got.SkipValidation)
	}
	if got.MaxVersion == nil || *got.MaxVersion != 7 {
		t.Errorf("MaxVersion = %v, want 7", got.MaxVersion)
	}
	if got.MaxVersionParseError != nil {
		t.Errorf("MaxVersionParseError = %v, want nil", got.MaxVersionParseError)
	}
}

func TestLoadMigrationEnv_VersionParseError(t *testing.T) {
	t.Setenv("WAVE_MAX_MIGRATION_VERSION", "not-a-number")

	got := LoadMigrationEnv()
	if got.MaxVersion != nil {
		t.Errorf("MaxVersion = %v, want nil on parse failure", got.MaxVersion)
	}
	if got.MaxVersionParseError == nil {
		t.Fatal("MaxVersionParseError = nil, want non-nil")
	}
	if got.MaxVersionRawValue != "not-a-number" {
		t.Errorf("MaxVersionRawValue = %q, want not-a-number", got.MaxVersionRawValue)
	}
}
