package contract

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// projectRoot returns the project root by walking up from the test file location.
func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get caller info")

	// Walk up from internal/contract/ to project root
	dir := filepath.Dir(filename)
	root := filepath.Join(dir, "..", "..")
	absRoot, err := filepath.Abs(root)
	require.NoError(t, err)
	return absRoot
}

func TestSharedSchemaSync(t *testing.T) {
	root := projectRoot(t)
	waveDir := filepath.Join(root, ".wave", "contracts")
	defaultsDir := filepath.Join(root, "internal", "defaults", "contracts")

	// Find all shared-*.schema.json files in .wave/contracts/
	entries, err := os.ReadDir(waveDir)
	require.NoError(t, err, "failed to read .wave/contracts/ directory")

	sharedSchemas := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) > 7 && name[:7] == "shared-" && filepath.Ext(name) == ".json" {
			sharedSchemas = append(sharedSchemas, name)
		}
	}

	require.NotEmpty(t, sharedSchemas, "expected at least one shared-*.schema.json in .wave/contracts/")

	for _, schemaName := range sharedSchemas {
		t.Run(schemaName, func(t *testing.T) {
			wavePath := filepath.Join(waveDir, schemaName)
			defaultsPath := filepath.Join(defaultsDir, schemaName)

			waveContent, err := os.ReadFile(wavePath)
			require.NoError(t, err, "failed to read .wave/contracts/%s", schemaName)

			defaultsContent, err := os.ReadFile(defaultsPath)
			require.NoError(t, err, "shared schema %s exists in .wave/contracts/ but not in internal/defaults/contracts/ — run: cp .wave/contracts/%s internal/defaults/contracts/", schemaName, schemaName)

			assert.Equal(t, string(waveContent), string(defaultsContent),
				"schema %s has diverged between .wave/contracts/ and internal/defaults/contracts/ — .wave/contracts/ is authoritative, sync with: cp .wave/contracts/%s internal/defaults/contracts/", schemaName, schemaName)
		})
	}
}
