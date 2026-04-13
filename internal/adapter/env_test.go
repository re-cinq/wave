package adapter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildAdapterEnv(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .env.local file
	envLocalPath := filepath.Join(tmpDir, ".env.local")
	err := os.WriteFile(envLocalPath, []byte("TEST_VAR=from_dotenv_local\n"), 0644)
	require.NoError(t, err)

	// Create .env file
	envPath := filepath.Join(tmpDir, ".env")
	err = os.WriteFile(envPath, []byte("TEST_VAR2=from_dotenv\n"), 0644)
	require.NoError(t, err)

	cfg := AdapterRunConfig{
		ManifestRoot: tmpDir,
		EnvOverrides: map[string]string{
			"OVERRIDE_VAR": "${TEST_VAR}",
		},
	}

	env, err := buildAdapterEnv(cfg)
	require.NoError(t, err)

	found := map[string]string{}
	for _, e := range env {
		key, val := splitEnvVar(e)
		if key != "" {
			found[key] = val
		}
	}

	require.Equal(t, "from_dotenv_local", found["TEST_VAR"])
	require.Equal(t, "from_dotenv", found["TEST_VAR2"])
	require.Equal(t, "from_dotenv_local", found["OVERRIDE_VAR"])
}

func splitEnvVar(s string) (string, string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:]
		}
	}
	return "", ""
}
