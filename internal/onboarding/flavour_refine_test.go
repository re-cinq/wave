package onboarding

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// Commands come from package.json scripts; scripts that do not exist are not
// invented (regression: `npm run format` was written without a format script).
func TestDetectFlavour_NodeUsesPackageScripts(t *testing.T) {
	tests := []struct {
		name                          string
		lockfile                      string
		scripts                       string
		test, lint, build, formatting string
	}{
		{"npm only test/build/lint", "", `{"test":"node --test","build":"tsc","lint":"eslint ."}`, "npm test", "npm run lint", "npm run build", ""},
		{"npm no scripts", "", `{}`, "", "", "", ""},
		{"yarn format:check preferred", "yarn.lock", `{"format":"prettier -w .","format:check":"prettier -c ."}`, "", "", "", "yarn format:check"},
		{"pnpm test", "pnpm-lock.yaml", `{"test":"vitest"}`, "pnpm test", "", "", ""},
		{"bun without test script uses bun test", "bun.lock", `{"build":"bun build"}`, "bun test", "", "bun run build", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x","scripts":`+tt.scripts+`}`)
			if tt.lockfile != "" {
				writeFile(t, filepath.Join(dir, tt.lockfile), "")
			}
			fi := DetectFlavour(dir)
			require.NotNil(t, fi)
			assert.Equal(t, tt.test, fi.TestCommand, "test")
			assert.Equal(t, tt.lint, fi.LintCommand, "lint")
			assert.Equal(t, tt.build, fi.BuildCommand, "build")
			assert.Equal(t, tt.formatting, fi.FormatCommand, "format")
		})
	}
}

// ruff commands are only written when the project configures ruff.
func TestDetectFlavour_PythonRuffOnlyWhenConfigured(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"p\"\n")
	fi := DetectFlavour(dir)
	require.NotNil(t, fi)
	assert.Equal(t, "pytest", fi.TestCommand)
	assert.Empty(t, fi.LintCommand)
	assert.Empty(t, fi.FormatCommand)

	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"p\"\n\n[tool.ruff]\nline-length = 100\n")
	fi = DetectFlavour(dir)
	require.NotNil(t, fi)
	assert.Equal(t, "ruff check .", fi.LintCommand)
	assert.Equal(t, "ruff format --check .", fi.FormatCommand)
}
