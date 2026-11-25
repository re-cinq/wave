package adapter

import (
	"os"
)

// BuildCuratedEnvironment constructs a curated environment for adapter subprocesses.
// It includes base variables (HOME, PATH, TERM, TMPDIR), explicitly allowed
// passthrough variables from the manifest, and step-specific env vars.
func BuildCuratedEnvironment(cfg AdapterRunConfig) []string {
	env := []string{
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
		"TERM=" + getenvDefault("TERM", "xterm-256color"),
		"TMPDIR=/tmp",
	}

	// Add explicitly allowed env vars from manifest
	for _, key := range cfg.EnvPassthrough {
		if val := os.Getenv(key); val != "" {
			env = append(env, key+"="+val)
		}
	}

	// Step-specific env vars (from pipeline config)
	env = append(env, cfg.Env...)
	return env
}
