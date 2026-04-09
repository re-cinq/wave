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
				"go.mod":     "module github.com/example/go-project\n",
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

			if tt.expected.Name != "" {
				assert.Equal(t, tt.expected.Name, got.Name)
			} else {
				// When no manifest provides a name, directory name fallback kicks in
				assert.NotEmpty(t, got.Name, "directory name fallback should provide a name")
			}
			assert.Equal(t, tt.expected.Description, got.Description)
		})
	}
}

func TestParseComposeServices(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []ComposeService
	}{
		{
			name: "standard compose with build contexts",
			content: `version: "3.8"
services:
  api:
    build: ./api
    ports:
      - "8080:8080"
  web:
    build: ./web
    ports:
      - "3000:3000"
  db:
    image: postgres:15
`,
			expected: []ComposeService{
				{Name: "api", Build: "./api"},
				{Name: "web", Build: "./web"},
				{Name: "db", Image: "postgres:15"},
			},
		},
		{
			name: "compose with nested build config",
			content: `services:
  app:
    build:
      context: ./app
      dockerfile: Dockerfile
`,
			expected: []ComposeService{
				{Name: "app", Build: "."},
			},
		},
		{
			name:     "empty content",
			content:  "",
			expected: nil,
		},
		{
			name: "no services key",
			content: `version: "3"
networks:
  default:
`,
			expected: nil,
		},
		{
			name: "compose with comments",
			content: `# My app
services:
  # Main API server
  api:
    build: ./api
  # Worker process
  worker:
    build: ./worker
`,
			expected: []ComposeService{
				{Name: "api", Build: "./api"},
				{Name: "worker", Build: "./worker"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseComposeServices(tt.content)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParseComposeFile(t *testing.T) {
	t.Run("discovers compose.yml", func(t *testing.T) {
		dir := t.TempDir()
		content := `services:
  api:
    build: ./api
  db:
    image: postgres:15
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "compose.yml"), []byte(content), 0o644))
		services := parseComposeFile(dir)
		require.Len(t, services, 2)
		assert.Equal(t, "api", services[0].Name)
		assert.Equal(t, "db", services[1].Name)
	})

	t.Run("discovers docker-compose.yml", func(t *testing.T) {
		dir := t.TempDir()
		content := `services:
  web:
    build: .
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(content), 0o644))
		services := parseComposeFile(dir)
		require.Len(t, services, 1)
		assert.Equal(t, "web", services[0].Name)
	})

	t.Run("no compose file returns nil", func(t *testing.T) {
		dir := t.TempDir()
		services := parseComposeFile(dir)
		assert.Nil(t, services)
	})
}

func TestScanNestedManifests(t *testing.T) {
	t.Run("finds sub-projects in services dir", func(t *testing.T) {
		dir := t.TempDir()

		// Create services/api with go.mod
		apiDir := filepath.Join(dir, "services", "api")
		require.NoError(t, os.MkdirAll(apiDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module github.com/example/api\n"), 0o644))

		// Create services/web with package.json
		webDir := filepath.Join(dir, "services", "web")
		require.NoError(t, os.MkdirAll(webDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(webDir, "package.json"), []byte(`{"name": "web-app"}`), 0o644))

		subs := scanNestedManifests(dir)
		require.Len(t, subs, 2)

		// Sort by path for deterministic comparison
		byPath := make(map[string]SubProject)
		for _, s := range subs {
			byPath[s.Path] = s
		}

		api := byPath[filepath.Join("services", "api")]
		assert.Equal(t, "api", api.Name)
		assert.Equal(t, "go", api.Language)

		web := byPath[filepath.Join("services", "web")]
		assert.Equal(t, "web-app", web.Name)
		assert.Equal(t, "node", web.Language)
	})

	t.Run("finds sub-projects in packages dir", func(t *testing.T) {
		dir := t.TempDir()

		pkgDir := filepath.Join(dir, "packages", "core")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "Cargo.toml"), []byte("[package]\nname = \"core-lib\"\n"), 0o644))

		subs := scanNestedManifests(dir)
		require.Len(t, subs, 1)
		assert.Equal(t, "core-lib", subs[0].Name)
		assert.Equal(t, "rust", subs[0].Language)
	})

	t.Run("empty directory returns nil", func(t *testing.T) {
		dir := t.TempDir()
		subs := scanNestedManifests(dir)
		assert.Nil(t, subs)
	})

	t.Run("skips non-standard directories", func(t *testing.T) {
		dir := t.TempDir()
		// Create a sub-project outside the standard dirs
		miscDir := filepath.Join(dir, "misc", "tool")
		require.NoError(t, os.MkdirAll(miscDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(miscDir, "go.mod"), []byte("module github.com/example/tool\n"), 0o644))

		subs := scanNestedManifests(dir)
		assert.Nil(t, subs)
	})
}

func TestParseREADME(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		expected string
	}{
		{
			name:     "markdown heading",
			filename: "README.md",
			content:  "# My Cool Project\n\nSome description here.\n",
			expected: "My Cool Project",
		},
		{
			name:     "RST heading with equals underline",
			filename: "README.rst",
			content:  "My RST Project\n===============\n\nDescription.\n",
			expected: "My RST Project",
		},
		{
			name:     "RST heading with dashes underline",
			filename: "README.rst",
			content:  "Dash Project\n------------\n\nMore text.\n",
			expected: "Dash Project",
		},
		{
			name:     "empty README",
			filename: "README.md",
			content:  "",
			expected: "",
		},
		{
			name:     "README with no heading",
			filename: "README.md",
			content:  "Just some text without a heading.\n",
			expected: "",
		},
		{
			name:     "lowercase readme.md",
			filename: "readme.md",
			content:  "# lowercase readme\n",
			expected: "lowercase readme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, tt.filename), []byte(tt.content), 0o644))
			got := parseREADME(dir)
			assert.Equal(t, tt.expected, got)
		})
	}

	t.Run("no README returns empty", func(t *testing.T) {
		dir := t.TempDir()
		got := parseREADME(dir)
		assert.Empty(t, got)
	})
}

func TestExtractProjectMetadata_ComposeAndSubprojects(t *testing.T) {
	t.Run("discovers compose services alongside main manifest", func(t *testing.T) {
		dir := t.TempDir()

		// Main project manifest
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"),
			[]byte("module github.com/example/monorepo\n"), 0o644))

		// Docker compose
		compose := `services:
  api:
    build: ./services/api
  web:
    build: ./services/web
  redis:
    image: redis:7
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "compose.yml"), []byte(compose), 0o644))

		// Nested services
		apiDir := filepath.Join(dir, "services", "api")
		require.NoError(t, os.MkdirAll(apiDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(apiDir, "go.mod"),
			[]byte("module github.com/example/monorepo/api\n"), 0o644))

		meta := ExtractProjectMetadata(dir)
		assert.Equal(t, "monorepo", meta.Name)
		require.Len(t, meta.Services, 3)
		assert.Equal(t, "api", meta.Services[0].Name)
		assert.Equal(t, "web", meta.Services[1].Name)
		assert.Equal(t, "redis", meta.Services[2].Name)
		require.Len(t, meta.SubProjects, 1)
		assert.Equal(t, "api", meta.SubProjects[0].Name)
		assert.Equal(t, "go", meta.SubProjects[0].Language)
	})

	t.Run("README fallback when no manifest", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"),
			[]byte("# My Unnamed Project\n\nNo language manifest here.\n"), 0o644))

		meta := ExtractProjectMetadata(dir)
		assert.Equal(t, "My Unnamed Project", meta.Name)
	})
}
