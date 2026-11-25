package adapter

import (
	"os"
	"strings"
	"testing"
)

func TestBuildCuratedEnvironment(t *testing.T) {
	t.Run("base vars present", func(t *testing.T) {
		cfg := AdapterRunConfig{}
		env := BuildCuratedEnvironment(cfg)
		envMap := envToMap(env)

		for _, key := range []string{"HOME", "PATH", "TERM", "TMPDIR"} {
			if _, ok := envMap[key]; !ok {
				t.Errorf("expected base var %s to be present", key)
			}
		}
	})

	t.Run("passthrough vars included when set", func(t *testing.T) {
		t.Setenv("TEST_PASSTHROUGH_KEY", "test-value-123")
		cfg := AdapterRunConfig{
			EnvPassthrough: []string{"TEST_PASSTHROUGH_KEY"},
		}
		env := BuildCuratedEnvironment(cfg)
		envMap := envToMap(env)

		if v, ok := envMap["TEST_PASSTHROUGH_KEY"]; !ok || v != "test-value-123" {
			t.Errorf("expected TEST_PASSTHROUGH_KEY=test-value-123, got %q (present=%v)", v, ok)
		}
	})

	t.Run("missing passthrough vars silently skipped", func(t *testing.T) {
		os.Unsetenv("NONEXISTENT_CURATED_TEST_VAR")
		cfg := AdapterRunConfig{
			EnvPassthrough: []string{"NONEXISTENT_CURATED_TEST_VAR"},
		}
		env := BuildCuratedEnvironment(cfg)
		envMap := envToMap(env)

		if _, ok := envMap["NONEXISTENT_CURATED_TEST_VAR"]; ok {
			t.Error("expected missing passthrough var to be absent")
		}
	})

	t.Run("step-specific env vars appended", func(t *testing.T) {
		cfg := AdapterRunConfig{
			Env: []string{"STEP_VAR=step-value"},
		}
		env := BuildCuratedEnvironment(cfg)
		envMap := envToMap(env)

		if v, ok := envMap["STEP_VAR"]; !ok || v != "step-value" {
			t.Errorf("expected STEP_VAR=step-value, got %q (present=%v)", v, ok)
		}
	})

	t.Run("canary var NOT leaked", func(t *testing.T) {
		t.Setenv("CANARY_SECRET_VAR", "should-not-leak")
		cfg := AdapterRunConfig{}
		env := BuildCuratedEnvironment(cfg)
		envMap := envToMap(env)

		if _, ok := envMap["CANARY_SECRET_VAR"]; ok {
			t.Error("expected canary var CANARY_SECRET_VAR to NOT be present in curated environment")
		}
	})
}

func envToMap(env []string) map[string]string {
	m := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}
