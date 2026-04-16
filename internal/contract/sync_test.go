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

func TestSchemaSync(t *testing.T) {
	root := projectRoot(t)
	waveDir := filepath.Join(root, ".wave", "contracts")
	defaultsDir := filepath.Join(root, "internal", "defaults", "contracts")

	// All *.schema.json files in .wave/contracts/ must exist and match in
	// internal/defaults/contracts/. .wave/contracts/ is the authoritative source.
	entries, err := os.ReadDir(waveDir)
	require.NoError(t, err, "failed to read .wave/contracts/ directory")

	schemas := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".json" {
			schemas = append(schemas, entry.Name())
		}
	}

	require.NotEmpty(t, schemas, "expected at least one *.schema.json in .wave/contracts/")

	for _, schemaName := range schemas {
		t.Run(schemaName, func(t *testing.T) {
			waveContent, err := os.ReadFile(filepath.Join(waveDir, schemaName))
			require.NoError(t, err, "failed to read .wave/contracts/%s", schemaName)

			defaultsContent, err := os.ReadFile(filepath.Join(defaultsDir, schemaName))
			require.NoError(t, err,
				"schema %s exists in .wave/contracts/ but not in internal/defaults/contracts/ — sync with: cp .wave/contracts/%s internal/defaults/contracts/",
				schemaName, schemaName)

			assert.Equal(t, string(waveContent), string(defaultsContent),
				"schema %s diverged between .wave/contracts/ and internal/defaults/contracts/ — .wave/contracts/ is authoritative, sync with: cp .wave/contracts/%s internal/defaults/contracts/",
				schemaName, schemaName)
		})
	}

	// Reverse check: every schema in internal/defaults/contracts/ must also
	// exist in .wave/contracts/. Prevents schemas added directly to defaults
	// without an authoritative copy in .wave/contracts/.
	defaultsEntries, err := os.ReadDir(defaultsDir)
	require.NoError(t, err, "failed to read internal/defaults/contracts/ directory")

	for _, entry := range defaultsEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		name := entry.Name()
		t.Run("defaults_has_wave_copy/"+name, func(t *testing.T) {
			_, err := os.Stat(filepath.Join(waveDir, name))
			assert.NoError(t, err,
				"schema %s exists in internal/defaults/contracts/ but not in .wave/contracts/ — add the authoritative copy: cp internal/defaults/contracts/%s .wave/contracts/",
				name, name)
		})
	}
}
