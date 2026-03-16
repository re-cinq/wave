package flavour

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetect_AllFlavours verifies that each supported language/toolchain is
// correctly identified from its marker files.
func TestDetect_AllFlavours(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string // filename -> content (empty string = touch)
		expected DetectionResult
	}{
		// --- Go ---
		{
			name:  "go",
			files: map[string]string{"go.mod": "module example.com/foo\n\ngo 1.21\n"},
			expected: DetectionResult{
				Flavour: "go", Language: "go",
				TestCommand: "go test ./...", LintCommand: "go vet ./...",
				BuildCommand: "go build ./...", FormatCommand: "gofmt -l .",
				SourceGlob: "*.go",
			},
		},
		// --- Rust ---
		{
			name:  "rust",
			files: map[string]string{"Cargo.toml": "[package]\nname = \"myapp\"\n"},
			expected: DetectionResult{
				Flavour: "rust", Language: "rust",
				TestCommand: "cargo test", LintCommand: "cargo clippy",
				BuildCommand: "cargo build", FormatCommand: "cargo fmt -- --check",
				SourceGlob: "*.rs",
			},
		},
		// --- Deno (deno.json) ---
		{
			name:  "deno_json",
			files: map[string]string{"deno.json": "{}"},
			expected: DetectionResult{
				Flavour: "deno", Language: "typescript",
				TestCommand: "deno test", LintCommand: "deno lint",
				BuildCommand: "deno compile", FormatCommand: "deno fmt --check",
				SourceGlob: "*.{ts,tsx}",
			},
		},
		// --- Deno (deno.jsonc) ---
		{
			name:  "deno_jsonc",
			files: map[string]string{"deno.jsonc": "{}"},
			expected: DetectionResult{
				Flavour: "deno", Language: "typescript",
				TestCommand: "deno test", LintCommand: "deno lint",
				BuildCommand: "deno compile", FormatCommand: "deno fmt --check",
				SourceGlob: "*.{ts,tsx}",
			},
		},
		// --- Bun (bun.lockb) ---
		{
			name:  "bun_lockb",
			files: map[string]string{"bun.lockb": "", "package.json": `{"name":"x"}`},
			expected: DetectionResult{
				Flavour: "bun", Language: "typescript",
				TestCommand: "bun test", LintCommand: "bun run lint",
				BuildCommand: "bun run build", FormatCommand: "",
				SourceGlob: "*.{ts,tsx}",
			},
		},
		// --- Bun (bun.lock) ---
		{
			name:  "bun_lock",
			files: map[string]string{"bun.lock": "", "package.json": `{"name":"x"}`},
			expected: DetectionResult{
				Flavour: "bun", Language: "typescript",
				TestCommand: "bun test", LintCommand: "bun run lint",
				BuildCommand: "bun run build", FormatCommand: "",
				SourceGlob: "*.{ts,tsx}",
			},
		},
		// --- pnpm ---
		{
			name:  "pnpm",
			files: map[string]string{"pnpm-lock.yaml": "", "package.json": `{"name":"x"}`},
			expected: DetectionResult{
				Flavour: "pnpm", Language: "javascript",
				TestCommand: "pnpm test", LintCommand: "pnpm run lint",
				BuildCommand: "pnpm run build", FormatCommand: "",
				SourceGlob: "*.{js,jsx}",
			},
		},
		// --- Yarn ---
		{
			name:  "yarn",
			files: map[string]string{"yarn.lock": "", "package.json": `{"name":"x"}`},
			expected: DetectionResult{
				Flavour: "yarn", Language: "javascript",
				TestCommand: "yarn test", LintCommand: "yarn run lint",
				BuildCommand: "yarn run build", FormatCommand: "",
				SourceGlob: "*.{js,jsx}",
			},
		},
		// --- Node (generic npm — package.json only, no lockfile) ---
		{
			name:  "node",
			files: map[string]string{"package.json": `{"name":"x"}`},
			expected: DetectionResult{
				Flavour: "node", Language: "javascript",
				TestCommand: "npm test", LintCommand: "npm run lint",
				BuildCommand: "npm run build", FormatCommand: "",
				SourceGlob: "*.{js,jsx}",
			},
		},
		// --- Python (modern — pyproject.toml) ---
		{
			name:  "python_modern",
			files: map[string]string{"pyproject.toml": "[project]\nname = \"mylib\"\n"},
			expected: DetectionResult{
				Flavour: "python", Language: "python",
				TestCommand: "pytest", LintCommand: "ruff check .",
				BuildCommand: "", FormatCommand: "ruff format --check .",
				SourceGlob: "*.py",
			},
		},
		// --- Python (legacy — setup.py) ---
		{
			name:  "python_legacy",
			files: map[string]string{"setup.py": "from setuptools import setup\nsetup()\n"},
			expected: DetectionResult{
				Flavour: "python", Language: "python",
				TestCommand: "pytest", LintCommand: "ruff check .",
				BuildCommand: "python setup.py build", FormatCommand: "ruff format --check .",
				SourceGlob: "*.py",
			},
		},
		// --- .NET (csproj) ---
		{
			name:  "dotnet",
			files: map[string]string{"MyProject.csproj": "<Project></Project>"},
			expected: DetectionResult{
				Flavour: "dotnet", Language: "csharp",
				TestCommand: "dotnet test", LintCommand: "dotnet format --verify-no-changes",
				BuildCommand: "dotnet build", FormatCommand: "dotnet format --verify-no-changes",
				SourceGlob: "*.cs",
			},
		},
		// --- Maven ---
		{
			name:  "maven",
			files: map[string]string{"pom.xml": "<project></project>"},
			expected: DetectionResult{
				Flavour: "maven", Language: "java",
				TestCommand: "mvn test", LintCommand: "mvn checkstyle:check",
				BuildCommand: "mvn package", FormatCommand: "",
				SourceGlob: "*.java",
			},
		},
		// --- Gradle ---
		{
			name:  "gradle",
			files: map[string]string{"build.gradle": "plugins { id 'java' }"},
			expected: DetectionResult{
				Flavour: "gradle", Language: "java",
				TestCommand: "./gradlew test", LintCommand: "./gradlew check",
				BuildCommand: "./gradlew build", FormatCommand: "",
				SourceGlob: "*.java",
			},
		},
		// --- Elixir ---
		{
			name:  "elixir",
			files: map[string]string{"mix.exs": "defmodule MyApp.MixProject do\nend\n"},
			expected: DetectionResult{
				Flavour: "elixir", Language: "elixir",
				TestCommand: "mix test", LintCommand: "mix credo",
				BuildCommand: "mix compile", FormatCommand: "mix format --check-formatted",
				SourceGlob: "*.{ex,exs}",
			},
		},
		// --- Dart ---
		{
			name:  "dart",
			files: map[string]string{"pubspec.yaml": "name: myapp\n"},
			expected: DetectionResult{
				Flavour: "dart", Language: "dart",
				TestCommand: "dart test", LintCommand: "dart analyze",
				BuildCommand: "dart compile exe", FormatCommand: "dart format --set-exit-if-changed .",
				SourceGlob: "*.dart",
			},
		},
		// --- CMake ---
		{
			name:  "cmake",
			files: map[string]string{"CMakeLists.txt": "cmake_minimum_required(VERSION 3.20)\n"},
			expected: DetectionResult{
				Flavour: "cmake", Language: "cpp",
				TestCommand: "ctest", LintCommand: "",
				BuildCommand: "cmake --build build", FormatCommand: "",
				SourceGlob: "*.{cpp,hpp,cc,h}",
			},
		},
		// --- Make ---
		{
			name:  "make",
			files: map[string]string{"Makefile": "all:\n\techo hello\n"},
			expected: DetectionResult{
				Flavour: "make", Language: "",
				TestCommand: "make test", LintCommand: "make lint",
				BuildCommand: "make build", FormatCommand: "",
				SourceGlob: "",
			},
		},
		// --- PHP ---
		{
			name:  "php",
			files: map[string]string{"composer.json": `{"name":"vendor/pkg"}`},
			expected: DetectionResult{
				Flavour: "php", Language: "php",
				TestCommand: "vendor/bin/phpunit", LintCommand: "vendor/bin/phpstan analyse",
				BuildCommand: "", FormatCommand: "vendor/bin/php-cs-fixer fix --dry-run",
				SourceGlob: "*.php",
			},
		},
		// --- Ruby ---
		{
			name:  "ruby",
			files: map[string]string{"Gemfile": "source 'https://rubygems.org'\n"},
			expected: DetectionResult{
				Flavour: "ruby", Language: "ruby",
				TestCommand: "bundle exec rspec", LintCommand: "bundle exec rubocop",
				BuildCommand: "", FormatCommand: "bundle exec rubocop -A --fail-level error",
				SourceGlob: "*.rb",
			},
		},
		// --- Swift ---
		{
			name:  "swift",
			files: map[string]string{"Package.swift": "// swift-tools-version:5.9\n"},
			expected: DetectionResult{
				Flavour: "swift", Language: "swift",
				TestCommand: "swift test", LintCommand: "swift package diagnose-api-breaking-changes",
				BuildCommand: "swift build", FormatCommand: "swift-format lint -r .",
				SourceGlob: "*.swift",
			},
		},
		// --- Zig ---
		{
			name:  "zig",
			files: map[string]string{"build.zig": "const std = @import(\"std\");\n"},
			expected: DetectionResult{
				Flavour: "zig", Language: "zig",
				TestCommand: "zig build test", LintCommand: "",
				BuildCommand: "zig build", FormatCommand: "zig fmt --check .",
				SourceGlob: "*.zig",
			},
		},
		// --- Scala ---
		{
			name:  "scala",
			files: map[string]string{"build.sbt": "name := \"myapp\"\n"},
			expected: DetectionResult{
				Flavour: "scala", Language: "scala",
				TestCommand: "sbt test", LintCommand: "sbt scalafmtCheck",
				BuildCommand: "sbt compile", FormatCommand: "sbt scalafmt",
				SourceGlob: "*.scala",
			},
		},
		// --- Haskell (Cabal — glob marker *.cabal) ---
		{
			name:  "cabal",
			files: map[string]string{"myproject.cabal": "name: myproject\n"},
			expected: DetectionResult{
				Flavour: "cabal", Language: "haskell",
				TestCommand: "cabal test", LintCommand: "hlint .",
				BuildCommand: "cabal build", FormatCommand: "",
				SourceGlob: "*.hs",
			},
		},
		// --- Haskell (Stack) ---
		{
			name:  "stack",
			files: map[string]string{"stack.yaml": "resolver: lts-21.0\n"},
			expected: DetectionResult{
				Flavour: "stack", Language: "haskell",
				TestCommand: "stack test", LintCommand: "hlint .",
				BuildCommand: "stack build", FormatCommand: "",
				SourceGlob: "*.hs",
			},
		},
		// --- TypeScript standalone (tsconfig.json only, no package.json) ---
		{
			name:  "typescript_standalone",
			files: map[string]string{"tsconfig.json": `{"compilerOptions":{}}`},
			expected: DetectionResult{
				Flavour: "typescript", Language: "typescript",
				TestCommand: "", LintCommand: "tsc --noEmit",
				BuildCommand: "tsc", FormatCommand: "",
				SourceGlob: "*.{ts,tsx}",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, tc.files)

			result := Detect(dir)
			assert.Equal(t, tc.expected.Flavour, result.Flavour, "Flavour mismatch")
			assert.Equal(t, tc.expected.Language, result.Language, "Language mismatch")
			assert.Equal(t, tc.expected.TestCommand, result.TestCommand, "TestCommand mismatch")
			assert.Equal(t, tc.expected.LintCommand, result.LintCommand, "LintCommand mismatch")
			assert.Equal(t, tc.expected.BuildCommand, result.BuildCommand, "BuildCommand mismatch")
			assert.Equal(t, tc.expected.FormatCommand, result.FormatCommand, "FormatCommand mismatch")
			assert.Equal(t, tc.expected.SourceGlob, result.SourceGlob, "SourceGlob mismatch")
		})
	}
}

// TestDetect_SpecificityOrdering verifies that more specific markers are
// matched before generic ones when multiple marker files are present.
func TestDetect_SpecificityOrdering(t *testing.T) {
	tests := []struct {
		name            string
		files           map[string]string
		expectedFlavour string
	}{
		{
			name:            "bun.lock beats package.json",
			files:           map[string]string{"bun.lock": "", "package.json": `{"name":"x"}`},
			expectedFlavour: "bun",
		},
		{
			name:            "pnpm-lock.yaml beats package.json",
			files:           map[string]string{"pnpm-lock.yaml": "", "package.json": `{"name":"x"}`},
			expectedFlavour: "pnpm",
		},
		{
			name:            "yarn.lock beats package.json",
			files:           map[string]string{"yarn.lock": "", "package.json": `{"name":"x"}`},
			expectedFlavour: "yarn",
		},
		{
			name:            "deno.json beats package.json",
			files:           map[string]string{"deno.json": "{}", "package.json": `{"name":"x"}`},
			expectedFlavour: "deno",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, tc.files)

			result := Detect(dir)
			assert.Equal(t, tc.expectedFlavour, result.Flavour,
				"expected %s to take priority", tc.expectedFlavour)
		})
	}
}

// TestDetect_NoMatch verifies that an empty directory yields a zero-value
// DetectionResult.
func TestDetect_NoMatch(t *testing.T) {
	dir := t.TempDir()

	result := Detect(dir)
	assert.Equal(t, DetectionResult{}, result, "expected zero-value result for empty directory")
}

// TestDetect_NodeWithTSConfig verifies that a Node-based flavour with a
// tsconfig.json file refines the language to "typescript" and the source glob
// to "*.{ts,tsx}".
func TestDetect_NodeWithTSConfig(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"package.json":  `{"name":"x"}`,
		"tsconfig.json": `{"compilerOptions":{}}`,
	})

	result := Detect(dir)
	assert.Equal(t, "node", result.Flavour, "flavour should still be node")
	assert.Equal(t, "typescript", result.Language, "language should be refined to typescript")
	assert.Equal(t, "*.{ts,tsx}", result.SourceGlob, "source glob should be refined for typescript")
}

// TestDetect_NodeWithPackageJSONScripts verifies that when package.json
// contains scripts, the detected commands are derived from those scripts.
func TestDetect_NodeWithPackageJSONScripts(t *testing.T) {
	dir := t.TempDir()

	pkgJSON := map[string]interface{}{
		"name": "my-app",
		"scripts": map[string]interface{}{
			"test":  "jest",
			"lint":  "eslint .",
			"build": "tsc",
		},
	}
	data, err := json.Marshal(pkgJSON)
	require.NoError(t, err)

	writeFiles(t, dir, map[string]string{
		"package.json": string(data),
	})

	result := Detect(dir)
	assert.Equal(t, "node", result.Flavour)
	// When scripts are present, commands should be derived from them.
	assert.Equal(t, "npm test", result.TestCommand)
	assert.Equal(t, "npm run lint", result.LintCommand)
	assert.Equal(t, "npm run build", result.BuildCommand)
}

// TestDetectMetadata verifies metadata extraction from language-specific
// manifest files.
func TestDetectMetadata(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		dirName  string // if set, the temp dir will be created with this name pattern
		expected MetadataResult
	}{
		{
			name: "go.mod extracts name from module path",
			files: map[string]string{
				"go.mod": "module github.com/org/myrepo\n\ngo 1.21\n",
			},
			expected: MetadataResult{
				Name: "myrepo",
			},
		},
		{
			name: "package.json extracts name and description",
			files: map[string]string{
				"package.json": `{"name":"my-app","description":"A cool application"}`,
			},
			expected: MetadataResult{
				Name:        "my-app",
				Description: "A cool application",
			},
		},
		{
			name: "Cargo.toml extracts name from package section",
			files: map[string]string{
				"Cargo.toml": "[package]\nname = \"my-crate\"\nversion = \"0.1.0\"\n",
			},
			expected: MetadataResult{
				Name: "my-crate",
			},
		},
		{
			name: "pyproject.toml extracts name from project section",
			files: map[string]string{
				"pyproject.toml": "[project]\nname = \"my-lib\"\ndescription = \"A Python library\"\n",
			},
			expected: MetadataResult{
				Name: "my-lib",
			},
		},
		{
			name:  "no manifest falls back to directory name",
			files: map[string]string{},
			expected: MetadataResult{
				Name: "", // Will be checked separately since dir name is dynamic
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, tc.files)

			result := DetectMetadata(dir)

			if tc.name == "no manifest falls back to directory name" {
				// The fallback uses the directory basename, which is the
				// temp dir name. Just verify it is non-empty.
				assert.NotEmpty(t, result.Name,
					"expected directory name as fallback")
				assert.Equal(t, filepath.Base(dir), result.Name,
					"expected fallback to match directory basename")
			} else {
				assert.Equal(t, tc.expected.Name, result.Name, "Name mismatch")
				if tc.expected.Description != "" {
					assert.Equal(t, tc.expected.Description, result.Description,
						"Description mismatch")
				}
			}
		})
	}
}

// writeFiles creates the given files in dir. Each key is a filename (no
// subdirectories), each value is the file content.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, name)
		err := os.WriteFile(path, []byte(content), 0o644)
		require.NoError(t, err, "failed to write %s", name)
	}
}
