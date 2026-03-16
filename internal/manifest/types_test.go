package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProjectVars(t *testing.T) {
	t.Run("nil project returns empty map", func(t *testing.T) {
		var p *Project
		vars := p.ProjectVars()
		assert.NotNil(t, vars)
		assert.Empty(t, vars)
	})

	t.Run("all fields set returns all keys", func(t *testing.T) {
		p := &Project{
			Language:      "go",
			Flavour:       "go",
			TestCommand:   "go test ./...",
			LintCommand:   "go vet ./...",
			BuildCommand:  "go build ./...",
			FormatCommand: "gofmt -l .",
			SourceGlob:    "**/*.go",
		}
		vars := p.ProjectVars()

		assert.Equal(t, "go", vars["project.language"])
		assert.Equal(t, "go", vars["project.flavour"])
		assert.Equal(t, "go test ./...", vars["project.test_command"])
		assert.Equal(t, "go vet ./...", vars["project.lint_command"])
		assert.Equal(t, "go build ./...", vars["project.build_command"])
		assert.Equal(t, "gofmt -l .", vars["project.format_command"])
		assert.Equal(t, "**/*.go", vars["project.source_glob"])
		assert.Len(t, vars, 7)
	})

	t.Run("only flavour set returns only flavour key", func(t *testing.T) {
		p := &Project{
			Flavour: "rust",
		}
		vars := p.ProjectVars()

		assert.Equal(t, "rust", vars["project.flavour"])
		assert.Len(t, vars, 1)
	})

	t.Run("empty flavour and format_command omitted from map", func(t *testing.T) {
		p := &Project{
			Language:     "python",
			TestCommand:  "pytest",
			LintCommand:  "ruff check .",
			BuildCommand: "python setup.py build",
			SourceGlob:   "**/*.py",
			// Flavour and FormatCommand intentionally left empty
		}
		vars := p.ProjectVars()

		assert.Equal(t, "python", vars["project.language"])
		assert.Equal(t, "pytest", vars["project.test_command"])
		assert.Equal(t, "ruff check .", vars["project.lint_command"])
		assert.Equal(t, "python setup.py build", vars["project.build_command"])
		assert.Equal(t, "**/*.py", vars["project.source_glob"])

		_, hasFlavour := vars["project.flavour"]
		assert.False(t, hasFlavour, "project.flavour should not be present when empty")

		_, hasFormat := vars["project.format_command"]
		assert.False(t, hasFormat, "project.format_command should not be present when empty")

		assert.Len(t, vars, 5)
	})

	t.Run("YAML round-trip preserves flavour and format_command", func(t *testing.T) {
		original := &Project{
			Language:      "go",
			Flavour:       "go",
			TestCommand:   "go test ./...",
			LintCommand:   "go vet ./...",
			BuildCommand:  "go build ./...",
			FormatCommand: "gofmt -l .",
			SourceGlob:    "**/*.go",
		}

		data, err := yaml.Marshal(original)
		require.NoError(t, err)

		var roundTripped Project
		err = yaml.Unmarshal(data, &roundTripped)
		require.NoError(t, err)

		assert.Equal(t, original.Language, roundTripped.Language)
		assert.Equal(t, original.Flavour, roundTripped.Flavour)
		assert.Equal(t, original.TestCommand, roundTripped.TestCommand)
		assert.Equal(t, original.LintCommand, roundTripped.LintCommand)
		assert.Equal(t, original.BuildCommand, roundTripped.BuildCommand)
		assert.Equal(t, original.FormatCommand, roundTripped.FormatCommand)
		assert.Equal(t, original.SourceGlob, roundTripped.SourceGlob)
	})
}
