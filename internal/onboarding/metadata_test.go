package onboarding

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractProjectMetadata(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected ProjectMetadata
	}{
		// go.mod extraction
		{
			name: "go.mod with full module path returns last segment as name",
			files: map[string]string{
				"go.mod": "module github.com/example/myproject\n\ngo 1.21\n",
			},
			expected: ProjectMetadata{Name: "myproject"},
		},
		{
			name: "go.mod with simple module name returns name directly",
			files: map[string]string{
				"go.mod": "module myapp\n\ngo 1.21\n",
			},
			expected: ProjectMetadata{Name: "myapp"},
		},
		{
			name: "go.mod with deep module path returns last segment",
			files: map[string]string{
				"go.mod": "module github.com/org/subgroup/my-tool\n",
			},
			expected: ProjectMetadata{Name: "my-tool"},
		},

		// Cargo.toml extraction
		{
			name: "Cargo.toml with package section returns crate name",
			files: map[string]string{
				"Cargo.toml": "[package]\nname = \"my-crate\"\nversion = \"0.1.0\"\n",
			},
			expected: ProjectMetadata{Name: "my-crate"},
		},
		{
			name: "Cargo.toml with package section and other sections stops at next section",
			files: map[string]string{
				"Cargo.toml": "[package]\nname = \"crate-name\"\n\n[dependencies]\nserde = \"1\"\n",
			},
			expected: ProjectMetadata{Name: "crate-name"},
		},

		// package.json extraction
		{
			name: "package.json with name and description returns both fields",
			files: map[string]string{
				"package.json": `{"name": "my-package", "description": "A cool package"}`,
			},
			expected: ProjectMetadata{Name: "my-package", Description: "A cool package"},
		},
		{
			name: "package.json with name only returns name with empty description",
			files: map[string]string{
				"package.json": `{"name": "my-package"}`,
			},
			expected: ProjectMetadata{Name: "my-package"},
		},

		// pyproject.toml extraction
		{
			name: "pyproject.toml with project section returns name",
			files: map[string]string{
				"pyproject.toml": "[project]\nname = \"my-project\"\nversion = \"1.0.0\"\n",
			},
			expected: ProjectMetadata{Name: "my-project"},
		},
		{
			name: "pyproject.toml with multiple sections returns name from project section",
			files: map[string]string{
				"pyproject.toml": "[build-system]\nrequires = [\"setuptools\"]\n\n[project]\nname = \"my-project\"\n",
			},
			expected: ProjectMetadata{Name: "my-project"},
		},

		// Empty directory
		{
			name:     "empty directory returns empty ProjectMetadata",
			files:    map[string]string{},
			expected: ProjectMetadata{},
		},

		// Malformed files — graceful degradation
		{
			name: "malformed package.json returns empty ProjectMetadata",
			files: map[string]string{
				"package.json": `{invalid json`,
			},
			expected: ProjectMetadata{},
		},
		{
			name: "go.mod with no module line returns empty ProjectMetadata",
			files: map[string]string{
				"go.mod": "go 1.21\n",
			},
			expected: ProjectMetadata{},
		},
		{
			name: "Cargo.toml with no package section returns empty ProjectMetadata",
			files: map[string]string{
				"Cargo.toml": "[dependencies]\nserde = \"1\"\n",
			},
			expected: ProjectMetadata{},
		},
		{
			name: "pyproject.toml with no project section returns empty ProjectMetadata",
			files: map[string]string{
				"pyproject.toml": "[build-system]\nrequires = [\"setuptools\"]\n",
			},
			expected: ProjectMetadata{},
		},

		// Priority: go.mod takes precedence over package.json
		{
			name: "go.mod present alongside package.json returns go.mod result",
			files: map[string]string{
				"go.mod":       "module github.com/example/go-project\n",
				"package.json": `{"name": "js-project", "description": "A JS project"}`,
			},
			expected: ProjectMetadata{Name: "go-project"},
		},
		// Priority: go.mod takes precedence over Cargo.toml
		{
			name: "go.mod present alongside Cargo.toml returns go.mod result",
			files: map[string]string{
				"go.mod":    "module github.com/example/go-project\n",
				"Cargo.toml": "[package]\nname = \"rust-crate\"\n",
			},
			expected: ProjectMetadata{Name: "go-project"},
		},
		// Priority: Cargo.toml takes precedence over package.json (checked before package.json)
		{
			name: "Cargo.toml present alongside package.json returns Cargo.toml result",
			files: map[string]string{
				"Cargo.toml":   "[package]\nname = \"my-crate\"\n",
				"package.json": `{"name": "my-package"}`,
			},
			expected: ProjectMetadata{Name: "my-crate"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			for filename, content := range tt.files {
				path := filepath.Join(dir, filename)
				err := os.WriteFile(path, []byte(content), 0o644)
				require.NoError(t, err, "failed to write test file %s", filename)
			}

			got := ExtractProjectMetadata(dir)

			assert.Equal(t, tt.expected, got)
		})
	}
}
